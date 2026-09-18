package keyringcmd_test

import (
	"bytes"
	"context"
	"testing"

	"github.com/spf13/cobra"

	"github.com/0xmhha/chainbench/cmd/chainbench/keyringcmd"

	_ "github.com/0xmhha/chainbench/internal/chains/all" // register chain plugins, as package main does
)

// run executes a keyring or validator command line the way an operator types
// it ("keyring new …", "validator roster …"). The groups are mounted on a bare
// root, so these tests exercise exactly what the real root command mounts,
// without depending on package main.
func run(t *testing.T, args ...string) (string, error) {
	t.Helper()
	root := &cobra.Command{Use: "chainbench", SilenceUsage: true, SilenceErrors: true}
	root.AddCommand(keyringcmd.New(), keyringcmd.NewValidator())
	var buf bytes.Buffer
	root.SetOut(&buf)
	root.SetErr(&buf)
	root.SetArgs(args)
	err := root.ExecuteContext(context.Background())
	return buf.String(), err
}
