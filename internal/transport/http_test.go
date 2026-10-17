package transport //nolint:testpackage // private guard configuration is intentionally unavailable to callers

import (
	"bytes"
	"context"
	"errors"
	"io"
	"log/slog"
	"maps"
	"net"
	"net/http"
	"net/http/httptest"
	"strings"
	"sync"
	"sync/atomic"
	"testing"
	"time"

	"github.com/mark3labs/mcp-go/client"
	clienttransport "github.com/mark3labs/mcp-go/client/transport"
	"github.com/mark3labs/mcp-go/mcp"
	"github.com/mark3labs/mcp-go/server"
)

var errTestCanary = errors.New(testToken)

func fixtureServer(calls *atomic.Int64) *server.MCPServer {
	s := server.NewMCPServer("fixture", "1", server.WithToolCapabilities(false),
		server.WithResourceCapabilities(false, false), server.WithPaginationLimit(1),
		server.WithToolHandlerMiddleware(SafeToolErrors), server.WithResourceHandlerMiddleware(SafeResourceErrors))
	s.AddTool(mcp.NewTool("identity"), func(ctx context.Context, _ mcp.CallToolRequest) (*mcp.CallToolResult, error) {
		calls.Add(1)

		p, ok := PrincipalFromContext(ctx)
		if !ok {
			return mcp.NewToolResultError("no principal"), nil
		}

		return mcp.NewToolResultText(p.ID + ":" + p.Transport), nil
	})
	s.AddTool(mcp.NewTool("failure"), func(context.Context, mcp.CallToolRequest) (*mcp.CallToolResult, error) {
		return mcp.NewToolResultError(testToken + " private-email-body"), nil
	})
	s.AddResource(mcp.NewResource("fixture://status", "status"),
		func(ctx context.Context, _ mcp.ReadResourceRequest) ([]mcp.ResourceContents, error) {
			calls.Add(1)

			p, _ := PrincipalFromContext(ctx)

			return []mcp.ResourceContents{mcp.TextResourceContents{URI: "fixture://status", Text: p.ID + ":" + p.Transport}}, nil
		})

	return s
}

func request(
	t *testing.T, handler http.Handler, method, path, body string, headers http.Header,
) *httptest.ResponseRecorder {
	t.Helper()

	r := httptest.NewRequestWithContext(
		context.Background(), method, "http://localhost:8080"+path, strings.NewReader(body),
	)
	r.Header.Set("Content-Type", "application/json")

	maps.Copy(r.Header, headers)

	w := httptest.NewRecorder()
	handler.ServeHTTP(w, r)

	return w
}

func TestAuthenticationBeforeDispatch(t *testing.T) {
	t.Parallel()

	var calls atomic.Int64

	h := NewHTTPHandler(configForTest(t), fixtureServer(&calls))

	bodies := []string{
		`{"jsonrpc":"2.0","id":1,"method":"initialize","params":` +
			`{"protocolVersion":"2025-03-26","capabilities":{},"clientInfo":{"name":"test","version":"1"}}}`,
		`{"jsonrpc":"2.0","id":2,"method":"tools/list"}`,
		`{"jsonrpc":"2.0","id":3,"method":"tools/call","params":{"name":"identity"}}`,
		`{"jsonrpc":"2.0","id":4,"method":"resources/list"}`,
		`{"jsonrpc":"2.0","id":5,"method":"resources/read","params":{"uri":"fixture://status"}}`,
	}
	for _, body := range bodies {
		for _, authorization := range [][]string{
			nil, {"Bearer wrong"}, {"Bearer " + testToken, "Bearer " + testToken},
			{"Bearer " + testToken + ", Bearer wrong"},
		} {
			w := request(t, h, http.MethodPost, "/mcp", body, http.Header{
				"Authorization": authorization, "X-User-Id": {"owner"}, "Mcp-Session-Id": {"forged"},
			})
			if w.Code != http.StatusUnauthorized || w.Header().Get("WWW-Authenticate") != "Bearer" {
				t.Fatalf("authentication: %d %s", w.Code, w.Body)
			}
		}
	}

	if calls.Load() != 0 {
		t.Fatal("unauthenticated dispatch")
	}

	for _, body := range bodies {
		w := request(t, h, http.MethodPost, "/mcp", body, http.Header{
			"Authorization": {"Bearer " + testToken}, "X-User-Id": {"attacker"},
		})
		if w.Code != http.StatusOK || strings.Contains(w.Body.String(), `"error"`) {
			t.Fatalf("authenticated protocol: %d %s", w.Code, w.Body)
		}

		if w.Header().Get("Mcp-Session-Id") != "" {
			t.Fatal("stateless initialize emitted session")
		}
	}

	if calls.Load() != 2 {
		t.Fatalf("calls=%d", calls.Load())
	}
}

