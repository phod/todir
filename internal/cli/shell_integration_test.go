//go:build !windows

package cli

import (
	"bytes"
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"runtime"
	"strings"
	"testing"
)

// todirBinDir holds the freshly built todir binary for the shell tests.
var todirBinDir string

func TestMain(m *testing.M) {
	os.Exit(runMain(m))
}

func runMain(m *testing.M) int {
	tmp, err := os.MkdirTemp("", "todir-shell-test-")
	if err != nil {
		fmt.Fprintln(os.Stderr, "mkdir temp:", err)
		return 1
	}
	defer os.RemoveAll(tmp)
	todirBinDir = filepath.Join(tmp, "bin")
	build := exec.Command("go", "build", "-o", filepath.Join(todirBinDir, "todir"), "../../cmd/todir")
	if out, err := build.CombinedOutput(); err != nil {
		fmt.Fprintf(os.Stderr, "go build: %v\n%s", err, out)
		return 1
	}
	return m.Run()
}

type testShell struct {
	name string // "bash" or "zsh"
	path string // resolved path
}

// shellsForTest returns the shells to run. On darwin and linux both bash and
// zsh must be present, so CI cannot silently lose coverage of either; on other
// systems a missing shell is a skip.
func shellsForTest(t *testing.T) []testShell {
	t.Helper()
	shells := availableShells()
	names := map[string]bool{}
	for _, sh := range shells {
		names[sh.name] = true
	}
	if len(names) < 2 {
		msg := fmt.Sprintf("need both bash and zsh, found %d distinct shell(s): %v", len(names), shells)
		if runtime.GOOS == "darwin" || runtime.GOOS == "linux" {
			t.Fatal(msg)
		}
		if len(shells) == 0 {
			t.Skip(msg)
		}
	}
	return shells
}

// availableShells returns bash and zsh binaries found on this machine,
// deduplicated by their resolved path.
func availableShells() []testShell {
	var shells []testShell
	seen := map[string]bool{}
	for _, cand := range []string{"/bin/bash", "bash", "zsh"} {
		p, err := exec.LookPath(cand)
		if err != nil {
			continue
		}
		real, err := filepath.EvalSymlinks(p)
		if err != nil {
			continue
		}
		if seen[real] {
			continue
		}
		seen[real] = true
		shells = append(shells, testShell{name: filepath.Base(cand), path: real})
	}
	return shells
}

// runShell runs script under sh in dir with an isolated HOME and config.
func runShell(t *testing.T, sh testShell, dir, script string, extraEnv ...string) (string, string, error) {
	t.Helper()
	tmp := t.TempDir()
	home := filepath.Join(tmp, "home")
	xdg := filepath.Join(tmp, "xdg")
	for _, d := range []string{home, xdg} {
		if err := os.MkdirAll(d, 0o755); err != nil {
			t.Fatal(err)
		}
	}
	var args []string
	if sh.name == "zsh" {
		args = []string{"-f", "-c", script}
	} else {
		args = []string{"-c", script}
	}
	cmd := exec.Command(sh.path, args...)
	cmd.Dir = dir
	env := []string{}
	for _, kv := range os.Environ() {
		k, _, _ := strings.Cut(kv, "=")
		switch k {
		case "PATH", "HOME", "XDG_CONFIG_HOME", "ZDOTDIR", "TODIR_CD_FD":
			continue
		}
		env = append(env, kv)
	}
	env = append(env,
		"PATH="+todirBinDir+string(os.PathListSeparator)+os.Getenv("PATH"),
		"HOME="+home,
		"XDG_CONFIG_HOME="+xdg,
	)
	if sh.name == "zsh" {
		env = append(env, "ZDOTDIR="+home)
	}
	cmd.Env = append(env, extraEnv...)
	var out, errb bytes.Buffer
	cmd.Stdout, cmd.Stderr = &out, &errb
	err := cmd.Run()
	return out.String(), errb.String(), err
}

func shellVersion(t *testing.T, sh testShell) string {
	t.Helper()
	out, err := exec.Command(sh.path, "--version").Output()
	if err != nil {
		return "unknown"
	}
	return strings.SplitN(strings.TrimSpace(string(out)), "\n", 2)[0]
}

