package app

import (
	"fmt"

	"github.com/0xmhha/chainbench/internal/core/lifecycle"
)

// Where a run begins.
//
// A run is asked to work against a network in one of four ways: compose the one
// the specs declare, attach to endpoints somebody typed, attach to the network
// a workspace already composed, or attach to the one an env declares. Which of
// the four is the first thing every run decides, and it was decided in three
// places that did not agree — the CLI's switch, the MCP tool's, and AttachRun's
// own branch on whether a workspace directory was set. The MCP tool had no
// fourth branch at all, so a spec declaring env.attach could not be run through
// it.
//
// Here it is decided once, and the answer is a state.

// Spelling is how one surface names the three things a caller can say.
//
// A refusal has to name what the operator actually typed, and the surfaces
// spell the same three differently — the CLI takes flags, the MCP tool takes
// object keys. The sentences are written once and the words come from here.
type Spelling struct {
	RPC       string
	Attach    string
	Workspace string
}

// CLISpelling is how `chainbench run` names them.
var CLISpelling = Spelling{RPC: "--rpc", Attach: "--attach", Workspace: "--workspace-dir"}

// ToolSpelling is how the chainbench_run tool names them.
var ToolSpelling = Spelling{RPC: "rpc", Attach: "attach", Workspace: "dataDir"}

// Named is what a caller said about which network to run against, before
// anything decides what that means.
type Named struct {
	// Spell is how this surface names the three below, for refusals.
	Spell Spelling
	// RPCURLs, Attach and WorkspaceDir are what the caller said.
	RPCURLs      []string
	Attach       bool
	WorkspaceDir string
	// Specs returns the raw spec blobs and the labels that name them in
	// refusals.
	//
	// It is a function because it is called only when nothing above named a
	// network. Reading the specs first made a run that combined two flags
	// report a missing file instead of the conflict — the caller's mistake was
	// in what they typed, and nothing had to be opened to see it.
	Specs func() ([][]byte, []string, error)
}

// Start is where a run begins: the state, and the declaration when the specs
// were the ones that named the network.
type Start struct {
	// At is one of ChainOpenWorkspace (compose) or the three adopt states.
	At lifecycle.Status
	// Declared is what env.attach said. It is set only for
	// AdoptChainByDeclaration and nil otherwise.
	Declared *AttachDecl
}

// StartFor resolves what a caller named into the state a run starts in.
//
// The order is the override order: what the caller typed is decided before the
// specs are read, so a document can never take a run away from the endpoints an
// operator gave it.
func StartFor(in Named) (Start, error) {
	s := in.Spell
	switch {
	case in.Attach && len(in.RPCURLs) > 0:
		return Start{}, fmt.Errorf("%s takes the endpoints from %s; it does not combine with %s",
			s.Attach, s.Workspace, s.RPC)
	case in.Attach && in.WorkspaceDir == "":
		return Start{}, fmt.Errorf("%s needs %s <dir>, the workspace whose network is already up",
			s.Attach, s.Workspace)
	case in.Attach:
		// The workspace supplies the endpoints and the capabilities its
		// composition advertised, which is what lets a gated spec run against
		// the network the operator set up rather than a fresh one built to
		// satisfy the gate.
		return Start{At: lifecycle.AdoptChainByWorkspace}, nil
	case len(in.RPCURLs) > 0 && in.WorkspaceDir != "":
		return Start{}, fmt.Errorf("%s composes a network; it does not combine with %s (use %s to run against the network it already composed)",
			s.Workspace, s.RPC, s.Attach)
	case len(in.RPCURLs) > 0:
		return Start{At: lifecycle.AdoptChainByRPC}, nil
	case in.WorkspaceDir != "":
		return Start{At: lifecycle.ChainOpenWorkspace}, nil
	}
	// Nothing the caller said named a network. Ask the specs.
	if in.Specs == nil {
		return Start{}, fmt.Errorf("no network was named and there are no specs to ask")
	}
	specs, labels, err := in.Specs()
	if err != nil {
		return Start{}, err
	}
	at, err := declaredAttachIn(specs, labels)
	if err != nil {
		return Start{}, err
	}
	if at == nil {
		return Start{}, fmt.Errorf(
			"provide %s <dir> (compose the network the specs declare), %s <dir> %s (run against the one it already composed), or %s <url> (attach to a running one) — or declare env.attach in the specs",
			s.Workspace, s.Workspace, s.Attach, s.RPC)
	}
	return Start{At: lifecycle.AdoptChainByDeclaration, Declared: at}, nil
}

// Adopts reports whether this start attaches to a network that is already up,
// rather than composing one.
func (s Start) Adopts() bool { return s.At.Block() == lifecycle.AdoptChain }
