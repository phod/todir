// Package store persists directory aliases in a TOML file.
package store

import (
	"bytes"
	"errors"
	"fmt"
	"os"
	"path/filepath"
	"regexp"
	"sort"

	"github.com/BurntSushi/toml"
)

const header = "# tkto aliases — safe to hand-edit\n"

var (
	ErrExists   = errors.New("alias already exists")
	ErrNotFound = errors.New("alias not found")
)

var nameRE = regexp.MustCompile(`^[A-Za-z0-9][A-Za-z0-9._-]*$`)

// Alias is a named directory target.
type Alias struct {
	Name     string `toml:"-"`
	Path     string `toml:"path"`
	Worktree bool   `toml:"worktree,omitempty"`
	Repo     string `toml:"repo,omitempty"`
}

// Store is the set of aliases keyed by name.
type Store struct {
	Aliases map[string]Alias `toml:"aliases"`
}

// ConfigPath returns the aliases file location.
func ConfigPath(getenv func(string) string, home string) string {
	base := getenv("XDG_CONFIG_HOME")
	if base == "" {
		base = filepath.Join(home, ".config")
	}
	return filepath.Join(base, "tkto", "aliases.toml")
}

// ValidName reports whether name is a legal alias name.
func ValidName(name string) bool { return nameRE.MatchString(name) }

// Load reads the store at path. A missing file yields an empty store.
func Load(path string) (*Store, error) {
	s := &Store{Aliases: map[string]Alias{}}
	data, err := os.ReadFile(path)
	if errors.Is(err, os.ErrNotExist) {
		return s, nil
	}
	if err != nil {
		return nil, fmt.Errorf("read %s: %w", path, err)
	}
	if _, err := toml.Decode(string(data), s); err != nil {
		return nil, fmt.Errorf("parse %s: %w", path, err)
	}
	if s.Aliases == nil {
		s.Aliases = map[string]Alias{}
	}
	for name, a := range s.Aliases {
		a.Name = name
		s.Aliases[name] = a
	}
	for _, a := range s.Sorted() {
		if err := validate(a); err != nil {
			return nil, fmt.Errorf("%s: %w", path, err)
		}
	}
	return s, nil
}

// validate rejects hand-edited entries that cannot be navigated to safely.
func validate(a Alias) error {
	switch {
	case a.Path == "":
		return fmt.Errorf("alias %s has no path (quote dotted names: [aliases.\"my.notes\"])", a.Name)
	case a.Worktree && a.Repo == "":
		return fmt.Errorf("worktree alias %s has no repo", a.Name)
	case !a.Worktree && !filepath.IsAbs(a.Path):
		return fmt.Errorf("alias %s has a relative path: %s", a.Name, a.Path)
	}
	return nil
}

// Save writes the store atomically with mode 0600.
func (s *Store) Save(path string) error {
	dir := filepath.Dir(path)
	if err := os.MkdirAll(dir, 0o700); err != nil {
		return fmt.Errorf("create %s: %w", dir, err)
	}
	aliases := s.Aliases
	if aliases == nil {
		aliases = map[string]Alias{}
	}
	var buf bytes.Buffer
	buf.WriteString(header)
	if err := toml.NewEncoder(&buf).Encode(Store{Aliases: aliases}); err != nil {
		return fmt.Errorf("encode aliases: %w", err)
	}
	tmp, err := os.CreateTemp(dir, ".aliases-*.tmp")
	if err != nil {
		return fmt.Errorf("write %s: %w", path, err)
	}
	tmpName := tmp.Name()
	if err := tmp.Chmod(0o600); err == nil {
		_, err = tmp.Write(buf.Bytes())
		if cerr := tmp.Close(); err == nil {
			err = cerr
		}
		if err == nil {
			err = os.Rename(tmpName, path)
		}
		if err != nil {
			os.Remove(tmpName)
			return fmt.Errorf("write %s: %w", path, err)
		}
		return nil
	} else {
		tmp.Close()
		os.Remove(tmpName)
		return fmt.Errorf("write %s: %w", path, err)
	}
}

// Get returns the alias with the given name.
func (s *Store) Get(name string) (Alias, bool) {
	a, ok := s.Aliases[name]
	return a, ok
}

// Put adds a, failing with ErrExists unless force is set.
func (s *Store) Put(a Alias, force bool) error {
	if s.Aliases == nil {
		s.Aliases = map[string]Alias{}
	}
	if _, ok := s.Aliases[a.Name]; ok && !force {
		return fmt.Errorf("%w: %s", ErrExists, a.Name)
	}
	s.Aliases[a.Name] = a
	return nil
}

// Delete removes the named alias.
func (s *Store) Delete(name string) error {
	if _, ok := s.Aliases[name]; !ok {
		return fmt.Errorf("%w: %s", ErrNotFound, name)
	}
	delete(s.Aliases, name)
	return nil
}

// Sorted returns all aliases ordered by name.
func (s *Store) Sorted() []Alias {
	out := make([]Alias, 0, len(s.Aliases))
	for name, a := range s.Aliases {
		a.Name = name
		out = append(out, a)
	}
	sort.Slice(out, func(i, j int) bool { return out[i].Name < out[j].Name })
	return out
}
