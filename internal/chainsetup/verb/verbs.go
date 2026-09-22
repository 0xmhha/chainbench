package verb

import (
	"context"
	"github.com/0xmhha/chainbench/internal/chainsetup"

	"github.com/0xmhha/chainbench/internal/resource"
)

// ChainNewIn initializes a composition workspace: the chain identity and where
// the network's data plane lives.
type ChainNewIn struct {
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
	// KeysDir is the key set the network composes from ("" = presets/keys).
	KeysDir string
	// Target is where the data plane lives; zero value = local, rooted at the
	// workspace directory.
	Target resource.Spec
	// Docker treats the servers as local docker containers: the harness's own
	// dials are translated through the localmap next to the server set.
	// Recorded on the workspace so every later step follows it.
	Docker bool
	// ServerSet is the server-set file, recordable here so --docker and the
	// set it translates through arrive as the pair they are.
	ServerSet string
	// WorkspaceConfigPath is the environment file owning the target data root and
	// purpose directories; recorded so later steps resolve portable references
	// under it.
	WorkspaceConfigPath string
}

// ChainNewOut reports what the workspace was initialized to.
type ChainNewOut struct {
	// Detail is the recorded step detail line.
	Detail string
}

// ChainNew initializes (or re-targets) the composition workspace — the `chain new`
// step, shared verbatim by the CLI subcommand and the MCP tool.
func ChainNew(_ context.Context, d chainsetup.Deps, in ChainNewIn) (ChainNewOut, error) {
	ws, err := chainsetup.Open(in.DataDir, d.Clock)
	if err != nil {
		return ChainNewOut{}, err
	}
	detail, err := ws.New(chainsetup.NewOpts{
		Chain: in.Chain, Binary: in.Binary, KeysDir: in.KeysDir, Target: in.Target,
		ManifestPath: in.ManifestPath, TemplatePath: in.TemplatePath, Docker: in.Docker,
		ServerSet: in.ServerSet, WorkspaceConfigPath: in.WorkspaceConfigPath,
	})
	if err != nil {
		return ChainNewOut{}, err
	}
	if err := ws.Save(); err != nil {
		return ChainNewOut{}, err
	}
	return ChainNewOut{Detail: detail}, nil
}

// ChainStatusIn identifies the workspace to inspect.
type ChainStatusIn struct {
	DataDir string
}

// ChainStatusOut is the workspace composition state.
type ChainStatusOut struct {
	// Dir is the workspace control directory.
	Dir string
	// State is the persisted composition state (chain, target, step table).
	State chainsetup.State
}

// ChainStatus reads the workspace composition state — the `chain status` step.
func ChainStatus(_ context.Context, d chainsetup.Deps, in ChainStatusIn) (ChainStatusOut, error) {
	ws, err := chainsetup.Open(in.DataDir, d.Clock)
	if err != nil {
		return ChainStatusOut{}, err
	}
	return ChainStatusOut{Dir: ws.Dir(), State: ws.State()}, nil
}

// ChainEndpointsIn asks for a composed network's reachable RPC endpoints.
type ChainEndpointsIn struct {
	DataDir string
}

// ChainEndpoints returns each node's HTTP RPC URL as this machine can reach it:
// the recorded per-node host, translated through the docker map when the
// workspace runs in docker mode — the same translation the health probe uses,
// so a caller attaching a test engine dials what actually answers.
func ChainEndpoints(_ context.Context, d chainsetup.Deps, in ChainEndpointsIn) ([]string, error) {
	ws, err := chainsetup.Open(in.DataDir, d.Clock)
	if err != nil {
		return nil, err
	}
	ws.SetEnv(d.Env)
	return ws.Endpoints()
}
