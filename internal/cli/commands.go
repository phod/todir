package cli

import (
	"errors"
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"text/tabwriter"

	"github.com/phod/tkto/internal/gitx"
	"github.com/phod/tkto/internal/picker"
	"github.com/phod/tkto/internal/store"
)

func (a *App) dispatch(m Mode) error {
	switch m.Kind {
	case KindVersion:
		fmt.Fprintf(a.Stdout, "tkto %s\n", a.Version)
		return nil
	case KindRegister:
		return a.register(m)
	case KindDelete:
		return a.deleteAlias(m)
	case KindListAliases:
		return a.listAliases()
	case KindNavigate:
		return a.navigate(m.Alias)
	case KindPickAlias:
		return a.pickAlias()
	case KindPickWorktree:
		return a.pickWorktree(m)
	case KindListWorktrees:
		return a.listWorktrees(m)
	case KindInit:
		return a.initShell(m.Shell)
	}
	return usagef("not implemented")
}

func (a *App) loadStore() (*store.Store, error) {
	return store.Load(a.ConfigPath)
}

func (a *App) register(m Mode) error {
	if m.Worktree {
		if err := requireGit(); err != nil {
			return err
		}
	}
	if !store.ValidName(m.Alias) {
		return usagef("invalid alias name %q: use letters, digits, '.', '_' or '-' and start with a letter or digit", m.Alias)
	}
	p := m.Path
	if !filepath.IsAbs(p) {
		p = filepath.Join(a.Cwd, p)
	}
	p = filepath.Clean(p)
	info, err := os.Stat(p)
	if errors.Is(err, os.ErrNotExist) {
		return fmt.Errorf("path does not exist: %s", p)
	}
	if err != nil {
		return fmt.Errorf("cannot access %s: %w", p, err)
	}
	if !info.IsDir() {
		return fmt.Errorf("not a directory: %s", p)
	}
	if m.Worktree {
		return a.registerWorktree(p, m.Alias, m.Force)
	}
	s, err := a.loadStore()
	if err != nil {
		return err
	}
	if err := s.Put(store.Alias{Name: m.Alias, Path: p}, m.Force); err != nil {
		if errors.Is(err, store.ErrExists) {
			return fmt.Errorf("alias %s already exists; use -f to overwrite", m.Alias)
		}
		return err
	}
	if err := s.Save(a.ConfigPath); err != nil {
		return err
	}
	fmt.Fprintf(a.Stderr, "registered %s → %s\n", m.Alias, p)
	return nil
}

func (a *App) deleteAlias(m Mode) error {
	s, err := a.loadStore()
	if err != nil {
		return err
	}
	if err := s.Delete(m.Alias); err != nil {
		if errors.Is(err, store.ErrNotFound) {
			return fmt.Errorf("alias %s not found", m.Alias)
		}
		return err
	}
	if err := s.Save(a.ConfigPath); err != nil {
		return err
	}
	fmt.Fprintf(a.Stderr, "deleted %s\n", m.Alias)
	return nil
}

func (a *App) listAliases() error {
	s, err := a.loadStore()
	if err != nil {
		return err
	}
	aliases := s.Sorted()
	if len(aliases) == 0 {
		fmt.Fprintln(a.Stderr, "tkto: no aliases registered; add one with: tkto -r <path> <name>")
		return nil
	}
	// Every row has three cells so tabwriter aligns one block across plain
	// and worktree rows; the padding left after an empty third cell is trimmed.
	var buf strings.Builder
	tw := tabwriter.NewWriter(&buf, 0, 0, 2, ' ', 0)
	for _, al := range aliases {
		note := ""
		if al.Worktree {
			note = "[worktree: " + gitx.DisplayName(al.Repo) + "]"
		}
		fmt.Fprintf(tw, "%s\t%s\t%s\n", al.Name, al.Path, note)
	}
	if err := tw.Flush(); err != nil {
		return err
	}
	for _, line := range strings.SplitAfter(buf.String(), "\n") {
		if line != "" {
			fmt.Fprintln(a.Stdout, strings.TrimRight(line, " \n"))
		}
	}
	return nil
}

func (a *App) navigate(name string) error {
	s, err := a.loadStore()
	if err != nil {
		return err
	}
	al, ok := s.Get(name)
	if !ok {
		return fmt.Errorf("alias %s not found", name)
	}
	target, err := a.resolveAlias(al)
	if err != nil {
		return err
	}
	return a.emitTarget(target)
}

// resolveAlias returns the directory an alias points at.
func (a *App) resolveAlias(al store.Alias) (string, error) {
	if al.Worktree {
		return a.resolveWorktreeAlias(al)
	}
	info, err := os.Stat(al.Path)
	if err != nil || !info.IsDir() {
		return "", fmt.Errorf("directory for alias %s does not exist: %s", al.Name, al.Path)
	}
	return al.Path, nil
}

func (a *App) pickAlias() error {
	s, err := a.loadStore()
	if err != nil {
		return err
	}
	aliases := s.Sorted()
	if len(aliases) == 0 {
		return errors.New("no aliases registered; add one with: tkto -r <path> <name>")
	}
	items := make([]picker.Item, 0, len(aliases))
	for _, al := range aliases {
		detail := al.Path
		if al.Worktree {
			detail += "  [worktree: " + gitx.DisplayName(al.Repo) + "]"
		}
		items = append(items, picker.Item{Label: al.Name, Detail: detail, Value: al.Name})
	}
	chosen, err := picker.Run("Aliases", items, a.Stdin, a.Stderr, a.StderrIsTTY)
	if err != nil {
		return err
	}
	return a.navigate(chosen.Value)
}

func (a *App) pickWorktree(m Mode) error {
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
	items := worktreeItems(wts, currentWorktreeIndex(wts, a.Cwd))
	chosen, err := picker.Run("Worktrees", items, a.Stdin, a.Stderr, a.StderrIsTTY)
	if err != nil {
		return err
	}
	return a.emitTarget(chosen.Value)
}

// worktreeItems builds the worktree picker rows, marking the current
// worktree with "* " and indenting the others to align.
func worktreeItems(wts []gitx.Worktree, current int) []picker.Item {
	items := make([]picker.Item, 0, len(wts))
	for i, wt := range wts {
		marker := "  "
		if i == current {
			marker = "* "
		}
		items = append(items, picker.Item{Label: marker + wt.Branch, Detail: wt.Path, Value: wt.Path})
	}
	return items
}