func TestHTTPGuards(t *testing.T) {
	t.Parallel()

	var calls atomic.Int64

	h := NewHTTPHandler(configForTest(t), fixtureServer(&calls))

	tests := []struct {
		name, method, path, body string
		headers                  http.Header
		status                   int
	}{
		{"health", "GET", "/healthz", "", nil, 200},
		{"health method", "POST", "/healthz", "", nil, 405},
		{"path", "GET", "/other", "", nil, 404},
		{"no alias", "GET", "/mcp/", "", nil, 404},
		{"encoded alias", "GET", "/m%63p", "", nil, 404},
		{"query not credentials", "GET", "/mcp?token=" + testToken, "", nil, 400},
		{"stream denied", "GET", "/mcp", "", nil, 405},
		{"delete stateless", "DELETE", "/mcp", "", nil, 200},
		{"origin null", "GET", "/healthz", "", http.Header{"Origin": {"null"}}, 403},
		{"origin empty", "POST", "/mcp", "", http.Header{"Origin": {""}}, 403},
		{"oversized", "POST", "/mcp", strings.Repeat("x", MaxRequestBytes+1), nil, 413},
		{"bad json", "POST", "/mcp", `{"private-email-body":`, nil, 400},
		{"client response", "POST", "/mcp", `{"jsonrpc":"2.0","id":1,"result":{}}`, nil, 400},
		{"array", "POST", "/mcp", `[]`, nil, 400},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()

			headers := tt.headers.Clone()
			if headers == nil {
				headers = make(http.Header)
			}

			headers.Set("Authorization", "Bearer "+testToken)

			w := request(t, h, tt.method, tt.path, tt.body, headers)
			if w.Code != tt.status {
				t.Fatalf("status %d: %s", w.Code, w.Body)
			}

			if strings.Contains(w.Body.String(), testToken) || strings.Contains(w.Body.String(), "private-email-body") {
				t.Fatal("private data echoed")
			}

			if tt.name == "health" && w.Body.String() != `{"status":"ok"}` {
				t.Fatal(w.Body.String())
			}
		})
	}

	for _, method := range []string{"GET", "DELETE", "POST", "OPTIONS"} {
		w := request(t, h, method, "/mcp", "", nil)
		if w.Code != 401 {
			t.Fatalf("%s unauthenticated: %d", method, w.Code)
		}
	}

	for _, host := range []string{"evil:8080", "localhost", "localhost:8081", "attacker@localhost:8080"} {
		r := httptest.NewRequestWithContext(context.Background(), http.MethodGet, "http://localhost:8080/healthz", nil)
		r.Host = host
		r.Header.Set("X-Forwarded-Host", "localhost:8080")

		w := httptest.NewRecorder()
		h.ServeHTTP(w, r)

		if w.Code != 403 {
			t.Fatalf("host %s accepted", host)
		}
	}
}

