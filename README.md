# todir

Short for "to directory": `todir account-service` takes you to account-service.

`todir` is a small CLI for jumping between directories with short aliases. Its
differentiator is worktree-aware aliases: register a path inside a git repo
once, and the alias resolves to the same relative location in whichever
worktree you are currently in.

## Install

```sh
go install github.com/phod/todir/cmd/todir@latest
```

Or download a release archive (darwin/linux, amd64/arm64) from the GitHub
releases page and put `todir` on your `PATH`. The release workflow builds these
archives (plus `checksums.txt`) with GoReleaser whenever a `v*` tag is pushed.
A Homebrew tap is coming later.

git is only needed for the worktree features. git 2.31 or newer is the tested
floor; older versions may work.

## Shell setup

A child process cannot change its parent shell's directory, so `todir` needs a
small shell function to do the `cd`. Add one line to your rc file:

```sh
eval "$(todir --init zsh)"   # ~/.zshrc
eval "$(todir --init bash)"  # ~/.bashrc
```

The binary generates the function and tab completions at every shell start.
Without the function, running `todir <alias>` only prints the path plus a
notice that the shell integration is not active.

zsh users: if your `~/.zshrc` calls `compinit` itself (directly or through a
framework such as oh-my-zsh), put the `eval` line after that call. If it does
not, todir runs `compinit -i` for you, which skips any completion directory
`compaudit` reports as insecure instead of asking about it at every shell
start; run `compaudit` to see and fix those directories.

Bash users: tab completion relies on the `bash-completion` package. Install it
(Homebrew: `brew install bash-completion@2`; Debian/Ubuntu:
`apt install bash-completion`) and make sure it is sourced before the `eval`
line. macOS's `/bin/bash` 3.2 runs the function fine, but completion there is
best-effort.

## Usage

```sh
todir -r ~/code/api api          # register a directory as "api"
todir api                        # jump to it
todir -l                         # list aliases
todir                            # pick an alias interactively (also: todir -li)
todir -d api                     # delete an alias
todir -r -f ~/code/api2 api      # overwrite an existing alias

todir -r -w ~/code/repo/services/web web   # register a worktree-aware alias
todir -wl                        # list worktrees of the current repo
todir -wl web                    # list worktrees of the repo behind an alias
todir -wli [alias]               # pick a worktree interactively

todir -v                         # print the version
todir -h                         # show help
```

Short flags combine: `todir -li`, `todir -wl`, `todir -wli` are the same as
`todir -l -i`, `todir -w -l`, `todir -w -l -i`.

### Worktree aliases

With `-w`, the alias is stored relative to the repository root. When you run
`todir web`, it resolves against whichever worktree you are currently in, so
the same alias lands you in `services/web` of each checkout. Outside the repo
you get an error naming the repo. `todir -wl` lists the worktrees with the
current one marked.

## Exit codes

| Code | Meaning          |
|------|------------------|
| 0    | OK               |
| 1    | Runtime error    |
| 2    | Usage error      |
| 130  | Picker cancelled |

## Configuration

Aliases live in `$XDG_CONFIG_HOME/todir/aliases.toml` (default
`~/.config/todir/aliases.toml`). The file is plain TOML and safe to edit by
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
