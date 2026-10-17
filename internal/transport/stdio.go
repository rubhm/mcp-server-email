package transport

import (
	"context"
	"errors"
	"io"
	"log"

	"github.com/mark3labs/mcp-go/server"
)

var errStdio = errors.New("stdio server failed")

// ServeStdio uses the original MCP framing with cancellation owned by main.
// Closing input on shutdown also releases a blocked protocol reader.
func ServeStdio(ctx context.Context, mcpServer *server.MCPServer, input io.ReadCloser, output io.Writer) error {
	ctx, cancel := context.WithCancel(ctx)
	defer cancel()

	done := make(chan struct{})
	stopped := make(chan struct{})

	go func() {
		defer close(stopped)

		select {
		case <-ctx.Done():
			_ = input.Close()
		case <-done:
		}
	}()

	s := server.NewStdioServer(mcpServer)
	s.SetErrorLogger(log.New(io.Discard, "", 0))
	err := s.Listen(authenticatedContext(ctx, Stdio), input, output)

	close(done)
	<-stopped

	if err != nil && ctx.Err() == nil {
		return errStdio
	}

	return nil
}
