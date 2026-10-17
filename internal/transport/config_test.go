package transport //nolint:testpackage // tests validate private startup parsing without changing process environment

import (
	"maps"
	"os"
	"path/filepath"
	"strconv"
	"strings"
	"testing"
)

const testToken = "dedicated-fake-token-for-transport-tests-0002"

func configForTest(t *testing.T) *Config {
	t.Helper()

	cfg, err := loadConfig(lookupForTest(t, nil))
	if err != nil {
		t.Fatal(err)
	}

	return cfg
}

func lookupForTest(t *testing.T, overrides map[string]string) func(string) (string, bool) {
	t.Helper()

	path := filepath.Join(t.TempDir(), "token")

	err := os.WriteFile(path, []byte(testToken+"\n"), 0o600)
	if err != nil {
		t.Fatal(err)
	}

	values := map[string]string{"EMAIL_TRANSPORT": HTTP, "EMAIL_HTTP_TOKEN_FILE": path}
	maps.Copy(values, overrides)

	return func(key string) (string, bool) {
		value, ok := values[key]

		return value, ok
	}
}

func TestConfiguration(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name, key, value string
		valid            bool
	}{
		{"default HTTP", "EMAIL_TRANSPORT", HTTP, true},
		{"stdio ignores HTTP", "EMAIL_TRANSPORT", Stdio, true},
		{"empty mode", "EMAIL_TRANSPORT", "", false},
		{"unknown mode", "EMAIL_TRANSPORT", "sse", false},
		{"empty address", "EMAIL_HTTP_ADDR", "", false},
		{"no port", "EMAIL_HTTP_ADDR", "localhost", false},
		{"invalid port", "EMAIL_HTTP_ADDR", "localhost:65536", false},
		{"private listener", "EMAIL_HTTP_ADDR", "0.0.0.0:8080", true},
		{"IPv6", "EMAIL_HTTP_ADDR", "[::1]:8080", true},
		{"wildcard host", "EMAIL_HTTP_ALLOWED_HOSTS", "*:8080", false},
		{"empty hosts", "EMAIL_HTTP_ALLOWED_HOSTS", "", false},
		{"missing host port", "EMAIL_HTTP_ALLOWED_HOSTS", "localhost", false},
		{"URL not host", "EMAIL_HTTP_ALLOWED_HOSTS", "https://mail:8080", false},
		{"hosts", "EMAIL_HTTP_ALLOWED_HOSTS", "Mail:8080, [::1]:8080", true},
		{"missing secret", "EMAIL_HTTP_TOKEN_FILE", "", false},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()

			_, err := loadConfig(lookupForTest(t, map[string]string{tt.key: tt.value}))
			if (err == nil) != tt.valid {
				t.Fatalf("valid=%v, error=%v", tt.valid, err)
			}
		})
	}

	cfg, err := loadConfig(func(string) (string, bool) { return "", false })
	if err != nil || cfg.Mode != Stdio {
		t.Fatalf("default mode: %+v %v", cfg, err)
	}
}

func TestSecretValidation(t *testing.T) {
	t.Parallel()

	for _, value := range []string{testToken, testToken + "\n", testToken + "\r\n", "", "short", testToken + "\r",
		testToken + "\n\n", testToken + " ", testToken + "\x00", testToken + "é",
		strings.Repeat("x", 257), strings.Repeat("x", 4097),
	} {
		t.Run(strings.ReplaceAll(value[:min(len(value), 12)], "\n", "LF")+"-"+strconv.Itoa(len(value)), func(t *testing.T) {
			t.Parallel()

			path := filepath.Join(t.TempDir(), "token")

			err := os.WriteFile(path, []byte(value), 0o600)
			if err != nil {
				t.Fatal(err)
			}

			token, err := readToken(path)

			valid := value == testToken || value == testToken+"\n" || value == testToken+"\r\n"
			if (err == nil) != valid {
				t.Fatalf("valid=%v error=%v", valid, err)
			}

			if valid && token != testToken {
				t.Fatal("line ending retained")
			}

			if err != nil && strings.Contains(err.Error(), testToken) {
				t.Fatal("secret in error")
			}
		})
	}

	_, err := readToken(t.TempDir())
	if err == nil {
		t.Fatal("directory accepted as secret")
	}
}
