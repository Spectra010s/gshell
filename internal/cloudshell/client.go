package cloudshell

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"strings"
	"time"

	"golang.org/x/crypto/ssh"
	"golang.org/x/oauth2"
)

const (
	apiBaseURL             = "https://cloudshell.googleapis.com/v1"
	defaultEnvironmentName = "users/me/environments/default"
)

// Environment holds the Google Cloud Shell environment state
type Environment struct {
	Name        string   `json:"name"`
	State       string   `json:"state"` // "STATE_UNSPECIFIED", "SUSPENDED", "PENDING", "RUNNING"
	SSHHost     string   `json:"sshHost"`
	SSHPort     int      `json:"sshPort"`
	SSHUsername string   `json:"sshUsername"`
	PublicKeys  []string `json:"publicKeys"`
}

// Account returns the owner email embedded in the environment resource name,
// e.g. "users/someone@example.com/environments/default".
func (e *Environment) Account() string {
	rest, ok := strings.CutPrefix(e.Name, "users/")
	if !ok {
		return ""
	}

	email, _, ok := strings.Cut(rest, "/environments/")
	if !ok || email == "me" {
		return ""
	}
	return email
}

// Operation represents a Google Long Running Operation (LRO)
type Operation struct {
	Name     string          `json:"name"`
	Done     bool            `json:"done"`
	Error    *OperationError `json:"error,omitempty"`
	Response *Environment    `json:"response,omitempty"`
}

// OperationError holds error details from a failed LRO
type OperationError struct {
	Code    int               `json:"code"`
	Message string            `json:"message"`
	Details []OperationDetail `json:"details,omitempty"`
}

// OperationDetail is a single typed entry in OperationError.Details
type OperationDetail struct {
	Type   string `json:"@type"`
	Code   string `json:"code"`
	Reason string `json:"reason"`
}

// Reason returns the machine-readable failure reason, if Google supplied one
func (e *OperationError) Reason() string {
	for _, d := range e.Details {
		if d.Code != "" {
			return d.Code
		}
		if d.Reason != "" {
			return d.Reason
		}
	}
	return ""
}

// operationError converts a failed LRO into a user-facing error, or nil if the
// operation succeeded. A done operation may still carry an error, so callers
// must check this before treating completion as success.
func operationError(op *Operation) error {
	if op == nil || op.Error == nil {
		return nil
	}

	switch reason := op.Error.Reason(); {
	case reason == "QUOTA_EXCEEDED":
		return fmt.Errorf("cloud shell quota exceeded: weekly free hours are used up")
	case op.Error.Message != "":
		return fmt.Errorf("cloud shell operation failed: %s", op.Error.Message)
	case reason != "":
		return fmt.Errorf("cloud shell operation failed: %s", strings.ToLower(reason))
	default:
		return fmt.Errorf("cloud shell operation failed (%d)", op.Error.Code)
	}
}

// ConnectionInfo contains all parameters needed to open an SSH or SFTP connection
type ConnectionInfo struct {
	Host     string
	Port     int
	Username string
	KeyPath  string
	Signer   ssh.Signer
}

// Client communicates with the Google Cloud Shell REST API
type Client struct {
	httpClient  *http.Client
	tokenSource oauth2.TokenSource
}

// NewClient creates an authenticated Cloud Shell API client using an OAuth TokenSource
func NewClient(ctx context.Context, tokenSource oauth2.TokenSource) *Client {
	return &Client{
		httpClient:  oauth2.NewClient(ctx, tokenSource),
		tokenSource: tokenSource,
	}
}

