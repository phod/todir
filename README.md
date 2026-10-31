# Take To (`tkto`)

Pronounced "take to". Type `tkto account-service` and read it as "take me to account-service".

`tkto` is a small CLI for jumping between directories with short aliases. Its
differentiator is worktree-aware aliases: register a path inside a git repo
once, and the alias resolves to the same relative location in whichever
worktree you are currently in.

## Install

```sh
go install github.com/phod/tkto/cmd/tkto@latest
```

Or download a release archive (darwin/linux, amd64/arm64) from the GitHub
releases page and put `tkto` on your `PATH`. The release workflow builds these
archives (plus `checksums.txt`) with GoReleaser whenever a `v*` tag is pushed.
A Homebrew tap is coming later.

git is only needed for the worktree features. git 2.31 or newer is the tested
floor; older versions may work.

## Shell setup

A child process cannot change its parent shell's directory, so `tkto` needs a
small shell function to do the `cd`. Add one line to your rc file:

```sh
eval "$(tkto --init zsh)"   # ~/.zshrc
eval "$(tkto --init bash)"  # ~/.bashrc
```

The binary generates the function and tab completions at every shell start.
Without the function, running `tkto <alias>` only prints the path plus a
notice that the shell integration is not active.

zsh users: if your `~/.zshrc` calls `compinit` itself (directly or through a
framework such as oh-my-zsh), put the `eval` line after that call.

Bash users: tab completion relies on the `bash-completion` package. Install it
(Homebrew: `brew install bash-completion@2`; Debian/Ubuntu:
`apt install bash-completion`) and make sure it is sourced before the `eval`
line. macOS's `/bin/bash` 3.2 runs the function fine, but completion there is
best-effort.

## Usage

```sh
tkto -r ~/code/api api          # register a directory as "api"
tkto api                        # jump to it
tkto -l                         # list aliases
tkto                            # pick an alias interactively (also: tkto -li)
tkto -d api                     # delete an alias
tkto -r -f ~/code/api2 api      # overwrite an existing alias

tkto -r -w ~/code/repo/services/web web   # register a worktree-aware alias
tkto -wl                        # list worktrees of the current repo
tkto -wl web                    # list worktrees of the repo behind an alias
tkto -wli [alias]               # pick a worktree interactively

tkto -v                         # print the version
tkto -h                         # show help
```

Short flags combine: `tkto -li`, `tkto -wl`, `tkto -wli` are the same as
`tkto -l -i`, `tkto -w -l`, `tkto -w -l -i`.

### Worktree aliases

With `-w`, the alias is stored relative to the repository root. When you run
`tkto web`, it resolves against whichever worktree you are currently in, so
the same alias lands you in `services/web` of each checkout. Outside the repo
you get an error naming the repo. `tkto -wl` lists the worktrees with the
current one marked.

## Exit codes

| Code | Meaning          |
|------|------------------|
| 0    | OK               |
| 1    | Runtime error    |
| 2    | Usage error      |
| 130  | Picker cancelled |

## Configuration

Aliases live in `$XDG_CONFIG_HOME/tkto/aliases.toml` (default
`~/.config/tkto/aliases.toml`). The file is plain TOML and safe to edit by
hand:

```toml
[aliases.api]
path = "/Users/me/code/api"

[aliases."my.notes"]   # names containing dots must be quoted
path = "/Users/me/notes"

[aliases.web]
path = "services/web"
worktree = true
repo = "/Users/me/code/repo/.git"
```

An unquoted dotted name such as `[aliases.my.notes]` is read as a nested
table and rejected. Plain aliases need an absolute `path`; worktree aliases
need a `repo`.
