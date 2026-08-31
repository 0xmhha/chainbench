// Package nodeconfig owns one node's configuration and renders it two ways:
// the TOML file a geth-family binary reads, and the argv it is launched with.
//
// The two are one fact with two spellings. They used to have two owners —
// this package rendered the file while launchopt's caller assembled the argv
// — and each caller of either re-derived the inputs from a plan or a record
// its own way. Spec is the single input; TOML and Argv are its renderers.
// What the file carries is not repeated on the command line (a node whose
// config names its auth port is not also told --authrpc.port), so the two
// renderings agree by construction rather than by care.
//
// launchopt stays a separate package beneath this one: it is the dialect
// table (which binary spells which flag how) and the layered override model,
// and Argv is its only caller that speaks for a node.
package nodeconfig

import (
	"fmt"
	"strings"

	"github.com/0xmhha/chainbench/internal/core/node"
	"github.com/0xmhha/chainbench/internal/core/registry"
)

// baseModules are the RPC namespaces every node exposes; the chain's consensus
// namespace is appended.
var baseModules = []string{"admin", "eth", "debug", "miner", "net", "txpool", "personal", "web3"}

// Chain is what a node's configuration takes from the chain it runs: facts
// from the manifest and the consensus family, none of them per node.
type Chain struct {
	// ID is the manifest id; it selects the flag dialect the binary speaks.
	ID string
	// RPCNamespace is the consensus namespace exposed over RPC ("istanbul",
	// "wemix").
	RPCNamespace string
	// MinerRecommit is the manifest's miner_recommit form: "duration"
	// (default) or "nanos" — which the target binary decodes.
	MinerRecommit string
	// NetworkID is the devp2p network id; 0 leaves it to the binary.
	NetworkID int64
	// FamilyFlags are the family's role-specific start flags, parsed into the
	// policy the argv carries (mining, unlock policy, RPC policy).
	FamilyFlags []string
}

// ChainOf reads a node's chain facts from its plugin for one role.
func ChainOf(plugin registry.ChainPlugin, role node.Role) Chain {
	m := plugin.Manifest()
	return Chain{
		ID:            m.ID,
		RPCNamespace:  m.Consensus.RPCNamespace,
		MinerRecommit: m.MinerRecommit,
		NetworkID:     m.NetworkID,
		FamilyFlags:   plugin.Family().StartFlags(role),
	}
}

// Spec is one node's configuration: everything both renderers draw on.
type Spec struct {
	Chain Chain
	Role  node.Role
	Ports node.Endpoints
	// SyncMode is the geth sync mode; empty renders the default ("full").
	SyncMode string

	// DataDir is the node's datadir; ConfigPath is where the TOML lands.
	DataDir    string
	ConfigPath string

	// Identity: where the node's keys are and, for a producer, which account
	// it seals with. Empty fields are simply not rendered.
	NodekeyPath  string
	KeystoreDir  string
	Unlock       string // 0x address to unlock, and the etherbase
	PasswordFile string

	// StaticNodes are the enode URLs this node dials (self included; geth
	// ignores self).
	StaticNodes []string

	// HTTPHost binds the HTTP and WS endpoints; empty is 0.0.0.0 in the file
	// and unset on the command line.
	HTTPHost string
	// MetricsHost binds the metrics endpoint; empty is 127.0.0.1.
	MetricsHost string
}

