package ssh

import (
	"fmt"
	"time"

	"gshell/internal/cloudshell"

	"golang.org/x/crypto/ssh"
)

// Dial establishes an authenticated SSH connection to Google Cloud Shell
func Dial(info *cloudshell.ConnectionInfo) (*ssh.Client, error) {
	config := &ssh.ClientConfig{
		User: info.Username,
		Auth: []ssh.AuthMethod{
			ssh.PublicKeys(info.Signer),
		},
		// Cloud Shell assigns a dynamic host:port and regenerates the VM's host key
		// when the environment is recreated, so a known_hosts entry would fail on
		// legitimate reconnects. gcloud's cloud-shell ssh behaves the same way.
		HostKeyCallback: ssh.InsecureIgnoreHostKey(),
		Timeout:         30 * time.Second,
	}

	addr := fmt.Sprintf("%s:%d", info.Host, info.Port)
	client, err := ssh.Dial("tcp", addr, config)
	if err != nil {
		return nil, fmt.Errorf("failed to connect to cloud shell at %s: %w", addr, err)
	}

	return client, nil
}
