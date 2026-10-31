# tele

`tele` is a small CLI for jumping between directories with short aliases. Its
differentiator is worktree-aware aliases: register a path inside a git repo
once, and the alias resolves to the same relative location in whichever
worktree you are currently in.

## Install

```sh
go install github.com/phodder/tele/cmd/tele@latest
```

Or download a release archive (darwin/linux, amd64/arm64) from the GitHub
releases page and put `tele` on your `PATH`. A Homebrew tap is coming later.

git is only needed for the worktree features.

## Shell setup

A child process cannot change its parent shell's directory, so `tele` needs a
small shell function to do the `cd`. Add one line to your rc file:

```sh
eval "$(tele --init zsh)"   # ~/.zshrc
eval "$(tele --init bash)"  # ~/.bashrc
```

The binary generates the function and tab completions at every shell start.
Without the function, running `tele <alias>` only prints the path plus a
notice that the shell integration is not active.

Bash users: tab completion relies on the `bash-completion` package. Install it
(Homebrew: `brew install bash-completion@2`; Debian/Ubuntu:
`apt install bash-completion`) and make sure it is sourced before the `eval`
line. macOS's `/bin/bash` 3.2 runs the function fine, but completion there is
best-effort.

## Usage

```sh
tele -r ~/code/api api          # register a directory as "api"
tele api                        # jump to it
tele -l                         # list aliases
tele                            # pick an alias interactively (also: tele -li)
tele -d api                     # delete an alias
tele -r -f ~/code/api2 api      # overwrite an existing alias

tele -r -w ~/code/repo/services/web web   # register a worktree-aware alias
tele -wl                        # list worktrees of the current repo
tele -wl web                    # list worktrees of the repo behind an alias
tele -wli [alias]               # pick a worktree interactively
```

### Worktree aliases

With `-w`, the alias is stored relative to the repository root. When you run
`tele web`, it resolves against whichever worktree you are currently in, so
the same alias lands you in `services/web` of each checkout. Outside the repo
you get an error naming the repo. `tele -wl` lists the worktrees with the
current one marked.

## Exit codes

| Code | Meaning          |
|------|------------------|
| 0    | OK               |
| 1    | Runtime error    |
| 2    | Usage error      |
| 130  | Picker cancelled |

## Configuration

Aliases live in `$XDG_CONFIG_HOME/tele/aliases.toml` (default
`~/.config/tele/aliases.toml`). The file is plain TOML and safe to edit by
hand.
