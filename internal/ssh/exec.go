package ssh

import (
	"fmt"
	"os"

	"golang.org/x/crypto/ssh"
)

// RunCommand executes a single command on Cloud Shell and streams stdout/stderr
func RunCommand(client *ssh.Client, command string) error {
	session, err := client.NewSession()
	if err != nil {
		return fmt.Errorf("failed to create SSH session: %w", err)
	}
	defer session.Close()

	session.Stdin = os.Stdin
	session.Stdout = os.Stdout
	session.Stderr = os.Stderr

	if err := session.Run(command); err != nil {
		return err
	}

	return nil
}
