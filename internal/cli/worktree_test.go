package cli

import (
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"testing"

	"github.com/phod/tkto/internal/gitx"
	"github.com/phod/tkto/internal/store"
)

func git(t *testing.T, args ...string) {
	t.Helper()
	t.Setenv("GIT_CONFIG_GLOBAL", os.DevNull)
	if out, err := exec.Command("git", args...).CombinedOutput(); err != nil {
		t.Fatalf("git %v: %v\n%s", args, err, out)
	}
}

func makeRepo(t *testing.T) string {
	t.Helper()
	main := t.TempDir()
	git(t, "init", "-q", "-b", "main", main)
	git(t, "-C", main, "config", "user.name", "t")
	git(t, "-C", main, "config", "user.email", "t@example.com")
	if err := os.WriteFile(filepath.Join(main, "f"), nil, 0o644); err != nil {
		t.Fatal(err)
	}
	git(t, "-C", main, "add", "f")
	git(t, "-C", main, "-c", "user.name=t", "-c", "user.email=t@example.com", "-c", "commit.gpgsign=false", "commit", "-q", "-m", "init")
	return main
}

func addWorktree(t *testing.T, main, name string) string {
	t.Helper()
	dir := filepath.Join(t.TempDir(), name)
	git(t, "-C", main, "worktree", "add", "-q", "-b", name, dir)
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

const svc = "services/step/account-service"

// wtFixture builds a repo with services/step/account-service committed and a
// linked worktree "feature".
func wtFixture(t *testing.T) (app *App, main, linked string) {
	t.Helper()
	main = makeRepo(t)
	dir := filepath.Join(main, filepath.FromSlash(svc))
	if err := os.MkdirAll(dir, 0o755); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(dir, "x"), nil, 0o644); err != nil {
		t.Fatal(err)
	}
	git(t, "-C", main, "add", ".")
	git(t, "-C", main, "-c", "user.name=t", "-c", "user.email=t@example.com", "-c", "commit.gpgsign=false", "commit", "-q", "-m", "svc")
	linked = addWorktree(t, main, "feature")
	app, _, _ = newTestApp(t)
	return
}

func registerAcct(t *testing.T, app *App, main string) {
	t.Helper()
	app.Cwd = filepath.Join(main, filepath.FromSlash(svc))
	if code, _, errs := run(t, app, "-r", "-w", ".", "acct"); code != 0 {
		t.Fatalf("register: code=%d err=%q", code, errs)
	}
}

func TestRegisterWorktreeStoresRelativePath(t *testing.T) {
	app, main, _ := wtFixture(t)
	app.Cwd = filepath.Join(main, filepath.FromSlash(svc))
	code, out, errs := run(t, app, "-r", "-w", ".", "acct")
	want := "registered acct → " + svc + " (worktree-aware, repo " + filepath.Base(main) + ")\n"
	if code != 0 || out != "" || errs != want {
		t.Fatalf("code=%d out=%q err=%q want err %q", code, out, errs, want)
	}
	s, err := store.Load(app.ConfigPath)
	if err != nil {
		t.Fatal(err)
	}
	al := s.Aliases["acct"]
	if al.Path != svc || !al.Worktree || norm(t, al.Repo) != norm(t, filepath.Join(main, ".git")) {
		t.Fatalf("alias = %+v", al)
	}
}

func TestRegisterWorktreeAtRootStoresDot(t *testing.T) {
	app, main, _ := wtFixture(t)
	app.Cwd = main
	if code, _, errs := run(t, app, "-r", "-w", ".", "root"); code != 0 {
		t.Fatalf("code=%d err=%q", code, errs)
	}
	s, _ := store.Load(app.ConfigPath)
	if s.Aliases["root"].Path != "." {
		t.Fatalf("Path = %q", s.Aliases["root"].Path)
	}
}

func TestRegisterWorktreeOutsideRepoFails(t *testing.T) {
	app, _, _ := wtFixture(t)
	code, _, errs := run(t, app, "-r", "-w", ".", "x")
	if code != 1 || !strings.Contains(errs, "not inside a git repository") {
		t.Fatalf("code=%d err=%q", code, errs)
	}
}

