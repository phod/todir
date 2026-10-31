package cli

import (
	"fmt"
	"path/filepath"
)

// emitTarget tells the shell function where to cd: the CdWriter when set,
// otherwise stdout.
func (a *App) emitTarget(path string) error {
	w := a.Stdout
	if a.CdWriter != nil {
		w = a.CdWriter
	}
	_, err := fmt.Fprintf(w, "%s\n", path)
	return err
}

// ShouldNotice reports whether the missing-shell-integration notice applies.
func ShouldNotice(k Kind, cdWriterSet, stderrIsTTY bool) bool {
	if k == KindInit || k == KindVersion {
		return false
	}
	return !cdWriterSet && stderrIsTTY
}

// printNotice writes the shell-integration notice to Stderr.
func (a *App) printNotice() {
	fmt.Fprintln(a.Stderr, "tele: shell integration is not loaded, so tele cannot change directory.")
	shell := filepath.Base(a.Getenv("SHELL"))
	if shell == "bash" || shell == "zsh" {
		fmt.Fprintf(a.Stderr, "      Add this to your shell rc file:  eval \"$(tele --init %s)\"\n", shell)
	}
}
