package main

import (
	"context"
	"errors"
	"fmt"
	"log/slog"
	"net"
	"os"
	"os/signal"
	"strconv"
	"syscall"
	"time"

	"github.com/boutquin/mcp-server-email/internal/config"
	"github.com/boutquin/mcp-server-email/internal/imap"
	applog "github.com/boutquin/mcp-server-email/internal/log"
	"github.com/boutquin/mcp-server-email/internal/resources"
	"github.com/boutquin/mcp-server-email/internal/smtp"
	"github.com/boutquin/mcp-server-email/internal/tools"
	"github.com/boutquin/mcp-server-email/internal/transport"
	"github.com/mark3labs/mcp-go/server"
)

// version is set at build time by GoReleaser via ldflags:
//
//	-X main.version={{.Version}}
var version = "dev"

var (
	errEmailConfig  = errors.New("invalid email configuration")
	errHTTPListener = errors.New("HTTP listener unavailable")
)

func main() {
	// Wire build-time version into the status resource.
	if version != "dev" {
		resources.Version = version
	}

	// Initialize structured logging from environment (LOG_LEVEL, LOG_FORMAT).
	logger := applog.Init(os.Stderr)
	slog.SetDefault(logger)

	ctx, stop := signal.NotifyContext(context.Background(), syscall.SIGINT, syscall.SIGTERM)

	err := run(ctx)
	if err != nil {
		stop()
		logger.Error("server stopped", slog.String("error", err.Error()))
		os.Exit(1)
	}

	stop()
}

func run(ctx context.Context) error {
	transportCfg, err := transport.LoadConfigFromEnv()
	if err != nil {
		return fmt.Errorf("transport configuration: %w", err)
	}

	if transportCfg.Mode == transport.HTTP {
		slog.SetDefault(slog.New(slog.NewJSONHandler(os.Stderr, &slog.HandlerOptions{
			ReplaceAttr: transport.RedactLogAttribute,
		})))
	}

	cfg, err := config.LoadFromEnv()
	if err != nil {
		return errEmailConfig
	}

	// Create connection pools
	imapPool := imap.NewPool(cfg)
	smtpPool := smtp.NewPool(cfg)

	defer func() {
		shutdownCtx, cancel := context.WithTimeout(context.WithoutCancel(ctx),
			time.Duration(cfg.PoolCloseTimeoutMS)*time.Millisecond)
		defer cancel()

		imapPool.Close(shutdownCtx)
		smtpPool.Close()
	}()

	// Create MCP server with tools and resources.
	s := newServer(cfg, imapPool, smtpPool, transportCfg.Mode)

	if transportCfg.Mode == transport.Stdio {
		err = transport.ServeStdio(ctx, s, os.Stdin, os.Stdout)
		if err != nil {
			return fmt.Errorf("stdio: %w", err)
		}

		return nil
	}

	listener, err := (&net.ListenConfig{}).Listen(ctx, "tcp", transportCfg.Address)
	if err != nil {
		return errHTTPListener
	}

	err = transport.ServeHTTP(ctx, transport.NewHTTPServer(transportCfg, s), listener)
	if err != nil {
		return fmt.Errorf("HTTP: %w", err)
	}

	return nil
}

// newServer creates the MCP server with all tools and resources registered.
func newServer(cfg *config.Config, imapPool *imap.Pool, smtpPool *smtp.Pool, mode string) *server.MCPServer {
	options := []server.ServerOption{
		server.WithToolCapabilities(mode == transport.Stdio),
		server.WithResourceCapabilities(false, mode == transport.Stdio),
		server.WithRecovery(),
		server.WithToolHandlerMiddleware(transport.SafeToolErrors),
		server.WithResourceHandlerMiddleware(transport.SafeResourceErrors),
		server.WithInstructions(
			"Email MCP server for IMAP/SMTP access. Supports multiple accounts, " +
				"full CRUD operations (read, send, move, delete, flag), folder management, and drafts. " +
				"Use email_accounts to list configured accounts. " +
				"Rate limit: " + strconv.Itoa(cfg.IMAPRateLimitRPM) + " IMAP req/min, " +
				strconv.Itoa(cfg.SMTPRateLimitRPH) + " SMTP sends/hour per account.",
		),
	}
	if mode == transport.HTTP {
		const discoveryPageSize = 10

		options = append(options, server.WithPaginationLimit(discoveryPageSize))
	}

	s := server.NewMCPServer("mcp-server-email", resources.Version, options...)

	tools.RegisterAll(s, imapPool, smtpPool, tools.LimitsFromConfig(cfg))
	s.AddResource(resources.StatusResource(), resources.StatusHandler(cfg, imapPool))

	return s
}