// GetEnvironment retrieves the current state of the default Cloud Shell environment
func (c *Client) GetEnvironment(ctx context.Context) (*Environment, error) {
	url := fmt.Sprintf("%s/%s", apiBaseURL, defaultEnvironmentName)
	req, err := http.NewRequestWithContext(ctx, http.MethodGet, url, nil)
	if err != nil {
		return nil, fmt.Errorf("failed to create request: %w", err)
	}

	resp, err := c.httpClient.Do(req)
	if err != nil {
		return nil, fmt.Errorf("request to cloudshell failed: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		body, _ := io.ReadAll(resp.Body)
		return nil, fmt.Errorf("failed to get environment (%d): %s", resp.StatusCode, string(body))
	}

	var env Environment
	if err := json.NewDecoder(resp.Body).Decode(&env); err != nil {
		return nil, fmt.Errorf("failed to decode environment response: %w", err)
	}

	return &env, nil
}

// AddPublicKey adds the user's public SSH key to the Cloud Shell environment
func (c *Client) AddPublicKey(ctx context.Context, publicKey string) (*Operation, error) {
	url := fmt.Sprintf("%s/%s:addPublicKey", apiBaseURL, defaultEnvironmentName)

	payload := map[string]string{"key": strings.TrimSpace(publicKey)}
	data, err := json.Marshal(payload)
	if err != nil {
		return nil, err
	}

	req, err := http.NewRequestWithContext(ctx, http.MethodPost, url, bytes.NewReader(data))
	if err != nil {
		return nil, err
	}
	req.Header.Set("Content-Type", "application/json")

	resp, err := c.httpClient.Do(req)
	if err != nil {
		return nil, fmt.Errorf("failed to send public key: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		body, _ := io.ReadAll(resp.Body)
		return nil, fmt.Errorf("add public key failed (%d): %s", resp.StatusCode, string(body))
	}

	var op Operation
	if err := json.NewDecoder(resp.Body).Decode(&op); err != nil {
		return nil, err
	}

	return &op, nil
}

// StartEnvironment requests Cloud Shell to boot up the VM
func (c *Client) StartEnvironment(ctx context.Context, accessToken string) (*Operation, error) {
	url := fmt.Sprintf("%s/%s:start", apiBaseURL, defaultEnvironmentName)

	payload := map[string]string{}
	if accessToken != "" {
		payload["accessToken"] = accessToken
	}
	data, err := json.Marshal(payload)
	if err != nil {
		return nil, err
	}

	req, err := http.NewRequestWithContext(ctx, http.MethodPost, url, bytes.NewReader(data))
	if err != nil {
		return nil, err
	}
	req.Header.Set("Content-Type", "application/json")

	resp, err := c.httpClient.Do(req)
	if err != nil {
		return nil, fmt.Errorf("start environment request failed: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		body, _ := io.ReadAll(resp.Body)
		return nil, fmt.Errorf("start environment failed (%d): %s", resp.StatusCode, string(body))
	}

	var op Operation
	if err := json.NewDecoder(resp.Body).Decode(&op); err != nil {
		return nil, err
	}

	return &op, nil
}

// AuthorizeEnvironment pushes the user's OAuth credentials into the running
// Cloud Shell session so in-shell gcloud commands work without a manual login.
// The call is synchronous and returns once the session accepts the token.
func (c *Client) AuthorizeEnvironment(ctx context.Context) error {
	tok, err := c.tokenSource.Token()
	if err != nil {
		return fmt.Errorf("failed to load access token: %w", err)
	}

	url := fmt.Sprintf("%s/%s:authorize", apiBaseURL, defaultEnvironmentName)
	data, err := json.Marshal(map[string]string{"accessToken": tok.AccessToken})
	if err != nil {
		return err
	}

	req, err := http.NewRequestWithContext(ctx, http.MethodPost, url, bytes.NewReader(data))
	if err != nil {
		return err
	}
	req.Header.Set("Content-Type", "application/json")

	resp, err := c.httpClient.Do(req)
	if err != nil {
		return fmt.Errorf("authorize environment request failed: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		body, _ := io.ReadAll(resp.Body)
		return fmt.Errorf("authorize environment failed (%d): %s", resp.StatusCode, string(body))
	}

	return nil
}

// WaitForOperation polls a Google LRO until it reports done
func (c *Client) WaitForOperation(ctx context.Context, opName string) error {
	ticker := time.NewTicker(1 * time.Second)
	defer ticker.Stop()

	for {
		select {
		case <-ctx.Done():
			return ctx.Err()
		case <-ticker.C:
			url := fmt.Sprintf("%s/%s", apiBaseURL, opName)
			req, err := http.NewRequestWithContext(ctx, http.MethodGet, url, nil)
			if err != nil {
				return err
			}

			resp, err := c.httpClient.Do(req)
			if err != nil {
				return err
			}

			var op Operation
			err = json.NewDecoder(resp.Body).Decode(&op)
			resp.Body.Close()
			if err != nil {
				return err
			}

			if op.Done {
				return operationError(&op)
			}
		}
	}
}

// waitForRunning polls the environment until it reports RUNNING. The start
// operation can complete before the machine is actually reachable.
func (c *Client) waitForRunning(ctx context.Context) (*Environment, error) {
	const timeout = 90 * time.Second

	deadline := time.Now().Add(timeout)
	ticker := time.NewTicker(2 * time.Second)
	defer ticker.Stop()

	for {
		env, err := c.GetEnvironment(ctx)
		if err != nil {
			return nil, err
		}
		if env.State == "RUNNING" {
			return env, nil
		}
		if time.Now().After(deadline) {
			return nil, fmt.Errorf("cloud shell did not start (state: %s)", strings.ToLower(env.State))
		}

		select {
		case <-ctx.Done():
			return nil, ctx.Err()
		case <-ticker.C:
		}
	}
}

// Prepare ensures the key is authorized and the machine is running, returning SSH connection info.
// onStage, if non-nil, is called before each long-running stage so the caller can render progress.
func (c *Client) Prepare(ctx context.Context, onStage func(string)) (*ConnectionInfo, error) {
	if onStage == nil {
		onStage = func(string) {}
	}

	keyMgr, err := NewSSHKeyManager()
	if err != nil {
		return nil, err
	}

	signer, pubKeyStr, err := keyMgr.GetOrCreateKey()
	if err != nil {
		return nil, fmt.Errorf("failed to initialize SSH keys: %w", err)
	}

	env, err := c.GetEnvironment(ctx)
	if err != nil {
		return nil, err
	}

	// 1. Check if public key is already registered with Cloud Shell
	hasKey := false
	for _, candidate := range env.PublicKeys {
		if strings.TrimSpace(candidate) == pubKeyStr {
			hasKey = true
			break
		}
	}

	if !hasKey {
		op, err := c.AddPublicKey(ctx, pubKeyStr)
		if err != nil {
			return nil, err
		}
		if err := operationError(op); err != nil {
			return nil, err
		}
		if !op.Done {
			if err := c.WaitForOperation(ctx, op.Name); err != nil {
				return nil, err
			}
		}
	}

	// 2. If machine is not running, start it
	if env.State != "RUNNING" {
		onStage("Starting your Cloud Shell machine")

		tok, tokErr := c.tokenSource.Token()
		var accessToken string
		if tokErr == nil {
			accessToken = tok.AccessToken
		}

		op, err := c.StartEnvironment(ctx, accessToken)
		if err != nil {
			return nil, err
		}

		if err := operationError(op); err != nil {
			return nil, err
		}

		if !op.Done {
			if err := c.WaitForOperation(ctx, op.Name); err != nil {
				return nil, err
			}
		}

		// The start operation can finish before the machine is reachable, so
		// wait for the environment to report RUNNING before using it
		env, err = c.waitForRunning(ctx)
		if err != nil {
			return nil, err
		}
	}

	// 3. Refresh credentials in the session so in-shell gcloud commands are
	// authenticated on every connect, not only when this run started the VM.
	if err := c.AuthorizeEnvironment(ctx); err != nil {
		return nil, err
	}

	if env.SSHHost == "" {
		return nil, fmt.Errorf("cloud shell is running but returned no SSH host")
	}

	port := env.SSHPort
	if port <= 0 {
		port = 22
	}

	return &ConnectionInfo{
		Host:     env.SSHHost,
		Port:     port,
		Username: env.SSHUsername,
		KeyPath:  keyMgr.PrivateKeyPath(),
		Signer:   signer,
	}, nil
}
