# Tele CLI — Design

Date: 2026-10-31
Status: draft for review
Human-facing spec: `HUMAN_SPEC.md` (this document refines it; where they differ, this document wins)

## 1. Intent

Tele is a directory-alias tool for the shell. You register a short name for a
directory once, then `tele <name>` takes you there, with tab completion. That
part is well-trodden ground.

The reason Tele exists is git worktrees. The author works in a monorepo with
several worktrees checked out at once, one per stream of work, often with a
different AI coding agent in each. The repo has the same shape in every
worktree, so "account-service" always means `services/step/account-service`
relative to *whichever worktree you are currently standing in*. A conventional
alias tool binds a name to one absolute path and is therefore useless the moment
you switch worktrees, unless you re-register every alias in every worktree,
which is laborious and goes stale within days.

Tele's differentiator is the **worktree-aware alias**: a name bound to a path
relative to a repo root, resolved at jump time against the worktree you are in.

Prior art: the author's own private tool `goto`, which worked well for plain
aliases. Its pain points shaped this spec: too many aliases to remember (hence
the interactive picker), no worktree support, and reserved words instead of
flags for commands (`goto add . name`), which is fragile. The name `goto` is
taken by an existing public tool, and Tele is intended for eventual public
distribution.

## 2. Goals, non-goals, anti-goals

### Goals

- `tele <name>` jumps to a registered directory, with tab completion of names.
- Worktree-aware aliases resolve relative to the current worktree of the
  registered repo.
- List registered aliases, plainly or in an interactive filterable picker.
- List the worktrees of a repo, plainly or in an interactive picker that jumps
  to the chosen worktree. Works from inside the repo or via an alias into it.
- One-line shell setup: add `eval "$(tele --init zsh)"` (or `bash`) to the rc
  file. The README documents it; the binary generates everything behind it.
- Set and forget: install once, register as you go, never think about it again.

### Non-goals

- Frecency, auto-tracking of visited directories, or any "smart" jumping
  (zoxide/autojump territory). Tele only knows what you registered.
- Matching across separate clones of the same repo. Worktrees made from one
  clone are the only case that matters.
- Creating, removing, or otherwise managing worktrees. Tele only reads them.
- fish, PowerShell, or other shells in v1. The design must not preclude them.
- Windows in v1.

### Anti-goals (would make it a failure even if it works)

- A jump that is perceptibly slower than a plain `cd`.
- Config that needs tending, migrating, or repairing.
- Requiring git for plain aliases.
- Behaviour that differs between bash and zsh, or between macOS and Linux.
- An alias name that can collide with a command. Commands are flags, never
  positional words.

## 3. Constraints

- **A child process cannot change its parent shell's working directory.** The
  user-facing `tele` is therefore a shell function, emitted by `tele --init`,
  that wraps the `tele` binary via `command tele`. This is the pyenv/rbenv
  pattern. See §6.
- git must be installed for any worktree feature. Plain aliases must work
  without it.
- Platforms: macOS and Linux, amd64 and arm64, from day one.
- Shells: bash and zsh in v1.
- The author's daily driver is zsh on macOS; agents run on Linux.

## 4. Technology (decided)

- **Language:** Go. Rationale: millisecond startup, single static binary, trivial
  cross-compilation, small enough codebase to read in a sitting.
- **CLI parsing and completion:** Cobra (with pflag). Handles combined short
  flags like `-wli`, generates bash and zsh completion scripts, and supports
  dynamic completion so alias names can be offered on tab.
- **Interactive pickers:** Bubble Tea with the Bubbles `list` component. Used
  only for the `-i` modes. Renders to **stderr**, never stdout.
- **Alias store:** a single TOML file. Human-readable and hand-editable.
- **Distribution:** `go install` and GitHub Releases with prebuilt binaries for
  the four platform targets. A Homebrew tap is intended later and nothing may
  preclude it.

## 5. Command surface (decided)

The command is `tele`. There are no subcommands. Flags select the operation;
positional arguments supply names and paths.

| Flag | Long form | Meaning |
|------|-----------|---------|
| `-r` | `--register` | Register an alias |
| `-w` | `--worktree` | Worktree mode (modifier on `-r`; selects worktree listing with `-l`) |
| `-f` | `--force` | Overwrite an existing alias on register |
| `-l` | `--list` | List aliases (or worktrees with `-w`) |
| `-i` | `--interactive` | Interactive picker (modifier on `-l` / `-wl`) |
| `-d` | `--delete` | Delete an alias |
|      | `--init <shell>` | Print the shell function and completions; evaluated at shell start |
| `-v` | `--version` | Print version |
| `-h` | `--help` | Help |

