package skills

import (
	"archive/tar"
	"compress/gzip"
	"context"
	"flag"
	"fmt"
	"io"
	"net/http"
	"net/url"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"time"
)

const (
	DefaultRepoURL    = "https://github.com/vigo999/mindspore-skills"
	DefaultRepoBranch = "refactor-arch-1.0"
	defaultRepoName   = "mindspore-skills"
	defaultSkillsDir  = "skills"
)

// RepoSync manages skills repository sync.
type RepoSync interface {
	Sync() error
}

// DefaultRepoSync keeps the bundled skills repo fresh under ~/.ms-cli.
type DefaultRepoSync struct {
	homeDir     string
	repoURL     string
	branch      string
	skipInTests bool
	httpClient  *http.Client
	lookPath    func(file string) (string, error)
	runCommand  func(name string, args ...string) error
}

// NewDefaultRepoSync creates the default startup syncer for the shared skills repo.
func NewDefaultRepoSync(homeDir string) *DefaultRepoSync {
	return &DefaultRepoSync{
		homeDir:     strings.TrimSpace(homeDir),
		repoURL:     DefaultRepoURL,
		branch:      DefaultRepoBranch,
		skipInTests: true,
		lookPath:    exec.LookPath,
		httpClient: &http.Client{
			Timeout: 2 * time.Minute,
		},
		runCommand: defaultRunCommand,
	}
}

// SyncedRepoDir returns the local checkout/download directory for the shared skills repo.
func SyncedRepoDir(homeDir string) string {
	return filepath.Join(strings.TrimSpace(homeDir), ".ms-cli", defaultRepoName)
}

// SyncedSkillsDir returns the highest-priority skills directory synced at startup.
func SyncedSkillsDir(homeDir string) string {
	return filepath.Join(SyncedRepoDir(homeDir), defaultSkillsDir)
}

// SkillsDir returns the synced skills directory for the receiver.
func (s *DefaultRepoSync) SkillsDir() string {
	return SyncedSkillsDir(s.homeDir)
}

// Sync keeps the shared skills repo available locally.
func (s *DefaultRepoSync) Sync() error {
	if strings.TrimSpace(s.homeDir) == "" {
		return fmt.Errorf("home directory is required")
	}
	if s.skipInTests && runningUnderGoTest() {
		return nil
	}

	repoDir := SyncedRepoDir(s.homeDir)
	skillsDir := SyncedSkillsDir(s.homeDir)
	if err := os.MkdirAll(filepath.Dir(repoDir), 0o755); err != nil {
		return fmt.Errorf("create skills parent dir: %w", err)
	}

	if s.hasGit() {
		if err := s.syncWithGit(repoDir); err != nil {
			if dirExists(skillsDir) {
				return nil
			}
			return err
		}
	} else {
		if dirExists(repoDir) {
			return nil
		}
		if err := s.downloadArchive(repoDir); err != nil {
			return err
		}
	}

	if !dirExists(skillsDir) {
		return fmt.Errorf("skills dir not found after sync: %s", skillsDir)
	}
	return nil
}

func (s *DefaultRepoSync) hasGit() bool {
	if s.lookPath == nil {
		return false
	}
	_, err := s.lookPath("git")
	return err == nil
}

func (s *DefaultRepoSync) syncWithGit(repoDir string) error {
	if !dirExists(repoDir) {
		if err := s.runCommand("git", "clone", "--branch", s.branch, "--single-branch", s.repoURL, repoDir); err != nil {
			return fmt.Errorf("git clone %s@%s: %w", s.repoURL, s.branch, err)
		}
		return nil
	}

	if !dirExists(filepath.Join(repoDir, ".git")) {
		if dirExists(filepath.Join(repoDir, defaultSkillsDir)) {
			return nil
		}
		return fmt.Errorf("skills repo path exists but is not a git repo: %s", repoDir)
	}

	if err := s.ensureOrigin(repoDir); err != nil {
		return err
	}
	if err := s.runCommand("git", "-C", repoDir, "fetch", "origin", s.branch); err != nil {
		return fmt.Errorf("git fetch %s: %w", s.branch, err)
	}
	if err := s.checkoutBranch(repoDir); err != nil {
		return err
	}
	if err := s.runCommand("git", "-C", repoDir, "pull", "--ff-only", "origin", s.branch); err != nil {
		return fmt.Errorf("git pull %s: %w", s.branch, err)
	}
	return nil
}

func (s *DefaultRepoSync) ensureOrigin(repoDir string) error {
	if err := s.runCommand("git", "-C", repoDir, "remote", "set-url", "origin", s.repoURL); err == nil {
		return nil
	}
	if err := s.runCommand("git", "-C", repoDir, "remote", "add", "origin", s.repoURL); err != nil {
		return fmt.Errorf("configure git origin: %w", err)
	}
	return nil
}

