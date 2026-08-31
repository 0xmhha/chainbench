package resourcecmd

import (
	"encoding/json"
	"fmt"
	"io"
	"sort"
	"text/tabwriter"

	"github.com/0xmhha/chainbench/internal/app"
)

// PrintMap renders a placement map the one way every surface prints it.
//
// Keeping a single renderer is deliberate: the port set once had three
// hand-written copies, two of which lost the etcd client port. Both the
// resource group (plan — a placement that would be) and the net group (show —
// the placement that is) print through here, so the two answers can never
// drift apart in shape.
func PrintMap(out io.Writer, m app.NetMapOut) {
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

// MapJSON renders a result machine-readably.
func MapJSON(out io.Writer, v any) error {
	return json.NewEncoder(out).Encode(v)
}
