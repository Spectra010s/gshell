package auth

import (
	"context"
	"fmt"
	"io"
	"net/http"
	"net/url"
	"strings"
	"time"

	"golang.org/x/oauth2"
)

const revokeURL = "https://oauth2.googleapis.com/revoke"

// RevokeToken invalidates the token with Google so it cannot be reused.
func RevokeToken(ctx context.Context, token *oauth2.Token) error {
	value := token.RefreshToken
	if value == "" {
		value = token.AccessToken
	}
	if value == "" {
		return nil
	}

	form := url.Values{"token": {value}}
	req, err := http.NewRequestWithContext(
		ctx, http.MethodPost, revokeURL, strings.NewReader(form.Encode()),
	)
	if err != nil {
		return fmt.Errorf("failed to create revocation request: %w", err)
	}
	req.Header.Set("Content-Type", "application/x-www-form-urlencoded")

	client := &http.Client{Timeout: 15 * time.Second}
	resp, err := client.Do(req)
	if err != nil {
		return fmt.Errorf("failed to reach Google: %w", err)
	}
	defer resp.Body.Close()
	_, _ = io.Copy(io.Discard, resp.Body)

	if resp.StatusCode != http.StatusOK {
		return fmt.Errorf("Google rejected the token: %s", resp.Status)
	}
	return nil
}