### Operations

| Invocation | Operation | Navigates? |
|------------|-----------|------------|
| `tele <alias>` | Jump to alias | yes |
| `tele` | Interactive alias picker (same as `tele -li`) | yes |
| `tele -r [-w] [-f] <path> <alias>` | Register | no |
| `tele -d <alias>` | Delete | no |
| `tele -l` | List aliases | no |
| `tele -li` (or `tele -i`) | Interactive alias picker | yes |
| `tele -wl [alias]` | List worktrees of current repo, or of the repo behind `alias` | no |
| `tele -wli [alias]` | Interactive worktree picker | yes |
| `tele --init <bash\|zsh>` | Print the shell function and completions (evaluated at shell start) | no |
| `tele -v`, `tele -h` | Version / help | no |

Mode resolution order: `--init`, `-r`, `-d`, `-w`+`-l`, `-l`/`-i`, then
positional-only. Any other combination (e.g. `-r -d`, `-l <alias>`, `-w` alone,
two positionals without `-r`) is a usage error.

`tele -i` without `-l` is accepted and means `tele -li`. (Author's call:
forgiving, harmless.)

### Alias names

Match `^[A-Za-z0-9][A-Za-z0-9._-]*$`. Must not start with `-`, so a name can
never be mistaken for a flag. Case-sensitive.

### Exit codes

| Code | Meaning |
|------|---------|
| 0 | Success |
| 1 | Runtime error: alias not found, directory missing, not in repo, wrong repo, git missing, alias exists without `-f` |
| 2 | Usage error (bad flag combination, missing argument, invalid alias name) |
| 130 | Interactive picker cancelled (Escape or Ctrl-C). Nothing is printed. |

All error messages go to stderr, are one line, and say what to do next where
there is an obvious next step (e.g. "use -f to overwrite").

## 6. Shell integration (decided)

### One binary, one name, one function

The binary is installed as `tele`. Shell integration defines a function also
named `tele` which shadows the binary and reaches it with `command tele`,
exactly as pyenv and rbenv do. There is no second binary name.

Install story (README):

```
brew install tele            # or: go install <module>/cmd/tele@latest
echo 'eval "$(tele --init zsh)"' >> ~/.zshrc     # or bash / ~/.bashrc
```

`tele --init <shell>` prints the `tele` function and the completion script for
that shell. Because the rc file holds only the `eval` line, the integration is
regenerated by the installed binary at every shell start and can never go stale
after an upgrade. Cost: one process spawn per shell start, a few milliseconds.

### How the function knows when to `cd`

The binary decides whether an invocation is a navigation. The function must not
re-implement flag parsing. The mechanism:

- The function runs `command tele` with the environment variable `TELE_CD_FD=3`
  and file descriptor 3 attached to a capture. Everything on the binary's stdout
  and stderr passes through untouched, so `tele -l | grep foo` behaves like an
  ordinary command.
- When the binary wants the shell to change directory it writes the absolute
  target path, followed by a newline, to the descriptor named by `TELE_CD_FD`.
  It writes nothing there otherwise.
- The function runs `cd -- "$target"` if the capture is non-empty, then returns
  the binary's exit status.
- When `TELE_CD_FD` is **not** set (binary run directly, tests, scripts),
  navigation modes print the target path on stdout instead.

Sketch (builder verifies on bash 3.2, bash 5, zsh 5.x):

```sh
tele() {
  local __tele_target __tele_rc
  { __tele_target=$(TELE_CD_FD=3 command tele "$@" 3>&1 1>&4); __tele_rc=$?; } 4>&1
  if [ -n "$__tele_target" ]; then
    cd -- "$__tele_target" || return
  fi
  return "$__tele_rc"
}
```

macOS ships bash 3.2 as `/bin/bash`; the function must work there.

### Notice when integration is not loaded

If `TELE_CD_FD` is unset and stderr is a terminal, the binary is being run
bare, without the shell function. For every operation other than `--init`,
`--help`, `--version`, and the hidden `__complete`, it prints one line to
stderr before doing its normal work:

```
tele: shell integration is not loaded, so tele cannot change directory.
      Add this to your shell rc file:  eval "$(tele --init zsh)"
```

(`zsh` or `bash` chosen from `$SHELL`; omit the second line if unknown.) The
operation still runs: a navigation prints the path on stdout, `-l` lists, and
so on. The notice is suppressed when stderr is not a TTY so scripts calling the
binary directly stay quiet. (User's decision: notice on anything other than
init when run bare.)

### Shell argument for `--init`

`--init` requires the shell name, `bash` or `zsh`. Anything else, or missing:
error to stderr listing supported shells, exit 2. The README's one-liner
always names the shell explicitly so startup never depends on `$SHELL`.

