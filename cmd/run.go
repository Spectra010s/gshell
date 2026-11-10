package cmd

import (
	"strings"

	gshellSSH "gshell/internal/ssh"

	"github.com/spf13/cobra"
)

// runCmd represents the run command
var runCmd = &cobra.Command{
	Use:   "run <command...>",
	Short: "Run a command on Cloud Shell",
	Long: `Execute arbitrary commands remotely on Google Cloud Shell and stream the output back.

Examples:
  gshell run ls -la
  gshell run "cd myproject && git status"`,
	Args: cobra.MinimumNArgs(1),
	RunE: func(cmd *cobra.Command, args []string) error {
		sshClient, err := connectCloudShell(cmd.Context())
		if err != nil {
			return err
		}
		defer sshClient.Close()

		remoteCommand := strings.Join(args, " ")
		return gshellSSH.RunCommand(sshClient, remoteCommand)
	},
}

func init() {
	rootCmd.AddCommand(runCmd)
}
