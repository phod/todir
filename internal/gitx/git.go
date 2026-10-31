// Package gitx shells out to git for repository discovery and worktree listing.
package gitx

import (
	"bytes"
	"errors"
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
)

var (
	ErrGitNotFound = errors.New("git is required for worktree aliases but was not found on PATH")
	ErrNotRepo     = errors.New("not inside a git repository")
	ErrNotWorkTree = errors.New("not inside a git work tree")
)

// firstLine returns the first non-empty line of git's stderr so errors stay
// on one line.
func firstLine(s string) string {
	for _, l := range strings.Split(s, "\n") {
		if l = strings.TrimSpace(l); l != "" {
			return l
		}
	}
	return ""
}

// Repo identifies a repository by its common git directory.
type Repo struct{ CommonDir, TopLevel string }

// Available reports whether git is on PATH.
func Available() bool {
	_, err := exec.LookPath("git")
	return err == nil
}

func git(dir string, args ...string) (string, error) {
	path, err := exec.LookPath("git")
	if err != nil {
		return "", ErrGitNotFound
	}
	cmd := exec.Command(path, append([]string{"-C", dir}, args...)...)
	cmd.Env = append(os.Environ(), "GIT_TERMINAL_PROMPT=0", "LC_ALL=C")
	var stdout, stderr bytes.Buffer
	cmd.Stdout, cmd.Stderr = &stdout, &stderr
	if err := cmd.Run(); err != nil {
		full := stderr.String()
		var ee *exec.ExitError
		if errors.As(err, &ee) && ee.ExitCode() == 128 {
			switch {
			case strings.Contains(full, "not a git repository"):
				return "", ErrNotRepo
			case strings.Contains(full, "must be run in a work tree"):
				return "", ErrNotWorkTree
			}
		}
		msg := firstLine(full)
		if msg == "" {
			msg = err.Error()
		}
		return "", fmt.Errorf("git %s: %s", strings.Join(args, " "), msg)
	}
	return stdout.String(), nil
}

func absJoin(dir, p string) string {
	if !filepath.IsAbs(p) {
		p = filepath.Join(dir, p)
	}
	return filepath.Clean(p)
}

// Discover finds the repository containing dir.
func Discover(dir string) (Repo, error) {
	out, err := git(dir, "rev-parse", "--git-common-dir", "--show-toplevel")
	if err != nil {
		return Repo{}, err
	}
	lines := strings.Split(strings.TrimRight(out, "\n"), "\n")
	if len(lines) != 2 {
		return Repo{}, fmt.Errorf("unexpected git rev-parse output: %q", out)
	}
	return Repo{CommonDir: absJoin(dir, lines[0]), TopLevel: absJoin(dir, lines[1])}, nil
}

// DisplayName returns a human name for a repo's common dir.
func DisplayName(commonDir string) string {
	base := filepath.Base(commonDir)
	if base == ".git" {
		return filepath.Base(filepath.Dir(commonDir))
	}
	return base
}

// Worktrees lists the worktrees of the repository containing dir.
func Worktrees(dir string) ([]Worktree, error) {
	out, err := git(dir, "worktree", "list", "--porcelain")
	if err != nil {
		return nil, err
	}
	return parsePorcelain(out)
}
