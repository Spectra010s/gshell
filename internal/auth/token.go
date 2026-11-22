package auth

import (
	"context"
	"encoding/base64"
	"encoding/json"
	"fmt"
	"strings"
	"sync"

	"golang.org/x/oauth2"
)

// Login runs the complete interactive authentication flow
func Login(ctx context.Context, p Prompter) (*oauth2.Token, error) {
	token, err := RequestAuthCodeLogin(ctx, p)
	if err != nil {
		return nil, err
	}

	if err := SaveToken(token, emailFromToken(token)); err != nil {
		return nil, fmt.Errorf("failed to save credentials: %w", err)
	}

	return token, nil
}

// emailFromToken reads the account email from the ID token issued by the
// sign-in flow.
func emailFromToken(token *oauth2.Token) string {
	raw, _ := token.Extra("id_token").(string)
	parts := strings.Split(raw, ".")
	if len(parts) != 3 {
		return ""
	}

	payload, err := base64.RawURLEncoding.DecodeString(parts[1])
	if err != nil {
		return ""
	}

	var claims struct {
		Email string `json:"email"`
	}
	if err := json.Unmarshal(payload, &claims); err != nil {
		return ""
	}
	return claims.Email
}

// persistentTokenSource wraps an oauth2.TokenSource and saves refreshed tokens to disk
type persistentTokenSource struct {
	mu     sync.Mutex
	source oauth2.TokenSource
	email  string
}

func (p *persistentTokenSource) Token() (*oauth2.Token, error) {
	p.mu.Lock()
	defer p.mu.Unlock()

	tok, err := p.source.Token()
	if err != nil {
		errStr := err.Error()
		if strings.Contains(errStr, "invalid_grant") || strings.Contains(errStr, "expired") || strings.Contains(errStr, "revoked") {
			return nil, fmt.Errorf("authentication expired or was revoked; run `gshell auth login` to sign in again")
		}
		return nil, err
	}

	// Persist the refreshed token to disk so future CLI calls don't need to re-auth
	_ = SaveToken(tok, p.email)
	return tok, nil
}

// GetTokenSource returns a valid, auto-refreshing oauth2.TokenSource.
// If no credentials are found, it returns ErrNoCredentials.
func GetTokenSource(ctx context.Context) (oauth2.TokenSource, error) {
	tok, email, err := LoadCredentials()
	if err != nil {
		return nil, err
	}

	conf := GetOAuthConfig()
	liveSource := conf.TokenSource(ctx, tok)

	return &persistentTokenSource{
		source: liveSource,
		email:  email,
	}, nil
}
