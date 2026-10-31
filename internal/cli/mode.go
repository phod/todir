package cli

// Kind identifies which operation the command line asks for.
type Kind int

const (
	KindNavigate Kind = iota
	KindPickAlias
	KindRegister
	KindDelete
	KindListAliases
	KindListWorktrees
	KindPickWorktree
	KindInit
	KindVersion
)

// Flags holds the parsed flag values relevant to mode resolution.
type Flags struct {
	Register, Worktree, Force, List, Interactive, Delete, Version bool
	Init                                                          string
}

// Mode is the single operation resolved from flags and positional arguments.
type Mode struct {
	Kind               Kind
	Alias, Path, Shell string
	Worktree, Force    bool
}

// Resolve turns flags and positional arguments into one Mode. The error is
// always a *UsageError.
func Resolve(f Flags, args []string) (Mode, error) {
	switch {
	case f.Init != "":
		if f.Register || f.Worktree || f.Force || f.List || f.Interactive || f.Delete || f.Version {
			return Mode{}, usagef("--init cannot be combined with other flags")
		}
		if len(args) != 0 {
			return Mode{}, usagef("--init takes no arguments")
		}
		return Mode{Kind: KindInit, Shell: f.Init}, nil

	case f.Version:
		if f.Register || f.Worktree || f.Force || f.List || f.Interactive || f.Delete {
			return Mode{}, usagef("--version cannot be combined with other flags")
		}
		if len(args) != 0 {
			return Mode{}, usagef("--version takes no arguments")
		}
		return Mode{Kind: KindVersion}, nil

	case f.Register:
		if f.Delete || f.List || f.Interactive {
			return Mode{}, usagef("-r cannot be combined with -d, -l or -i")
		}
		if len(args) != 2 {
			return Mode{}, usagef("-r takes a path and an alias name")
		}
		return Mode{Kind: KindRegister, Path: args[0], Alias: args[1], Worktree: f.Worktree, Force: f.Force}, nil

	case f.Delete:
		if f.Worktree || f.Force || f.List || f.Interactive {
			return Mode{}, usagef("-d cannot be combined with -w, -f, -l or -i")
		}
		if len(args) != 1 {
			return Mode{}, usagef("-d takes exactly one alias name")
		}
		return Mode{Kind: KindDelete, Alias: args[0]}, nil

	case f.Force:
		return Mode{}, usagef("-f requires -r")

	case f.Worktree:
		if !f.List && !f.Interactive {
			return Mode{}, usagef("-w requires -r or -l")
		}
		if len(args) > 1 {
			return Mode{}, usagef("too many arguments")
		}
		m := Mode{Kind: KindListWorktrees}
		if f.Interactive {
			m.Kind = KindPickWorktree
		}
		if len(args) == 1 {
			m.Alias = args[0]
		}
		return m, nil

	case f.List || f.Interactive:
		if len(args) != 0 {
			return Mode{}, usagef("-l and -i take no arguments")
		}
		if f.Interactive {
			return Mode{Kind: KindPickAlias}, nil
		}
		return Mode{Kind: KindListAliases}, nil
	}

	switch len(args) {
	case 0:
		return Mode{Kind: KindPickAlias}, nil
	case 1:
		return Mode{Kind: KindNavigate, Alias: args[0]}, nil
	}
	return Mode{}, usagef("too many arguments")
}
