package cmd

import (
	gshellSSH "gshell/internal/ssh"

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

		sshClient, err := connectCloudShell(cmd.Context())
		if err != nil {
			return err
		}
		defer sshClient.Close()

		sftpClient, err := gshellSSH.NewSFTPClient(sshClient)
		if err != nil {
			return err
		}
		defer sftpClient.Close()

		return sftpClient.Sync(source, dest, syncRecursive, syncDryRun)
	},
}

func init() {
	// Register flags for sync
	syncCmd.Flags().BoolVarP(&syncRecursive, "recurse", "r", false, "Sync directories recursively")
	syncCmd.Flags().BoolVarP(&syncDryRun, "dry-run", "n", false, "Preview which files would be synced without transferring")

	// Attach to root command
	rootCmd.AddCommand(syncCmd)
}
