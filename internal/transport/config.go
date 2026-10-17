// Package transport owns process transports and their authentication boundary.
package transport

import (
	"errors"
	"io"
	"net"
	"os"
	"strconv"
	"strings"
)

const (
	Stdio          = "stdio"
	HTTP           = "http"
	defaultAddress = "127.0.0.1:8080"
	defaultHosts   = "localhost:8080,127.0.0.1:8080"
	maxSecretFile  = 4096
	minTokenLength = 32
	maxTokenLength = 256
)

var (
	errTransport    = errors.New("EMAIL_TRANSPORT must be stdio or http")
	errAddress      = errors.New("invalid EMAIL_HTTP_ADDR: expected host:port")
	errHosts        = errors.New("invalid EMAIL_HTTP_ALLOWED_HOSTS: exact host:port entries required")
	errAuthority    = errors.New("invalid authority")
	errPort         = errors.New("invalid port")
	errSecretFile   = errors.New("EMAIL_HTTP_TOKEN_FILE must name a readable secret file")
	errSecretType   = errors.New("EMAIL_HTTP_TOKEN_FILE must be a regular file")
	errSecretSize   = errors.New("invalid HTTP secret file size")
	errSecretFormat = errors.New("HTTP secret must contain 32-256 non-whitespace printable ASCII bytes")
)

// Config is validated once, before any mail pools or listeners are constructed.
// Secret material is intentionally private and must never be logged.
type Config struct {
	Mode    string
	Address string
	token   string
	hosts   map[string]struct{}
}

// LoadConfigFromEnv loads transport configuration independently of mail accounts.
func LoadConfigFromEnv() (*Config, error) {
	return loadConfig(os.LookupEnv)
}

func loadConfig(lookup func(string) (string, bool)) (*Config, error) {
	cfg := &Config{Mode: Stdio}
	mode, exists := lookup("EMAIL_TRANSPORT")

	if exists {
		cfg.Mode = mode
	}

	if cfg.Mode == Stdio {
		return cfg, nil
	}

	if cfg.Mode != HTTP {
		return nil, errTransport
	}

	cfg.Address = defaultAddress
	value, exists := lookup("EMAIL_HTTP_ADDR")

	if exists {
		cfg.Address = value
	}

	_, err := authority(cfg.Address)
	if err != nil {
		return nil, errAddress
	}

	hosts := defaultHosts
	value, exists = lookup("EMAIL_HTTP_ALLOWED_HOSTS")

	if exists {
		hosts = value
	}

	cfg.hosts = make(map[string]struct{})
	for value := range strings.SplitSeq(hosts, ",") {
		host, err := authority(strings.TrimSpace(value))
		if err != nil {
			return nil, errHosts
		}

		cfg.hosts[host] = struct{}{}
	}

	file, _ := lookup("EMAIL_HTTP_TOKEN_FILE")

	token, err := readToken(file)
	if err != nil {
		return nil, err
	}

	cfg.token = token

	return cfg, nil
}

func authority(value string) (string, error) {
	host, port, err := net.SplitHostPort(value)
	if err != nil || host == "" || strings.ContainsAny(host, "/*%?#@\\ \t\r\n") {
		return "", errAuthority
	}

	number, err := strconv.Atoi(port)
	if err != nil || number < 1 || number > 65535 || strconv.Itoa(number) != port {
		return "", errPort
	}

	return net.JoinHostPort(strings.ToLower(host), port), nil
}

func readToken(path string) (string, error) {
	file, err := os.Open(path) //nolint:gosec // startup-only secret path supplied by the operator
	if err != nil {
		return "", errSecretFile
	}

	defer func() { _ = file.Close() }()

	info, err := file.Stat()
	if err != nil || !info.Mode().IsRegular() {
		return "", errSecretType
	}

	data, err := io.ReadAll(io.LimitReader(file, maxSecretFile+1))
	if err != nil || len(data) > maxSecretFile {
		return "", errSecretSize
	}

	token := strings.TrimSuffix(string(data), "\r\n")
	if token == string(data) {
		token = strings.TrimSuffix(token, "\n")
	}

	if len(token) < minTokenLength || len(token) > maxTokenLength {
		return "", errSecretFormat
	}

	for _, char := range []byte(token) {
		if char < '!' || char > '~' {
			return "", errSecretFormat
		}
	}

	return token, nil
}
