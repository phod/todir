package shellinit

import (
	"errors"
	"strings"
	"testing"
)

const fakeCompletion = "# fake completion script\n"

const zshGuard = "if ! (( $+functions[compdef] )); then autoload -Uz compinit && compinit; fi"

func TestScriptBashContainsFunctionAndCompletion(t *testing.T) {
	got, err := Script("bash", fakeCompletion)
	if err != nil {
		t.Fatalf("Script: %v", err)
	}
	for _, want := range []string{"todir() {", "command todir", "TODIR_CD_FD=3", fakeCompletion} {
		if !strings.Contains(got, want) {
			t.Errorf("bash script missing %q:\n%s", want, got)
		}
	}
	if strings.Contains(got, "compinit") {
		t.Errorf("bash script must not contain compinit guard:\n%s", got)
	}
	if !strings.HasPrefix(got, Function) {
		t.Errorf("bash script must start with Function:\n%s", got)
	}
}

func TestScriptZshHasCompinitGuardBeforeCompletion(t *testing.T) {
	got, err := Script("zsh", fakeCompletion)
	if err != nil {
		t.Fatalf("Script: %v", err)
	}
	fn := strings.Index(got, "todir() {")
	guard := strings.Index(got, zshGuard)
	comp := strings.Index(got, fakeCompletion)
	if fn < 0 || guard < 0 || comp < 0 {
		t.Fatalf("missing part: fn=%d guard=%d comp=%d\n%s", fn, guard, comp, got)
	}
	if !(fn < guard && guard < comp) {
		t.Fatalf("wrong order: fn=%d guard=%d comp=%d", fn, guard, comp)
	}
}

func TestScriptUnsupportedShell(t *testing.T) {
	_, err := Script("fish", fakeCompletion)
	if !errors.Is(err, ErrUnsupportedShell) {
		t.Fatalf("err = %v, want ErrUnsupportedShell", err)
	}
	if err.Error() != `unsupported shell "fish"; supported: bash, zsh` {
		t.Fatalf("message = %q", err.Error())
	}
}
