package main

import (
	"context"
	_ "embed"
	"encoding/json"
	"fmt"
	"log/slog"
	"net/http"
	"os"
	"os/signal"
	"syscall"
	"time"

	"gtm-mcp-server/auth"
	"gtm-mcp-server/config"
	"gtm-mcp-server/gtm"
	"gtm-mcp-server/middleware"

	"github.com/modelcontextprotocol/go-sdk/mcp"
)

//go:embed llms.txt
var llmsTxt string

const (
	serverName    = "gtm-mcp-server"
	serverVersion = "1.6.0"
)

func main() {
	logger := slog.New(slog.NewJSONHandler(os.Stderr, &slog.HandlerOptions{
		Level: slog.LevelInfo,
	}))
	slog.SetDefault(logger)

	cfg, err := config.Load()
	if err != nil {
		logger.Error("failed to load configuration", "error", err)
		os.Exit(1)
	}

	if cfg.LogLevel == "debug" {
		logger = slog.New(slog.NewJSONHandler(os.Stderr, &slog.HandlerOptions{
			Level: slog.LevelDebug,
		}))
		slog.SetDefault(logger)
	}

	// Initialize service account token source for GTM API calls.
	saTokenSource, err := auth.NewServiceAccountTokenSource(context.Background(), cfg.ServiceAccountKeyJSON, cfg.ServiceAccountKeyFile)
	if err != nil {
		logger.Error("failed to initialize service account",
			"error", err,
			"hint", "set GOOGLE_SERVICE_ACCOUNT_KEY_FILE or GOOGLE_SERVICE_ACCOUNT_KEY_JSON, or deploy on GCP for Workload Identity",
		)
		os.Exit(1)
	}

	credSource := "workload_identity"
	switch {
	case cfg.ServiceAccountKeyFile != "":
		credSource = "key_file"
	case cfg.ServiceAccountKeyJSON != "":
		credSource = "key_json"
	}
	logger.Info("service_account_initialized", "credential_source", credSource)

	server := mcp.NewServer(&mcp.Implementation{
		Name:    serverName,
		Version: serverVersion,
	}, nil)

	server.AddReceivingMiddleware(middleware.NewLoggingMiddleware(logger))
	registerTools(server)

	mcpHandler := mcp.NewStreamableHTTPHandler(func(r *http.Request) *mcp.Server {
		return server
	}, nil)

	mux := http.NewServeMux()

	mux.HandleFunc("GET /health", func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusOK)
		json.NewEncoder(w).Encode(map[string]string{
			"status":  "healthy",
			"service": serverName,
			"version": serverVersion,
		})
	})

	mux.HandleFunc("GET /llms.txt", func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "text/plain; charset=utf-8")
		w.WriteHeader(http.StatusOK)
		w.Write([]byte(llmsTxt))
	})

	authMiddleware := auth.Middleware(logger, saTokenSource, cfg.ServiceAccountAPIKey)
	mux.Handle("/", authMiddleware(maxBytesHandler(5<<20, mcpHandler)))

	if cfg.ServiceAccountAPIKey != "" {
		logger.Info("api_key_auth_enabled")
	} else {
		logger.Warn("SERVICE_ACCOUNT_API_KEY not set, running without API key authentication")
	}

	addr := fmt.Sprintf(":%d", cfg.Port)
	httpServer := &http.Server{
		Addr:              addr,
		Handler:           mux,
		ReadTimeout:       30 * time.Second,
		ReadHeaderTimeout: 10 * time.Second,
		WriteTimeout:      0, // Disabled for SSE streams
		IdleTimeout:       120 * time.Second,
	}

	ctx, stop := signal.NotifyContext(context.Background(), syscall.SIGINT, syscall.SIGTERM)
	defer stop()

	go func() {
		logger.Info("starting GTM MCP server",
			"port", cfg.Port,
			"base_url", cfg.BaseURL,
		)
		if err := httpServer.ListenAndServe(); err != nil && err != http.ErrServerClosed {
			logger.Error("server error", "error", err)
			os.Exit(1)
		}
	}()

	<-ctx.Done()
	logger.Info("shutting down server")

	shutdownCtx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()

	if err := httpServer.Shutdown(shutdownCtx); err != nil {
		logger.Error("shutdown error", "error", err)
	}

	logger.Info("server stopped")
}

func registerTools(server *mcp.Server) {
	registerUtilityTools(server)
	gtm.RegisterTools(server)
}

func maxBytesHandler(maxBytes int64, next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.Body != nil {
			r.Body = http.MaxBytesReader(w, r.Body, maxBytes)
		}
		next.ServeHTTP(w, r)
	})
}

func registerUtilityTools(server *mcp.Server) {
	type PingInput struct {
		Message string `json:"message,omitempty" jsonschema:"Optional message to echo back"`
	}
	type PingOutput struct {
		Reply     string `json:"reply"`
		Timestamp string `json:"timestamp"`
	}

	mcp.AddTool(server, &mcp.Tool{
		Name:        "ping",
		Description: "Test connectivity to the GTM MCP server",
	}, func(ctx context.Context, req *mcp.CallToolRequest, input PingInput) (*mcp.CallToolResult, PingOutput, error) {
		reply := "pong"
		if input.Message != "" {
			reply = fmt.Sprintf("pong: %s", input.Message)
		}
		return nil, PingOutput{Reply: reply, Timestamp: time.Now().UTC().Format(time.RFC3339)}, nil
	})

	type AuthStatusInput struct{}
	type AuthStatusOutput struct {
		Authenticated bool   `json:"authenticated"`
		Message       string `json:"message"`
	}

	mcp.AddTool(server, &mcp.Tool{
		Name:        "auth_status",
		Description: "Check authentication status with Google Tag Manager",
	}, func(ctx context.Context, req *mcp.CallToolRequest, input AuthStatusInput) (*mcp.CallToolResult, AuthStatusOutput, error) {
		tokenInfo := auth.GetTokenInfo(ctx)
		output := AuthStatusOutput{Authenticated: tokenInfo != nil}
		if tokenInfo != nil {
			output.Message = "Authenticated via service account"
		} else {
			output.Message = "Not authenticated. Set SERVICE_ACCOUNT_API_KEY to enable authentication."
		}
		return nil, output, nil
	})
}
