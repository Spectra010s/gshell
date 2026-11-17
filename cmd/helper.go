package cmd

import (
	"context"
	"fmt"
	"os"

	"gshell/internal/auth"
	"gshell/internal/cloudshell"
	appconfig "gshell/internal/config"
	gshellSSH "gshell/internal/ssh"
	appui "gshell/internal/ui"

	"golang.org/x/crypto/ssh"
)

// resolveProject returns the project gshell should activate in Cloud Shell.
// Precedence: --project, GSHELL_PROJECT, saved config.
func resolveProject() (string, error) {
	id := projectFlag
	if id == "" {
		id = os.Getenv("GSHELL_PROJECT")
	}
	if id == "" {
		cfg, err := appconfig.Load()
		if err != nil {
			return "", err
		}
		id = cfg.Project
	}

	if id == "" {
		return "", nil
	}
	if err := appconfig.ValidateProject(id); err != nil {
		return "", err
	}
	return id, nil
}

// connectCloudShell prepares the environment and establishes an authenticated SSH connection
func connectCloudShell(ctx context.Context) (*ssh.Client, error) {
	tokenSource, err := auth.GetTokenSource(ctx)
	if err != nil {
		return nil, err
	}

	csClient := cloudshell.NewClient(ctx, tokenSource)

	var step *appui.Step
	finish := func(ok bool) {
		if step == nil {
			return
		}
		if ok {
			step.Done()
		} else {
			step.Fail()
		}
		step = nil
	}

	connInfo, err := csClient.Prepare(ctx, func(msg string) {
		finish(true)
		step = getUI().Step(msg)
	})
	if err != nil {
		finish(false)
		return nil, fmt.Errorf("failed to prepare cloud shell session: %w", err)
	}
	finish(true)

	sshClient, err := gshellSSH.Dial(connInfo)
	if err != nil {
		return nil, err
	}

	return sshClient, nil
}
