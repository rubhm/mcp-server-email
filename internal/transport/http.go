package transport

import (
	"bytes"
	"context"
	"crypto/sha256"
	"crypto/subtle"
	"encoding/json"
	"errors"
	"io"
	"log"
	"maps"
	"mime"
	"net"
	"net/http"
	"strconv"
	"strings"
	"time"

	"github.com/mark3labs/mcp-go/server"
)

const (
	MaxRequestBytes   = 1 << 20
	maxHeaderBytes    = 16 << 10
	requestTimeout    = 30 * time.Second
	shutdownTimeout   = 5 * time.Second
	readHeaderTimeout = 5 * time.Second
	readTimeout       = 15 * time.Second
	writeTimeout      = 60 * time.Second
	idleTimeout       = 60 * time.Second
)

var errHTTPServer = errors.New("HTTP server failed")

// NewHTTPHandler authenticates before entering any MCP library handler.
func NewHTTPHandler(cfg *Config, mcpServer *server.MCPServer) http.Handler {
	protocol := server.NewStreamableHTTPServer(mcpServer,
		server.WithStateLess(true), server.WithDisableStreaming(true),
		server.WithLogger(discardLogger{}),
	)

	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if !allowedRequest(cfg, r) {
			http.Error(w, "forbidden", http.StatusForbidden)

			return
		}

		if r.URL.RawPath != "" && r.URL.RawPath != r.URL.Path {
			http.Error(w, "not_found", http.StatusNotFound)

			return
		}

		if r.URL.RawQuery != "" || r.URL.ForceQuery {
			http.Error(w, "invalid_request", http.StatusBadRequest)

			return
		}

		switch r.URL.Path {
		case "/healthz":
			serveHealth(w, r)
		case "/mcp":
			serveAuthenticated(cfg, protocol, w, r)
		default:
			http.Error(w, "not_found", http.StatusNotFound)
		}
	})
}

type discardLogger struct{}

func (discardLogger) Infof(string, ...any)  {}
func (discardLogger) Errorf(string, ...any) {}

func allowedRequest(cfg *Config, r *http.Request) bool {
	_, hasOrigin := r.Header["Origin"]
	host, err := authority(r.Host)
	_, allowed := cfg.hosts[host]

	return err == nil && allowed && !hasOrigin
}

func serveHealth(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodGet {
		w.Header().Set("Allow", http.MethodGet)
		http.Error(w, "method_not_allowed", http.StatusMethodNotAllowed)

		return
	}

	w.Header().Set("Content-Type", "application/json")
	_, _ = io.WriteString(w, `{"status":"ok"}`)
}

func validAuthorization(cfg *Config, headers http.Header) bool {
	values := headers.Values("Authorization")
	if cfg.token == "" || len(values) != 1 {
		return false
	}

	scheme, token, ok := strings.Cut(values[0], " ")
	if !ok || !strings.EqualFold(scheme, "Bearer") {
		return false
	}

	provided := sha256.Sum256([]byte(token))
	expected := sha256.Sum256([]byte(cfg.token))

	return subtle.ConstantTimeCompare(provided[:], expected[:]) == 1
}

func serveAuthenticated(cfg *Config, protocol http.Handler, w http.ResponseWriter, r *http.Request) {
	if !validAuthorization(cfg, r.Header) {
		w.Header().Set("WWW-Authenticate", "Bearer")
		http.Error(w, "unauthorized", http.StatusUnauthorized)

		return
	}

	ctx, cancel := context.WithTimeout(authenticatedContext(r.Context(), HTTP), requestTimeout)
	defer cancel()

	r = r.WithContext(ctx)
	if r.Method == http.MethodPost && !prepareBody(w, r) {
		return
	}
	// Stateless P0 has no client-owned session state. Do not pass fabricated IDs
	// into the library's session stores or diagnostic logging.
	r.Header.Del("Mcp-Session-Id")

	response := &protocolResponse{header: make(http.Header)}
	protocol.ServeHTTP(response, r)

	if ctx.Err() != nil {
		http.Error(w, "timeout", http.StatusGatewayTimeout)

		return
	}

	response.send(w)
}

