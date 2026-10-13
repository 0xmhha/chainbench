package feature

import (
	"context"

	"github.com/0xmhha/chainbench/internal/app"
)

type resolveNodesInput struct {
	DataDir string   `cb:"data-dir" help:"Composition workspace directory"`
	Chain   string   `cb:"chain" help:"Chain identifier for explicit endpoints"`
	RPCURLs []string `cb:"rpc" help:"Explicit node RPC endpoints"`
}

type networkStateInput struct {
	StateDir string `cb:"state-dir,required" help:"Directory holding attached network records"`
}

type namedNetworkInput struct {
	networkStateInput
	Name string `cb:"name,required" help:"Name of an attached network"`
}

type attachNetworkInput struct {
	networkStateInput
	Network app.NodeSet `json:"network"`
}

type rpcInput struct {
	RPC string `cb:"rpc,required" help:"Node RPC endpoint"`
}

type nodeInput struct {
	Node app.Node `json:"node"`
}

type callNodeInput struct {
	nodeInput
	Method string `cb:"method,required" help:"RPC method to invoke"`
	Params []any  `json:"params"`
}

type resolveServerInput struct {
	Ref           app.ServerRef `json:"ref"`
	MinValidators int           `cb:"min-validators" help:"Minimum validator capacity"`
	PortBand      int           `cb:"port-band" help:"Ports reserved for each node slot"`
}

type validatorsInput struct {
	Chain    string `cb:"chain" help:"Chain identifier"`
	Manifest string `cb:"manifest" help:"Optional plugin manifest"`
	Template string `cb:"template" help:"Optional plugin template"`
	RPC      string `cb:"rpc,required" help:"Node RPC endpoint"`
}

func init() {
	Register(Registration{Name: "network.resolve", Stage: StageReport, ReadOnly: true, Summary: "Resolve recorded or explicit endpoints into a node set"}, func(ctx context.Context, deps app.Deps, in resolveNodesInput) (app.NodeSet, error) {
		return app.ResolveNodes(ctx, deps, in.DataDir, in.Chain, in.RPCURLs)
	})
	Register(Registration{Name: "network.attach", Stage: StageCompose, Summary: "Record an existing network without controlling its processes"}, func(_ context.Context, deps app.Deps, in attachNetworkInput) (struct{}, error) {
		return struct{}{}, app.AttachNetwork(deps, in.StateDir, in.Network)
	})
	Register(Registration{Name: "network.info", Stage: StageReport, ReadOnly: true, Summary: "Read one attached network's record"}, func(_ context.Context, deps app.Deps, in namedNetworkInput) (app.NodeSet, error) {
		return app.Network(deps, in.StateDir, in.Name)
	})
	Register(Registration{Name: "network.list", Stage: StageReport, ReadOnly: true, Summary: "List attached network records"}, func(_ context.Context, deps app.Deps, in networkStateInput) ([]app.NodeSet, error) {
		return app.Networks(deps, in.StateDir)
	})
	Register(Registration{Name: "network.detach", Stage: StageCompose, Summary: "Forget an attached network without stopping its nodes"}, func(_ context.Context, deps app.Deps, in namedNetworkInput) (struct{}, error) {
		return struct{}{}, app.DetachNetwork(deps, in.StateDir, in.Name)
	})
	Register(Registration{Name: "node.peers", Stage: StageReport, ReadOnly: true, Summary: "Read a node's peer count"}, func(ctx context.Context, deps app.Deps, in rpcInput) (uint64, error) {
		return app.PeersOf(ctx, deps, in.RPC)
	})
	Register(Registration{Name: "node.status", Stage: StageReport, ReadOnly: true, Summary: "Read a node's height, chain ID, peers and sync status"}, func(ctx context.Context, deps app.Deps, in rpcInput) (app.NodeReadings, error) {
		return app.ReadNode(ctx, deps, in.RPC)
	})
	Register(Registration{Name: "network.node-peers", Stage: StageReport, ReadOnly: true, Summary: "Read a recorded node's peer count through its transport"}, func(ctx context.Context, deps app.Deps, in nodeInput) (uint64, error) {
		return app.PeersOfNode(ctx, deps, in.Node)
	})
	Register(Registration{Name: "network.node-call", Stage: StageTest, Summary: "Call an arbitrary method on a recorded node"}, func(ctx context.Context, deps app.Deps, in callNodeInput) ([]byte, error) {
		return app.CallOnNode(ctx, deps, in.Node, in.Method, in.Params...)
	})
	Register(Registration{Name: "resource.resolve", Stage: StageCompose, Summary: "Resolve server placement and target access settings"}, func(_ context.Context, deps app.Deps, in resolveServerInput) (app.ResolveServerOut, error) {
		return app.ResolveServer(deps, in.Ref, in.MinValidators, in.PortBand)
	})
	Register(Registration{Name: "chain.validators", Stage: StageReport, ReadOnly: true, Summary: "Read the running validator set through its consensus family"}, func(ctx context.Context, deps app.Deps, in validatorsInput) (app.ValidatorsOut, error) {
		return app.Validators(ctx, deps, in.Chain, in.Manifest, in.Template, in.RPC)
	})
}
