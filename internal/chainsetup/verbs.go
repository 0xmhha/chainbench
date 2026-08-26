package chainsetup

import (
	"context"
	"fmt"
	"time"

	"github.com/0xmhha/chainbench/internal/core/driver"
	"github.com/0xmhha/chainbench/internal/core/machine"
)

// Deps is what the verbs need from their caller: a clock for step stamps, an
// environment, a command line for the workspace lock's owner note, and a
// reporter for operational side notes. All may be zero — the defaults are
// time.Now, the process environment, an empty owner, and silence.
type Deps struct {
	Clock   func() time.Time
	Env     func(string) string
	Command string
	Report  func(format string, args ...any)
	// Driver overrides the transport used to control already-running node
	// processes; nil uses the local driver. Injected for tests and for
	// surfaces that route the same verb over SSH.
	Driver func() (driver.Driver, error)
}

func (d Deps) nodeDriver() (driver.Driver, error) {
	if d.Driver == nil {
		return driver.NewLocalDriver(), nil
	}
	return d.Driver()
}

func (d Deps) command() string { return d.Command }

func (d Deps) logf(format string, args ...any) {
	if d.Report != nil {
		d.Report(format, args...)
	}
}

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
	Target machine.Spec
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
	ws, err := Open(in.DataDir, d.Clock)
	if err != nil {
		return NetNewOut{}, err
	}
	detail, err := ws.New(NewOpts{
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
	State State
}

// NetStatus reads the workspace composition state — the `net status` step.
func NetStatus(_ context.Context, d Deps, in NetStatusIn) (NetStatusOut, error) {
	ws, err := Open(in.DataDir, d.Clock)
	if err != nil {
		return NetStatusOut{}, err
	}
	return NetStatusOut{Dir: ws.Dir(), State: ws.State()}, nil
}

// NetEndpointsIn asks for a composed network's reachable RPC endpoints.
type NetEndpointsIn struct {
	DataDir string
}

// NetEndpoints returns each node's HTTP RPC URL as this machine can reach it:
// the recorded per-node host, translated through the docker map when the
// workspace runs in docker mode — the same translation the health probe uses,
// so a caller attaching a test engine dials what actually answers.
func NetEndpoints(_ context.Context, d Deps, in NetEndpointsIn) ([]string, error) {
	ws, err := Open(in.DataDir, d.Clock)
	if err != nil {
		return nil, err
	}
	ws.SetEnv(d.Env)
	st := ws.State()
	if len(st.Nodes) == 0 {
		return nil, fmt.Errorf("chainsetup: endpoints: no node table — run `net allocate` first")
	}
	m, err := ws.opener().AddrMap()
	if err != nil {
		return nil, err
	}
	urls := make([]string, 0, len(st.Nodes))
	for _, ns := range st.Nodes {
		host, port := ns.Host, ns.HTTP
		if host == "" {
			host = ws.RPCHost()
		}
		if m != nil {
			host, port = m(host, port)
		}
		urls = append(urls, fmt.Sprintf("http://%s:%d", host, port))
	}
	return urls, nil
}
