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

// Operation represents a Google Long Running Operation (LRO)
type Operation struct {
	Name     string          `json:"name"`
	Done     bool            `json:"done"`
	Error    *OperationError `json:"error,omitempty"`
	Response *Environment    `json:"response,omitempty"`
}

// OperationError holds error details from a failed LRO
type OperationError struct {
	Code    int    `json:"code"`
	Message string `json:"message"`
}

// ConnectionInfo contains all parameters needed to open an SSH or SFTP connection
type ConnectionInfo struct {
	Host     string
	Port     int
	Username string
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

// WaitForOperation polls a Google LRO until it reports done
func (c *Client) WaitForOperation(ctx context.Context, opName string, statusMsg string) error {
	ticker := time.NewTicker(1 * time.Second)
	defer ticker.Stop()

	if statusMsg != "" {
		fmt.Printf("%s...", statusMsg)
		defer fmt.Println(" done.")
	}

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
				if op.Error != nil {
					return fmt.Errorf("operation error (%d): %s", op.Error.Code, op.Error.Message)
				}
				return nil
			}
		}
	}
}

// Prepare ensures the key is authorized and the machine is running, returning SSH connection info
func (c *Client) Prepare(ctx context.Context) (*ConnectionInfo, error) {
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
		if !op.Done {
			if err := c.WaitForOperation(ctx, op.Name, "Registering SSH key with Cloud Shell"); err != nil {
				return nil, err
			}
		}
	}

	// 2. If machine is not running, start it
	if env.State != "RUNNING" {
		tok, tokErr := c.tokenSource.Token()
		var accessToken string
		if tokErr == nil {
			accessToken = tok.AccessToken
		}

		op, err := c.StartEnvironment(ctx, accessToken)
		if err != nil {
			return nil, err
		}

		if !op.Done {
			if err := c.WaitForOperation(ctx, op.Name, "Starting Cloud Shell machine"); err != nil {
				return nil, err
			}
		}

		// Refresh environment details after boot to get host and port
		env, err = c.GetEnvironment(ctx)
		if err != nil {
			return nil, fmt.Errorf("failed to refresh environment after start: %w", err)
		}
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
		Signer:   signer,
	}, nil
}
