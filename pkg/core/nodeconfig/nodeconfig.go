// Package nodeconfig generates a per-node geth-family TOML config for the setup
// phase's environment-build step (requirement #8). It is chain-aware only in
// the RPC namespace exposed (istanbul vs wemix, from the consensus family) and
// the miner section (validators only); everything else is shared geth config.
package nodeconfig

import (
	"fmt"
	"strings"

	"github.com/0xmhha/chainbench/pkg/core/node"
)

// baseModules are the RPC namespaces every node exposes; the chain's consensus
// namespace is appended.
var baseModules = []string{"admin", "eth", "debug", "miner", "net", "txpool", "personal", "web3"}

// Params fully describes one node's config.
type Params struct {
	Role         node.Role
	Ports        node.Endpoints
	KeystoreDir  string
	RPCNamespace string   // consensus namespace, e.g. "istanbul" or "wemix"
	HTTPHost     string   // default "0.0.0.0"
	MetricsHost  string   // default "127.0.0.1"
	StaticNodes  []string // enode URLs (all nodes, self included; geth ignores self)
	SyncMode     string   // default "full"
}

// Generate renders the TOML config bytes for one node.
func Generate(p Params) []byte {
	httpHost := orDefault(p.HTTPHost, "0.0.0.0")
	metricsHost := orDefault(p.MetricsHost, "127.0.0.1")
	syncMode := orDefault(p.SyncMode, "full")

	modules := append(append([]string{}, baseModules...), p.RPCNamespace)

	var b strings.Builder
	fmt.Fprintf(&b, "[Eth]\nSyncMode = %q\n\n", syncMode)

	if p.Role == node.RoleValidator || p.Role == node.RoleBoot {
		b.WriteString("[Eth.Miner]\nRecommit = \"2s\"\n\n")
	}

	b.WriteString("[Node]\n")
	fmt.Fprintf(&b, "KeyStoreDir = %q\n", p.KeystoreDir)
	fmt.Fprintf(&b, "AuthPort = %d\n", p.Ports.Auth)
	fmt.Fprintf(&b, "HTTPHost = %q\n", httpHost)
	fmt.Fprintf(&b, "HTTPPort = %d\n", p.Ports.HTTP)
	b.WriteString("HTTPCors = [\"*\"]\n")
	b.WriteString("HTTPVirtualHosts = [\"*\"]\n")
	fmt.Fprintf(&b, "HTTPModules = [%s]\n\n", quoteList(modules))

	b.WriteString("[Node.P2P]\n")
	fmt.Fprintf(&b, "ListenAddr = \":%d\"\n", p.Ports.P2P)
	b.WriteString("NoDiscovery = true\n")
	if len(p.StaticNodes) > 0 {
		b.WriteString("StaticNodes = [\n")
		for i, en := range p.StaticNodes {
			sep := ","
			if i == len(p.StaticNodes)-1 {
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
	fmt.Fprintf(&b, "Port = %d\n", p.Ports.Metrics)
	b.WriteString("EnableInfluxDB = false\n")
	b.WriteString("EnableInfluxDBV2 = false\n")

	return []byte(b.String())
}

// Enode builds a static-node enode URL from a devp2p public key and p2p port.
func Enode(publicKey, host string, p2pPort int) string {
	return fmt.Sprintf("enode://%s@%s:%d?discport=0", publicKey, host, p2pPort)
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
