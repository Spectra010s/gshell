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

// Action describes one file copied or skipped.
type Action struct {
	Source  string
	Dest    string
	Skipped bool
}

// Result summarizes a copy operation.
type Result struct {
	Files   int
	Bytes   int64
	Skipped int
	Actions []Action
}

// Merge adds another result to this one.
func (r *Result) Merge(other *Result) {
	if other == nil {
		return
	}
	r.Files += other.Files
	r.Bytes += other.Bytes
	r.Skipped += other.Skipped
	r.Actions = append(r.Actions, other.Actions...)
}

func (r *Result) copied(src, dst string, size int64) {
	r.Files++
	r.Bytes += size
	r.Actions = append(r.Actions, Action{Source: src, Dest: dst})
}

func (r *Result) skip(src, dst string) {
	r.Skipped++
	r.Actions = append(r.Actions, Action{Source: src, Dest: dst, Skipped: true})
}

// remoteDisplay renders a resolved remote path the way the user would write it.
func remoteDisplay(p string) string {
	switch {
	case p == ".":
		return "remote:~"
	case strings.HasPrefix(p, "/"):
		return "remote:" + p
	default:
		return "remote:~/" + p
	}
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
func (s *SFTPClient) Upload(localPath, remotePath string, recurse, dryRun bool) (*Result, error) {
	localStat, err := os.Stat(localPath)
	if err != nil {
		return nil, fmt.Errorf("local file not found: %w", err)
	}

	targetPath := s.resolveRemotePath(remotePath)
	res := &Result{}

	if localStat.IsDir() {
		if !recurse {
			return nil, fmt.Errorf("%s is a directory (use -r / --recursive to upload)", localPath)
		}
		return res, s.uploadDir(localPath, targetPath, dryRun, res)
	}

	return res, s.uploadFile(localPath, targetPath, dryRun, res)
}

func (s *SFTPClient) uploadFile(localPath, remotePath string, dryRun bool, res *Result) error {
	// If remotePath looks like a directory (ends with / or is an existing dir), append local filename
	if strings.HasSuffix(remotePath, "/") {
		if !dryRun {
			_ = s.client.MkdirAll(remotePath)
		}
		remotePath = path.Join(remotePath, filepath.Base(localPath))
	} else if stat, err := s.client.Stat(remotePath); err == nil && stat.IsDir() {
		remotePath = path.Join(remotePath, filepath.Base(localPath))
	}

	var size int64
	if info, err := os.Stat(localPath); err == nil {
		size = info.Size()
	}

	if dryRun {
		res.copied(localPath, remoteDisplay(remotePath), size)
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

	res.copied(localPath, remoteDisplay(remotePath), n)
	return nil
}

func (s *SFTPClient) uploadDir(localDir, remoteDir string, dryRun bool, res *Result) error {
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
				return nil
			}
			return s.client.MkdirAll(currentRemote)
		}

		return s.uploadFile(currentLocal, currentRemote, dryRun, res)
	})
}

// Download transfers a file or directory from Cloud Shell to local machine
func (s *SFTPClient) Download(remotePath, localPath string, recurse, dryRun bool) (*Result, error) {
	resolvedRemote := s.resolveRemotePath(remotePath)

	remoteStat, err := s.client.Stat(resolvedRemote)
	if err != nil {
		return nil, fmt.Errorf("remote path not found: %w", err)
	}

	res := &Result{}
	if remoteStat.IsDir() {
		if !recurse {
			return nil, fmt.Errorf("remote path %s is a directory (use -r / --recursive)", remotePath)
		}
		return res, s.downloadDir(resolvedRemote, localPath, dryRun, res)
	}

	return res, s.downloadFile(resolvedRemote, localPath, dryRun, res)
}

