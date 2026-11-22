package ssh

import (
	"fmt"
	"io"
	"os"
	"os/signal"
	"syscall"

	"golang.org/x/crypto/ssh"
	"golang.org/x/term"
)

// InteractiveSession allocates a pseudo-terminal (PTY) and connects standard I/O
func InteractiveSession(client *ssh.Client, project string) error {
	session, err := client.NewSession()
	if err != nil {
		return fmt.Errorf("failed to create SSH session: %w", err)
	}
	defer session.Close()

	stdinFd := int(os.Stdin.Fd())

	// Default terminal window dimensions
	width, height := 80, 24
	if term.IsTerminal(stdinFd) {
		if w, h, err := term.GetSize(stdinFd); err == nil {
			width, height = w, h
		}

		// Put terminal in raw mode so keystrokes, Ctrl+C, and Tab pass directly to Cloud Shell
		oldState, err := term.MakeRaw(stdinFd)
		if err != nil {
			return fmt.Errorf("failed to set raw terminal mode: %w", err)
		}
		defer term.Restore(stdinFd, oldState)
	}

	// Request pseudo-terminal from Cloud Shell
	modes := ssh.TerminalModes{
		ssh.ECHO:          1,
		ssh.TTY_OP_ISPEED: 14400,
		ssh.TTY_OP_OSPEED: 14400,
	}

	termType := os.Getenv("TERM")
	if termType == "" {
		termType = "xterm-256color"
	}

	if err := session.RequestPty(termType, height, width, modes); err != nil {
		return fmt.Errorf("failed to allocate PTY: %w", err)
	}

	session.Stdin = os.Stdin
	session.Stdout = os.Stdout
	session.Stderr = os.Stderr

	// Watch for window resize signals (SIGWINCH) and update remote PTY
	sigChan := make(chan os.Signal, 1)
	signal.Notify(sigChan, syscall.SIGWINCH)
	go func() {
		for range sigChan {
			if w, h, err := term.GetSize(stdinFd); err == nil {
				_ = session.WindowChange(h, w)
			}
		}
	}()
	defer signal.Stop(sigChan)

	// Start the remote interactive shell. If a project is selected, pass it the
	// same way gcloud does so Cloud Shell configures gcloud for the session.
	if project != "" {
		err = session.Start(fmt.Sprintf("DEVSHELL_PROJECT_ID=%s bash -l", project))
	} else {
		err = session.Shell()
	}
	if err != nil {
		return fmt.Errorf("failed to start shell: %w", err)
	}

	if err := session.Wait(); err != nil {
		// Exit status is normal on shell exit, ignore ExitMissingError
		if err != io.EOF {
			return err
		}
	}

	return nil
}
