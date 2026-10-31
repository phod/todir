package cli

import (
	"bytes"
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/phod/tkto/internal/picker"
	"github.com/phod/tkto/internal/store"
)

func TestPickAliasWithoutTTYErrors(t *testing.T) {
	app, _, _ := newTestApp(t)
	if code, _, errs := run(t, app, "-r", t.TempDir(), "foo"); code != 0 {
		t.Fatalf("register: %d %q", code, errs)
	}
	app.StderrIsTTY = false
	code, out, errs := run(t, app)
	if code != 1 || out != "" || errs != "tkto: interactive mode needs a terminal\n" {
		t.Fatalf("code=%d out=%q err=%q", code, out, errs)
	}
}

func TestPickAliasEmptyStoreHint(t *testing.T) {
	app, _, _ := newTestApp(t)
	app.StderrIsTTY = true
	app.CdWriter = &bytes.Buffer{} // shell integration loaded: no notice
	code, out, errs := run(t, app, "-li")
	want := "tkto: no aliases registered; add one with: tkto -r <path> <name>\n"
	if code != 1 || out != "" || errs != want {
		t.Fatalf("code=%d out=%q err=%q", code, out, errs)
	}
}

func TestPickWorktreeWithoutTTYErrors(t *testing.T) {
	app, main, _ := wtFixture(t)
	app.Cwd = main
	app.StderrIsTTY = false
	code, out, errs := run(t, app, "-wli")
	if code != 1 || out != "" || errs != "tkto: interactive mode needs a terminal\n" {
		t.Fatalf("code=%d out=%q err=%q", code, out, errs)
	}
}

func TestExitCodeCancelledIs130(t *testing.T) {
	if got := exitCode(picker.ErrCancelled); got != 130 {
		t.Fatalf("exitCode = %d", got)
	}
	if got := exitCode(fmt.Errorf("wrap: %w", picker.ErrCancelled)); got != 130 {
		t.Fatalf("wrapped exitCode = %d", got)
	}
	if got := exitCode(fmt.Errorf("wrap: %w", usagef("x"))); got != 2 {
		t.Fatalf("wrapped usage exitCode = %d", got)
	}
}

// run executes the app with fresh stdout/stderr buffers.
func run(t *testing.T, app *App, args ...string) (int, string, string) {
	t.Helper()
	out, errb := &bytes.Buffer{}, &bytes.Buffer{}
	app.Stdout, app.Stderr = out, errb
	code := Run(app, args)
	return code, out.String(), errb.String()
}

func TestRegisterThenNavigate(t *testing.T) {
	app, _, _ := newTestApp(t)
	dir := t.TempDir()
	code, out, errs := run(t, app, "-r", dir, "foo")
	if code != 0 || out != "" || errs != "registered foo → "+dir+"\n" {
		t.Fatalf("register: code=%d out=%q err=%q", code, out, errs)
	}
	code, out, errs = run(t, app, "foo")
	if code != 0 || out != dir+"\n" || errs != "" {
		t.Fatalf("navigate: code=%d out=%q err=%q", code, out, errs)
	}
}

func TestRegisterRelativePathStoredAbsolute(t *testing.T) {
	app, _, _ := newTestApp(t)
	if code, _, errs := run(t, app, "-r", ".", "foo"); code != 0 {
		t.Fatalf("code=%d err=%q", code, errs)
	}
	s, err := store.Load(app.ConfigPath)
	if err != nil {
		t.Fatal(err)
	}
	if got := s.Aliases["foo"].Path; got != app.Cwd {
		t.Fatalf("Path = %q, want %q", got, app.Cwd)
	}
}

func TestRegisterDuplicateFailsWithoutForce(t *testing.T) {
	app, _, _ := newTestApp(t)
	dir := t.TempDir()
	run(t, app, "-r", dir, "foo")
	code, _, errs := run(t, app, "-r", dir, "foo")
	if code != 1 || errs != "tkto: alias foo already exists; use -f to overwrite\n" {
		t.Fatalf("code=%d err=%q", code, errs)
	}
}

func TestRegisterDuplicateWithForceOverwrites(t *testing.T) {
	app, _, _ := newTestApp(t)
	d1, d2 := t.TempDir(), t.TempDir()
	run(t, app, "-r", d1, "foo")
	if code, _, errs := run(t, app, "-r", "-f", d2, "foo"); code != 0 {
		t.Fatalf("code=%d err=%q", code, errs)
	}
	_, out, _ := run(t, app, "foo")
	if out != d2+"\n" {
		t.Fatalf("out=%q want %q", out, d2)
	}
}

func TestRegisterInvalidNameIsUsage(t *testing.T) {
	for _, name := range []string{"-x", "foo/bar"} {
		app, _, _ := newTestApp(t)
		code, _, errs := run(t, app, "-r", t.TempDir(), "--", name)
		if code != 2 || !strings.HasPrefix(errs, "tkto: invalid alias name") {
			t.Fatalf("%s: code=%d err=%q", name, code, errs)
		}
		if _, err := os.Stat(app.ConfigPath); err == nil {
			t.Fatalf("%s: store file created", name)
		}
	}
}

