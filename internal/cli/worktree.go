package cli

import (
	"errors"
	"fmt"
	"os"
	"path/filepath"
	"strings"

	"github.com/phodder/tele/internal/gitx"
	"github.com/phodder/tele/internal/store"
)

// requireGit fails fast when git is unavailable. Only worktree operations call it.
func requireGit() error {
	if !gitx.Available() {
		return gitx.ErrGitNotFound
	}
	return nil
}

// registerWorktree stores absPath as a path relative to its repository's
// top level, bound to the repository's common git dir.
func (a *App) registerWorktree(absPath, name string, force bool) error {
	repo, err := gitx.Discover(absPath)
	if err != nil {
		return err
	}
	rel, err := filepath.Rel(repo.TopLevel, absPath)
	if err != nil {
		return err
	}
	if rel == ".." || strings.HasPrefix(rel, ".."+string(filepath.Separator)) {
		// absPath was given through a symlink; git reports the real top level.
		real, rerr := filepath.EvalSymlinks(absPath)
		if rerr != nil {
			return rerr
		}
		if rel, err = filepath.Rel(repo.TopLevel, real); err != nil {
			return err
		}
	}
	rel = filepath.ToSlash(rel)
	s, err := a.loadStore()
	if err != nil {
		return err
	}
	if err := s.Put(store.Alias{Name: name, Path: rel, Worktree: true, Repo: repo.CommonDir}, force); err != nil {
		if errors.Is(err, store.ErrExists) {
			return fmt.Errorf("alias %s already exists; use -f to overwrite", name)
		}
		return err
	}
	if err := s.Save(a.ConfigPath); err != nil {
		return err
	}
	fmt.Fprintf(a.Stderr, "registered %s → %s (worktree-aware, repo %s)\n", name, rel, gitx.DisplayName(repo.CommonDir))
	return nil
}

// sameRepo reports whether two common dirs identify the same repository.
func sameRepo(a, b string) bool {
	a, b = filepath.Clean(a), filepath.Clean(b)
	if a == b {
		return true
	}
	ra, err1 := filepath.EvalSymlinks(a)
	rb, err2 := filepath.EvalSymlinks(b)
	return err1 == nil && err2 == nil && ra == rb
}

// resolveWorktreeAlias maps a worktree alias onto the worktree containing a.Cwd.
func (a *App) resolveWorktreeAlias(al store.Alias) (string, error) {
	if err := requireGit(); err != nil {
		return "", err
	}
	repo, err := gitx.Discover(a.Cwd)
	if errors.Is(err, gitx.ErrNotRepo) {
		return "", fmt.Errorf("not inside a git repository; alias %s belongs to repo %s (%s)", al.Name, gitx.DisplayName(al.Repo), al.Repo)
	}
	if err != nil {
		return "", err
	}
	if !sameRepo(al.Repo, repo.CommonDir) {
		return "", fmt.Errorf("in repository %s; alias %s belongs to %s", gitx.DisplayName(repo.CommonDir), al.Name, gitx.DisplayName(al.Repo))
	}
	top := filepath.Clean(repo.TopLevel)
	target := filepath.Join(top, filepath.FromSlash(al.Path))
	if filepath.IsAbs(al.Path) || (target != top && !strings.HasPrefix(target, top+string(filepath.Separator))) {
		return "", fmt.Errorf("alias %s has an invalid worktree path: %s", al.Name, al.Path)
	}
	info, err := os.Stat(target)
	if err != nil || !info.IsDir() {
		return "", fmt.Errorf("directory for alias %s does not exist in this worktree: %s", al.Name, target)
	}
	return target, nil
}
