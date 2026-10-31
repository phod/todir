package cli

import (
	"bytes"
	"strings"
	"testing"
)

func newTestApp(t *testing.T) (*App, *bytes.Buffer, *bytes.Buffer) {
	t.Helper()
	out, errb := &bytes.Buffer{}, &bytes.Buffer{}
	app := &App{
		Stdout:     out,
		Stderr:     errb,
		Stdin:      strings.NewReader(""),
		Getenv:     func(string) string { return "" },
		Cwd:        t.TempDir(),
		ConfigPath: t.TempDir() + "/config.json",
		Version:    "1.2.3",
	}
	return app, out, errb
}

func TestRunVersionPrintsToStdout(t *testing.T) {
	app, out, _ := newTestApp(t)
	if code := Run(app, []string{"--version"}); code != 0 {
		t.Fatalf("exit = %d, want 0", code)
	}
	if got := out.String(); got != "tele 1.2.3\n" {
		t.Fatalf("stdout = %q", got)
	}
}

func TestRunUnknownFlagIsUsageError(t *testing.T) {
	app, _, errb := newTestApp(t)
	if code := Run(app, []string{"--bogus"}); code != 2 {
		t.Fatalf("exit = %d, want 2", code)
	}
	if !strings.HasPrefix(errb.String(), "tele: ") {
		t.Fatalf("stderr = %q", errb.String())
	}
}

func TestParseCdFD(t *testing.T) {
	if fd, set, err := ParseCdFD(""); fd != 0 || set || err != nil {
		t.Fatalf("empty: %d %v %v", fd, set, err)
	}
	if fd, set, err := ParseCdFD("3"); fd != 3 || !set || err != nil {
		t.Fatalf("3: %d %v %v", fd, set, err)
	}
	if _, _, err := ParseCdFD("x"); err == nil {
		t.Fatal("want error for x")
	}
	if _, _, err := ParseCdFD("-1"); err == nil {
		t.Fatal("want error for -1")
	}
}

func TestInitWithoutShellListsShells(t *testing.T) {
	for _, args := range [][]string{{"--init"}, {"--init="}} {
		app, out, errb := newTestApp(t)
		if code := Run(app, args); code != 2 {
			t.Fatalf("%v: exit = %d, want 2", args, code)
		}
		if got := errb.String(); got != "tele: --init requires a shell name: bash or zsh\n" || out.Len() != 0 {
			t.Fatalf("%v: stderr = %q stdout = %q", args, got, out.String())
		}
	}
}
