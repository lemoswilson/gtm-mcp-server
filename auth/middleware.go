package auth

import (
	"context"
	"crypto/subtle"
	"encoding/json"
	"log/slog"
	"net/http"
	"strings"
	"time"

	"golang.org/x/oauth2"
)

// ContextKey is the type for context keys used by this package.
type ContextKey string

const (
	TokenInfoKey ContextKey = "token_info"
)

// TokenInfo holds metadata about an authenticated request.
type TokenInfo struct {
	ClientID  string
	CreatedAt time.Time
	ExpiresAt time.Time
}

// Middleware validates requests using the SERVICE_ACCOUNT_API_KEY bearer token.
// If apiKey is empty, all requests are allowed through without authentication.
func Middleware(logger *slog.Logger, saTokenSource oauth2.TokenSource, apiKey string) func(http.Handler) http.Handler {
	return func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			if apiKey == "" {
				ctx := context.WithValue(r.Context(), SATokenSourceKey, saTokenSource)
				next.ServeHTTP(w, r.WithContext(ctx))
				return
			}

			authHeader := r.Header.Get("Authorization")
			if authHeader == "" {
				logger.Warn("auth_failed", "reason", "missing_header")
				unauthorized(w, "Missing authorization header")
				return
			}

			parts := strings.SplitN(authHeader, " ", 2)
			if len(parts) != 2 || !strings.EqualFold(parts[0], "bearer") {
				logger.Warn("auth_failed", "reason", "invalid_format")
				unauthorized(w, "Invalid authorization header format")
				return
			}

			if subtle.ConstantTimeCompare([]byte(parts[1]), []byte(apiKey)) != 1 {
				logger.Warn("auth_failed", "reason", "invalid_key", "token_prefix", truncateToken(parts[1]))
				unauthorized(w, "Invalid API key")
				return
			}

			ctx := context.WithValue(r.Context(), SATokenSourceKey, saTokenSource)
			ctx = context.WithValue(ctx, TokenInfoKey, &TokenInfo{
				ClientID:  "service-account",
				CreatedAt: time.Now(),
				ExpiresAt: time.Now().Add(24 * time.Hour * 365 * 10),
			})
			logger.Debug("authenticated request", "auth_mode", "service_account")
			next.ServeHTTP(w, r.WithContext(ctx))
		})
	}
}

// GetTokenInfo retrieves TokenInfo from context.
func GetTokenInfo(ctx context.Context) *TokenInfo {
	if info, ok := ctx.Value(TokenInfoKey).(*TokenInfo); ok {
		return info
	}
	return nil
}

func unauthorized(w http.ResponseWriter, message string) {
	w.Header().Set("WWW-Authenticate", `Bearer realm="gtm-mcp-server"`)
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(http.StatusUnauthorized)
	json.NewEncoder(w).Encode(map[string]string{
		"error":             "unauthorized",
		"error_description": message,
	})
}

func truncateToken(token string) string {
	if len(token) <= 8 {
		return token + "..."
	}
	return token[:8] + "..."
}
