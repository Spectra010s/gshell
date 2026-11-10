package ssh

import (
	"crypto/md5"
	"encoding/hex"
	"fmt"
	"io"
	"os"
	"path"
	"path/filepath"
	"strings"

	"github.com/pkg/sftp"
	"golang.org/x/crypto/ssh"
)

// SFTPClient wraps the SFTP connection for file transfers
type SFTPClient struct {
	client *sftp.Client
}

// NewSFTPClient initializes an SFTP session over the existing SSH connection
func NewSFTPClient(sshClient *ssh.Client) (*SFTPClient, error) {
	client, err := sftp.NewClient(sshClient)
	if err != nil {
		return nil, fmt.Errorf("failed to initialize SFTP subsystem: %w", err)
	}
	return &SFTPClient{client: client}, nil
}

// Close terminates the SFTP session
func (s *SFTPClient) Close() error {
	return s.client.Close()
}

// resolveRemotePath handles ~ expansion relative to Cloud Shell home directory
func (s *SFTPClient) resolveRemotePath(remotePath string) string {
	remotePath = strings.TrimPrefix(remotePath, "remote:")
	if remotePath == "~" || remotePath == "~/" {
		return "."
	}
	if strings.HasPrefix(remotePath, "~/") {
		return strings.TrimPrefix(remotePath, "~/")
	}
	return remotePath
}

// Upload transfers a single file or directory from local machine to Cloud Shell
func (s *SFTPClient) Upload(localPath, remotePath string, recurse, dryRun bool) error {
	localStat, err := os.Stat(localPath)
	if err != nil {
		return fmt.Errorf("local file not found: %w", err)
	}

	targetPath := s.resolveRemotePath(remotePath)

	if localStat.IsDir() {
		if !recurse {
			return fmt.Errorf("%s is a directory (use -r / --recurse to upload)", localPath)
		}
		return s.uploadDir(localPath, targetPath, dryRun)
	}

	return s.uploadFile(localPath, targetPath, dryRun)
}

func (s *SFTPClient) uploadFile(localPath, remotePath string, dryRun bool) error {
	// If remotePath is a directory, append local filename
	if stat, err := s.client.Stat(remotePath); err == nil && stat.IsDir() {
		remotePath = path.Join(remotePath, filepath.Base(localPath))
	}

	if dryRun {
		fmt.Printf("[dry-run] upload: %s -> %s\n", localPath, remotePath)
		return nil
	}

	srcFile, err := os.Open(localPath)
	if err != nil {
		return err
	}
	defer srcFile.Close()

	// Ensure remote parent directory exists
	remoteDir := path.Dir(remotePath)
	if remoteDir != "." && remoteDir != "/" {
		_ = s.client.MkdirAll(remoteDir)
	}

	dstFile, err := s.client.Create(remotePath)
	if err != nil {
		return fmt.Errorf("failed to create remote file %s: %w", remotePath, err)
	}
	defer dstFile.Close()

	n, err := io.Copy(dstFile, srcFile)
	if err != nil {
		return fmt.Errorf("failed to transfer data: %w", err)
	}

	fmt.Printf("Uploaded %s (%d bytes)\n", localPath, n)
	return nil
}

func (s *SFTPClient) uploadDir(localDir, remoteDir string, dryRun bool) error {
	return filepath.Walk(localDir, func(currentLocal string, info os.FileInfo, err error) error {
		if err != nil {
			return err
		}

		relPath, err := filepath.Rel(localDir, currentLocal)
		if err != nil {
			return err
		}

		currentRemote := path.Join(remoteDir, filepath.ToSlash(relPath))

		if info.IsDir() {
			if dryRun {
				fmt.Printf("[dry-run] mkdir: %s\n", currentRemote)
				return nil
			}
			return s.client.MkdirAll(currentRemote)
		}

		return s.uploadFile(currentLocal, currentRemote, dryRun)
	})
}

// Download transfers a file or directory from Cloud Shell to local machine
func (s *SFTPClient) Download(remotePath, localPath string, recurse, dryRun bool) error {
	resolvedRemote := s.resolveRemotePath(remotePath)

	remoteStat, err := s.client.Stat(resolvedRemote)
	if err != nil {
		return fmt.Errorf("remote path not found: %w", err)
	}

	if remoteStat.IsDir() {
		if !recurse {
			return fmt.Errorf("remote path %s is a directory (use -r / --recurse)", remotePath)
		}
		return s.downloadDir(resolvedRemote, localPath, dryRun)
	}

	return s.downloadFile(resolvedRemote, localPath, dryRun)
}

