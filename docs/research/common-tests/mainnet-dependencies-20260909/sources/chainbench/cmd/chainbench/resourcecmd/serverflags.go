package resourcecmd

import (
	"github.com/spf13/cobra"

	"github.com/0xmhha/chainbench/internal/resource"
)

// ServerFlags is the one binding for the server-selection flags every
// placing command shares (--server-set, --server, --server-index,
// --all-servers). Host addresses and ports live in the server-set file,
// never on the command line; these flags only select within it. The resource
// surface owns it; `net` and `run` borrow it.
type ServerFlags struct {
	config     string
	server     string
	index      int
	allServers bool
}

// Bind registers the flags on cmd.
func (f *ServerFlags) Bind(cmd *cobra.Command) {
	cmd.Flags().StringVar(&f.config, "server-set", "",
		"server-set file: which servers exist and how to reach them (default: "+resource.DefaultSetFile+" when present)")
	cmd.Flags().StringVar(&f.server, "server", "", "server to place nodes on, by name from the server set")
	cmd.Flags().IntVar(&f.index, "server-index", 0, "server to place nodes on, by index from the server set")
	cmd.Flags().BoolVar(&f.allServers, "all-servers", false, "spread the network across every server in the set, one node per host")
}

// Ref is the module-layer selection.
func (f *ServerFlags) Ref() resource.ServerRef {
	return resource.ServerRef{SetPath: f.config, Name: f.server, Index: f.index, All: f.allServers}
}
