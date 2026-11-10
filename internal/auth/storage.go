package auth

import (
	"encoding/json"
	"errors"
	"fmt"
	"os"
	"path/filepath"

	"golang.org/x/oauth2"
)

var (
	// ErrNoCredentials indicates that no stored credentials file was found
	ErrNoCredentials = errors.New("no saved credentials found, please login first")
)

// getConfigFilePath returns ~/.config/gshell/credentials.json across OSes
func getConfigFilePath() (string, error) {
	configDir, err := os.UserConfigDir()
	if err != nil {
		// Fallback to user home directory if UserConfigDir fails
		homeDir, homeErr := os.UserHomeDir()
		if homeErr != nil {
			return "", fmt.Errorf("unable to determine user configuration path: %w", err)
		}
		configDir = filepath.Join(homeDir, ".config")
	}

	return filepath.Join(configDir, "gshell", "credentials.json"), nil
}

// SaveToken persists an OAuth token securely with 0600 permissions
func SaveToken(token *oauth2.Token) error {
	filePath, err := getConfigFilePath()
	if err != nil {
		return err
	}

	// Ensure the ~/.config/gshell directory exists with owner-only access
	dir := filepath.Dir(filePath)
	if err := os.MkdirAll(dir, 0700); err != nil {
		return fmt.Errorf("failed to create config directory: %w", err)
	}

	data, err := json.MarshalIndent(token, "", "  ")
	if err != nil {
		return fmt.Errorf("failed to encode credentials: %w", err)
	}

	// 0600 ensures only the current user can read or write the credentials
	if err := os.WriteFile(filePath, data, 0600); err != nil {
		return fmt.Errorf("failed to write credentials file: %w", err)
	}

	return nil
}

// LoadToken reads and parses the stored OAuth token if present
func LoadToken() (*oauth2.Token, error) {
	filePath, err := getConfigFilePath()
	if err != nil {
		return nil, err
	}

	data, err := os.ReadFile(filePath)
	if err != nil {
		if os.IsNotExist(err) {
			return nil, ErrNoCredentials
		}
		return nil, fmt.Errorf("failed to read credentials file: %w", err)
	}

	var token oauth2.Token
	if err := json.Unmarshal(data, &token); err != nil {
		return nil, fmt.Errorf("failed to parse credentials file: %w", err)
	}

	return &token, nil
}

// DeleteToken removes the stored credentials file
func DeleteToken() error {
	filePath, err := getConfigFilePath()
	if err != nil {
		return err
	}

	if err := os.Remove(filePath); err != nil && !os.IsNotExist(err) {
		return fmt.Errorf("failed to remove credentials file: %w", err)
	}

	return nil
}