func (s *DefaultRepoSync) checkoutBranch(repoDir string) error {
	if err := s.runCommand("git", "-C", repoDir, "checkout", s.branch); err == nil {
		return nil
	}
	if err := s.runCommand("git", "-C", repoDir, "checkout", "-b", s.branch, "--track", "origin/"+s.branch); err != nil {
		return fmt.Errorf("git checkout %s: %w", s.branch, err)
	}
	return nil
}

func (s *DefaultRepoSync) downloadArchive(repoDir string) error {
	archiveURL, err := archiveURL(s.repoURL, s.branch)
	if err != nil {
		return err
	}

	resp, err := s.httpClient.Get(archiveURL)
	if err != nil {
		return fmt.Errorf("download skills archive: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return fmt.Errorf("download skills archive: unexpected status %d", resp.StatusCode)
	}

	tempDir, err := os.MkdirTemp(filepath.Dir(repoDir), defaultRepoName+"-download-*")
	if err != nil {
		return fmt.Errorf("create temp dir: %w", err)
	}
	defer os.RemoveAll(tempDir)

	extractDir := filepath.Join(tempDir, defaultRepoName)
	if err := extractTarGz(resp.Body, extractDir); err != nil {
		return fmt.Errorf("extract skills archive: %w", err)
	}
	if err := os.Rename(extractDir, repoDir); err != nil {
		return fmt.Errorf("install downloaded skills repo: %w", err)
	}
	return nil
}

func archiveURL(repoURL, branch string) (string, error) {
	repoURL = strings.TrimSpace(repoURL)
	repoURL = strings.TrimSuffix(repoURL, ".git")
	repoURL = strings.TrimSuffix(repoURL, "/")
	const githubPrefix = "https://github.com/"
	if !strings.HasPrefix(repoURL, githubPrefix) {
		return "", fmt.Errorf("unsupported skills repo url: %s", repoURL)
	}
	repoPath := strings.TrimPrefix(repoURL, githubPrefix)
	return "https://codeload.github.com/" + repoPath + "/tar.gz/refs/heads/" + url.PathEscape(branch), nil
}

func extractTarGz(src io.Reader, destDir string) error {
	gzr, err := gzip.NewReader(src)
	if err != nil {
		return err
	}
	defer gzr.Close()

	tr := tar.NewReader(gzr)
	for {
		hdr, err := tr.Next()
		if err == io.EOF {
			return nil
		}
		if err != nil {
			return err
		}

		relPath := stripArchiveRoot(hdr.Name)
		if relPath == "" {
			continue
		}

		targetPath, err := safeJoin(destDir, relPath)
		if err != nil {
			return err
		}

		switch hdr.Typeflag {
		case tar.TypeDir:
			if err := os.MkdirAll(targetPath, hdr.FileInfo().Mode().Perm()); err != nil {
				return err
			}
		case tar.TypeReg, tar.TypeRegA:
			if err := os.MkdirAll(filepath.Dir(targetPath), 0o755); err != nil {
				return err
			}
			file, err := os.OpenFile(targetPath, os.O_CREATE|os.O_WRONLY|os.O_TRUNC, hdr.FileInfo().Mode().Perm())
			if err != nil {
				return err
			}
			if _, err := io.Copy(file, tr); err != nil {
				_ = file.Close()
				return err
			}
			if err := file.Close(); err != nil {
				return err
			}
		}
	}
}

func stripArchiveRoot(name string) string {
	name = strings.TrimSpace(name)
	name = strings.TrimPrefix(name, "./")
	if name == "" {
		return ""
	}
	parts := strings.SplitN(name, "/", 2)
	if len(parts) < 2 {
		return ""
	}
	return strings.TrimPrefix(parts[1], "/")
}

func safeJoin(rootDir, relPath string) (string, error) {
	rootDir = filepath.Clean(rootDir)
	relPath = filepath.Clean(relPath)
	targetPath := filepath.Join(rootDir, relPath)
	if targetPath == rootDir {
		return targetPath, nil
	}
	prefix := rootDir + string(os.PathSeparator)
	if !strings.HasPrefix(targetPath, prefix) {
		return "", fmt.Errorf("archive path escapes destination: %s", relPath)
	}
	return targetPath, nil
}

func dirExists(path string) bool {
	info, err := os.Stat(path)
	return err == nil && info.IsDir()
}

func runningUnderGoTest() bool {
	return flag.Lookup("test.v") != nil || flag.Lookup("test.run") != nil
}

func defaultRunCommand(name string, args ...string) error {
	ctx, cancel := context.WithTimeout(context.Background(), 2*time.Minute)
	defer cancel()

	cmd := exec.CommandContext(ctx, name, args...)
	output, err := cmd.CombinedOutput()
	if err == nil {
		return nil
	}

	msg := strings.TrimSpace(string(output))
	if msg == "" {
		return err
	}
	return fmt.Errorf("%w: %s", err, msg)
}
