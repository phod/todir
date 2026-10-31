package main

import (
	"fmt"
	"os"

	"golang.org/x/term"

	"github.com/phodder/tele/internal/cli"
)

var version = "dev"

func main() {
	app := &cli.App{
		Stdout:      os.Stdout,
		Stderr:      os.Stderr,
		Stdin:       os.Stdin,
		StderrIsTTY: term.IsTerminal(int(os.Stderr.Fd())),
		Getenv:      os.Getenv,
		ConfigPath:  "", // set from the store package in Task 2
		Version:     version,
	}
	if cwd, err := os.Getwd(); err == nil {
		app.Cwd = cwd
	}
	fd, set, err := cli.ParseCdFD(os.Getenv("TELE_CD_FD"))
	if err != nil {
		fmt.Fprintln(os.Stderr, "tele: invalid TELE_CD_FD")
		os.Exit(2)
	}
	if set {
		app.CdWriter = os.NewFile(uintptr(fd), "tele-cd")
	}
	os.Exit(cli.Run(app, os.Args[1:]))
}