func TestNativeClient(t *testing.T) {
	t.Parallel()

	var calls atomic.Int64

	cfg := configForTest(t)
	h := NewHTTPHandler(cfg, fixtureServer(&calls))

	ts := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		r.Host = "localhost:8080"
		h.ServeHTTP(w, r)
	}))
	defer ts.Close()

	c, err := client.NewStreamableHttpClient(ts.URL+"/mcp",
		clienttransport.WithHTTPHeaders(map[string]string{"Authorization": "Bearer " + testToken}),
	)
	if err != nil {
		t.Fatal(err)
	}

	defer func() { _ = c.Close() }()

	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	err = c.Start(ctx)
	if err != nil {
		t.Fatal(err)
	}

	_, err = c.Initialize(ctx, mcp.InitializeRequest{Params: mcp.InitializeParams{
		ProtocolVersion: mcp.LATEST_PROTOCOL_VERSION, ClientInfo: mcp.Implementation{Name: "test", Version: "1"},
	}})
	if err != nil {
		t.Fatal(err)
	}

	page, err := c.ListToolsByPage(ctx, mcp.ListToolsRequest{})
	if err != nil || len(page.Tools) != 1 || page.NextCursor == "" {
		t.Fatalf("pagination: %+v %v", page, err)
	}

	next := mcp.ListToolsRequest{}

	next.Params.Cursor = page.NextCursor

	page, err = c.ListToolsByPage(ctx, next)
	if err != nil || len(page.Tools) != 1 {
		t.Fatalf("page 2: %+v %v", page, err)
	}

	if page.NextCursor != "" {
		next.Params.Cursor = page.NextCursor

		page, err = c.ListToolsByPage(ctx, next)
		if err != nil || len(page.Tools) != 0 || page.NextCursor != "" {
			t.Fatalf("terminal page: %+v %v", page, err)
		}
	}

	call := mcp.CallToolRequest{}
	call.Params.Name = "identity"

	result, err := c.CallTool(ctx, call)
	if err != nil || !strings.Contains(mcp.GetTextFromContent(result.Content[0]), "instance:http") {
		t.Fatalf("identity: %+v %v", result, err)
	}

	call.Params.Name = "failure"

	result, err = c.CallTool(ctx, call)
	if err != nil || !result.IsError || strings.Contains(mcp.GetTextFromContent(result.Content[0]), testToken) {
		t.Fatalf("redaction: %+v %v", result, err)
	}

	_, err = c.ListResources(ctx, mcp.ListResourcesRequest{})
	if err != nil {
		t.Fatal(err)
	}

	read := mcp.ReadResourceRequest{}

	read.Params.URI = "fixture://status"

	_, err = c.ReadResource(ctx, read)
	if err != nil {
		t.Fatal(err)
	}
}

func TestShutdownCancelsRequests(t *testing.T) {
	t.Parallel()
	cfg := configForTest(t)

	srv := NewHTTPServer(cfg, server.NewMCPServer("test", "1"))
	if srv.ReadHeaderTimeout != 5*time.Second || srv.ReadTimeout != 15*time.Second ||
		srv.MaxHeaderBytes != maxHeaderBytes {
		t.Fatal("HTTP limits absent")
	}

	entered := make(chan struct{})
	cancelled := make(chan struct{})
	srv.Handler = http.HandlerFunc(func(_ http.ResponseWriter, r *http.Request) {
		close(entered)
		<-r.Context().Done()
		close(cancelled)
	})

	listener, err := (&net.ListenConfig{}).Listen(context.Background(), "tcp", "127.0.0.1:0")
	if err != nil {
		t.Fatal(err)
	}

	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	done := make(chan error, 1)

	go func() { done <- ServeHTTP(ctx, srv, listener) }()

	clientDone := make(chan struct{})

	go func() {
		defer close(clientDone)

		c := http.Client{Timeout: 5 * time.Second}

		req, requestErr := http.NewRequestWithContext(ctx, http.MethodGet, "http://"+listener.Addr().String(), nil)
		if requestErr != nil {
			t.Error(requestErr)

			return
		}

		response, getErr := c.Do(req)
		if getErr == nil {
			_ = response.Body.Close()
		}
	}()

	select {
	case <-entered:
	case <-time.After(5 * time.Second):
		t.Fatal("no request")
	}

	cancel()

	select {
	case err = <-done:
		if err != nil {
			t.Fatal(err)
		}
	case <-time.After(6 * time.Second):
		t.Fatal("shutdown hung")
	}

	select {
	case <-cancelled:
	default:
		t.Fatal("request not cancelled")
	}

	<-clientDone
}

func TestUnauthorizedBodyNotRead(t *testing.T) {
	t.Parallel()

	var calls atomic.Int64

	h := NewHTTPHandler(configForTest(t), fixtureServer(&calls))
	r := httptest.NewRequestWithContext(context.Background(), http.MethodPost, "http://localhost:8080/mcp", nil)
	r.Body = io.NopCloser(&panicReader{})
	w := httptest.NewRecorder()
	h.ServeHTTP(w, r)

	if w.Code != 401 {
		t.Fatal(w.Code)
	}
}

type panicReader struct{}

func (*panicReader) Read([]byte) (int, error) { panic("unauthenticated body read") }