// TOML renders the config file a geth-family binary reads.
func TOML(s Spec) []byte {
	httpHost := orDefault(s.HTTPHost, "0.0.0.0")
	metricsHost := orDefault(s.MetricsHost, "127.0.0.1")
	syncMode := orDefault(s.SyncMode, "full")

	modules := append(append([]string{}, baseModules...), s.Chain.RPCNamespace)

	var b strings.Builder
	fmt.Fprintf(&b, "[Eth]\nSyncMode = %q\n\n", syncMode)

	if node.Is(s.Role, node.RoleBP) || node.Is(s.Role, node.RoleBoot) {
		// miner.Config.Recommit is a time.Duration. Most geth-family binaries
		// (go-stablenet/go-wbft) decode it from a TOML string ("2s"); the older
		// go-ethereum in go-wemix decodes it only from an integer number of
		// nanoseconds. Which form the target binary accepts is a manifest fact
		// (miner_recommit) — do not re-derive it from the chain here.
		recommit := `"2s"`
		if s.Chain.MinerRecommit == "nanos" {
			recommit = "2000000000" // 2s in nanoseconds
		}
		fmt.Fprintf(&b, "[Eth.Miner]\nRecommit = %s\n\n", recommit)
	}

	b.WriteString("[Node]\n")
	fmt.Fprintf(&b, "KeyStoreDir = %q\n", s.KeystoreDir)
	fmt.Fprintf(&b, "AuthPort = %d\n", s.Ports.Auth)
	fmt.Fprintf(&b, "HTTPHost = %q\n", httpHost)
	fmt.Fprintf(&b, "HTTPPort = %d\n", s.Ports.HTTP)
	b.WriteString("HTTPCors = [\"*\"]\n")
	b.WriteString("HTTPVirtualHosts = [\"*\"]\n")
	fmt.Fprintf(&b, "HTTPModules = [%s]\n\n", quoteList(modules))

	// WebSocket endpoint (same host/modules as HTTP) so eth_subscribe is served;
	// the port is also set via the --ws.port launch flag.
	fmt.Fprintf(&b, "WSHost = %q\n", httpHost)
	fmt.Fprintf(&b, "WSPort = %d\n", s.Ports.WS)
	b.WriteString("WSOrigins = [\"*\"]\n")
	fmt.Fprintf(&b, "WSModules = [%s]\n\n", quoteList(modules))

	b.WriteString("[Node.P2P]\n")
	fmt.Fprintf(&b, "ListenAddr = \":%d\"\n", s.Ports.P2P)
	b.WriteString("NoDiscovery = true\n")
	if len(s.StaticNodes) > 0 {
		b.WriteString("StaticNodes = [\n")
		for i, en := range s.StaticNodes {
			sep := ","
			if i == len(s.StaticNodes)-1 {
				sep = ""
			}
			fmt.Fprintf(&b, "  %q%s\n", en, sep)
		}
		b.WriteString("]\n")
	} else {
		b.WriteString("StaticNodes = []\n")
	}
	b.WriteString("\n")

	b.WriteString("[Metrics]\n")
	b.WriteString("Enabled = true\n")
	fmt.Fprintf(&b, "HTTP = %q\n", metricsHost)
	fmt.Fprintf(&b, "Port = %d\n", s.Ports.Metrics)
	b.WriteString("EnableInfluxDB = false\n")
	b.WriteString("EnableInfluxDBV2 = false\n")

	return []byte(b.String())
}

// Argv renders the launch argv through the launchopt Builder: the family's
// policy from its start flags, the identity, storage, p2p, and RPC modules
// from the spec, then the caller's overrides on their own layer.
//
// A spec with a ConfigPath leaves the auth port to the file; one without
// (a handoff relaunch that carries no config) says it on the command line.
func Argv(s Spec, overrides ...Override) ([]string, error) {
	policy, err := ParseFamilyFlags(s.Chain.FamilyFlags)
	if err != nil {
		return nil, err
	}
	id := Identity{
		NodeKeyFile:         s.NodekeyPath,
		AllowInsecureUnlock: policy.AllowInsecureUnlock,
	}
	if s.Unlock != "" {
		id.Unlock = s.Unlock
		id.PasswordFile = s.PasswordFile
		id.Etherbase = s.Unlock
	}
	modules := []Module{
		id,
		Storage{DataDir: s.DataDir, ConfigFile: s.ConfigPath},
		// The manifest's network id is emitted rather than left to the
		// binary's default: a chain whose devp2p network id differs from its
		// genesis chain id (the handoff produces one) must say so. An
		// operator's --network-id still wins, arriving on a later layer.
		P2P{Port: s.Ports.P2P, NetworkID: s.Chain.NetworkID},
		HTTPRPC{Enabled: true, Addr: s.HTTPHost, Port: s.Ports.HTTP},
		WSRPC{Enabled: true, Port: s.Ports.WS},
	}
	if s.ConfigPath == "" && s.Ports.Auth > 0 {
		modules = append(modules, AuthIPC{AuthPort: s.Ports.Auth})
	}
	modules = append(modules,
		RPCPolicy{DeprecatedPersonal: policy.DeprecatedPersonal, UnprotectedTxs: policy.UnprotectedTxs},
		Mining{Mine: policy.Mine},
	)
	return New(DialectFor(s.Chain.ID), modules...).WithOverrides(overrides...).Build()
}

func quoteList(ss []string) string {
	parts := make([]string, len(ss))
	for i, s := range ss {
		parts[i] = fmt.Sprintf("%q", s)
	}
	return strings.Join(parts, ", ")
}

func orDefault(v, def string) string {
	if v == "" {
		return def
	}
	return v
}
