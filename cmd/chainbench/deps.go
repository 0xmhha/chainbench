package main

import (
	"fmt"
	"os"
	"strings"

	"github.com/spf13/cobra"

	"github.com/0xmhha/chainbench/internal/app"
)

// cliDeps is the dependency set the CLI hands to a use case.
//
// It records the command line so the workspace lock can name what is using a
// workspace. An operator who finds a network busy needs "net up --chain wemix,
// pid 4211, since 14:02", not "locked" — the second sends them looking for a
// file to delete.
func cliDeps(cmd *cobra.Command) app.Deps {
	return app.Deps{
		Command: commandLine(cmd),
		Logf: func(format string, args ...any) {
			fmt.Fprintf(cmd.ErrOrStderr(), format+"\n", args...)
		},
	}
}

// commandLine renders this invocation the way the operator typed it, minus the
// absolute path of the binary.
func commandLine(cmd *cobra.Command) string {
	parts := []string{"chainbench"}
	if len(os.Args) > 1 {
		parts = append(parts, os.Args[1:]...)
	} else if cmd != nil {
		parts = append(parts, cmd.CommandPath())
	}
	return strings.Join(parts, " ")
}
