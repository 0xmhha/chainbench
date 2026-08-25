package app

import (
	"context"

	"github.com/0xmhha/chainbench/internal/core/target"
	"github.com/0xmhha/chainbench/internal/netcompose"
)

// NetNewIn initializes a composition workspace: the chain identity and where
// the network's data plane lives.
type NetNewIn struct {
	// DataDir is the local workspace (control-plane) directory.
	DataDir string
	// Chain is the registry chain id.
	Chain string
	// Binary is the node binary path (may also be set at start).
	Binary string
	// ManifestPath selects an external, project-supplied chain manifest instead
	// of an embedded chain; TemplatePath is its genesis template.
	ManifestPath string
	TemplatePath string
	// KeysDir is the key set the network composes from ("" = keys/preset).
	KeysDir string
	// Target is where the data plane lives; zero value = local, rooted at the
	// workspace directory.
	Target target.TargetSpec
	// Docker treats the servers as local docker containers: the harness's own
	// dials are translated through the localmap next to the server set.
	// Recorded on the workspace so every later step follows it.
	Docker bool
}

// NetNewOut reports what the workspace was initialized to.
type NetNewOut struct {
	// Detail is the recorded step detail line.
	Detail string
}

// NetNew initializes (or re-targets) the composition workspace — the `net new`
// step, shared verbatim by the CLI subcommand and the MCP tool.
func NetNew(_ context.Context, d Deps, in NetNewIn) (NetNewOut, error) {
	ws, err := netcompose.Open(in.DataDir, d.Clock)
	if err != nil {
		return NetNewOut{}, err
	}
	detail, err := ws.New(netcompose.NewOpts{
		Chain: in.Chain, Binary: in.Binary, KeysDir: in.KeysDir, Target: in.Target,
		ManifestPath: in.ManifestPath, TemplatePath: in.TemplatePath, Docker: in.Docker,
	})
	if err != nil {
		return NetNewOut{}, err
	}
	if err := ws.Save(); err != nil {
		return NetNewOut{}, err
	}
	return NetNewOut{Detail: detail}, nil
}

// NetStatusIn identifies the workspace to inspect.
type NetStatusIn struct {
	DataDir string
}

// NetStatusOut is the workspace composition state.
type NetStatusOut struct {
	// Dir is the workspace control directory.
	Dir string
	// State is the persisted composition state (chain, target, step table).
	State netcompose.State
}

// NetStatus reads the workspace composition state — the `net status` step.
func NetStatus(_ context.Context, d Deps, in NetStatusIn) (NetStatusOut, error) {
	ws, err := netcompose.Open(in.DataDir, d.Clock)
	if err != nil {
		return NetStatusOut{}, err
	}
	return NetStatusOut{Dir: ws.Dir(), State: ws.State()}, nil
}
