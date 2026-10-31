package cli

import (
	"errors"
	"fmt"
	"os"
	"path/filepath"
	"sort"
	"strings"
	"text/tabwriter"

	"github.com/phod/todir/internal/gitx"
	"github.com/phod/todir/internal/store"
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
	rel, ok, err := relWithin(repo.TopLevel, absPath)
	if err != nil {
		return err
	}
	if !ok {
		return fmt.Errorf("alias %s has an invalid worktree path: %s", name, rel)
	}
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

func escapes(rel string) bool {
	return rel == ".." || strings.HasPrefix(rel, ".."+string(filepath.Separator))
}

// relWithin returns absPath relative to top in slash form, retrying through
// symlinks (git reports the real top level). ok is false when the path still
// lies outside top.
func relWithin(top, absPath string) (rel string, ok bool, err error) {
	if rel, err = filepath.Rel(top, absPath); err != nil {
		return "", false, err
	}
	if escapes(rel) {
		real, rerr := filepath.EvalSymlinks(absPath)
		if rerr != nil {
			return "", false, rerr
		}
		if rel, err = filepath.Rel(top, real); err != nil {
			return "", false, err
		}
	}
	return filepath.ToSlash(rel), !escapes(rel), nil
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

// worktreeRepoDir picks the directory whose repository -wl should list.
func (a *App) worktreeRepoDir(alias string) (string, error) {
	if alias == "" {
		if _, err := gitx.Discover(a.Cwd); err != nil {
			return "", err
		}
		return a.Cwd, nil
	}
	s, err := a.loadStore()
	if err != nil {
		return "", err
	}
	al, ok := s.Get(alias)
	if !ok {
		return "", fmt.Errorf("alias %s not found", alias)
	}
	if al.Worktree {
		return al.Repo, nil
	}
	if _, err := gitx.Discover(al.Path); err != nil {
		if errors.Is(err, gitx.ErrNotRepo) {
			return "", fmt.Errorf("alias %s does not point into a git repository", alias)
		}
		return "", err
	}
	return al.Path, nil
}

// worktreeRows lists the worktrees of the repo at dir, main first then by branch.
func (a *App) worktreeRows(dir string) ([]gitx.Worktree, error) {
	wts, err := gitx.Worktrees(dir)
	if err != nil {
		return nil, err
	}
	sort.SliceStable(wts, func(i, j int) bool {
		if wts[i].Main != wts[j].Main {
			return wts[i].Main
		}
		return wts[i].Branch < wts[j].Branch
	})
	return wts, nil
}

func pathWithin(p, cwd string) bool {
	p, cwd = filepath.Clean(p), filepath.Clean(cwd)
	return cwd == p || strings.HasPrefix(cwd, strings.TrimSuffix(p, string(filepath.Separator))+string(filepath.Separator))
}

// currentWorktreeIndex returns the index of the worktree containing cwd
// (longest matching path), or -1.
func currentWorktreeIndex(wts []gitx.Worktree, cwd string) int {
	realCwd, cerr := filepath.EvalSymlinks(cwd)
	best, bestLen := -1, -1
	for i, w := range wts {
		match := pathWithin(w.Path, cwd)
		if !match && cerr == nil {
			if rp, err := filepath.EvalSymlinks(w.Path); err == nil {
				match = pathWithin(rp, realCwd)
			}
		}
		if match && len(w.Path) > bestLen {
			best, bestLen = i, len(w.Path)
		}
	}
	return best
}

// formatWorktrees renders marker, branch and path columns.
func formatWorktrees(wts []gitx.Worktree, current int) string {
	var b strings.Builder
	tw := tabwriter.NewWriter(&b, 0, 4, 2, ' ', 0)
	for i, w := range wts {
		marker := " "
		if i == current {
			marker = "*"
		}
		fmt.Fprintf(tw, "%s\t%s\t%s\n", marker, w.Branch, w.Path)
	}
	tw.Flush()
	return b.String()
}

func (a *App) listWorktrees(m Mode) error {
	if err := requireGit(); err != nil {
		return err
	}
	dir, err := a.worktreeRepoDir(m.Alias)
	if err != nil {
		return err
	}
	wts, err := a.worktreeRows(dir)
	if err != nil {
		return err
	}
	fmt.Fprint(a.Stdout, formatWorktrees(wts, currentWorktreeIndex(wts, a.Cwd)))
	return nil
}