func (s *SFTPClient) downloadFile(remotePath, localPath string, dryRun bool, res *Result) error {
	// If localPath looks like a directory (ends with / or is an existing dir), append the remote base name
	if strings.HasSuffix(localPath, "/") || strings.HasSuffix(localPath, string(filepath.Separator)) {
		if !dryRun {
			_ = os.MkdirAll(localPath, 0755)
		}
		localPath = filepath.Join(localPath, path.Base(remotePath))
	} else if stat, err := os.Stat(localPath); err == nil && stat.IsDir() {
		localPath = filepath.Join(localPath, path.Base(remotePath))
	}

	if dryRun {
		var size int64
		if stat, err := s.client.Stat(remotePath); err == nil {
			size = stat.Size()
		}
		res.copied(remoteDisplay(remotePath), localPath, size)
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

	res.copied(remoteDisplay(remotePath), localPath, n)
	return nil
}

func (s *SFTPClient) downloadDir(remoteDir, localDir string, dryRun bool, res *Result) error {
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
				continue
			}
			if err := os.MkdirAll(currentLocal, 0755); err != nil {
				return err
			}
			continue
		}

		if err := s.downloadFile(walker.Path(), currentLocal, dryRun, res); err != nil {
			return err
		}
	}
	return nil
}

// Sync uploads only modified or missing files by comparing checksums over the open connection
func (s *SFTPClient) Sync(localPath, remotePath string, recurse, dryRun bool) (*Result, error) {
	localStat, err := os.Stat(localPath)
	if err != nil {
		return nil, err
	}

	targetRemote := s.resolveRemotePath(remotePath)
	res := &Result{}

	if !localStat.IsDir() {
		return res, s.syncSingleFile(localPath, targetRemote, dryRun, res)
	}

	if !recurse {
		return nil, fmt.Errorf("%s is a directory (use -r / --recursive to sync)", localPath)
	}

	err = filepath.Walk(localPath, func(currentLocal string, info os.FileInfo, err error) error {
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

		return s.syncSingleFile(currentLocal, currentRemote, dryRun, res)
	})
	return res, err
}

func (s *SFTPClient) syncSingleFile(localFile, remoteFile string, dryRun bool, res *Result) error {
	// If remoteFile is an existing directory, compare against the file inside it
	if stat, err := s.client.Stat(remoteFile); err == nil && stat.IsDir() {
		remoteFile = path.Join(remoteFile, filepath.Base(localFile))
	}

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
				res.skip(localFile, remoteDisplay(remoteFile))
				return nil
			}
		}
	}

	// File is different or missing -> upload
	return s.uploadFile(localFile, remoteFile, dryRun, res)
}

// SyncDown downloads only modified or missing files by comparing checksums
func (s *SFTPClient) SyncDown(remotePath, localPath string, recurse, dryRun bool) (*Result, error) {
	resolvedRemote := s.resolveRemotePath(remotePath)

	remoteStat, err := s.client.Stat(resolvedRemote)
	if err != nil {
		return nil, fmt.Errorf("remote path not found: %w", err)
	}

	res := &Result{}
	if !remoteStat.IsDir() {
		return res, s.syncDownFile(resolvedRemote, localPath, dryRun, res)
	}

	if !recurse {
		return nil, fmt.Errorf("remote path %s is a directory (use -r / --recursive)", remotePath)
	}

	walker := s.client.Walk(resolvedRemote)
	for walker.Step() {
		if err := walker.Err(); err != nil {
			continue
		}

		relPath, err := filepath.Rel(resolvedRemote, walker.Path())
		if err != nil {
			continue
		}

		currentLocal := filepath.Join(localPath, filepath.FromSlash(relPath))
		stat := walker.Stat()

		if stat.IsDir() {
			if dryRun {
				continue
			}
			_ = os.MkdirAll(currentLocal, 0755)
			continue
		}

		if err := s.syncDownFile(walker.Path(), currentLocal, dryRun, res); err != nil {
			return res, err
		}
	}
	return res, nil
}

func (s *SFTPClient) syncDownFile(remoteFile, localFile string, dryRun bool, res *Result) error {
	// If localFile is an existing directory, append the remote base name
	if stat, err := os.Stat(localFile); err == nil && stat.IsDir() {
		localFile = filepath.Join(localFile, path.Base(remoteFile))
	}

	remoteMD5, err := s.getRemoteMD5(remoteFile)
	if err != nil {
		return err
	}

	// If local file exists and checksums match, skip
	if localMD5, err := calculateMD5(localFile); err == nil && localMD5 == remoteMD5 {
		res.skip(remoteDisplay(remoteFile), localFile)
		return nil
	}

	// File is different or missing -> download
	return s.downloadFile(remoteFile, localFile, dryRun, res)
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