func TestRegisterNonDirectoryFails(t *testing.T) {
	dir := t.TempDir()
	file := filepath.Join(dir, "f.txt")
	if err := os.WriteFile(file, nil, 0o644); err != nil {
		t.Fatal(err)
	}
	for _, p := range []string{file, filepath.Join(dir, "missing")} {
		app, _, _ := newTestApp(t)
		code, _, errs := run(t, app, "-r", p, "foo")
		if code != 1 || !strings.Contains(errs, p) {
			t.Fatalf("%s: code=%d err=%q", p, code, errs)
		}
		if _, err := os.Stat(app.ConfigPath); err == nil {
			t.Fatalf("%s: store file created", p)
		}
	}
}

func TestNavigateUnknownAlias(t *testing.T) {
	app, _, _ := newTestApp(t)
	code, _, errs := run(t, app, "nope")
	if code != 1 || errs != "tkto: alias nope not found\n" {
		t.Fatalf("code=%d err=%q", code, errs)
	}
}

func TestNavigateMissingDirectory(t *testing.T) {
	app, _, _ := newTestApp(t)
	dir := t.TempDir()
	run(t, app, "-r", dir, "foo")
	os.Remove(dir)
	code, _, errs := run(t, app, "foo")
	if code != 1 || !strings.Contains(errs, "does not exist") || !strings.Contains(errs, dir) {
		t.Fatalf("code=%d err=%q", code, errs)
	}
}

func TestNavigateWritesToCdWriter(t *testing.T) {
	app, _, _ := newTestApp(t)
	dir := t.TempDir()
	run(t, app, "-r", dir, "foo")
	cd := &bytes.Buffer{}
	app.CdWriter = cd
	code, out, _ := run(t, app, "foo")
	if code != 0 || out != "" || cd.String() != dir+"\n" {
		t.Fatalf("code=%d out=%q cd=%q", code, out, cd.String())
	}
}

func TestDeleteRemovesAlias(t *testing.T) {
	app, _, _ := newTestApp(t)
	run(t, app, "-r", t.TempDir(), "foo")
	code, _, errs := run(t, app, "-d", "foo")
	if code != 0 || errs != "deleted foo\n" {
		t.Fatalf("code=%d err=%q", code, errs)
	}
	code, _, errs = run(t, app, "foo")
	if code != 1 || !strings.Contains(errs, "alias foo not found") {
		t.Fatalf("code=%d err=%q", code, errs)
	}
}

func TestDeleteUnknownAlias(t *testing.T) {
	app, _, _ := newTestApp(t)
	code, _, errs := run(t, app, "-d", "nope")
	if code != 1 || errs != "tkto: alias nope not found\n" {
		t.Fatalf("code=%d err=%q", code, errs)
	}
}

func TestListSortedPlain(t *testing.T) {
	app, _, _ := newTestApp(t)
	d1, d2 := t.TempDir(), t.TempDir()
	run(t, app, "-r", d2, "bb")
	run(t, app, "-r", d1, "a")
	code, out, errs := run(t, app, "-l")
	want := "a   " + d1 + "\nbb  " + d2 + "\n"
	if code != 0 || out != want || errs != "" {
		t.Fatalf("code=%d out=%q want=%q err=%q", code, out, want, errs)
	}
}

func TestListEmptyPrintsHint(t *testing.T) {
	app, _, _ := newTestApp(t)
	code, out, errs := run(t, app, "-l")
	if code != 0 || out != "" || errs != "tkto: no aliases registered; add one with: tkto -r <path> <name>\n" {
		t.Fatalf("code=%d out=%q err=%q", code, out, errs)
	}
}

func TestNoticeOnlyWhenBareAndTTY(t *testing.T) {
	app, _, _ := newTestApp(t)
	app.StderrIsTTY = true
	_, _, errs := run(t, app, "-l")
	if !strings.HasPrefix(errs, "tkto: shell integration is not loaded") {
		t.Fatalf("expected notice, got %q", errs)
	}
	app.CdWriter = &bytes.Buffer{}
	_, _, errs = run(t, app, "-l")
	if strings.Contains(errs, "shell integration") {
		t.Fatalf("unexpected notice with CdWriter: %q", errs)
	}
	app.CdWriter = nil
	_, _, errs = run(t, app, "--version")
	if strings.Contains(errs, "shell integration") {
		t.Fatalf("unexpected notice for --version: %q", errs)
	}
}

func TestInitEmptyValueIsUsage(t *testing.T) {
	app, _, _ := newTestApp(t)
	code, _, errs := run(t, app, "--init", "")
	if code != 2 || errs != "tkto: --init requires a shell name: bash or zsh\n" {
		t.Fatalf("code=%d err=%q", code, errs)
	}
}

func TestInitBadShellIsUsage(t *testing.T) {
	app, _, _ := newTestApp(t)
	code, out, errs := run(t, app, "--init", "fish")
	want := "tkto: unsupported shell \"fish\"; supported: bash, zsh\n"
	if code != 2 || out != "" || errs != want {
		t.Fatalf("code=%d out=%q err=%q", code, out, errs)
	}
}

func TestInitZshToStdoutNoNotice(t *testing.T) {
	app, _, _ := newTestApp(t)
	app.StderrIsTTY = true
	app.CdWriter = nil
	code, out, errs := run(t, app, "--init", "zsh")
	if code != 0 || !strings.Contains(out, "tkto() {") || errs != "" {
		t.Fatalf("code=%d err=%q out=%q", code, errs, out)
	}
}