func TestNavigateWorktreeFromLinked(t *testing.T) {
	app, main, linked := wtFixture(t)
	registerAcct(t, app, main)
	app.Cwd = linked
	code, out, errs := run(t, app, "acct")
	want := norm(t, linked) + "/" + svc + "\n"
	if code != 0 || out != want || errs != "" {
		t.Fatalf("code=%d out=%q err=%q want %q", code, out, errs, want)
	}
}

func TestNavigateWorktreeFromMainSubdir(t *testing.T) {
	app, main, _ := wtFixture(t)
	registerAcct(t, app, main)
	app.Cwd = filepath.Join(main, "services")
	code, out, _ := run(t, app, "acct")
	want := norm(t, main) + "/" + svc + "\n"
	if code != 0 || out != want {
		t.Fatalf("code=%d out=%q want %q", code, out, want)
	}
}

func TestNavigateWorktreeOutsideRepo(t *testing.T) {
	app, main, _ := wtFixture(t)
	registerAcct(t, app, main)
	app.Cwd = t.TempDir()
	code, out, errs := run(t, app, "acct")
	s, _ := store.Load(app.ConfigPath)
	want := "tkto: not inside a git repository; alias acct belongs to repo " + filepath.Base(main) + " (" + s.Aliases["acct"].Repo + ")\n"
	if code != 1 || out != "" || errs != want {
		t.Fatalf("code=%d out=%q err=%q want %q", code, out, errs, want)
	}
}

func TestNavigateWorktreeWrongRepo(t *testing.T) {
	app, main, _ := wtFixture(t)
	registerAcct(t, app, main)
	other := makeRepo(t)
	app.Cwd = other
	code, out, errs := run(t, app, "acct")
	want := "tkto: in repository " + filepath.Base(other) + "; alias acct belongs to " + filepath.Base(main) + "\n"
	if code != 1 || out != "" || errs != want {
		t.Fatalf("code=%d out=%q err=%q want %q", code, out, errs, want)
	}
}

func TestNavigateWorktreeMissingDirInBranch(t *testing.T) {
	app, main, linked := wtFixture(t)
	registerAcct(t, app, main)
	if err := os.RemoveAll(filepath.Join(linked, "services")); err != nil {
		t.Fatal(err)
	}
	app.Cwd = linked
	code, out, errs := run(t, app, "acct")
	if code != 1 || out != "" || !strings.Contains(errs, "directory for alias acct does not exist in this worktree: ") {
		t.Fatalf("code=%d out=%q err=%q", code, out, errs)
	}
}

func TestNavigateWorktreeEscapingPathRefused(t *testing.T) {
	for _, bad := range []string{"../x", "/etc", "a/../../x", ".."} {
		app, main, _ := wtFixture(t)
		s, _ := store.Load(app.ConfigPath)
		repo := norm(t, filepath.Join(main, ".git"))
		if err := s.Put(store.Alias{Name: "bad", Path: bad, Worktree: true, Repo: repo}, false); err != nil {
			t.Fatal(err)
		}
		if err := s.Save(app.ConfigPath); err != nil {
			t.Fatal(err)
		}
		app.Cwd = main
		code, out, errs := run(t, app, "bad")
		if code != 1 || out != "" || !strings.Contains(errs, "alias bad has an invalid worktree path: "+bad) {
			t.Fatalf("%q: code=%d out=%q err=%q", bad, code, out, errs)
		}
	}
}

func TestWorktreeOpsWithoutGit(t *testing.T) {
	app, main, _ := wtFixture(t)
	registerAcct(t, app, main)
	plain := t.TempDir()
	if code, _, errs := run(t, app, "-r", plain, "plain"); code != 0 {
		t.Fatalf("code=%d err=%q", code, errs)
	}
	t.Setenv("PATH", t.TempDir())
	want := "tkto: " + gitx.ErrGitNotFound.Error() + "\n"
	code, _, errs := run(t, app, "-r", "-w", ".", "w2")
	if code != 1 || errs != want {
		t.Fatalf("register: code=%d err=%q", code, errs)
	}
	code, _, errs = run(t, app, "acct")
	if code != 1 || errs != want {
		t.Fatalf("navigate: code=%d err=%q", code, errs)
	}
	code, out, errs := run(t, app, "plain")
	if code != 0 || out != plain+"\n" {
		t.Fatalf("plain: code=%d out=%q err=%q", code, out, errs)
	}
}

