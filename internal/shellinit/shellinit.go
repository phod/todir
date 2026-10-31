// Package shellinit renders the shell integration printed by `tele --init`.
// It knows nothing about Cobra: the completion script is passed in as text.
package shellinit

import (
	"errors"
	"fmt"
)

// ErrUnsupportedShell is returned by Script for shells other than bash and zsh.
var ErrUnsupportedShell = errors.New("unsupported shell")

// Function is the `tele` shell function. It shadows the binary, runs it with
// TELE_CD_FD=3 so navigation targets arrive on fd 3 while stdout and stderr
// pass through, cds when a target was written, and returns the binary's exit
// status. It must work on bash 3.2, bash 5 and zsh 5.x.
const Function = `tele() {
  local __tele_target __tele_rc
  { __tele_target=$(TELE_CD_FD=3 command tele "$@" 3>&1 1>&4); __tele_rc=$?; } 4>&1
  if [ -n "$__tele_target" ]; then
    cd -- "$__tele_target" || return
  fi
  return "$__tele_rc"
}
`

// zshCompinitGuard initialises the completion system only when the user has
// not already done so, because Cobra's zsh script ends in compdef.
const zshCompinitGuard = "if ! (( $+functions[compdef] )); then autoload -Uz compinit && compinit; fi\n"

type unsupportedShellError struct{ shell string }

func (e *unsupportedShellError) Error() string {
	return fmt.Sprintf("unsupported shell %q; supported: bash, zsh", e.shell)
}

func (e *unsupportedShellError) Is(target error) bool { return target == ErrUnsupportedShell }

// Script returns the integration for shell: the function, then (zsh only) the
// compinit guard, then the completion script.
func Script(shell, completion string) (string, error) {
	switch shell {
	case "bash":
		return Function + completion, nil
	case "zsh":
		return Function + zshCompinitGuard + completion, nil
	}
	return "", &unsupportedShellError{shell: shell}
}
