// Package upgrade composes the atomic setup primitives (genesis config-section
// merge, port planning, network-id resolution, preflight) into a single,
// validated launch plan for a hardfork handoff: a from-chain (e.g. go-wemix +
// etcd, poa) that produces blocks up to a fork, and a to-chain (e.g. go-wbft,
// bft) that syncs the pre-fork blocks and takes over producing after it.
//
// BuildPlan is pure and engine-agnostic. Nothing here is wemix- or wbft-
// specific: the pair of chains, the fork name, the network id, the roles and
// the genesis values all arrive as inputs (from manifests + a golden profile),
// and the fork's consensus config section is DATA lifted out of the to-chain's
// own genesis — never a constant baked into this package. The side-effecting
// steps (shelling out to build the from-chain's deploy-time genesis, launching
// mixed binaries) stay in their adapters; BuildPlan only assembles and checks.
package upgrade

import (
	"encoding/json"
	"fmt"
	"github.com/0xmhha/chainbench/internal/resource"
	"math/big"
	"strings"

	"github.com/0xmhha/chainbench/internal/core/genesis"
	"github.com/0xmhha/chainbench/internal/core/node"
	"github.com/0xmhha/chainbench/internal/core/registry"
)

// wbftQuorumMin is the minimum number of BFT validators that can make progress
// (a 3f+1 tolerating one fault). Fewer post-fork validators stall the handoff.
const wbftQuorumMin = 4

// Roles says how many nodes of each kind a handoff network runs.
type Roles struct {
	// Producers are from-chain block producers that mine up to the fork.
	Producers int
	// Validators are to-chain post-fork validators that take over. Must be at
	// least wbftQuorumMin for the BFT engine to make progress.
	Validators int
}

// Inputs are the fully-resolved values a handoff network is built from. Every
// field is supplied by the caller (manifest + golden profile); there are no
// code defaults, so a plan records exactly the environment under test.
type Inputs struct {
	Roles Roles
	// NetworkID is the single devp2p network id forced onto every node. The
	// handoff requires a uniform id (go-wemix otherwise defaults to its own,
	// independent of chain id), so it is stated once here rather than taken
	// per-chain from the two manifests.
	NetworkID int64
	// ForkBlock is the handoff height (the to-chain's fork-activation block).
	ForkBlock *big.Int
	// FromGenesis is the from-chain's base genesis bytes (produced out-of-band,
	// e.g. by the poa gwemix genesis shell-out). It must already carry the
	// pre-fork config (chain id, petersburgBlock). The fork section is merged in.
	FromGenesis []byte
	// ToGenesis are the to-chain genesis inputs (validators, BLS keys, extra
	// data, alloc) used to build the to-chain genesis, from which the fork's
	// consensus config section is lifted.
	ToGenesis genesis.Inputs
	// ProducerAddrs are the from-chain producer/member addresses (registered as
	// governance members). They must be disjoint from the to-chain validators.
	ProducerAddrs []string
	// NodePubkeys are each node's 128-hex devp2p public key in plan order
	// (index 0 = first producer). Optional; when supplied it is carried onto the
	// node specs so the mesh can be wired from the plan. Length, if non-zero,
	// must equal the total node count.
	NodePubkeys []string
	// Port bases/steps for resource.Plan (per-node p2p/etcd/http/ws/auth).
	P2PBase, P2PStep, RPCBase, RPCStep int
}

// NodeSpec is one node's resolved launch assignment.
type NodeSpec struct {
	Index int
	// Chain is the manifest id whose binary and consensus family this node runs
	// (from-chain for producers, to-chain for validators).
	Chain string
	// Role is the node's operational role.
	Role node.Role
	// Producer is true for from-chain producers (pre-fork miners), false for
	// to-chain post-fork validators.
	Producer bool
	// NetworkID is the uniform devp2p network id (same for every node).
	NetworkID int64
	// Recommit is the from/to manifest's miner_recommit TOML encoding
	// ("duration" | "nanos"); the driver formats miner.Recommit accordingly.
	Recommit string
	// Ports is the node's resolved port set.
	Ports node.Endpoints
	// Pubkey is the node's 128-hex devp2p public key (no 0x prefix), used to
	// build its enode for mesh wiring. Empty when identities are not supplied.
	Pubkey string
}

