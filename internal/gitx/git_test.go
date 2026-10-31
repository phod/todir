package gitx

import (
	"errors"
	"os"
	"os/exec"
	"path/filepath"
	"testing"
)

func run(t *testing.T, args ...string) {
	t.Helper()
	cmd := exec.Command("git", args...)
	if out, err := cmd.CombinedOutput(); err != nil {
		t.Fatalf("git %v: %v\n%s", args, err, out)
	}
}

func makeRepo(t *testing.T) string {
	t.Helper()
	main := t.TempDir()
	run(t, "init", "-q", "-b", "main", main)
	run(t, "-C", main, "config", "user.name", "t")
	run(t, "-C", main, "config", "user.email", "t@example.com")
	if err := os.WriteFile(filepath.Join(main, "f"), nil, 0o644); err != nil {
		t.Fatal(err)
	}
	run(t, "-C", main, "add", "f")
	run(t, "-C", main, "-c", "user.name=t", "-c", "user.email=t@example.com", "commit", "-q", "-m", "init")
	return main
}

func addWorktree(t *testing.T, main, name string) string {
	t.Helper()
	dir := filepath.Join(t.TempDir(), name)
	run(t, "-C", main, "worktree", "add", "-q", "-b", name, dir)
	return dir
}

func norm(t *testing.T, p string) string {
	t.Helper()
	r, err := filepath.EvalSymlinks(p)
	if err != nil {
		t.Fatal(err)
	}
	return r
}

func TestDiscoverMainAndLinked(t *testing.T) {
	main := makeRepo(t)
	linked := addWorktree(t, main, "feature")
	want := norm(t, filepath.Join(main, ".git"))
	a, err := Discover(main)
	if err != nil {
		t.Fatal(err)
	}
	b, err := Discover(linked)
	if err != nil {
		t.Fatal(err)
	}
	if norm(t, a.CommonDir) != want || norm(t, b.CommonDir) != want {
		t.Errorf("common dirs %q %q want %q", a.CommonDir, b.CommonDir, want)
	}
	if norm(t, a.TopLevel) != norm(t, main) || norm(t, b.TopLevel) != norm(t, linked) {
		t.Errorf("toplevels %q %q", a.TopLevel, b.TopLevel)
	}
}

func TestDiscoverSubdirectory(t *testing.T) {
	main := makeRepo(t)
	linked := addWorktree(t, main, "feature")
	sub := filepath.Join(linked, "sub", "dir")
	if err := os.MkdirAll(sub, 0o755); err != nil {
		t.Fatal(err)
	}
	r, err := Discover(sub)
	if err != nil {
		t.Fatal(err)
	}
	if norm(t, r.TopLevel) != norm(t, linked) {
		t.Errorf("toplevel %q", r.TopLevel)
	}
}

func TestDiscoverNotRepo(t *testing.T) {
	dir := t.TempDir()
	t.Setenv("GIT_CEILING_DIRECTORIES", filepath.Dir(dir))
	_, err := Discover(dir)
	if !errors.Is(err, ErrNotRepo) {
		t.Fatalf("err = %v", err)
	}
}

func TestWorktreesListsBoth(t *testing.T) {
	main := makeRepo(t)
	addWorktree(t, main, "feature")
	wts, err := Worktrees(main)
	if err != nil {
		t.Fatal(err)
	}
	if len(wts) != 2 || !wts[0].Main || wts[0].Branch != "main" || wts[1].Branch != "feature" {
		t.Fatalf("got %+v", wts)
	}
}

func TestAvailableFalseWithEmptyPath(t *testing.T) {
	t.Setenv("PATH", t.TempDir())
	if Available() {
		t.Error("Available() = true")
	}
	if _, err := Discover(t.TempDir()); !errors.Is(err, ErrGitNotFound) {
		t.Errorf("err = %v", err)
	}
}
