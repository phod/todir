package cli

import (
	"errors"
	"fmt"
	"os"
	"path/filepath"
	"text/tabwriter"

	"github.com/phodder/tele/internal/store"
)

func (a *App) dispatch(m Mode) error {
	switch m.Kind {
	case KindVersion:
		fmt.Fprintf(a.Stdout, "tele %s\n", a.Version)
		return nil
	case KindRegister:
		return a.register(m)
	case KindDelete:
		return a.deleteAlias(m)
	case KindListAliases:
		return a.listAliases()
	case KindNavigate:
		return a.navigate(m.Alias)
	}
	return usagef("not implemented")
}

func (a *App) loadStore() (*store.Store, error) {
	return store.Load(a.ConfigPath)
}

func (a *App) register(m Mode) error {
	if m.Worktree {
		return usagef("not implemented")
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
		fmt.Fprintln(a.Stderr, "tele: no aliases registered; add one with: tele -r <path> <name>")
		return nil
	}
	tw := tabwriter.NewWriter(a.Stdout, 0, 0, 2, ' ', 0)
	for _, al := range aliases {
		fmt.Fprintf(tw, "%s\t%s\n", al.Name, al.Path)
	}
	return tw.Flush()
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
	info, err := os.Stat(al.Path)
	if err != nil || !info.IsDir() {
		return "", fmt.Errorf("directory for alias %s does not exist: %s", al.Name, al.Path)
	}
	return al.Path, nil
}