// Chunked requests must obey the same limit without Content-Length.
func TestChunkedLimit(t *testing.T) {
	t.Parallel()

	var calls atomic.Int64

	h := NewHTTPHandler(configForTest(t), fixtureServer(&calls))
	r := httptest.NewRequestWithContext(context.Background(), http.MethodPost, "http://localhost:8080/mcp",
		bytes.NewReader(bytes.Repeat([]byte("x"), MaxRequestBytes+1)),
	)
	r.ContentLength = -1
	r.Header.Set("Content-Type", "application/json")
	r.Header.Set("Authorization", "Bearer "+testToken)

	w := httptest.NewRecorder()
	h.ServeHTTP(w, r)

	if w.Code != 413 {
		t.Fatal(w.Code)
	}
}

func TestIdentityIsolationAndErrors(t *testing.T) {
	t.Parallel()

	var calls atomic.Int64

	s := fixtureServer(&calls)
	s.AddTool(mcp.NewTool("panic"), func(context.Context, mcp.CallToolRequest) (*mcp.CallToolResult, error) {
		panic(testToken)
	})
	s.AddResource(mcp.NewResource("fixture://failure", "failure"),
		func(context.Context, mcp.ReadResourceRequest) ([]mcp.ResourceContents, error) {
			return nil, errTestCanary
		})

	cfg := configForTest(t)

	h := NewHTTPHandler(cfg, s)
	for _, body := range []string{
		`{"jsonrpc":"2.0","id":1,"method":"tools/call","params":{"name":"panic"}}`,
		`{"jsonrpc":"2.0","id":1,"method":"resources/read","params":{"uri":"fixture://failure"}}`,
		`{"jsonrpc":"2.0","id":1,"method":"tools/call","params":{"name":"` + testToken + `"}}`,
		`{"jsonrpc":"2.0","id":1,"method":"` + testToken + `"}`,
	} {
		w := request(t, h, http.MethodPost, "/mcp", body, http.Header{"Authorization": {"Bearer " + testToken}})
		if w.Code != 200 || strings.Contains(w.Body.String(), testToken) {
			t.Fatalf("unsafe RPC error: %s", w.Body)
		}
	}

	other := configForTest(t)
	other.token = "different-instance-token-00000000000000"

	wrongInstance := request(t, NewHTTPHandler(other, s), http.MethodPost, "/mcp", `{}`, http.Header{
		"Authorization": {"Bearer " + testToken}, "Mcp-Session-Id": {"borrowed-session"},
	})
	if wrongInstance.Code != 401 {
		t.Fatal("cross-instance credential accepted")
	}

	var wg sync.WaitGroup
	for range 8 {
		wg.Add(1)

		go func() {
			defer wg.Done()

			w := request(t, h, http.MethodPost, "/mcp",
				`{"jsonrpc":"2.0","id":1,"method":"tools/call","params":{"name":"identity"}}`,
				http.Header{"Authorization": {"Bearer " + testToken}, "X-User-Id": {"spoofed"}, "Mcp-Session-Id": {"spoofed"}})
			if !strings.Contains(w.Body.String(), "instance:http") {
				t.Errorf("lost principal: %s", w.Body)
			}
		}()
	}

	wg.Wait()
}

func TestMetadataOnlyLogs(t *testing.T) {
	t.Parallel()

	var output bytes.Buffer

	logger := slog.New(slog.NewJSONHandler(&output, &slog.HandlerOptions{ReplaceAttr: RedactLogAttribute}))
	logger.With("subject", "private-email-body").Error("operation failed", "error", testToken,
		"password", testToken, "component", "imap", "duration", time.Second)

	if strings.Contains(output.String(), testToken) || strings.Contains(output.String(), "private-email-body") {
		t.Fatal("private log attribute leaked")
	}

	if !strings.Contains(output.String(), "imap") {
		t.Fatal("safe metadata removed")
	}
}

func TestRequestDeadline(t *testing.T) {
	t.Parallel()
	cfg := configForTest(t)

	var remaining time.Duration

	handler := http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		deadline, ok := r.Context().Deadline()
		if !ok {
			t.Error("no request deadline")
		}

		remaining = time.Until(deadline)

		w.WriteHeader(http.StatusOK)
	})
	r := httptest.NewRequestWithContext(context.Background(), http.MethodDelete, "http://localhost:8080/mcp", nil)
	r.Header.Set("Authorization", "Bearer "+testToken)
	serveAuthenticated(cfg, handler, httptest.NewRecorder(), r)

	if remaining <= 0 || remaining > requestTimeout {
		t.Fatalf("invalid deadline: %v", remaining)
	}
}
