package main

import (
	"bufio"
	"bytes"
	"context"
	"encoding/json"
	"os"
	"os/exec"
	"strings"
	"syscall"
	"testing"
	"time"
)

// TestEntrypointProcess isolates the stdio library's process-global session.
func TestEntrypointProcess(t *testing.T) {
	t.Parallel()

	if os.Getenv("EMAIL_TEST_ENTRYPOINT") == "1" {
		main()
	}
}

func subprocess(t *testing.T, settings ...string) *exec.Cmd {
	t.Helper()

	path, err := os.Executable()
	if err != nil {
		t.Fatal(err)
	}

	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	t.Cleanup(cancel)

	cmd := exec.CommandContext(ctx, path, "-test.run=^TestEntrypointProcess$") //nolint:gosec // only this test executable

	for _, entry := range os.Environ() {
		if !strings.HasPrefix(entry, "EMAIL_") && !strings.HasPrefix(entry, "LOG_") {
			cmd.Env = append(cmd.Env, entry)
		}
	}

	cmd.Env = append(cmd.Env, "EMAIL_TEST_ENTRYPOINT=1",
		`EMAIL_ACCOUNTS=[{"id":"fake","email":"fake@example.com","imap_host":"invalid.example",`+
			`"smtp_host":"invalid.example"}]`)
	cmd.Env = append(cmd.Env, settings...)

	return cmd
}

func TestInvalidStartup(t *testing.T) {
	t.Parallel()

	for _, setting := range []string{"EMAIL_TRANSPORT=", "EMAIL_TRANSPORT=sse", "EMAIL_TRANSPORT=http"} {
		t.Run(setting, func(t *testing.T) {
			t.Parallel()
			cmd := subprocess(t, setting, "EMAIL_HTTP_TOKEN_FILE=/unreadable/private-secret-canary")

			var stdout, stderr bytes.Buffer

			cmd.Stdout = &stdout

			cmd.Stderr = &stderr
			if cmd.Run() == nil {
				t.Fatal("invalid startup succeeded")
			}

			if stdout.Len() != 0 || strings.Contains(stderr.String(), "private-secret-canary") {
				t.Fatalf("unsafe startup output: %s %s", &stdout, &stderr)
			}
		})
	}
}

func TestStdioProtocolAndSignal(t *testing.T) {
	t.Parallel()
	cmd := subprocess(t, "EMAIL_HTTP_ADDR=invalid", "EMAIL_HTTP_TOKEN_FILE=/missing")

	stdin, err := cmd.StdinPipe()
	if err != nil {
		t.Fatal(err)
	}

	stdout, err := cmd.StdoutPipe()
	if err != nil {
		t.Fatal(err)
	}

	var stderr bytes.Buffer

	cmd.Stderr = &stderr

	err = cmd.Start()
	if err != nil {
		t.Fatal(err)
	}

	t.Cleanup(func() { _ = cmd.Process.Kill() })

	reader := bufio.NewReader(stdout)

	for _, body := range []string{
		`{"jsonrpc":"2.0","id":1,"method":"initialize","params":` +
			`{"protocolVersion":"2025-03-26","capabilities":{},"clientInfo":{"name":"regression","version":"1"}}}`,
		`{"jsonrpc":"2.0","id":2,"method":"tools/list"}`,
		`{"jsonrpc":"2.0","id":3,"method":"tools/call","params":{"name":"email_accounts","arguments":{}}}`,
		`{"jsonrpc":"2.0","id":4,"method":"resources/read","params":{"uri":"email://status"}}`,
	} {
		_, err = stdin.Write([]byte(body + "\n"))
		if err != nil {
			t.Fatal(err)
		}

		line, readErr := reader.ReadBytes('\n')
		if readErr != nil {
			t.Fatal(readErr)
		}

		var result map[string]json.RawMessage
		if json.Unmarshal(line, &result) != nil || result["result"] == nil || result["error"] != nil {
			t.Fatalf("invalid protocol stdout: %s", line)
		}
	}

	err = cmd.Process.Signal(syscall.SIGTERM)
	if err != nil {
		t.Fatal(err)
	}

	err = cmd.Wait()
	if err != nil {
		t.Fatalf("signal exit: %v, %s", err, &stderr)
	}
}