const navigateScript = `set -e
eval "$(todir --init SHELLNAME)"
todir -r "$TARGET" foo 2>/dev/null
cd /
todir foo
echo "pwd=$(pwd -P)"
out=$(todir -l | cat)
echo "list=$out"
echo "pwd2=$(pwd -P)"
set +e
todir nope 2>err.txt; echo "rc=$?"
echo "pwd3=$(pwd -P)"
grep -c 'shell integration is not loaded' err.txt || true
type todir | head -1
`

func TestShellIntegrationNavigatesAndPropagatesExit(t *testing.T) {
	shells := shellsForTest(t)
	for _, sh := range shells {
		t.Run(sh.name+"@"+sh.path, func(t *testing.T) {
			t.Logf("shell: %s", shellVersion(t, sh))
			target := t.TempDir()
			want, err := filepath.EvalSymlinks(target)
			if err != nil {
				t.Fatal(err)
			}
			script := strings.ReplaceAll(navigateScript, "SHELLNAME", sh.name)
			out, errs, err := runShell(t, sh, t.TempDir(), script, "TARGET="+target)
			if err != nil {
				t.Fatalf("shell failed: %v\nstdout:\n%s\nstderr:\n%s", err, out, errs)
			}
			if errs != "" {
				t.Errorf("unexpected stderr: %q", errs)
			}
			lines := strings.Split(strings.TrimRight(out, "\n"), "\n")
			if len(lines) != 7 {
				t.Fatalf("want 7 output lines, got %d:\n%s", len(lines), out)
			}
			checks := []struct{ got, want string }{
				{lines[0], "pwd=" + want},
				{lines[2], "pwd2=" + want},
				{lines[3], "rc=1"},
				{lines[4], "pwd3=" + want},
				{lines[5], "0"},
			}
			for _, c := range checks {
				if c.got != c.want {
					t.Errorf("got %q, want %q\nfull output:\n%s", c.got, c.want, out)
				}
			}
			if !strings.HasPrefix(lines[1], "list=") || !strings.Contains(lines[1], "foo") {
				t.Errorf("list line = %q, want it to contain foo", lines[1])
			}
			if !strings.Contains(lines[6], "function") {
				t.Errorf("type todir = %q, want a function", lines[6])
			}
			errTxt, err := os.ReadFile(filepath.Join(target, "err.txt"))
			if err != nil {
				t.Fatalf("read err.txt: %v", err)
			}
			if string(errTxt) != "todir: alias nope not found\n" {
				t.Errorf("err.txt = %q", errTxt)
			}
		})
	}
}

func TestShellIntegrationCompletionLoads(t *testing.T) {
	shells := shellsForTest(t)
	for _, sh := range shells {
		t.Run(sh.name+"@"+sh.path, func(t *testing.T) {
			t.Logf("shell: %s", shellVersion(t, sh))
			var script string
			if sh.name == "zsh" {
				script = `eval "$(todir --init zsh)" || exit 1
whence -w _todir`
			} else {
				script = `eval "$(todir --init bash)" || exit 1
if [ "${BASH_VERSINFO[0]}" -lt 4 ]; then echo "SKIP bash $BASH_VERSION"; exit 0; fi
complete -p todir`
			}
			out, errs, err := runShell(t, sh, t.TempDir(), script)
			if err != nil {
				t.Fatalf("shell failed: %v\nstdout:\n%s\nstderr:\n%s", err, out, errs)
			}
			if errs != "" {
				t.Errorf("unexpected stderr: %q", errs)
			}
			out = strings.TrimSpace(out)
			if strings.HasPrefix(out, "SKIP ") {
				t.Skipf("completion check needs bash >= 4 (%s); function loaded without error", strings.TrimPrefix(out, "SKIP "))
			}
			if sh.name == "zsh" {
				if out != "_todir: function" {
					t.Fatalf("whence -w _todir = %q", out)
				}
			} else if !strings.Contains(out, "todir") {
				t.Fatalf("complete -p todir = %q", out)
			}
		})
	}
}
