// Package registry is the chain-agnostic plugin registry. It holds the
// declarative per-chain Manifest data and the ChainPlugin/ConsensusFamily
// contracts that core code (pipeline, drivers, mcp) uses without importing any
// specific chain — chain knowledge lives in pkg/chains/* and pkg/consensus/*,
// which register here at init (docs/CHAINBENCH_GO_REDESIGN.md §4).
package registry

import (
	"encoding/json"
	"fmt"
)

// Manifest is the declarative static profile of one chain, mirroring
// pkg/chains/<id>/manifest.json. It data-izes the facts that were previously
// hardcoded across lib/*.sh and network/internal/probe (binary name, consensus
// namespace, hardfork fields, supported tx types, probe signature) so adding a
// chain is mostly data, not code.
type Manifest struct {
	// ID is the chain identifier ("stablenet"|"wbft"|"wemix").
	ID string `json:"id"`
	// Binary is the node binary name (gstable|gwbft|gwemix).
	Binary string `json:"binary"`
	// ChainID is the default local-network chain id (operator-overridable via
	// profile). Also used by the probe for chains that disambiguate by id.
	ChainID int64 `json:"chain_id"`
	// NetworkID is the devp2p network id. It MUST be set explicitly and be
	// identical across every node of a network: go-wemix defaults its network
	// id to 1111 (Wemix mainnet) independent of ChainID, while go-wbft defaults
	// it to ChainID, so an unset value silently prevents cross-binary peering.
	// There is deliberately no code default — set it here so a run's network id
	// is always traceable to the manifest.
	NetworkID int64 `json:"network_id"`
	// MinerRecommit selects the TOML encoding of miner.Config.Recommit that
	// this chain's binary accepts: "duration" (a TOML string like "2s", used by
	// go-stablenet/go-wbft) or "nanos" (an integer number of nanoseconds, used
	// by the older go-ethereum in go-wemix). The wrong form crashes the node at
	// config load. No code default.
	MinerRecommit string `json:"miner_recommit"`
	// Bootstrap describes how a network of this chain is brought up.
	Bootstrap BootstrapSpec `json:"bootstrap"`
	// Build describes how to build the binary.
	Build BuildSpec `json:"build"`
	// ConsensusFamily is the consensus algorithm family this chain uses
	// ("wbft" for stablenet+wbft, "poa" for wemix). It selects the
	// ConsensusFamily plugin (docs §4, decision D9).
	ConsensusFamily string `json:"consensus_family"`
	// Protocol names the accounts SDK protocol profile (tx types, account model)
	// this chain uses; empty defaults to ID. An externally-supplied manifest on
	// an existing family sets this to borrow a built-in protocol (e.g.
	// "stablenet") since the SDK does not know the external chain's id.
	Protocol string `json:"protocol,omitempty"`
	// Genesis describes how this chain's genesis is structured.
	Genesis GenesisSpec `json:"genesis"`
	// Consensus holds the RPC-facing consensus facts.
	Consensus ConsensusSpec `json:"consensus"`
	// TxTypes is the set of EIP-2718 tx type bytes this chain accepts,
	// as hex strings (e.g. "0x16"). Mirrors accounts/protocol.
	TxTypes []string `json:"tx_types"`
	// Probe is the chain-detection signature (namespace probe method).
	Probe ProbeSpec `json:"probe"`
	// Capabilities is the provider-independent capability set the chain
	// supports (e.g. "consensus").
	Capabilities []string `json:"capabilities"`
	// Upgrade, when present, declares that a network of this chain hands block
	// production off to another chain's binary/consensus at a fork block (the
	// wemix+etcd -> wbft hardfork). Absent for chains with no upgrade.
	Upgrade *UpgradeSpec `json:"upgrade,omitempty"`
}

// BootstrapSpec describes how a network of a chain is brought up to producing
// blocks — the boundary between the two consensus families.
type BootstrapSpec struct {
	// Type is one of:
	//   "static"          - validators and BLS keys come from the genesis, and
	//                       nodes peer via a static-node list (anzeon/wbft).
	//   "governance-etcd" - a boot node deploys the governance contracts and
	//                       initializes an etcd cluster; peers and the producer
	//                       rotation are driven by the on-chain member list and
	//                       etcd (wemix).
	Type string `json:"type"`
}