// Plan is the composed, preflight-validated description of a handoff network.
type Plan struct {
	From, To registry.Manifest
	AtFork   string
	// Genesis is the from-chain genesis carrying the to-chain's fork section and
	// fork-activation block: every node initializes from these identical bytes.
	Genesis []byte
	Nodes   []NodeSpec
	Network NetworkPlan
}

// BuildPlan assembles a handoff launch plan from the two chain plugins and the
// inputs, validating it through preflight before returning. It is pure: it
// shells out to nothing and mutates no state.
func BuildPlan(from, to registry.ChainPlugin, in Inputs) (Plan, error) {
	fm, tm := from.Manifest(), to.Manifest()

	// The from-chain must declare the handoff, and it must point at the to-chain.
	up := fm.Upgrade
	if up == nil {
		return Plan{}, fmt.Errorf("upgrade: chain %q declares no upgrade in its manifest", fm.ID)
	}
	if up.ToChain != tm.ID {
		return Plan{}, fmt.Errorf("upgrade: chain %q hands off to %q, not %q", fm.ID, up.ToChain, tm.ID)
	}
	if in.Roles.Producers < 1 {
		return Plan{}, fmt.Errorf("upgrade: need at least one producer, got %d", in.Roles.Producers)
	}
	if in.Roles.Validators < wbftQuorumMin {
		return Plan{}, fmt.Errorf("upgrade: %q needs at least %d post-fork validators for BFT progress, got %d", tm.ID, wbftQuorumMin, in.Roles.Validators)
	}
	if in.ForkBlock == nil {
		return Plan{}, fmt.Errorf("upgrade: fork block must be set")
	}
	if len(in.FromGenesis) == 0 {
		return Plan{}, fmt.Errorf("upgrade: from-chain base genesis must be provided")
	}

	// 1. Build the to-chain genesis and lift the fork's consensus config section
	//    out of it. The section is thus data from the to-chain's own template,
	//    fully substituted with the real post-fork validator set — not a
	//    constant in this package.
	toGenesis, err := genesis.Build(to, in.ToGenesis)
	if err != nil {
		return Plan{}, fmt.Errorf("upgrade: build to-chain genesis: %w", err)
	}
	section, err := genesis.ExtractConfigSection(toGenesis, up.AtFork)
	if err != nil {
		return Plan{}, fmt.Errorf("upgrade: read %q section from %q genesis: %w", up.AtFork, tm.ID, err)
	}
	if len(section) == 0 {
		return Plan{}, fmt.Errorf("upgrade: %q genesis has no %q config section to hand off", tm.ID, up.AtFork)
	}

	// 2. Merge the fork section and activation block into the from-chain genesis
	//    so its producers embed the fork and the to-chain accepts the pre-fork
	//    chain. The block key follows the "<fork>Block" convention.
	merged, err := genesis.SetConfigSection(in.FromGenesis, up.AtFork, section)
	if err != nil {
		return Plan{}, fmt.Errorf("upgrade: merge fork section: %w", err)
	}
	blockJSON, err := json.Marshal(in.ForkBlock)
	if err != nil {
		return Plan{}, err
	}
	merged, err = genesis.SetConfigSection(merged, up.AtFork+"Block", blockJSON)
	if err != nil {
		return Plan{}, fmt.Errorf("upgrade: set fork block: %w", err)
	}
	if err := genesis.ValidateForks(merged); err != nil {
		return Plan{}, err
	}

	// 3. Assign roles, ports and the uniform network id. Producers run the
	//    from-chain binary/consensus; validators run the to-chain's.
	total := in.Roles.Producers + in.Roles.Validators
	if len(in.NodePubkeys) != 0 && len(in.NodePubkeys) != total {
		return Plan{}, fmt.Errorf("upgrade: %d node pubkeys but %d nodes", len(in.NodePubkeys), total)
	}
	nodes := make([]NodeSpec, 0, total)
	ports := make([]node.Endpoints, 0, total)
	netids := make([]int64, 0, total)
	for i := 0; i < total; i++ {
		p, err := resource.Plan(i+1, in.P2PBase, in.P2PStep, in.RPCBase, in.RPCStep, node.DefaultReservation)
		if err != nil {
			return Plan{}, fmt.Errorf("upgrade: port plan node %d: %w", i, err)
		}
		producer := i < in.Roles.Producers
		spec := NodeSpec{
			Index: i, Role: node.RoleValidator, Producer: producer,
			NetworkID: in.NetworkID, Ports: p,
		}
		if len(in.NodePubkeys) != 0 {
			spec.Pubkey = in.NodePubkeys[i]
		}
		if producer {
			spec.Chain, spec.Recommit = fm.ID, fm.MinerRecommit
		} else {
			spec.Chain, spec.Recommit = tm.ID, tm.MinerRecommit
		}
		nodes = append(nodes, spec)
		ports = append(ports, p)
		netids = append(netids, in.NetworkID)
	}

	// 4. Validate the assembled plan: each builder checks its own artifact
	// (network ids uniform, ports disjoint, forks ordered) and the handoff
	// checks the one rule that is its own — a validator is not a member.
	net := NetworkPlan{
		NetworkIDs:     netids,
		Ports:          ports,
		Genesis:        merged,
		WemixMembers:   in.ProducerAddrs,
		WbftValidators: in.ToGenesis.Validators,
	}
	if err := net.validate(); err != nil {
		return Plan{}, err
	}

	return Plan{From: fm, To: tm, AtFork: up.AtFork, Genesis: merged, Nodes: nodes, Network: net}, nil
}

