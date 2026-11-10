package cloudshell

import (
	"crypto/ed25519"
	"crypto/rand"
	"encoding/pem"
	"errors"
	"fmt"
	"os"
	"path/filepath"
	"strings"

	"golang.org/x/crypto/ssh"
)

// SSHKeyManager manages the local Ed25519 keypair for Cloud Shell sessions
type SSHKeyManager struct {
	keyDir string
}

// NewSSHKeyManager initializes key storage in ~/.config/gshell/
func NewSSHKeyManager() (*SSHKeyManager, error) {
	configDir, err := os.UserConfigDir()
	if err != nil {
		homeDir, homeErr := os.UserHomeDir()
		if homeErr != nil {
			return nil, fmt.Errorf("unable to resolve home directory: %w", err)
		}
		configDir = filepath.Join(homeDir, ".config")
	}

	keyDir := filepath.Join(configDir, "gshell")
	if err := os.MkdirAll(keyDir, 0700); err != nil {
		return nil, fmt.Errorf("failed to create key directory: %w", err)
	}

	return &SSHKeyManager{keyDir: keyDir}, nil
}

// GetOrCreateKey loads an existing Ed25519 keypair or generates a new one
func (m *SSHKeyManager) GetOrCreateKey() (ssh.Signer, string, error) {
	privPath := filepath.Join(m.keyDir, "id_ed25519")
	pubPath := filepath.Join(m.keyDir, "id_ed25519.pub")

	// If keys exist on disk, read and return them
	if privBytes, err := os.ReadFile(privPath); err == nil {
		signer, parseErr := ssh.ParsePrivateKey(privBytes)
		if parseErr == nil {
			if pubBytes, err := os.ReadFile(pubPath); err == nil {
				return signer, strings.TrimSpace(string(pubBytes)), nil
			}
			// If public key file is missing, derive it from signer
			pubKeyStr := strings.TrimSpace(string(ssh.MarshalAuthorizedKey(signer.PublicKey())))
			_ = os.WriteFile(pubPath, []byte(pubKeyStr+"\n"), 0644)
			return signer, pubKeyStr, nil
		}
	}

	// Generate new Ed25519 keypair
	pubKey, privKey, err := ed25519.GenerateKey(rand.Reader)
	if err != nil {
		return nil, "", fmt.Errorf("failed to generate ed25519 key: %w", err)
	}

	// Convert private key to OpenSSH PEM format
	privBlock, err := ssh.MarshalPrivateKey(privKey, "gshell")
	if err != nil {
		return nil, "", fmt.Errorf("failed to encode private key: %w", err)
	}
	privPEM := pem.EncodeToMemory(privBlock)

	// Save private key with 0600 permissions
	if err := os.WriteFile(privPath, privPEM, 0600); err != nil {
		return nil, "", fmt.Errorf("failed to write private key: %w", err)
	}

	// Format public key in authorized_keys format
	sshPubKey, err := ssh.NewPublicKey(pubKey)
	if err != nil {
		return nil, "", fmt.Errorf("failed to convert public key: %w", err)
	}

	pubKeyStr := strings.TrimSpace(string(ssh.MarshalAuthorizedKey(sshPubKey)))
	if err := os.WriteFile(pubPath, []byte(pubKeyStr+"\n"), 0644); err != nil {
		return nil, "", fmt.Errorf("failed to write public key: %w", err)
	}

	signer, err := ssh.NewSignerFromKey(privKey)
	if err != nil {
		return nil, "", fmt.Errorf("failed to create signer: %w", err)
	}

	return signer, pubKeyStr, nil
}

// RemoveKeys cleans up the local keypair files
func (m *SSHKeyManager) RemoveKeys() error {
	privPath := filepath.Join(m.keyDir, "id_ed25519")
	pubPath := filepath.Join(m.keyDir, "id_ed25519.pub")

	var errs []error
	if err := os.Remove(privPath); err != nil && !os.IsNotExist(err) {
		errs = append(errs, err)
	}
	if err := os.Remove(pubPath); err != nil && !os.IsNotExist(err) {
		errs = append(errs, err)
	}

	if len(errs) > 0 {
		return errors.Join(errs...)
	}
	return nil
}
