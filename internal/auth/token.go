package auth

import (
	"context"
	"fmt"
	"os"
	"sync"

	"golang.org/x/oauth2"
	"golang.org/x/oauth2/google"
)

// Login runs the complete interactive Device Flow authentication
func Login(ctx context.Context) (*oauth2.Token, error) {
	clientID := os.Getenv("GSHELL_CLIENT_ID")
	clientSecret := os.Getenv("GSHELL_CLIENT_SECRET")

	authResp, err := RequestDeviceCode(ctx, clientID)
	if err != nil {
		return nil, err
	}

	fmt.Println()
	fmt.Println("  Google Cloud Shell Authentication")
	fmt.Println("  ---------------------------------")
	fmt.Printf("  1. Open this link:   %s\n", authResp.VerificationURL)
	fmt.Printf("  2. Enter this code:  %s\n", authResp.UserCode)
	fmt.Println()
	fmt.Println("  Waiting for authorization in browser...")

	token, err := PollForToken(ctx, clientID, clientSecret, authResp.DeviceCode, authResp.Interval)
	if err != nil {
		return nil, err
	}

	if err := SaveToken(token); err != nil {
		return nil, fmt.Errorf("failed to save credentials: %w", err)
	}

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

	clientID := os.Getenv("GSHELL_CLIENT_ID")
	if clientID == "" {
		clientID = defaultClientID
	}
	clientSecret := os.Getenv("GSHELL_CLIENT_SECRET")
	if clientSecret == "" {
		clientSecret = defaultClientSecret
	}

	conf := &oauth2.Config{
		ClientID:     clientID,
		ClientSecret: clientSecret,
		Endpoint:     google.Endpoint,
		Scopes:       []string{CloudPlatformScope},
	}

	// Create an auto-refreshing token source backed by our saved token
	liveSource := conf.TokenSource(ctx, tok)

	return &persistentTokenSource{
		source: liveSource,
	}, nil
}
