package auth

import (
	"context"
	"fmt"
	"os"

	"golang.org/x/oauth2"
	"golang.org/x/oauth2/google"
)

const (
	// CloudPlatformScope grants access to Cloud Shell management APIs
	CloudPlatformScope = "https://www.googleapis.com/auth/cloud-platform"

	// UserEmailScope lets the sign-in response include the account email
	UserEmailScope = "https://www.googleapis.com/auth/userinfo.email"

	// OpenIDScope makes the token response include an ID token
	OpenIDScope = "openid"

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
		Scopes:       []string{OpenIDScope, UserEmailScope, CloudPlatformScope},
	}
}

// Prompter is the minimal terminal interface needed during interactive login.
type Prompter interface {
	Infof(format string, a ...any)
	Prompt(label string) (string, error)
}

// RequestAuthCodeLogin runs the interactive PKCE authorization code flow
func RequestAuthCodeLogin(ctx context.Context, p Prompter) (*oauth2.Token, error) {
	conf := GetOAuthConfig()

	verifier := oauth2.GenerateVerifier()
	authURL := conf.AuthCodeURL(
		"gshell-cli",
		oauth2.AccessTypeOffline,
		oauth2.S256ChallengeOption(verifier),
		oauth2.SetAuthURLParam("prompt", "consent"),
	)

	p.Infof("Go to the following link in your browser, and complete the sign-in prompts:")
	p.Infof("")
	p.Infof("    %s", authURL)
	p.Infof("")

	code, err := p.Prompt("Once finished, enter the verification code provided in your browser: ")
	if err != nil {
		return nil, fmt.Errorf("failed to read verification code: %w", err)
	}

	if code == "" {
		return nil, fmt.Errorf("verification code cannot be empty")
	}

	token, err := conf.Exchange(ctx, code, oauth2.VerifierOption(verifier))
	if err != nil {
		return nil, fmt.Errorf("authentication failed: %w", err)
	}

	return token, nil
}
