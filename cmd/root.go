package cmd

import (
	"fmt"
	"os"

	gshellSSH "gshell/internal/ssh"

	"github.com/spf13/cobra"
)

var rootCmd = &cobra.Command{
	Use:          "gshell",
	Short:        "Terminal client for Google Cloud Shell",
	Version:      "2.0.0",
	SilenceUsage: true,
	RunE: func(cmd *cobra.Command, args []string) error {
		sshClient, err := connectCloudShell(cmd.Context())
		if err != nil {
			return err
		}
		defer sshClient.Close()

		return gshellSSH.InteractiveSession(sshClient)
	},
}

// Execute adds all child commands to the root command and sets flags appropriately.
// This is called by main.main(). It only needs to happen once to the rootCmd.
func Execute() {
	if err := rootCmd.Execute(); err != nil {
		fmt.Fprintln(os.Stderr, err)
		os.Exit(1)
	}
}