func (s *SFTPClient) downloadFile(remotePath, localPath string, dryRun bool) error {
	// If localPath is an existing directory, append the remote base name
	if stat, err := os.Stat(localPath); err == nil && stat.IsDir() {
		localPath = filepath.Join(localPath, path.Base(remotePath))
	}

	if dryRun {
		fmt.Printf("[dry-run] download: %s -> %s\n", remotePath, localPath)
		return nil
	}

	srcFile, err := s.client.Open(remotePath)
	if err != nil {
		return fmt.Errorf("failed to open remote file: %w", err)
	}
	defer srcFile.Close()

	if err := os.MkdirAll(filepath.Dir(localPath), 0755); err != nil {
		return err
	}

	dstFile, err := os.Create(localPath)
	if err != nil {
		return fmt.Errorf("failed to create local file: %w", err)
	}
	defer dstFile.Close()

	n, err := io.Copy(dstFile, srcFile)
	if err != nil {
		return fmt.Errorf("failed to write local file: %w", err)
	}

	fmt.Printf("Downloaded %s (%d bytes)\n", localPath, n)
	return nil
}

func (s *SFTPClient) downloadDir(remoteDir, localDir string, dryRun bool) error {
	walker := s.client.Walk(remoteDir)
	for walker.Step() {
		if err := walker.Err(); err != nil {
			return err
		}

		relPath, err := filepath.Rel(remoteDir, walker.Path())
		if err != nil {
			return err
		}

		currentLocal := filepath.Join(localDir, filepath.FromSlash(relPath))
		stat := walker.Stat()

		if stat.IsDir() {
			if dryRun {
				fmt.Printf("[dry-run] mkdir local: %s\n", currentLocal)
				continue
			}
			if err := os.MkdirAll(currentLocal, 0755); err != nil {
				return err
			}
			continue
		}

		if err := s.downloadFile(walker.Path(), currentLocal, dryRun); err != nil {
			return err
		}
	}
	return nil
}

// Sync uploads only modified or missing files by comparing checksums over the open connection
func (s *SFTPClient) Sync(localPath, remotePath string, recurse, dryRun bool) error {
	localStat, err := os.Stat(localPath)
	if err != nil {
		return err
	}

	targetRemote := s.resolveRemotePath(remotePath)

	if !localStat.IsDir() {
		return s.syncSingleFile(localPath, targetRemote, dryRun)
	}

	if !recurse {
		return fmt.Errorf("%s is a directory (use -r / --recurse to sync)", localPath)
	}

	return filepath.Walk(localPath, func(currentLocal string, info os.FileInfo, err error) error {
		if err != nil {
			return err
		}

		relPath, err := filepath.Rel(localPath, currentLocal)
		if err != nil {
			return err
		}

		currentRemote := path.Join(targetRemote, filepath.ToSlash(relPath))

		if info.IsDir() {
			if !dryRun {
				_ = s.client.MkdirAll(currentRemote)
			}
			return nil
		}

		return s.syncSingleFile(currentLocal, currentRemote, dryRun)
	})
}

func (s *SFTPClient) syncSingleFile(localFile, remoteFile string, dryRun bool) error {
	localMD5, err := calculateMD5(localFile)
	if err != nil {
		return err
	}

	remoteStat, err := s.client.Stat(remoteFile)
	if err == nil {
		localInfo, _ := os.Stat(localFile)
		// Quick check: if size matches, check remote checksum
		if remoteStat.Size() == localInfo.Size() {
			remoteMD5, err := s.getRemoteMD5(remoteFile)
			if err == nil && localMD5 == remoteMD5 {
				// File is identical, skip transfer!
				return nil
			}
		}
	}

	// File is different or missing -> upload
	return s.uploadFile(localFile, remoteFile, dryRun)
}

func (s *SFTPClient) getRemoteMD5(remoteFile string) (string, error) {
	file, err := s.client.Open(remoteFile)
	if err != nil {
		return "", err
	}
	defer file.Close()

	hash := md5.New()
	if _, err := io.Copy(hash, file); err != nil {
		return "", err
	}

	return hex.EncodeToString(hash.Sum(nil)), nil
}

func calculateMD5(filePath string) (string, error) {
	file, err := os.Open(filePath)
	if err != nil {
		return "", err
	}
	defer file.Close()

	hash := md5.New()
	if _, err := io.Copy(hash, file); err != nil {
		return "", err
	}

	return hex.EncodeToString(hash.Sum(nil)), nil
}