### zsh `compinit`

Cobra's zsh script ends in `compdef`, which fails if the user has never run
`compinit`. The zsh `--init` output checks whether `compdef` is defined and,
if not, runs `autoload -Uz compinit && compinit` first. It never runs
`compinit` when the user already has. (Author's call.)

### bash versions

The `tele` function must work on macOS `/bin/bash` 3.2 and on bash 5. Tab
completion must work on bash ≥ 4 and is best-effort on 3.2; a completion
failure on 3.2 is not a release blocker, but the function must still load
without error there. macOS CI runs the function tests against `/bin/bash`.
(Author's call.)

### Completion

- Completion is registered for the name `tele` (the function). The generated
  Cobra scripts call back into `tele __complete ...`; this routes through the
  function, which passes it straight through since `__complete` is not a
  navigation. The builder verifies this works in both shells, and may rewrite
  the callback to invoke the binary directly in the emitted script if it does
  not.
- Dynamic completions: alias names for the first positional of a navigation,
  for `-d`, and for `-wl`/`-wli`. Directories for the `<path>` of `-r`. Flags
  always.
- Cobra's `completion` and `help` subcommands are hidden or disabled so the
  surface stays flag-only. The `__complete` hidden command must remain.

## 7. Alias store (decided)

Location: `$XDG_CONFIG_HOME/tele/aliases.toml`, defaulting to
`~/.config/tele/aliases.toml` on both macOS and Linux (deliberately not
`~/Library/...`). Created on first register with mode 0600, directory 0700.

Writes are atomic: write to a temp file in the same directory, then rename.

Schema:

```toml
# tele aliases — safe to hand-edit
[aliases.notes]
path = "/Users/phodder/notes"

[aliases.account-service]
path = "services/step/account-service"
worktree = true
repo = "/Users/phodder/code/monorepo/.git"
```

- Plain alias: `path` is absolute. Stored as given after cleaning (`..` and
  `.` resolved, no symlink resolution).
- Worktree alias: `path` is relative to the repo root (`.` for the root itself,
  forward slashes), `worktree = true`, and `repo` is the absolute path of the
  repo's **common git directory** (see §8).
- Unknown keys are preserved on rewrite? **No.** Rewrite emits only known keys;
  the file is Tele's. Comments are not preserved. (Author's call: simpler, and
  the file is small.)
- A malformed file is a runtime error naming the file; Tele never silently
  resets it.

## 8. Worktree semantics (decided)

### Repo identity

All worktrees created from one clone share a single *common git directory*
(`<main checkout>/.git`, or the bare repo directory for a bare-repo setup).
Tele uses its absolute path as the repo's identity. Obtain it with
`git rev-parse --path-format=absolute --git-common-dir` (git ≥ 2.31); fall back
to resolving the relative result against the cwd for older git. Builder's call
whether to require 2.31.

Separate clones do not match each other. This is intended.

For display, a repo is named by the directory containing the common dir
(`monorepo` in the example), or the common dir's own basename for bare repos.

### Registration (`tele -r -w <path> <alias>`)

1. Require git (see §9).
2. Resolve `<path>` to an absolute directory; must exist.
3. From that directory, obtain the common dir (identity) and the worktree
   top-level (`git rev-parse --show-toplevel`). Not in a repo → error.
4. Store `path` as `<path>` relative to the top-level, `worktree = true`, `repo`
   = common dir.

### Navigation (`tele <alias>` where alias is worktree-aware)

1. Require git.
2. From the cwd, obtain common dir and top-level.
   - Not in a repo → error: `not inside a git repository; alias <name> belongs to repo <display name> (<common dir>)`.
   - Common dir ≠ stored → error: `in repository <current display name>; alias <name> belongs to <stored display name>`.
3. Target = top-level joined with stored relative path.
4. Target does not exist or is not a directory → error naming the expected
   path. (A branch may legitimately lack the directory.)
5. Emit target per §6.

Plain aliases never invoke git. If the stored directory no longer exists, error;
never fall back to a parent.

### Worktree listing (`tele -wl [alias]`)

Determine the repo to list:

- No alias: the repo containing the cwd. Not in a repo → error.
- Plain alias: the repo containing the alias's directory. Not in a repo → error
  saying the alias does not point into a git repository.
- Worktree alias: the stored common dir directly.

Run `git worktree list --porcelain` (via `git -C <dir>`) and parse it. For each
worktree capture path, branch (short ref name), or short SHA when detached, and
whether it is the main worktree. Mark the worktree containing the cwd, if any.

Plain output: one line per worktree, `branch` then `path`, aligned, current one
marked with `*`. Sorted with the main worktree first, then by branch name.

Interactive (`-wli`): picker over the same rows; enter emits the worktree's
path per §6.

## 9. Git availability

Tele looks for `git` on `PATH` only when an operation needs it. Missing →
`git is required for worktree aliases but was not found on PATH`, exit 1.
Plain register, navigate, list, delete, and `--init` never touch git.

## 10. Interactive pickers (decided)

Both pickers share one implementation.

- Render to stderr via Bubble Tea. Require a TTY on stderr; otherwise error
  (`interactive mode needs a terminal`), exit 1.
- Rows: for aliases, name plus path, with a visible marker for worktree-aware
  aliases and the repo display name. For worktrees, branch plus path, current
  marked.
- Typing filters (fuzzy, matching anywhere in name/branch or path). Arrow keys
  (and Ctrl-N/Ctrl-P) move. Enter selects. Escape or Ctrl-C cancels: exit 130,
  no output.
- Selecting an alias resolves it exactly as `tele <name>` would, including
  worktree errors, which are reported after the picker closes.
- The picker lists, at most, a few hundred rows; no pagination design needed
  beyond what the list component provides.
- Visual detail (colours, borders, prompt text) is the builder's call. No colour
  when `NO_COLOR` is set.

## 11. Plain listing (`tele -l`)

One alias per line, sorted by name. Columns: name, then path. Worktree aliases
show the relative path followed by `[worktree: <repo display name>]`. Plain
text; alignment via spaces; no colour when stdout is not a TTY or `NO_COLOR` is
set. Empty store: prints a one-line hint on stderr about `tele -r`, exit 0.

## 12. Register and delete details

- `tele -r <path> <alias>`: path must be an existing directory. Alias must be
  valid (§5). Exists already → error `alias <name> already exists; use -f to
  overwrite`, exit 1, unless `-f`. On success print one line to stderr:
  `registered <name> → <path>` (worktree variant adds `(worktree-aware, repo
  <display name>)`). stdout stays empty so the function has nothing to misread.
- `tele -d <alias>`: not found → error, exit 1. Success → one line to stderr.
- Informational output for non-navigation operations goes to stdout (lists,
  `--init`, help, version). Confirmations of mutations go to stderr. (Author's
  call: keeps stdout clean for scripting and keeps `tele -l | wc -l` honest.)

## 13. Performance budget

`tele <plain-alias>` end to end, including the shell function, must stay under
roughly 20 ms on the author's machine. Worktree aliases add two git invocations
and are expected to land under 50 ms. Reading the TOML file must not dominate.

## 14. Quality and testing (decided at the level of requirements)

- Unit tests for mode resolution, alias-name validation, store read/write
  (including atomic write and malformed file), and porcelain parsing.
- Integration tests that create real temporary git repos with linked worktrees
  and exercise registration, navigation (match, not-in-repo, wrong-repo,
  missing-directory), and worktree listing.
- Shell tests: for each of bash and zsh, evaluate the `--init` output in a
  fresh shell, run `tele <alias>`, and assert `pwd` changed; also assert
  `tele -l | cat` prints the list and does not `cd`; assert the bare-binary
  notice appears on stderr only when integration is absent; assert Escape in a picker
  leaves the directory unchanged (may use a pseudo-terminal or be skipped on
  CI if a TTY is unavailable, in which case it is run locally and documented).
- CI on GitHub Actions: ubuntu-latest and macos-latest, `go test ./...`,
  `go vet`, and the shell tests.
- Release: goreleaser (or equivalent) producing darwin/linux × amd64/arm64
  archives containing the `tele` binary.
- README: install, the one-line `eval` for each shell, and the five commands
  in §5 with one example each.

## 15. Left to the builder

- Go module path and repository layout (`cmd/tele`, `internal/...`). The
  author has not named a GitHub repo yet; use a placeholder and make it one
  edit.
- Exact TOML library, Bubble Tea styling, table alignment code.
- Whether to require git ≥ 2.31 or support the relative-path fallback.
- Whether completion callbacks route through the function or use
  `command tele` directly, provided tab completion works in both shells.
- Error message wording beyond the content specified above.
- How the shell tests obtain a TTY in CI, or whether they are documented as
  local-only.
- Release tooling specifics.

## 16. Decision log

- 2026-10-31 Builder check raised four questions. Resolved: binary ships as
  `tele` with a same-name wrapper function (user); the earlier `--rc` hook was
  dropped in favour of `--init` plus a README one-liner (user); bare-binary
  notice on every operation except init/help/version (user); zsh `compinit`
  handling and bash 3.2 completion scope (author's calls, §6).
