package transport

import (
	"context"
	"errors"
	"log/slog"

	"github.com/mark3labs/mcp-go/mcp"
	"github.com/mark3labs/mcp-go/server"
)

var errUpstream = errors.New("upstream_error")

// SafeToolErrors retains legacy stdio errors but never returns provider text over HTTP.
//
//nolint:nonamedreturns // deferred panic recovery must replace the returned error/result
func SafeToolErrors(next server.ToolHandlerFunc) server.ToolHandlerFunc {
	return func(ctx context.Context, request mcp.CallToolRequest) (result *mcp.CallToolResult, err error) {
		principal, _ := PrincipalFromContext(ctx)
		if principal.Transport != HTTP {
			return next(ctx, request)
		}

		defer func() {
			if recover() != nil {
				result, err = mcp.NewToolResultError("upstream_error"), nil
			}
		}()

		result, err = next(ctx, request)
		if err != nil || (result != nil && result.IsError) {
			return mcp.NewToolResultError("upstream_error"), nil
		}

		return result, nil
	}
}

// SafeResourceErrors removes private provider errors at the resource boundary too.
//
//nolint:nonamedreturns // deferred panic recovery must replace the returned error/result
func SafeResourceErrors(next server.ResourceHandlerFunc) server.ResourceHandlerFunc {
	return func(ctx context.Context, request mcp.ReadResourceRequest) (result []mcp.ResourceContents, err error) {
		principal, _ := PrincipalFromContext(ctx)
		if principal.Transport != HTTP {
			return next(ctx, request)
		}

		defer func() {
			if recover() != nil {
				result, err = nil, errUpstream
			}
		}()

		result, err = next(ctx, request)
		if err != nil {
			return nil, errUpstream
		}

		return result, nil
	}
}

// RedactLogAttribute permits only known metadata in HTTP process logs.
func RedactLogAttribute(_ []string, attr slog.Attr) slog.Attr {
	switch attr.Key {
	case slog.TimeKey, slog.LevelKey, slog.MessageKey, "component", "account_id",
		"duration", "timeout", "recipients", "attempt":
		return attr
	default:
		return slog.Attr{}
	}
}
