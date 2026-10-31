package cli

import (
	"errors"
	"fmt"
	"io"
	"strconv"
	"strings"

	"github.com/phodder/tele/internal/picker"
	"github.com/spf13/cobra"
)

// App carries the process environment so Run is testable.
type App struct {
	Stdout, Stderr io.Writer
	Stdin          io.Reader
	CdWriter       io.Writer // nil when TELE_CD_FD is unset
	StderrIsTTY    bool
	Getenv         func(string) string
	Cwd            string
	ConfigPath     string
	Version        string
}

// ParseCdFD parses the TELE_CD_FD value. An empty value means unset.
func ParseCdFD(v string) (fd int, set bool, err error) {
	if v == "" {
		return 0, false, nil
	}
	fd, err = strconv.Atoi(v)
	if err != nil || fd < 0 {
		return 0, false, fmt.Errorf("invalid TELE_CD_FD")
	}
	return fd, true, nil
}

// Run executes tele with args and returns the process exit code.
func Run(app *App, args []string) int {
	root := newRoot(app)
	root.SetArgs(args)
	if err := root.Execute(); err != nil {
		if !errors.Is(err, picker.ErrCancelled) {
			fmt.Fprintf(app.Stderr, "tele: %s\n", err)
		}
		return exitCode(err)
	}
	return 0
}

func newRoot(app *App) *cobra.Command {
	var (
		register, worktree, force, list, interactive, del, version bool
		initShell                                                  string
	)
	root := &cobra.Command{
		Use:           "tele",
		SilenceUsage:  true,
		SilenceErrors: true,
		Args:          cobra.ArbitraryArgs,
		RunE: func(cmd *cobra.Command, args []string) error {
			if cmd.Flags().Changed("init") && initShell == "" {
				return usagef("--init requires a shell name: bash or zsh")
			}
			m, err := Resolve(Flags{
				Register: register, Worktree: worktree, Force: force, List: list,
				Interactive: interactive, Delete: del, Version: version, Init: initShell,
			}, args)
			if err != nil {
				return err
			}
			if ShouldNotice(m.Kind, app.CdWriter != nil, app.StderrIsTTY) {
				app.printNotice()
			}
			return app.dispatch(m)
		},
	}
	root.CompletionOptions.DisableDefaultCmd = true
	root.SetHelpCommand(&cobra.Command{Hidden: true})
	root.SetOut(app.Stdout)
	root.SetErr(app.Stderr)
	root.SetFlagErrorFunc(func(_ *cobra.Command, err error) error {
		if strings.Contains(err.Error(), "flag needs an argument: --init") {
			return usagef("--init requires a shell name: bash or zsh")
		}
		return &UsageError{Msg: err.Error()}
	})

	f := root.Flags()
	f.BoolVarP(&register, "register", "r", false, "register a directory alias")
	f.BoolVarP(&worktree, "worktree", "w", false, "target a git worktree")
	f.BoolVarP(&force, "force", "f", false, "overwrite an existing alias")
	f.BoolVarP(&list, "list", "l", false, "list aliases")
	f.BoolVarP(&interactive, "interactive", "i", false, "pick an alias interactively")
	f.BoolVarP(&del, "delete", "d", false, "delete an alias")
	f.StringVar(&initShell, "init", "", "print shell integration for <shell>")
	f.BoolVarP(&version, "version", "v", false, "print version")
	root.ValidArgsFunction = app.completeArgs
	_ = root.RegisterFlagCompletionFunc("init", completeInit)
	return root
}
