package cli

import (
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"testing"

	"github.com/phodder/tele/internal/gitx"
	"github.com/phodder/tele/internal/store"
)

func git(t *testing.T, args ...string) {
	t.Helper()
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
	git(t, "-C", main, "-c", "user.name=t", "-c", "user.email=t@example.com", "commit", "-q", "-m", "init")
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
	git(t, "-C", main, "-c", "user.name=t", "-c", "user.email=t@example.com", "commit", "-q", "-m", "svc")
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
	want := "tele: not inside a git repository; alias acct belongs to repo " + filepath.Base(main) + " (" + s.Aliases["acct"].Repo + ")\n"
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
	want := "tele: in repository " + filepath.Base(other) + "; alias acct belongs to " + filepath.Base(main) + "\n"
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
	want := "tele: " + gitx.ErrGitNotFound.Error() + "\n"
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
