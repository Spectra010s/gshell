package cmd

import (
	gshellSSH "gshell/internal/ssh"

	"github.com/spf13/cobra"
)

// Flags for download command
var (
	downloadRecursive bool
	downloadDryRun    bool
)

// downloadCmd represents the download/get command
var downloadCmd = &cobra.Command{
	Use:     "download [flags] <source...> [destination]",
	Aliases: []string{"get", "pull"},
	Short:   "Download files or folders from Cloud Shell",
	Long: `Download remote files or directories from Google Cloud Shell to your local machine.

Source paths can use the 'remote:' prefix (e.g. remote:~/logs/app.log).
If multiple sources are provided, the destination must be a local directory.
If destination is omitted, files download to your current local directory (./).`,
	Args: cobra.MinimumNArgs(1),
	RunE: func(cmd *cobra.Command, args []string) error {
		var dest string
		var sources []string

		if len(args) == 1 {
			// Default destination to current working directory when omitted
			sources = []string{args[0]}
			dest = "./"
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
			if err := sftpClient.Download(src, dest, downloadRecursive, downloadDryRun); err != nil {
				return err
			}
		}

		return nil
	},
}

func init() {
	// Register flags for download
	downloadCmd.Flags().BoolVarP(&downloadRecursive, "recurse", "r", false, "Download directories recursively")
	downloadCmd.Flags().BoolVarP(&downloadDryRun, "dry-run", "n", false, "Preview download without transferring")

	// Attach to root command
	rootCmd.AddCommand(downloadCmd)
}