func TestListShowsWorktreeMarker(t *testing.T) {
	app, main, _ := wtFixture(t)
	registerAcct(t, app, main)
	plain := t.TempDir()
	run(t, app, "-r", plain, "plain")
	t.Setenv("PATH", t.TempDir())
	code, out, errs := run(t, app, "-l")
	if code != 0 || errs != "" {
		t.Fatalf("code=%d err=%q", code, errs)
	}
	lines := strings.Split(strings.TrimRight(out, "\n"), "\n")
	if len(lines) != 2 {
		t.Fatalf("out=%q", out)
	}
	got := strings.Join(strings.Fields(lines[0]), " ")
	want := "acct " + svc + " [worktree: " + filepath.Base(main) + "]"
	if got != want {
		t.Fatalf("acct row = %q, want %q", got, want)
	}
	if strings.Contains(lines[1], "[") {
		t.Fatalf("plain row = %q", lines[1])
	}
}

func TestListAlignsMixedRows(t *testing.T) {
	app, main, _ := wtFixture(t)
	app.Cwd = filepath.Join(main, filepath.FromSlash(svc))
	if code, _, errs := run(t, app, "-r", "-w", ".", "a"); code != 0 {
		t.Fatalf("register a: %d %q", code, errs)
	}
	if code, _, errs := run(t, app, "-r", t.TempDir(), "b"); code != 0 {
		t.Fatalf("register b: %d %q", code, errs)
	}
	app.Cwd = main
	if code, _, errs := run(t, app, "-r", "-w", ".", "c"); code != 0 {
		t.Fatalf("register c: %d %q", code, errs)
	}
	code, out, errs := run(t, app, "-l")
	if code != 0 || errs != "" {
		t.Fatalf("code=%d err=%q", code, errs)
	}
	lines := strings.Split(strings.TrimRight(out, "\n"), "\n")
	if len(lines) != 3 {
		t.Fatalf("out=%q", out)
	}
	ia, ic := strings.Index(lines[0], "[worktree:"), strings.Index(lines[2], "[worktree:")
	if ia < 0 || ia != ic {
		t.Fatalf("worktree column not aligned across a plain row (%d vs %d):\n%s", ia, ic, out)
	}
	for _, l := range lines {
		if l != strings.TrimRight(l, " ") {
			t.Fatalf("trailing whitespace in %q", l)
		}
	}
}

func TestListWorktreesFromInsideRepo(t *testing.T) {
	app, _, linked := wtFixture(t)
	app.Cwd = linked
	code, out, errs := run(t, app, "-wl")
	lines := strings.Split(strings.TrimRight(out, "\n"), "\n")
	if code != 0 || errs != "" || len(lines) != 2 {
		t.Fatalf("code=%d out=%q err=%q", code, out, errs)
	}
	if !strings.Contains(lines[0], "main") || strings.HasPrefix(lines[0], "*") {
		t.Fatalf("first line = %q", lines[0])
	}
	if !strings.HasPrefix(lines[1], "*") || !strings.Contains(lines[1], "feature") {
		t.Fatalf("second line = %q", lines[1])
	}
}

func TestListWorktreesViaPlainAlias(t *testing.T) {
	app, main, _ := wtFixture(t)
	app.Cwd = main
	if code, _, errs := run(t, app, "-r", ".", "repo"); code != 0 {
		t.Fatalf("register: %d %q", code, errs)
	}
	app.Cwd = t.TempDir()
	code, out, errs := run(t, app, "-wl", "repo")
	lines := strings.Split(strings.TrimRight(out, "\n"), "\n")
	if code != 0 || errs != "" || len(lines) != 2 || strings.Contains(out, "*") {
		t.Fatalf("code=%d out=%q err=%q", code, out, errs)
	}
}

func TestListWorktreesViaWorktreeAlias(t *testing.T) {
	app, main, _ := wtFixture(t)
	registerAcct(t, app, main)
	app.Cwd = t.TempDir()
	code, out, errs := run(t, app, "-wl", "acct")
	lines := strings.Split(strings.TrimRight(out, "\n"), "\n")
	if code != 0 || errs != "" || len(lines) != 2 {
		t.Fatalf("code=%d out=%q err=%q", code, out, errs)
	}
}

