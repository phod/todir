package main

import (
	"fmt"
	"os"

	"golang.org/x/term"

	"github.com/phod/todir/internal/cli"
	"github.com/phod/todir/internal/store"
)

var version = "dev"

func main() {
	home, err := os.UserHomeDir()
	if err != nil {
		fmt.Fprintln(os.Stderr, "todir: cannot determine home directory")
		os.Exit(1)
	}
	app := &cli.App{
		Stdout:      os.Stdout,
		Stderr:      os.Stderr,
		Stdin:       os.Stdin,
		StderrIsTTY: term.IsTerminal(int(os.Stderr.Fd())),
		Getenv:      os.Getenv,
		ConfigPath:  store.ConfigPath(os.Getenv, home),
		Version:     version,
	}
	if cwd, err := os.Getwd(); err == nil {
		app.Cwd = cwd
	}
	fd, set, err := cli.ParseCdFD(os.Getenv("TODIR_CD_FD"))
	if err != nil {
		fmt.Fprintln(os.Stderr, "todir: invalid TODIR_CD_FD")
		os.Exit(2)
	}
	if set {
		app.CdWriter = os.NewFile(uintptr(fd), "todir-cd")
	}
	os.Exit(cli.Run(app, os.Args[1:]))
}
