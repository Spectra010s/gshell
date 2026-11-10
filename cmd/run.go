package cmd

import (
	"fmt"
	"strings"

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
		remoteCommand := strings.Join(args, " ")
		fmt.Printf("Running remotely on Cloud Shell: %s\n", remoteCommand)
		// Phase 4 will call SSH remote execution session here
		return nil
	},
}

func init() {
	rootCmd.AddCommand(runCmd)
}
