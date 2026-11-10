package cmd

import (
	"context"
	"fmt"

	"gshell/internal/auth"
	"gshell/internal/cloudshell"
	gshellSSH "gshell/internal/ssh"

	"golang.org/x/crypto/ssh"
)

// connectCloudShell prepares the environment and establishes an authenticated SSH connection
func connectCloudShell(ctx context.Context) (*ssh.Client, error) {
	tokenSource, err := auth.GetTokenSource(ctx)
	if err != nil {
		return nil, err
	}

	csClient := cloudshell.NewClient(ctx, tokenSource)

	connInfo, err := csClient.Prepare(ctx)
	if err != nil {
		return nil, fmt.Errorf("failed to prepare cloud shell session: %w", err)
	}

	sshClient, err := gshellSSH.Dial(connInfo)
	if err != nil {
		return nil, err
	}

	return sshClient, nil
}
