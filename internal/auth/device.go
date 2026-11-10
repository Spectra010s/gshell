package auth

import (
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"net/url"
	"strings"
	"time"

	"golang.org/x/oauth2"
)

const (
	deviceCodeURL = "https://oauth2.googleapis.com/device/code"
	tokenURL      = "https://oauth2.googleapis.com/token"

	// CloudPlatformScope grants access to Cloud Shell management APIs
	CloudPlatformScope = "https://www.googleapis.com/auth/cloud-platform"

	// Default Client ID used for Google Cloud terminal tools.
	// Users can override via GSHELL_CLIENT_ID and GSHELL_CLIENT_SECRET environment variables.
	defaultClientID     = "32555940559.apps.googleusercontent.com"
	defaultClientSecret = "ZmssLNjJy2998hD4CTg2ejr2"
)

// DeviceAuthResponse holds the initial code challenge returned by Google
type DeviceAuthResponse struct {
	DeviceCode      string `json:"device_code"`
	UserCode        string `json:"user_code"`
	VerificationURL string `json:"verification_url"`
	ExpiresIn       int    `json:"expires_in"`
	Interval        int    `json:"interval"`
}

type tokenResponse struct {
	AccessToken  string `json:"access_token"`
	RefreshToken string `json:"refresh_token"`
	TokenType    string `json:"token_type"`
	ExpiresIn    int    `json:"expires_in"`
	Error        string `json:"error"`
}

// RequestDeviceCode initiates the device authorization flow with Google OAuth2
func RequestDeviceCode(ctx context.Context, clientID string) (*DeviceAuthResponse, error) {
	if clientID == "" {
		clientID = defaultClientID
	}

	form := url.Values{
		"client_id": {clientID},
		"scope":     {CloudPlatformScope},
	}

	req, err := http.NewRequestWithContext(ctx, http.MethodPost, deviceCodeURL, strings.NewReader(form.Encode()))
	if err != nil {
		return nil, fmt.Errorf("failed to create device auth request: %w", err)
	}
	req.Header.Set("Content-Type", "application/x-www-form-urlencoded")

	resp, err := http.DefaultClient.Do(req)
	if err != nil {
		return nil, fmt.Errorf("failed to initiate device auth: %w", err)
	}
	defer resp.Body.Close()

	body, err := io.ReadAll(resp.Body)
	if err != nil {
		return nil, fmt.Errorf("failed to read response: %w", err)
	}

	if resp.StatusCode != http.StatusOK {
		return nil, fmt.Errorf("device auth failed (%d): %s", resp.StatusCode, string(body))
	}

	var authResp DeviceAuthResponse
	if err := json.Unmarshal(body, &authResp); err != nil {
		return nil, fmt.Errorf("failed to parse device auth response: %w", err)
	}

	if authResp.Interval <= 0 {
		authResp.Interval = 5
	}

	return &authResp, nil
}

// PollForToken polls Google's token endpoint until the user approves the device code
func PollForToken(ctx context.Context, clientID, clientSecret, deviceCode string, interval int) (*oauth2.Token, error) {
	if clientID == "" {
		clientID = defaultClientID
	}
	if clientSecret == "" {
		clientSecret = defaultClientSecret
	}

	ticker := time.NewTicker(time.Duration(interval) * time.Second)
	defer ticker.Stop()

	for {
		select {
		case <-ctx.Done():
			return nil, ctx.Err()
		case <-ticker.C:
			form := url.Values{
				"client_id":     {clientID},
				"client_secret": {clientSecret},
				"device_code":   {deviceCode},
				"grant_type":    {"urn:ietf:params:oauth:grant-type:device_code"},
			}

			req, err := http.NewRequestWithContext(ctx, http.MethodPost, tokenURL, strings.NewReader(form.Encode()))
			if err != nil {
				return nil, err
			}
			req.Header.Set("Content-Type", "application/x-www-form-urlencoded")

			resp, err := http.DefaultClient.Do(req)
			if err != nil {
				return nil, fmt.Errorf("polling error: %w", err)
			}

			body, err := io.ReadAll(resp.Body)
			resp.Body.Close()
			if err != nil {
				return nil, err
			}

			var tokenResp tokenResponse
			if err := json.Unmarshal(body, &tokenResp); err != nil {
				return nil, fmt.Errorf("failed to parse token response: %w", err)
			}

			switch tokenResp.Error {
			case "":
				// Success: convert to standard oauth2.Token
				expiry := time.Now().Add(time.Duration(tokenResp.ExpiresIn) * time.Second)
				return &oauth2.Token{
					AccessToken:  tokenResp.AccessToken,
					RefreshToken: tokenResp.RefreshToken,
					TokenType:    tokenResp.TokenType,
					Expiry:       expiry,
				}, nil

			case "authorization_pending":
				// User has not approved yet, continue waiting
				continue

			case "slow_down":
				// Google requested longer interval
				ticker.Reset(time.Duration(interval+5) * time.Second)
				continue

			case "access_denied":
				return nil, fmt.Errorf("login was denied by user")

			case "expired_token":
				return nil, fmt.Errorf("login session timed out, please run login again")

			default:
				return nil, fmt.Errorf("authentication error: %s", tokenResp.Error)
			}
		}
	}
}