// UpgradeSpec declares a hardfork handoff to another chain at a fork block.
// Pre-fork, this chain's nodes produce; at the fork block they stop and the
// to-chain's nodes (running concurrently and syncing until then) take over.
type UpgradeSpec struct {
	// ToChain is the chain id whose binary/consensus takes over (e.g. "wbft").
	ToChain string `json:"to_chain"`
	// AtFork is the fork name whose activation block is the handoff point
	// (e.g. "croissant").
	AtFork string `json:"at_fork"`
	// ValidatorSource says where the post-fork validator set comes from
	// ("croissant_init" = the genesis croissant.init validators/BLS keys).
	ValidatorSource string `json:"validator_source"`
}

// BuildSpec describes how to obtain the node binary.
type BuildSpec struct {
	Repo       string `json:"repo"`        // e.g. "go-wbft"
	MakeTarget string `json:"make_target"` // e.g. "gwbft"
}

// GenesisSpec describes a chain's genesis structure. EngineField is the config
// key that carries consensus config ("anzeon" for stablenet, "croissant" for
// wbft; empty for the poa/registry family whose genesis is deploy-time).
// Template names the chain's genesis template (embedded as
// pkg/chains/<id>/genesis.json by the plugin).
type GenesisSpec struct {
	EngineField string   `json:"engine_field"`
	Hardforks   []string `json:"hardforks"`
	Template    string   `json:"template"` // "" = no static template (poa)
}

// ConsensusSpec holds the RPC-facing consensus facts used by verify/consensus.
type ConsensusSpec struct {
	RPCNamespace     string `json:"rpc_namespace"`     // "istanbul" | "wemix"
	ValidatorsMethod string `json:"validators_method"` // "istanbul_getValidators" | ...
}

// ProbeSpec is the chain-detection signature (see network/internal/probe).
type ProbeSpec struct {
	Method   string  `json:"method"`              // RPC method whose presence identifies the chain
	ChainIDs []int64 `json:"chain_ids,omitempty"` // optional chain-id gate for disambiguation
}

// ParseManifest decodes a Manifest from JSON bytes and validates required
// fields.
func ParseManifest(b []byte) (Manifest, error) {
	var m Manifest
	if err := json.Unmarshal(b, &m); err != nil {
		return Manifest{}, fmt.Errorf("registry: parse manifest: %w", err)
	}
	if err := m.validate(); err != nil {
		return Manifest{}, err
	}
	return m, nil
}

func (m Manifest) validate() error {
	if m.ID == "" {
		return fmt.Errorf("registry: manifest missing id")
	}
	if m.Binary == "" {
		return fmt.Errorf("registry: manifest %q missing binary", m.ID)
	}
	if m.ConsensusFamily == "" {
		return fmt.Errorf("registry: manifest %q missing consensus_family", m.ID)
	}
	if m.ChainID <= 0 {
		return fmt.Errorf("registry: manifest %q missing/invalid chain_id", m.ID)
	}
	if m.NetworkID <= 0 {
		return fmt.Errorf("registry: manifest %q missing/invalid network_id (set it explicitly; there is no default)", m.ID)
	}
	switch m.MinerRecommit {
	case "duration", "nanos":
	default:
		return fmt.Errorf("registry: manifest %q miner_recommit must be \"duration\" or \"nanos\", got %q", m.ID, m.MinerRecommit)
	}
	switch m.Bootstrap.Type {
	case "static", "governance-etcd":
	default:
		return fmt.Errorf("registry: manifest %q bootstrap.type must be \"static\" or \"governance-etcd\", got %q", m.ID, m.Bootstrap.Type)
	}
	if u := m.Upgrade; u != nil {
		if u.ToChain == "" || u.AtFork == "" || u.ValidatorSource == "" {
			return fmt.Errorf("registry: manifest %q upgrade requires to_chain, at_fork, and validator_source", m.ID)
		}
	}
	return nil
}
