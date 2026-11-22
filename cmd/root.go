package cmd

import (
	"errors"
	"fmt"
	"os"

	"gshell/internal/auth"
	gshellSSH "gshell/internal/ssh"
	appui "gshell/internal/ui"

	"github.com/spf13/cobra"
)

var (
	rootUI      *appui.UI
	projectFlag string
	uiOpts      struct {
		verbose bool
		noColor bool
	}

	// version is the release version. It can be overridden at build time with
	// -ldflags "-X gshell/cmd.version=<version>".
	version = "2.0.0"
)

// getUI returns the shared UI, initialized after flags have been parsed.
func getUI() *appui.UI {
	if rootUI == nil {
		rootUI = appui.New(appui.Options{
			Verbose: uiOpts.verbose,
			NoColor: uiOpts.noColor,
		})
	}
	return rootUI
}

var rootCmd = &cobra.Command{
	Use:   "gshell",
	Short: "Terminal client for Google Cloud Shell",
	Example: `  gshell                          # open an interactive session
  gshell run "git status"         # run a command in Cloud Shell
  gshell status                   # show environment details
  gshell cp ./project remote:~/   # copy files to Cloud Shell`,
	Version:       version,
	SilenceUsage:  true,
	SilenceErrors: true,
	RunE: func(cmd *cobra.Command, args []string) error {
		project, err := resolveProject()
		if err != nil {
			return err
		}

		sshClient, err := connectCloudShell(cmd.Context())
		if err != nil {
			return err
		}
		defer sshClient.Close()

		return gshellSSH.InteractiveSession(sshClient, project)
	},
}

func init() {
	rootCmd.PersistentFlags().BoolVar(&uiOpts.verbose, "verbose", false, "Show detailed output")
	rootCmd.PersistentFlags().BoolVar(&uiOpts.noColor, "no-color", false, "Disable colored output")
	rootCmd.PersistentFlags().StringVar(&projectFlag, "project", "", "Cloud Shell project to activate (defaults to GSHELL_PROJECT or saved config)")
}

// Execute adds all child commands to the root command and sets flags appropriately.
// This is called by main.main(). It only needs to happen once to the rootCmd.
func Execute() {
	if err := rootCmd.Execute(); err != nil {
		if errors.Is(err, auth.ErrNoCredentials) {
			fmt.Fprintln(os.Stderr, "Not authenticated.")
			fmt.Fprintln(os.Stderr, "Run: gshell auth login")
		} else {
			fmt.Fprintln(os.Stderr, "Error:", err)
		}
		os.Exit(1)
	}
}