// Buffer the non-streaming response so library RPC errors cannot echo provider
// details or caller-controlled tool names. Successful mail content is unchanged.
type protocolResponse struct {
	header http.Header
	status int
	body   bytes.Buffer
}

func (r *protocolResponse) Header() http.Header { return r.header }
func (r *protocolResponse) WriteHeader(status int) {
	if r.status == 0 {
		r.status = status
	}
}
func (r *protocolResponse) Write(data []byte) (int, error) {
	r.WriteHeader(http.StatusOK)
	_, _ = r.body.Write(data)

	return len(data), nil
}
func (r *protocolResponse) send(w http.ResponseWriter) {
	if r.status == 0 {
		r.status = http.StatusOK
	}

	if r.status >= http.StatusBadRequest {
		http.Error(w, "invalid_request", r.status)

		return
	}

	data := r.body.Bytes()

	var message map[string]json.RawMessage
	if json.Unmarshal(data, &message) == nil && message["error"] != nil {
		var rpcError struct {
			Code int `json:"code"`
		}

		_ = json.Unmarshal(message["error"], &rpcError)
		message["error"] = json.RawMessage(`{"code":` + strconv.Itoa(rpcError.Code) + `,"message":"request_failed"}`)

		var err error

		data, err = json.Marshal(message)
		if err != nil {
			http.Error(w, "request_failed", http.StatusInternalServerError)

			return
		}
	}

	maps.Copy(w.Header(), r.header)

	w.WriteHeader(r.status)
	_, _ = w.Write(data)
}

func prepareBody(w http.ResponseWriter, r *http.Request) bool {
	mediaType, _, err := mime.ParseMediaType(r.Header.Get("Content-Type"))
	if err != nil || mediaType != "application/json" {
		http.Error(w, "invalid_content_type", http.StatusBadRequest)

		return false
	}

	data, err := io.ReadAll(http.MaxBytesReader(w, r.Body, MaxRequestBytes))
	if err != nil {
		var tooLarge *http.MaxBytesError
		if errors.As(err, &tooLarge) {
			http.Error(w, "request_too_large", http.StatusRequestEntityTooLarge)
		} else {
			http.Error(w, "invalid_request", http.StatusBadRequest)
		}

		return false
	}

	var envelope struct {
		JSONRPC string `json:"jsonrpc"`
		Method  string `json:"method"`
	}
	if json.Unmarshal(data, &envelope) != nil || envelope.JSONRPC != "2.0" || envelope.Method == "" {
		http.Error(w, "invalid_request", http.StatusBadRequest)

		return false
	}

	r.Body = io.NopCloser(bytes.NewReader(data))

	return true
}

// NewHTTPServer fixes timeouts independently of MCP library defaults.
func NewHTTPServer(cfg *Config, mcpServer *server.MCPServer) *http.Server {
	return &http.Server{
		Addr:              cfg.Address,
		Handler:           NewHTTPHandler(cfg, mcpServer),
		ReadHeaderTimeout: readHeaderTimeout,
		ReadTimeout:       readTimeout,
		WriteTimeout:      writeTimeout,
		IdleTimeout:       idleTimeout,
		MaxHeaderBytes:    maxHeaderBytes,
		ErrorLog:          log.New(io.Discard, "", 0),
	}
}

// ServeHTTP stops admission, cancels requests, and waits for bounded HTTP cleanup.
// The caller owns pool cleanup after this returns.
func ServeHTTP(ctx context.Context, srv *http.Server, listener net.Listener) error {
	requestCtx, cancel := context.WithCancel(ctx)
	defer cancel()

	srv.BaseContext = func(net.Listener) context.Context { return requestCtx }
	done := make(chan struct{})
	stopped := make(chan struct{})

	go func() {
		defer close(stopped)

		select {
		case <-ctx.Done():
			cancel()

			shutdownCtx, shutdownCancel := context.WithTimeout(context.WithoutCancel(ctx), shutdownTimeout)
			defer shutdownCancel()

			if srv.Shutdown(shutdownCtx) != nil {
				_ = srv.Close()
			}
		case <-done:
		}
	}()

	err := srv.Serve(listener)

	close(done)
	<-stopped

	if errors.Is(err, http.ErrServerClosed) {
		return nil
	}

	return errHTTPServer
}
