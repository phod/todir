# todir

todir is a CLI application that is used to create aliases for directories to make it
convenient to jump between directories.

One differentiator todir has over similar CLI tools is that it can also set up worktree aware
aliases.


# Simple usage

Simple usage is limited to navigation, registration, and listing alias.


## Registration

Registration requires the `-r` or `--register` flag:

```
todir -r /path/to/directory my-alias
```

A relative or absolute directory may be used. Commonly we would see `todir -r . my-alias`.

Registering a name that already exists fails. Use the `-f` or `--force` flag to overwrite it:

```
todir -r -f . my-alias
```


## Navigation

With no CLI flags, we assume usage of an alias to navigate:

```
todir my-alias  # Navigates the user to /path/to/directory, as per above registration example
```

Any usage of flags means we are not navigating, and that the user is trying to do something else. This is how we
avoid conflicts with alias and commands.


## Listing alias

Using the `-l` or `--list` flag, we can list the alias registered:

```
todir -l # List all registered aliases
```

With an additional `-i` or `--interactive` flag, we keep this list open and allow for keyboard navigation or typing. If using the navigation, the user may navigate across
the aliases. If typing, we should filter down the list. Pressing enter will then move the user to that directory, as if they used a `todir my-alias` command directly.
This is a convenience feature for uncommonly used aliases. Exit the session without navigating using the escape key.

```
todir -li # List all registered aliases in an interactive session
```


## Deregistration

Deregistration uses the `-d` or `--delete` flag:

```
todir -d my-alias # Removes the alias
```

## Autocomplete

Autocomplete via the tab key should be supported, so we should expose a command that outputs the shell
function and completion bindings. The user adds one line to their rc file and the binary generates the
rest at shell start (the pyenv approach):

```
eval "$(todir --init bash)"   # or zsh
```


# Worktree usage

Requirement: git must be installed.

We need to explore how this could work. But some ideas:
1. Set up an alias that is configured as "worktree aware". This should also store what git repository it is in.
2. When using a worktree aware alias, we should check to see if we're in a git repository for the matching alias. If we're in the matching git repository, we will navigate to the directory within the worktree.

Registration could look similar to simple usage, but with an additional `-w` or `--worktree` flag. e.g.
```
todir -r -w . my-dir-alias
```
This checks to see where the root of the repository is, and sets up the aliased directory to navigate within that context.

Usage would then be the same as the simple case:
```
todir my-dir-alias
```

## Error scenarios

### Not in a repo

When using a worktree alias, if we're not in a repo, surface an error and exit. Error should state what the expected repo is.

### Git not installed

If git is not installed, surface and error message telling the user git must be installed and exit.

## Questions

### Additional feature: Worktree navigation

Does git expose the ability to see what worktrees are available locally? It'd be a nice convenience. If not, all good. Just a nice to have feature if it's available.

If it is available, we should have something like the following to list the worktrees and navigating between them:
```
todir -w -l
```

Where `-w` is also `--worktree` and `-l` is `--list`

With a `-i` or `--interactive` flag, we can navigate via the CLI to a specific worktree. Either with arrow keys, or typing to filter the names down. Pressing enter will navigate to that worktree. 
Exit the session without navigation using the escape key.

This command would only work in the relevant repo. So a user would be expected to navigate to an existing repo first, perhaps the main branch via a stable alias.
```
todir my-repo # Stable repo of main branch, worktrees are made off of this
todir -wli # Lists worktrees and navigate
todir my-dir-alias # Navigate to a directory within that worktree
```

Some fiddliness here. So this needs further discussion. May deem as too janky to implement.
