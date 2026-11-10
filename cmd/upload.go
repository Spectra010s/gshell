package cmd

import (
	gshellSSH "gshell/internal/ssh"

	"github.com/spf13/cobra"
)

// Flags for upload command
var (
	uploadRecursive bool
	uploadDryRun    bool
)

// uploadCmd represents the upload/send command
var uploadCmd = &cobra.Command{
	Use:     "upload [flags] <source...> [destination]",
	Aliases: []string{"send", "push"},
	Short:   "Upload files or folders to Cloud Shell",
	Long: `Upload local files or directories to Google Cloud Shell.

Destination paths can use the 'remote:' prefix (e.g. remote:~/myproject/).
If multiple sources are provided, the destination must be a directory.
If destination is omitted, files default to remote:~/`,
	Args: cobra.MinimumNArgs(1),
	RunE: func(cmd *cobra.Command, args []string) error {
		var dest string
		var sources []string

		if len(args) == 1 {
			// Default destination to Cloud Shell home when omitted
			sources = []string{args[0]}
			dest = "remote:~/"
		} else {
			// Last argument is destination, everything before is source files
			dest = args[len(args)-1]
			sources = args[:len(args)-1]
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

		for _, src := range sources {
			if err := sftpClient.Upload(src, dest, uploadRecursive, uploadDryRun); err != nil {
				return err
			}
		}

		return nil
	},
}

func init() {
	// Register flags for upload
	uploadCmd.Flags().BoolVarP(&uploadRecursive, "recurse", "r", false, "Upload directories recursively")
	uploadCmd.Flags().BoolVarP(&uploadDryRun, "dry-run", "n", false, "Preview upload without transferring")

	// Attach to root command
	rootCmd.AddCommand(uploadCmd)
}