// NetworkPlan is the fully-resolved description of the handoff network about
// to launch, kept on the Plan so a surface can show what was validated.
type NetworkPlan struct {
	// NetworkIDs is each node's configured devp2p network id.
	NetworkIDs []int64
	// Ports is each node's resolved port set.
	Ports []node.Endpoints
	// Genesis is the genesis bytes every node initializes from (identical).
	Genesis []byte
	// WemixMembers are the addresses registered as wemix+etcd producers.
	WemixMembers []string
	// WbftValidators are the croissant.init post-fork validator addresses.
	// For an upgrade network these must be disjoint from WemixMembers.
	WbftValidators []string
}

// validate runs the builders' own checks plus the handoff's role rule. Every
// condition here caused a silent, hard-to-diagnose failure at least once.
func (p NetworkPlan) validate() error {
	if err := resource.ValidateUniform(p.NetworkIDs); err != nil {
		return err
	}
	if err := resource.ValidatePorts(p.Ports); err != nil {
		return err
	}
	if len(p.NetworkIDs) != len(p.Ports) {
		return fmt.Errorf("upgrade: %d network ids but %d port sets", len(p.NetworkIDs), len(p.Ports))
	}
	if len(p.Genesis) > 0 {
		if err := genesis.ValidateForks(p.Genesis); err != nil {
			return err
		}
	}
	// A go-wbft validator listed as a wemix member registers in governance but
	// never joins etcd (it runs wbft), shows as "down", and stalls the producer.
	member := map[string]bool{}
	for _, a := range p.WemixMembers {
		member[strings.ToLower(a)] = true
	}
	for _, v := range p.WbftValidators {
		if member[strings.ToLower(v)] {
			return fmt.Errorf("upgrade: address %s is both a wemix member and a wbft validator; keep validators out of the wemix member list", v)
		}
	}
	return nil
}
