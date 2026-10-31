package store

import (
	"errors"
	"os"
	"path/filepath"
	"strings"
	"testing"
)

func TestConfigPathXDG(t *testing.T) {
	get := func(k string) string {
		if k == "XDG_CONFIG_HOME" {
			return "/x"
		}
		return ""
	}
	if got, want := ConfigPath(get, "/home/u"), filepath.Join("/x", "todir", "aliases.toml"); got != want {
		t.Fatalf("got %q want %q", got, want)
	}
	none := func(string) string { return "" }
	if got, want := ConfigPath(none, "/home/u"), filepath.Join("/home/u", ".config", "todir", "aliases.toml"); got != want {
		t.Fatalf("got %q want %q", got, want)
	}
}

func TestValidName(t *testing.T) {
	for _, n := range []string{"a", "account-service", "x.y_z"} {
		if !ValidName(n) {
			t.Errorf("%q should be valid", n)
		}
	}
	for _, n := range []string{"", "-x", ".hidden", "a/b", "a b", "é"} {
		if ValidName(n) {
			t.Errorf("%q should be invalid", n)
		}
	}
}

func TestLoadMissingFileIsEmpty(t *testing.T) {
	s, err := Load(filepath.Join(t.TempDir(), "nope", "aliases.toml"))
	if err != nil {
		t.Fatal(err)
	}
	if s == nil || s.Aliases == nil || len(s.Aliases) != 0 {
		t.Fatalf("expected empty non-nil map, got %+v", s)
	}
}

func TestLoadMalformedNamesFile(t *testing.T) {
	p := filepath.Join(t.TempDir(), "aliases.toml")
	if err := os.WriteFile(p, []byte("not = [toml"), 0o600); err != nil {
		t.Fatal(err)
	}
	_, err := Load(p)
	if err == nil || !strings.Contains(err.Error(), p) {
		t.Fatalf("expected error naming %q, got %v", p, err)
	}
}

func TestLoadRejectsInvalidEntries(t *testing.T) {
	cases := []struct{ name, body, want string }{
		{"unquoted dotted name", "[aliases.my.notes]\npath = \"/n\"\n", "alias my has no path"},
		{"empty path", "[aliases.a]\npath = \"\"\n", "alias a has no path"},
		{"worktree without repo", "[aliases.w]\npath = \"svc\"\nworktree = true\n", "worktree alias w has no repo"},
		{"relative plain path", "[aliases.rel]\npath = \"some/dir\"\n", "alias rel has a relative path: some/dir"},
	}
	for _, c := range cases {
		t.Run(c.name, func(t *testing.T) {
			p := filepath.Join(t.TempDir(), "aliases.toml")
			if err := os.WriteFile(p, []byte(c.body), 0o600); err != nil {
				t.Fatal(err)
			}
			_, err := Load(p)
			if err == nil || !strings.Contains(err.Error(), p) || !strings.Contains(err.Error(), c.want) {
				t.Fatalf("err = %v, want it to name %q and contain %q", err, p, c.want)
			}
		})
	}
}

func TestLoadAcceptsQuotedDottedName(t *testing.T) {
	p := filepath.Join(t.TempDir(), "aliases.toml")
	body := "[aliases.\"my.notes\"]\npath = \"/n\"\n\n[aliases.w]\npath = \"svc\"\nworktree = true\nrepo = \"/r/.git\"\n"
	if err := os.WriteFile(p, []byte(body), 0o600); err != nil {
		t.Fatal(err)
	}
	s, err := Load(p)
	if err != nil {
		t.Fatal(err)
	}
	if a, ok := s.Get("my.notes"); !ok || a.Path != "/n" {
		t.Fatalf("my.notes = %+v", a)
	}
}

func TestSaveThenLoadRoundTrip(t *testing.T) {
	p := filepath.Join(t.TempDir(), "todir", "aliases.toml")
	s := &Store{Aliases: map[string]Alias{}}
	if err := s.Put(Alias{Name: "plain", Path: "/a/b"}, false); err != nil {
		t.Fatal(err)
	}
	if err := s.Put(Alias{Name: "wt", Path: "/a/c", Worktree: true, Repo: "/a/repo"}, false); err != nil {
		t.Fatal(err)
	}
	if err := s.Save(p); err != nil {
		t.Fatal(err)
	}
	fi, err := os.Stat(p)
	if err != nil {
		t.Fatal(err)
	}
	if fi.Mode().Perm() != 0o600 {
		t.Errorf("file mode %v", fi.Mode().Perm())
	}
	di, err := os.Stat(filepath.Dir(p))
	if err != nil {
		t.Fatal(err)
	}
	if di.Mode().Perm() != 0o700 {
		t.Errorf("dir mode %v", di.Mode().Perm())
	}
	b, err := os.ReadFile(p)
	if err != nil {
		t.Fatal(err)
	}
	if !strings.HasPrefix(string(b), "# todir aliases — safe to hand-edit\n") {
		t.Errorf("missing header: %q", b)
	}
	got, err := Load(p)
	if err != nil {
		t.Fatal(err)
	}
	if a, ok := got.Get("plain"); !ok || a != (Alias{Name: "plain", Path: "/a/b"}) {
		t.Errorf("plain: %+v", a)
	}
	if a, ok := got.Get("wt"); !ok || a != (Alias{Name: "wt", Path: "/a/c", Worktree: true, Repo: "/a/repo"}) {
		t.Errorf("wt: %+v", a)
	}
}

func TestPutRejectsDuplicateUnlessForce(t *testing.T) {
	s := &Store{}
	if err := s.Put(Alias{Name: "a", Path: "/1"}, false); err != nil {
		t.Fatal(err)
	}
	if err := s.Put(Alias{Name: "a", Path: "/2"}, false); !errors.Is(err, ErrExists) {
		t.Fatalf("want ErrExists, got %v", err)
	}
	if err := s.Put(Alias{Name: "a", Path: "/2"}, true); err != nil {
		t.Fatal(err)
	}
	if a, _ := s.Get("a"); a.Path != "/2" {
		t.Fatalf("not overwritten: %+v", a)
	}
}

func TestDeleteMissingReturnsErrNotFound(t *testing.T) {
	s := &Store{}
	if err := s.Delete("x"); !errors.Is(err, ErrNotFound) {
		t.Fatalf("got %v", err)
	}
	if err := s.Put(Alias{Name: "x", Path: "/"}, false); err != nil {
		t.Fatal(err)
	}
	if err := s.Delete("x"); err != nil {
		t.Fatal(err)
	}
}

func TestSortedByName(t *testing.T) {
	s := &Store{}
	for _, n := range []string{"c", "a", "b"} {
		if err := s.Put(Alias{Name: n, Path: "/" + n}, false); err != nil {
			t.Fatal(err)
		}
	}
	var names []string
	for _, a := range s.Sorted() {
		names = append(names, a.Name)
	}
	if strings.Join(names, ",") != "a,b,c" {
		t.Fatalf("got %v", names)
	}
}

func TestSaveIsAtomic(t *testing.T) {
	dir := t.TempDir()
	p := filepath.Join(dir, "aliases.toml")
	s := &Store{}
	if err := s.Put(Alias{Name: "a", Path: "/a"}, false); err != nil {
		t.Fatal(err)
	}
	if err := s.Save(p); err != nil {
		t.Fatal(err)
	}
	m, err := filepath.Glob(filepath.Join(dir, "*.tmp*"))
	if err != nil {
		t.Fatal(err)
	}
	if len(m) != 0 {
		t.Fatalf("leftover temp files: %v", m)
	}
}
