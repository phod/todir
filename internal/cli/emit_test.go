package cli

import (
	"bytes"
	"strings"
	"testing"
)

func TestEmitTargetUsesCdWriterWhenSet(t *testing.T) {
	var out, cd bytes.Buffer
	a := &App{Stdout: &out, CdWriter: &cd}
	if err := a.emitTarget("/p"); err != nil {
		t.Fatal(err)
	}
	if cd.String() != "/p\n" {
		t.Errorf("cd = %q", cd.String())
	}
	if out.Len() != 0 {
		t.Errorf("stdout = %q", out.String())
	}
}

func TestEmitTargetFallsBackToStdout(t *testing.T) {
	var out bytes.Buffer
	a := &App{Stdout: &out}
	if err := a.emitTarget("/p"); err != nil {
		t.Fatal(err)
	}
	if out.String() != "/p\n" {
		t.Errorf("stdout = %q", out.String())
	}
}

func TestShouldNotice(t *testing.T) {
	kinds := []Kind{KindNavigate, KindPickAlias, KindRegister, KindDelete, KindListAliases,
		KindListWorktrees, KindPickWorktree, KindInit, KindVersion}
	for _, k := range kinds {
		for _, cd := range []bool{false, true} {
			for _, tty := range []bool{false, true} {
				want := !cd && tty && k != KindInit && k != KindVersion
				if got := ShouldNotice(k, cd, tty); got != want {
					t.Errorf("ShouldNotice(%v,%v,%v)=%v want %v", k, cd, tty, got, want)
				}
			}
		}
	}
}

func TestPrintNoticeText(t *testing.T) {
	first := "tele: shell integration is not loaded, so tele cannot change directory.\n"
	var errb bytes.Buffer
	a := &App{Stderr: &errb, Getenv: func(string) string { return "/bin/zsh" }}
	a.printNotice()
	want := first + "      Add this to your shell rc file:  eval \"$(tele --init zsh)\"\n"
	if errb.String() != want {
		t.Errorf("got %q want %q", errb.String(), want)
	}
	if !strings.Contains(errb.String(), `eval "$(tele --init zsh)"`) {
		t.Error("missing eval line")
	}

	errb.Reset()
	a.Getenv = func(string) string { return "/bin/fish" }
	a.printNotice()
	if errb.String() != first {
		t.Errorf("fish got %q", errb.String())
	}
}
