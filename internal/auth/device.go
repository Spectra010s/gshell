package auth

import (
	"bufio"
	"context"
	"fmt"
	"os"
	"strings"

	"golang.org/x/oauth2"
	"golang.org/x/oauth2/google"
)

const (
	// CloudPlatformScope grants access to Cloud Shell management APIs
	CloudPlatformScope = "https://www.googleapis.com/auth/cloud-platform"

	// AuthProxyRedirectURL is Google's OAuth2 redirect landing page that displays
	// the authorization code for terminal and headless environments.
	AuthProxyRedirectURL = "https://sdk.cloud.google.com/authcode.html"

	// Default Client ID used for Google Cloud terminal tools.
	// Users can override via GSHELL_CLIENT_ID and GSHELL_CLIENT_SECRET environment variables.
	defaultClientID     = "32555940559.apps.googleusercontent.com"
	defaultClientSecret = "ZmssLNjJy2998hD4CTg2ejr2"
)

// GetOAuthConfig returns an OAuth2 config configured for PKCE authorization
func GetOAuthConfig() *oauth2.Config {
	clientID := os.Getenv("GSHELL_CLIENT_ID")
	if clientID == "" {
		clientID = defaultClientID
	}
	clientSecret := os.Getenv("GSHELL_CLIENT_SECRET")
	if clientSecret == "" {
		clientSecret = defaultClientSecret
	}

	return &oauth2.Config{
		ClientID:     clientID,
		ClientSecret: clientSecret,
		Endpoint:     google.Endpoint,
		RedirectURL:  AuthProxyRedirectURL,
		Scopes:       []string{CloudPlatformScope},
	}
}

// RequestAuthCodeLogin runs the interactive PKCE authorization code flow
func RequestAuthCodeLogin(ctx context.Context) (*oauth2.Token, error) {
	conf := GetOAuthConfig()

	verifier := oauth2.GenerateVerifier()
	authURL := conf.AuthCodeURL(
		"gshell-cli",
		oauth2.AccessTypeOffline,
		oauth2.S256ChallengeOption(verifier),
		oauth2.SetAuthURLParam("prompt", "consent"),
	)

	fmt.Println()
	fmt.Println("  Google Cloud Shell Authentication")
	fmt.Println("  ---------------------------------")
	fmt.Println("  1. Open this link in your browser:")
	fmt.Println()
	fmt.Printf("     %s\n", authURL)
	fmt.Println()
	fmt.Println("  2. Complete sign-in and copy the verification code.")
	fmt.Println()
	fmt.Print("  Enter verification code: ")

	reader := bufio.NewReader(os.Stdin)
	input, err := reader.ReadString('\n')
	if err != nil && len(input) == 0 {
		return nil, fmt.Errorf("failed to read verification code: %w", err)
	}

	code := strings.TrimSpace(input)
	if code == "" {
		return nil, fmt.Errorf("verification code cannot be empty")
	}

	token, err := conf.Exchange(ctx, code, oauth2.VerifierOption(verifier))
	if err != nil {
		return nil, fmt.Errorf("authentication failed: %w", err)
	}

	return token, nil
}
