package auth

import (
	"context"
	"fmt"
	"os"

	"golang.org/x/oauth2"
	"golang.org/x/oauth2/google"
)

const SATokenSourceKey ContextKey = "sa_token_source"

// GoogleScopes defines the OAuth2 scopes required for GTM API access.
var GoogleScopes = []string{
	"https://www.googleapis.com/auth/tagmanager.delete.containers",
	"https://www.googleapis.com/auth/tagmanager.edit.containers",
	"https://www.googleapis.com/auth/tagmanager.edit.containerversions",
	"https://www.googleapis.com/auth/tagmanager.manage.accounts",
	"https://www.googleapis.com/auth/tagmanager.publish",
}

// NewServiceAccountTokenSource creates an oauth2.TokenSource scoped to GTM API access.
// Priority: keyJSON (inline) > keyFile (path to JSON file) > Application Default Credentials.
func NewServiceAccountTokenSource(ctx context.Context, keyJSON, keyFile string) (oauth2.TokenSource, error) {
	if keyJSON == "" && keyFile != "" {
		data, err := os.ReadFile(keyFile)
		if err != nil {
			return nil, fmt.Errorf("cannot read GOOGLE_SERVICE_ACCOUNT_KEY_FILE %q: %w", keyFile, err)
		}
		keyJSON = string(data)
	}

	if keyJSON != "" {
		config, err := google.JWTConfigFromJSON([]byte(keyJSON), GoogleScopes...)
		if err != nil {
			return nil, fmt.Errorf("invalid service account key JSON: %w", err)
		}
		return config.TokenSource(ctx), nil
	}

	creds, err := google.FindDefaultCredentials(ctx, GoogleScopes...)
	if err != nil {
		return nil, fmt.Errorf("no service account credentials found (set GOOGLE_SERVICE_ACCOUNT_KEY_FILE, GOOGLE_SERVICE_ACCOUNT_KEY_JSON, or use Workload Identity): %w", err)
	}
	return creds.TokenSource, nil
}

// GetSATokenSource retrieves the service account token source from context.
// Returns nil when not in S2S mode.
func GetSATokenSource(ctx context.Context) oauth2.TokenSource {
	if ts, ok := ctx.Value(SATokenSourceKey).(oauth2.TokenSource); ok {
		return ts
	}
	return nil
}
