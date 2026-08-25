// Package netmapcmd mounts the netmap command group: the network map — which
// node runs where, in which role, on which ports — asked as questions.
//
// The group is queries only. Nothing here composes, launches, or writes; the
// commands read a server set or a composed workspace and answer. Changing a
// network is `net`'s job, and keeping the two apart is what lets a placement
// change be exercised without composing anything.
//
// File placement follows the keyringcmd rule: the group starts as one file per
// verb cluster (netmap.go for the group and shared flags, one file per verb),
// and tests mirror the group.
package netmapcmd

import (
	"encoding/json"
	"fmt"
	"io"
	"sort"
	"text/tabwriter"

	"github.com/spf13/cobra"

	"github.com/0xmhha/chainbench/internal/app"
	"github.com/0xmhha/chainbench/internal/netmap"
)

// New builds the netmap command group.
func New() *cobra.Command {
	cmd := &cobra.Command{
		Use:   "netmap",
		Short: "The network map: which node runs where, in which role, on which ports",
		Long: "Queries over node placement. `plan` computes the placement a network shape\n" +
			"would get, from the server set alone; `show` reads a composed workspace's map;\n" +
			"`pool` reports the addresses and port slots a network may be composed from.\n" +
			"Nothing in this group changes anything — composing is `net`'s job.",
	}
	cmd.AddCommand(newShowCmd(), newPoolCmd(), newPlanCmd())
	return cmd
}

// serverFlags is the server-set selection shared by the verbs that read one.
// Host addresses and ports live in the server-set file, never on the command
// line.
type serverFlags struct {
	config string
	server string
	index  int
	fleet  bool
}

func (f *serverFlags) bind(cmd *cobra.Command) {
	cmd.Flags().StringVar(&f.config, "server-set", "",
		"server-set file: which servers exist and how to reach them (default: "+netmap.DefaultSetFile+" when present)")
	cmd.Flags().StringVar(&f.server, "server", "", "server to place nodes on, by name from the server set")
	cmd.Flags().IntVar(&f.index, "server-index", 0, "server to place nodes on, by index from the server set")
	cmd.Flags().BoolVar(&f.fleet, "fleet", false, "spread the network across every server in the set, one node per host")
}

func (f *serverFlags) ref() app.ServerRef {
	return app.ServerRef{SetPath: f.config, Name: f.server, Index: f.index, Fleet: f.fleet}
}

// deps is the Deps every netmap verb runs with: operational side notes print
// as they happen.
func deps(cmd *cobra.Command) app.Deps {
	out := cmd.OutOrStdout()
	return app.Deps{Logf: func(format string, args ...any) {
		fmt.Fprintf(out, format+"\n", args...)
	}}
}

// printMap renders the one placement projection every verb shares. Keeping a
// single renderer is deliberate: the port set once had three hand-written
// copies, two of which lost the etcd client port.
func printMap(out io.Writer, m app.NetMapOut) {
	w := tabwriter.NewWriter(out, 0, 0, 2, ' ', 0)
	fmt.Fprintln(w, "NODE\tROLE\tLABEL\tHOST\tP2P\tETCD\tHTTP\tDATADIR")
	for _, e := range m.Entries {
		etcd := "-"
		if e.Etcd != 0 {
			etcd = fmt.Sprint(e.Etcd)
		}
		// A family that embeds etcd listens on two ports; showing one would
		// hide the other from the operator reading a bind failure.
		if e.EtcdClient != 0 {
			etcd += "/" + fmt.Sprint(e.EtcdClient)
		}
		dataDir := e.DataDir
		if dataDir == "" {
			dataDir = "-"
		}
		fmt.Fprintf(w, "%s\t%s\t%s\t%s\t%d\t%s\t%d\t%s\n",
			e.Label, e.Role, e.Alias, e.Host, e.P2P, etcd, e.HTTP, dataDir)
	}
	_ = w.Flush()

	roles := make([]string, 0, len(m.Roles))
	for r := range m.Roles {
		roles = append(roles, r)
	}
	sort.Strings(roles)
	summary := ""
	for i, r := range roles {
		if i > 0 {
			summary += ", "
		}
		summary += fmt.Sprintf("%d %s", m.Roles[r], r)
	}
	shown := ""
	if len(m.Entries) != m.Total {
		shown = fmt.Sprintf("%d of ", len(m.Entries))
	}
	fmt.Fprintf(out, "%s%d node(s): %s\n", shown, m.Total, summary)
}

// emitJSON renders a result machine-readably.
func emitJSON(out io.Writer, v any) error {
	return json.NewEncoder(out).Encode(v)
}
