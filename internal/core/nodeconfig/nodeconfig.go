// Package nodeconfig generates a per-node geth-family TOML config for the setup
// phase's environment-build step (requirement #8). It is chain-aware only in
// the RPC namespace exposed (from the consensus family) and the miner-recommit
// encoding; both come from the chain manifest (RPCNamespace, MinerRecommit), so
// no chain name is hardcoded here. Everything else is shared geth config.
package nodeconfig

import (
	"fmt"
	"github.com/0xmhha/chainbench/internal/core/node"
	"strconv"
	"strings"
)

// baseModules are the RPC namespaces every node exposes; the chain's consensus
// namespace is appended.
var baseModules = []string{"admin", "eth", "debug", "miner", "net", "txpool", "personal", "web3"}

// Params fully describes one node's config.
type Params struct {
	Role          node.Role
	Ports         node.Endpoints
	KeystoreDir   string
	RPCNamespace  string   // consensus namespace, e.g. "istanbul" or "wemix"
	MinerRecommit string   // manifest miner_recommit form: "duration" (default) | "nanos"
	HTTPHost      string   // default "0.0.0.0"
	MetricsHost   string   // default "127.0.0.1"
	StaticNodes   []string // enode URLs (all nodes, self included; geth ignores self)
	SyncMode      string   // default "full"
}

// Generate renders the TOML config bytes for one node.
func Generate(p Params) []byte {
	httpHost := orDefault(p.HTTPHost, "0.0.0.0")
	metricsHost := orDefault(p.MetricsHost, "127.0.0.1")
	syncMode := orDefault(p.SyncMode, "full")

	modules := append(append([]string{}, baseModules...), p.RPCNamespace)

	var b strings.Builder
	fmt.Fprintf(&b, "[Eth]\nSyncMode = %q\n\n", syncMode)

	if node.Is(p.Role, node.RoleBP) || node.Is(p.Role, node.RoleBoot) {
		// miner.Config.Recommit is a time.Duration. Most geth-family binaries
		// (go-stablenet/go-wbft) decode it from a TOML string ("2s"); the older
		// go-ethereum in go-wemix decodes it only from an integer number of
		// nanoseconds. Which form the target binary accepts is a manifest fact
		// (miner_recommit), threaded in via MinerRecommit — do not re-derive it
		// from the chain/namespace here.
		recommit := `"2s"`
		if p.MinerRecommit == "nanos" {
			recommit = "2000000000" // 2s in nanoseconds
		}
		fmt.Fprintf(&b, "[Eth.Miner]\nRecommit = %s\n\n", recommit)
	}

	b.WriteString("[Node]\n")
	fmt.Fprintf(&b, "KeyStoreDir = %q\n", p.KeystoreDir)
	fmt.Fprintf(&b, "AuthPort = %d\n", p.Ports.Auth)
	fmt.Fprintf(&b, "HTTPHost = %q\n", httpHost)
	fmt.Fprintf(&b, "HTTPPort = %d\n", p.Ports.HTTP)
	b.WriteString("HTTPCors = [\"*\"]\n")
	b.WriteString("HTTPVirtualHosts = [\"*\"]\n")
	fmt.Fprintf(&b, "HTTPModules = [%s]\n\n", quoteList(modules))

	// WebSocket endpoint (same host/modules as HTTP) so eth_subscribe is served;
	// the port is also set via the --ws.port launch flag.
	fmt.Fprintf(&b, "WSHost = %q\n", httpHost)
	fmt.Fprintf(&b, "WSPort = %d\n", p.Ports.WS)
	b.WriteString("WSOrigins = [\"*\"]\n")
	fmt.Fprintf(&b, "WSModules = [%s]\n\n", quoteList(modules))

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

// LaunchArgs assembles the common geth-family node launch flags (datadir, config,
// and the port flags) plus the family-specific flags. These are geth-family
// conventions shared by both consensus families and by the binary-swap hardfork,
// so they live here (next to the config generation) rather than in a pipeline
// stage — the hardfork executor need not depend on pkg/core/pipeline/setup.
func LaunchArgs(dataDir, configPath string, ports node.Endpoints, familyFlags []string) []string {
	args := []string{
		"--datadir", dataDir,
		"--config", configPath,
		"--port", strconv.Itoa(ports.P2P),
		"--http",
		"--http.port", strconv.Itoa(ports.HTTP),
		"--ws",
		"--ws.port", strconv.Itoa(ports.WS),
	}
	return append(args, familyFlags...)
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
