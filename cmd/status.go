package cmd

import (
	"fmt"

	"github.com/spf13/cobra"
)

// statusCmd represents the status command
var statusCmd = &cobra.Command{
	Use:   "status",
	Short: "Show current Cloud Shell account and project info",
	Long:  `Display the active authenticated Google account, project ID, and Cloud Shell environment state.`,
	RunE: func(cmd *cobra.Command, args []string) error {
		fmt.Println("Fetching Cloud Shell status...")
		// Phase 3 will query cloudshell.googleapis.com environment info here
		fmt.Println("Environment: (pending auth)")
		return nil
	},
}

func init() {
	rootCmd.AddCommand(statusCmd)
}
