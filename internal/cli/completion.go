package cli

import (
	"bytes"
	"errors"
	"fmt"

	"github.com/spf13/cobra"

	"github.com/phod/tkto/internal/shellinit"
)

// initShell prints the shell function and completion script for shell.
func (a *App) initShell(shell string) error {
	root := newRoot(a)
	var buf bytes.Buffer
	var err error
	switch shell {
	case "bash":
		err = root.GenBashCompletionV2(&buf, true)
	case "zsh":
		err = root.GenZshCompletion(&buf)
	}
	if err != nil {
		return fmt.Errorf("generating %s completion: %w", shell, err)
	}
	script, err := shellinit.Script(shell, buf.String())
	if errors.Is(err, shellinit.ErrUnsupportedShell) {
		return &UsageError{Msg: err.Error()}
	}
	if err != nil {
		return err
	}
	_, err = fmt.Fprint(a.Stdout, script)
	return err
}

// completeArgs supplies dynamic completions for tkto's positional arguments:
// directories for the <path> of -r, nothing for its <name>, and alias names
// everywhere else.
func (a *App) completeArgs(cmd *cobra.Command, args []string, _ string) ([]string, cobra.ShellCompDirective) {
	if reg, _ := cmd.Flags().GetBool("register"); reg {
		if len(args) == 0 {
			return nil, cobra.ShellCompDirectiveFilterDirs
		}
		return nil, cobra.ShellCompDirectiveNoFileComp
	}
	s, err := a.loadStore()
	if err != nil {
		return nil, cobra.ShellCompDirectiveNoFileComp
	}
	aliases := s.Sorted()
	names := make([]string, 0, len(aliases))
	for _, al := range aliases {
		names = append(names, al.Name)
	}
	return names, cobra.ShellCompDirectiveNoFileComp
}

// completeInit offers the supported shells for --init.
func completeInit(*cobra.Command, []string, string) ([]string, cobra.ShellCompDirective) {
	return []string{"bash", "zsh"}, cobra.ShellCompDirectiveNoFileComp
}
