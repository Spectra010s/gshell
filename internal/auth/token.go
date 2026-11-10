package auth

import (
	"context"
	"fmt"
	"sync"

	"golang.org/x/oauth2"
)

// Login runs the complete interactive authentication flow
func Login(ctx context.Context) (*oauth2.Token, error) {
	token, err := RequestAuthCodeLogin(ctx)
	if err != nil {
		return nil, err
	}

	if err := SaveToken(token); err != nil {
		return nil, fmt.Errorf("failed to save credentials: %w", err)
	}

	fmt.Println()
	fmt.Println("  Authentication successful! Credentials saved.")
	fmt.Println()
	return token, nil
}

// persistentTokenSource wraps an oauth2.TokenSource and saves refreshed tokens to disk
type persistentTokenSource struct {
	mu     sync.Mutex
	source oauth2.TokenSource
}

func (p *persistentTokenSource) Token() (*oauth2.Token, error) {
	p.mu.Lock()
	defer p.mu.Unlock()

	tok, err := p.source.Token()
	if err != nil {
		return nil, err
	}

	// Persist the refreshed token to disk so future CLI calls don't need to re-auth
	_ = SaveToken(tok)
	return tok, nil
}

// GetTokenSource returns a valid, auto-refreshing oauth2.TokenSource.
// If no credentials are found, it prompts the user to authenticate.
func GetTokenSource(ctx context.Context) (oauth2.TokenSource, error) {
	tok, err := LoadToken()
	if err != nil {
		if err == ErrNoCredentials {
			// Trigger interactive login if not authenticated yet
			tok, err = Login(ctx)
			if err != nil {
				return nil, err
			}
		} else {
			return nil, err
		}
	}

	conf := GetOAuthConfig()
	liveSource := conf.TokenSource(ctx, tok)

	return &persistentTokenSource{
		source: liveSource,
	}, nil
}
