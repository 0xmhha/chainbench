package main

import (
	"sort"
	"strings"

	"github.com/spf13/cobra"

	"github.com/0xmhha/chainbench/cmd/chainbench/surface"
)

// newQueryCmd projects every command that declared itself read-only under one
// group, so "what can I look at" has a single answer.
//
// It is generated from the tree rather than written (surface-unification-design
// §4.4). A hand-built query group would be a second spelling of every command
// it lists, and the two would drift — which is the failure the keyring
// consolidation removed and which would come straight back in the read path.
//
// The noun group stays canonical: `keyring list` is the spelling documentation
// teaches, and `query keyring list` is the same registration rendered a second
// time. They cannot disagree, because the projection shares the original's RunE
// and its flag set — setting --json on one sets the variable the other reads.
func newQueryCmd(root *cobra.Command) *cobra.Command {
	q := &cobra.Command{
		Use:   "query",
		Short: "Everything that only looks: the read-only commands, gathered",
		Long: "Every command that declares itself read-only, projected under one group.\n\n" +
			"These change no file, no process and no chain state, and print no secret, so\n" +
			"an operator exploring inside `query` cannot cause an accident. Each is the\n" +
			"same registration as its canonical spelling — `query keyring list` and\n" +
			"`keyring list` run the same code — and the noun group is what the docs teach.",
	}
	for _, c := range root.Commands() {
		if c.Name() == "query" || c.Name() == "help" || c.Name() == "completion" {
			continue
		}
		if p := project(c); p != nil {
			q.AddCommand(p)
		}
	}
	return q
}

// project returns c rendered for the query group, or nil when neither c nor
// anything under it is read-only.
//
// A group is kept only for the read-only children it has, so `query chain`
// holds status and health without holding up or rm. A group that declares
// itself read-only and has none is still shown: it is a leaf that happens to
// have subcommands, like `capabilities`.
func project(c *cobra.Command) *cobra.Command {
	var kids []*cobra.Command
	for _, sub := range c.Commands() {
		if sub.Name() == "help" {
			continue
		}
		if p := project(sub); p != nil {
			kids = append(kids, p)
		}
	}
	self := surface.IsReadOnly(c)
	if !self && len(kids) == 0 {
		return nil
	}

	p := &cobra.Command{
		Use:     c.Use,
		Short:   c.Short,
		Long:    c.Long,
		Example: c.Example,
		Args:    c.Args,
		// The annotation travels with the projection so a test can hold the
		// group to the same rule the tree was walked by.
		Annotations: c.Annotations,
	}
	if self {
		// Share the behaviour and the flag variables rather than copying them.
		// A copy is a second registration however carefully it is made, and the
		// point of the projection is that there is only ever one.
		p.RunE = c.RunE
		p.Run = c.Run
		p.PreRunE = c.PreRunE
		p.Flags().AddFlagSet(c.Flags())
		p.PersistentFlags().AddFlagSet(c.PersistentFlags())
	}
	for _, k := range kids {
		p.AddCommand(k)
	}
	return p
}

// ReadOnlyPaths lists the canonical spelling of every command the query group
// projects, sorted. It is what the tests and the MCP read-only tool list are
// both derived from, so the three cannot disagree about what is safe.
func ReadOnlyPaths(root *cobra.Command) []string {
	var out []string
	var walk func(c *cobra.Command, path []string)
	walk = func(c *cobra.Command, path []string) {
		here := append(append([]string(nil), path...), c.Name())
		if surface.IsReadOnly(c) {
			out = append(out, strings.Join(here[1:], " "))
		}
		for _, sub := range c.Commands() {
			walk(sub, here)
		}
	}
	walk(root, nil)
	sort.Strings(out)
	return out
}
