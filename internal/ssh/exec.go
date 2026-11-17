package ssh

import (
	"fmt"
	"os"

	"golang.org/x/crypto/ssh"
)

// RunCommand executes a single command on Cloud Shell and streams stdout/stderr.
// If project is set, it is exported for the command so gcloud and client
// libraries use it.
func RunCommand(client *ssh.Client, command, project string) error {
	session, err := client.NewSession()
	if err != nil {
		return fmt.Errorf("failed to create SSH session: %w", err)
	}
	defer session.Close()

	if project != "" {
		command = fmt.Sprintf(
			"export DEVSHELL_PROJECT_ID=%s GOOGLE_CLOUD_PROJECT=%s GOOGLE_CLOUD_QUOTA_PROJECT=%s CLOUDSDK_CORE_PROJECT=%s; %s",
			project, project, project, project, command,
		)
	}

	session.Stdin = os.Stdin
	session.Stdout = os.Stdout
	session.Stderr = os.Stderr

	if err := session.Run(command); err != nil {
		return err
	}

	return nil
}
