package cmd

import (
	"fmt"
	"strings"

	"github.com/spf13/cobra"
)

// Flags for sync command
var (
	syncRecursive bool
	syncDryRun    bool
)

// syncCmd represents the sync command
var syncCmd = &cobra.Command{
	Use:   "sync [flags] <source> [destination]",
	Short: "Sync changed files to Cloud Shell",
	Long: `Sync local files or directories to Cloud Shell using checksum comparisons.
Only files that have changed will be transferred.

Example:
  gshell sync ./myproject remote:~/myproject -r
  gshell sync config.json remote:~/config.json`,
	Args: cobra.MinimumNArgs(1),
	RunE: func(cmd *cobra.Command, args []string) error {
		source := args[0]
		dest := "remote:~/"

		if len(args) > 1 {
			dest = args[1]
		}

		remoteDest := strings.TrimPrefix(dest, "remote:")

		if syncDryRun {
			fmt.Printf("[dry-run] Would sync %s to remote:%s (recursive: %v)\n", source, remoteDest, syncRecursive)
			return nil
		}

		fmt.Printf("Syncing %s to remote:%s (recursive: %v)...\n", source, remoteDest, syncRecursive)
		// Phase 4 will call the checksum diff + transfer engine here
		return nil
	},
}

func init() {
	// Register flags for sync
	syncCmd.Flags().BoolVarP(&syncRecursive, "recurse", "r", false, "Sync directories recursively")
	syncCmd.Flags().BoolVarP(&syncDryRun, "dry-run", "n", false, "Preview which files would be synced without transferring")

	// Attach to root command
	rootCmd.AddCommand(syncCmd)
}
