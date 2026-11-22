package cmd

import (
	"github.com/spf13/cobra"
)

var mountCmd = &cobra.Command{
	Use:   "get-mount-command <mount-dir>",
	Short: "Print the sshfs command to mount Cloud Shell",
	Long: `Print the sshfs command that mounts your Cloud Shell home directory at the
given local path.

gshell does not run the command. sshfs and FUSE must be installed, and the user
must have permission to mount.`,
	Args: cobra.ExactArgs(1),
	RunE: func(cmd *cobra.Command, args []string) error {
		connInfo, err := prepareCloudShell(cmd.Context())
		if err != nil {
			return err
		}

		getUI().Printf(
			"sshfs %s@%s: %s -p %d -oIdentityFile=%s -oStrictHostKeyChecking=no\n",
			connInfo.Username, connInfo.Host, args[0], connInfo.Port, connInfo.KeyPath,
		)
		return nil
	},
}

func init() {
	rootCmd.AddCommand(mountCmd)
}
