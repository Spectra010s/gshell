package cmd

import (
	"fmt"
	"strings"

	gshellSSH "gshell/internal/ssh"

	"github.com/spf13/cobra"
	"golang.org/x/crypto/ssh"
)

var (
	cpRecursive bool
	cpExpand    bool
	cpSync      bool
	cpDryRun    bool
)

var cpCmd = &cobra.Command{
	Use:   "cp [flags] <sources...> <dest>",
	Short: "Copy files between local machine and Cloud Shell",
	Long: `Copy files between the local and remote file systems.

A 'remote:' prefix on any path indicates it refers to the Cloud Shell
file system, resolved relative to the remote home directory.

Direction is determined by where 'remote:' appears:
  Upload:   gshell cp local.txt remote:~/project/
  Download: gshell cp remote:~/file.txt ./local/

The --recursive flag is required if any source is a directory.

With the --expand flag, remote file names are treated as bash expressions,
subject to expansion of tildes, braces, globs, and environment variables.

With the --sync flag, only files that differ (by checksum) are transferred.

With the --dry-run flag, no files are transferred; the planned transfers are
printed instead.`,
	Args: cobra.MinimumNArgs(2),
	RunE: func(cmd *cobra.Command, args []string) error {
		dest := args[len(args)-1]
		sources := args[:len(args)-1]

		destIsRemote := strings.HasPrefix(dest, "remote:")
		sourceIsRemote := false
		for _, s := range sources {
			if strings.HasPrefix(s, "remote:") {
				sourceIsRemote = true
				break
			}
		}

		if destIsRemote && sourceIsRemote {
			return fmt.Errorf("both source and destination cannot be remote")
		}
		if !destIsRemote && !sourceIsRemote {
			return fmt.Errorf("either source or destination must use the 'remote:' prefix")
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

		if sourceIsRemote {
			// Download: remote -> local
			if cpExpand {
				expanded, err := expandRemotePaths(sshClient, sources)
				if err != nil {
					return err
				}
				sources = expanded
			}

			total := &gshellSSH.Result{}
			for _, src := range sources {
				var res *gshellSSH.Result
				if cpSync {
					res, err = sftpClient.SyncDown(src, dest, cpRecursive, cpDryRun)
				} else {
					res, err = sftpClient.Download(src, dest, cpRecursive, cpDryRun)
				}
				if err != nil {
					return err
				}
				total.Merge(res)
			}
			reportTransfer(getUI(), total, cpDryRun)
		} else {
			// Upload: local -> remote
			total := &gshellSSH.Result{}
			for _, src := range sources {
				var res *gshellSSH.Result
				if cpSync {
					res, err = sftpClient.Sync(src, dest, cpRecursive, cpDryRun)
				} else {
					res, err = sftpClient.Upload(src, dest, cpRecursive, cpDryRun)
				}
				if err != nil {
					return err
				}
				total.Merge(res)
			}
			reportTransfer(getUI(), total, cpDryRun)
		}

		return nil
	},
}

// expandRemotePaths runs bash expansion on remote path patterns via SSH
func expandRemotePaths(sshClient *ssh.Client, patterns []string) ([]string, error) {
	var expanded []string
	for _, pattern := range patterns {
		p := strings.TrimPrefix(pattern, "remote:")

		session, err := sshClient.NewSession()
		if err != nil {
			return nil, fmt.Errorf("failed to open SSH session for expansion: %w", err)
		}

		out, err := session.Output(fmt.Sprintf(`bash -c 'for f in %s; do [ -e "$f" ] && echo "$f"; done'`, p))
		session.Close()
		if err != nil {
			return nil, fmt.Errorf("failed to expand remote path %s: %w", pattern, err)
		}

		lines := strings.Split(strings.TrimSpace(string(out)), "\n")
		for _, line := range lines {
			line = strings.TrimSpace(line)
			if line != "" {
				expanded = append(expanded, "remote:"+line)
			}
		}
	}

	if len(expanded) == 0 {
		return nil, fmt.Errorf("no matching remote files found")
	}
	return expanded, nil
}

func init() {
	cpCmd.Flags().BoolVarP(&cpRecursive, "recursive", "r", false, "Recursively copy directories")
	cpCmd.Flags().BoolVarP(&cpExpand, "expand", "e", false, "Expand remote file names on remote shell")
	cpCmd.Flags().BoolVar(&cpSync, "sync", false, "Only transfer files that differ by checksum")
	cpCmd.Flags().BoolVarP(&cpDryRun, "dry-run", "n", false, "Print planned transfers without copying")
	rootCmd.AddCommand(cpCmd)
}