func TestListWorktreesOutsideRepoNoAlias(t *testing.T) {
	app, _, _ := wtFixture(t)
	app.Cwd = t.TempDir()
	code, _, errs := run(t, app, "-wl")
	if code != 1 || !strings.Contains(errs, "not inside a git repository") {
		t.Fatalf("code=%d err=%q", code, errs)
	}
}

func TestListWorktreesAliasNotInRepo(t *testing.T) {
	app, _, _ := wtFixture(t)
	app.Cwd = t.TempDir()
	plain := t.TempDir()
	if code, _, errs := run(t, app, "-r", plain, "plain"); code != 0 {
		t.Fatalf("register: %d %q", code, errs)
	}
	code, _, errs := run(t, app, "-wl", "plain")
	if code != 1 || !strings.Contains(errs, "alias plain does not point into a git repository") {
		t.Fatalf("code=%d err=%q", code, errs)
	}
}

func TestCurrentWorktreeIndexLongestPrefix(t *testing.T) {
	wts := []gitx.Worktree{{Path: "/r"}, {Path: "/r-feature"}}
	if got := currentWorktreeIndex(wts, "/r-feature/x"); got != 1 {
		t.Fatalf("index = %d", got)
	}
	if got := currentWorktreeIndex(wts, "/other"); got != -1 {
		t.Fatalf("index = %d", got)
	}
}

func TestRelWithin(t *testing.T) {
	top := norm(t, t.TempDir())
	sub := filepath.Join(top, "a", "b")
	if err := os.MkdirAll(sub, 0o755); err != nil {
		t.Fatal(err)
	}
	if rel, ok, err := relWithin(top, sub); err != nil || !ok || rel != "a/b" {
		t.Fatalf("inside: %q %v %v", rel, ok, err)
	}
	if rel, ok, err := relWithin(top, top); err != nil || !ok || rel != "." {
		t.Fatalf("top: %q %v %v", rel, ok, err)
	}
	link := filepath.Join(t.TempDir(), "link")
	if err := os.Symlink(sub, link); err != nil {
		t.Fatal(err)
	}
	if rel, ok, err := relWithin(top, link); err != nil || !ok || rel != "a/b" {
		t.Fatalf("via symlink: %q %v %v", rel, ok, err)
	}
	outside := t.TempDir()
	rel, ok, err := relWithin(top, outside)
	if err != nil || ok || !strings.HasPrefix(rel, "../") {
		t.Fatalf("outside: %q %v %v", rel, ok, err)
	}
}

func TestBareWorktreeLabelled(t *testing.T) {
	// As gitx.parsePorcelain reports a bare entry.
	wts := []gitx.Worktree{{Path: "/srv/bare.git", Branch: "(bare)", Bare: true, Main: true}, {Path: "/srv/feat", Branch: "feat"}}
	lines := strings.Split(strings.TrimRight(formatWorktrees(wts, 1), "\n"), "\n")
	if got := strings.Fields(lines[0]); len(got) != 2 || got[0] != "(bare)" || got[1] != "/srv/bare.git" {
		t.Fatalf("bare row = %q", lines[0])
	}
	if items := worktreeItems(wts, 1); items[0].Label != "  (bare)" {
		t.Fatalf("bare item = %+v", items[0])
	}
}

func TestWorktreeItemsMarkCurrent(t *testing.T) {
	wts := []gitx.Worktree{{Path: "/r", Branch: "main"}, {Path: "/r-feature", Branch: "feature"}}
	items := worktreeItems(wts, currentWorktreeIndex(wts, "/r-feature/sub"))
	if len(items) != 2 || items[0].Label != "  main" || items[1].Label != "* feature" {
		t.Fatalf("items = %+v", items)
	}
	if items[1].Value != "/r-feature" || items[1].Detail != "/r-feature" {
		t.Fatalf("item = %+v", items[1])
	}
	for _, it := range worktreeItems(wts, -1) {
		if !strings.HasPrefix(it.Label, "  ") {
			t.Fatalf("unexpected marker: %+v", it)
		}
	}
}
