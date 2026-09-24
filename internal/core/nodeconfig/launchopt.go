// Launch-option assembly (part of package nodeconfig): the single assembly
// point for a node's launch command line (background requirement #2, algorithm
// step 7). It replaces the five scattered argv sites measured in
// docs/dev/architecture/code-graph.md §3 with one Builder over ten concern
// modules. (Formerly the standalone launchopt package, folded into nodeconfig.)
//
// The design (formerly docs/dev/archive/chain-binary-flag-graph.md §3.3, since
// retired; `git show cde3a08f:<that path>`) rests on one measured
// fact: the three chain binaries expose two flag generations, not three, so a
// Dialect — the flag vocabulary of one binary generation — is the only place
// that knows a binary's spelling. Option modules stay chain-agnostic and speak
// typed Keys; the Dialect translates or reports the knob unsupported.
//
// Everything here is a pure function of its inputs: no I/O, no globals, fully
// unit-testable.
package nodeconfig

import (
	"fmt"
	"sort"
	"strings"
)

// OptionKey is the chain-agnostic name of one launch knob. Typed so a knob is never
// a magic string; the Dialect maps it to (or refuses) a concrete flag.
type OptionKey string

// The vocabulary. Grouped by owning module (see modules.go). Spellings on the
// right are the geth114 dialect; deviations live in the dialect tables.
const (
	// Identity
	KeyNodeKey             OptionKey = "nodekey"               // --nodekey <file>
	KeyKeystore            OptionKey = "keystore"              // --keystore <dir>
	KeyUnlock              OptionKey = "unlock"                // --unlock <addr>
	KeyPassword            OptionKey = "password"              // --password <file>
	KeyAllowInsecureUnlock OptionKey = "allow-insecure-unlock" // --allow-insecure-unlock
	KeyEtherbase           OptionKey = "miner.etherbase"       // --miner.etherbase <addr>

	// Storage
	KeyDataDir  OptionKey = "datadir"  // --datadir <dir>
	KeyConfig   OptionKey = "config"   // --config <file>
	KeySyncMode OptionKey = "syncmode" // --syncmode full|snap
	KeyGCMode   OptionKey = "gcmode"   // --gcmode full|archive

	// P2P
	KeyPort       OptionKey = "port"       // --port <n>
	KeyBootnodes  OptionKey = "bootnodes"  // --bootnodes <enodes>
	KeyNoDiscover OptionKey = "nodiscover" // --nodiscover
	KeyMaxPeers   OptionKey = "maxpeers"   // --maxpeers <n>
	KeyNAT        OptionKey = "nat"        // --nat none|any|...
	KeyNetworkID  OptionKey = "networkid"  // --networkid <n>

	// HTTPRPC
	KeyHTTP           OptionKey = "http"            // --http
	KeyHTTPAddr       OptionKey = "http.addr"       // --http.addr <ip>
	KeyHTTPPort       OptionKey = "http.port"       // --http.port <n>
	KeyHTTPAPI        OptionKey = "http.api"        // --http.api <list>
	KeyHTTPVHosts     OptionKey = "http.vhosts"     // --http.vhosts <list>
	KeyHTTPCorsDomain OptionKey = "http.corsdomain" // --http.corsdomain <list>

	// WSRPC
	KeyWS        OptionKey = "ws"         // --ws
	KeyWSAddr    OptionKey = "ws.addr"    // --ws.addr <ip>
	KeyWSPort    OptionKey = "ws.port"    // --ws.port <n>
	KeyWSAPI     OptionKey = "ws.api"     // --ws.api <list>
	KeyWSOrigins OptionKey = "ws.origins" // --ws.origins <list>

	// AuthIPC
	KeyAuthAddr   OptionKey = "authrpc.addr" // --authrpc.addr <ip>
	KeyAuthPort   OptionKey = "authrpc.port" // --authrpc.port <n>
	KeyIPCPath    OptionKey = "ipcpath"      // --ipcpath <path>
	KeyIPCDisable OptionKey = "ipcdisable"   // --ipcdisable

	// RPCPolicy
	KeyRPCDeprecatedPersonal OptionKey = "rpc.enabledeprecatedpersonal" // --rpc.enabledeprecatedpersonal
	KeyRPCUnprotectedTxs     OptionKey = "rpc.allow-unprotected-txs"    // --rpc.allow-unprotected-txs
	KeyRPCGasCap             OptionKey = "rpc.gascap"                   // --rpc.gascap <n>
	KeyRPCTxFeeCap           OptionKey = "rpc.txfeecap"                 // --rpc.txfeecap <n>

	// Mining
	KeyMine          OptionKey = "mine"           // --mine
	KeyMinerGasLimit OptionKey = "miner.gaslimit" // --miner.gaslimit <n>
	KeyMinerGasPrice OptionKey = "miner.gasprice" // --miner.gasprice <n>
	KeyMinerRecommit OptionKey = "miner.recommit" // --miner.recommit <dur|nanos>

	// Metrics
	KeyMetrics     OptionKey = "metrics"      // --metrics
	KeyMetricsAddr OptionKey = "metrics.addr" // --metrics.addr <ip>
	KeyMetricsPort OptionKey = "metrics.port" // --metrics.port <n>

	// Txpool — mempool admission and retention. These decide whether a
	// transaction a test submits is kept, replaced or dropped, so a test that
	// exercises nonce gaps or replacement needs to say what the pool does
	// rather than hope the default suits it.
	KeyTxPoolLocals       OptionKey = "txpool.locals"       // --txpool.locals <addrs>
	KeyTxPoolNoLocals     OptionKey = "txpool.nolocals"     // --txpool.nolocals
	KeyTxPoolJournal      OptionKey = "txpool.journal"      // --txpool.journal <path>
	KeyTxPoolRejournal    OptionKey = "txpool.rejournal"    // --txpool.rejournal <dur>
	KeyTxPoolPriceLimit   OptionKey = "txpool.pricelimit"   // --txpool.pricelimit <n>
	KeyTxPoolPriceBump    OptionKey = "txpool.pricebump"    // --txpool.pricebump <n>
	KeyTxPoolAccountSlots OptionKey = "txpool.accountslots" // --txpool.accountslots <n>
	KeyTxPoolGlobalSlots  OptionKey = "txpool.globalslots"  // --txpool.globalslots <n>
	KeyTxPoolAccountQueue OptionKey = "txpool.accountqueue" // --txpool.accountqueue <n>
	KeyTxPoolGlobalQueue  OptionKey = "txpool.globalqueue"  // --txpool.globalqueue <n>
	KeyTxPoolLifetime     OptionKey = "txpool.lifetime"     // --txpool.lifetime <dur>

	// Cache — memory split across database, trie, pruning and snapshots. A
	// sync or pruning test that does not set these is measuring the default.
	KeyCache           OptionKey = "cache"            // --cache <mb>
	KeyCacheDatabase   OptionKey = "cache.database"   // --cache.database <pct>
	KeyCacheTrie       OptionKey = "cache.trie"       // --cache.trie <pct>
	KeyCacheGC         OptionKey = "cache.gc"         // --cache.gc <pct>
	KeyCacheSnapshot   OptionKey = "cache.snapshot"   // --cache.snapshot <pct>
	KeyCacheNoPrefetch OptionKey = "cache.noprefetch" // --cache.noprefetch
	KeyCachePreimages  OptionKey = "cache.preimages"  // --cache.preimages

	// GPO — the gas price oracle behind eth_gasPrice and eth_maxPriorityFee.
	// The gas-policy suite asserts on those answers, and they come from here.
	KeyGPOBlocks      OptionKey = "gpo.blocks"      // --gpo.blocks <n>
	KeyGPOPercentile  OptionKey = "gpo.percentile"  // --gpo.percentile <n>
	KeyGPOMaxPrice    OptionKey = "gpo.maxprice"    // --gpo.maxprice <wei>
	KeyGPOIgnorePrice OptionKey = "gpo.ignoreprice" // --gpo.ignoreprice <wei>

	// State — what the node keeps and for how long. Archive-vs-pruned changes
	// which historical reads answer at all.
	KeySnapshot       OptionKey = "snapshot"        // --snapshot
	KeyDataDirAncient OptionKey = "datadir.ancient" // --datadir.ancient <path>
	KeyTxLookupLimit  OptionKey = "txlookuplimit"   // --txlookuplimit <n>

	// Peering shape beyond the port: how many pending peers, which networks
	// may connect, where discovery looks.
	KeyMaxPendPeers  OptionKey = "maxpendpeers"   // --maxpendpeers <n>
	KeyNetRestrict   OptionKey = "netrestrict"    // --netrestrict <cidr>
	KeyDiscoveryDNS  OptionKey = "discovery.dns"  // --discovery.dns <url>
	KeyDiscoveryPort OptionKey = "discovery.port" // geth114 --discovery.port <n>

	// Dev mode — a single-node chain that seals on demand. Useful for tests
	// that need a chain and not a network.
	KeyDev         OptionKey = "dev"          // --dev
	KeyDevPeriod   OptionKey = "dev.period"   // --dev.period <s>
	KeyDevGasLimit OptionKey = "dev.gaslimit" // --dev.gaslimit <n>

	// Miner extras beyond the ones the Mining module already owns.
	KeyMinerExtraData OptionKey = "miner.extradata" // --miner.extradata <bytes>

	// History pruning (geth114 generation only).
	KeyHistoryState        OptionKey = "history.state"        // --history.state <n>
	KeyHistoryTransactions OptionKey = "history.transactions" // --history.transactions <n>
	KeyCacheBlockLogs      OptionKey = "cache.blocklogs"      // --cache.blocklogs <n>

	// ChainExt — generation-specific consensus knobs. geth114 has none of
	// these; requesting one there is a classified error, never a silent skip.
	KeyConsensusMethod   OptionKey = "chain.consensusmethod"    // gwemix --consensusmethod
	KeyBlocksPerTurn     OptionKey = "chain.blocksperturn"      // gwemix --blocksperturn
	KeyNonceLimit        OptionKey = "chain.noncelimit"         // gwemix --noncelimit
	KeyMaxTxsPerBlock    OptionKey = "chain.maxtxsperblock"     // gwemix --maxtxsperblock
	KeyBlockInterval     OptionKey = "chain.block.interval"     // gwemix --wemix.block.interval
	KeyBlockTimeAdj      OptionKey = "chain.block.timeadj"      // gwemix --wemix.block.timeadjblocks
	KeyBlockMinBuildTime OptionKey = "chain.block.minbuildtime" // gwemix --wemix.block.minbuildtime
	KeyBlockMinBuildTxs  OptionKey = "chain.block.minbuildtxs"  // gwemix --wemix.block.minbuildtxs
	KeyBlockTrailTime    OptionKey = "chain.block.trailtime"    // gwemix --wemix.block.trailtime
	KeyBootnodeCount     OptionKey = "chain.bootnodecount"      // gwemix --wemix.bootnodecount
	KeyMaxIdleBlock      OptionKey = "chain.maxidleblock"       // gwemix --maxidleblockinterval
	KeyFixedDifficulty   OptionKey = "chain.fixeddifficulty"    // gwemix --fixeddifficulty
	KeyFixedGasLimit     OptionKey = "chain.fixedgaslimit"      // gwemix --fixedgaslimit
	KeyMinerGasTarget    OptionKey = "miner.gastarget"          // gwemix --miner.gastarget
)

// Layer names one precedence level of the value stack: a later layer setting
// the same OptionKey wins.
//
// It travels so a refusal can say WHO asked for a knob the binary does not
// have, which is the difference between "this dialect has no such flag" and
// "the case you wrote asked for a flag this dialect has no such flag". The
// winner used to be stored alongside each value as well, for a `chain status`
// display that was never built — nothing read it in the two years it existed,
// so the value went and the diagnostic stayed.
//
// These four are not the three tiers a declaration is merged through
// (chain-preset, then the case's own override, then the command). The first
// two name who COMPUTED a value and the last two which document SUPPLIED one,
// and the first two tiers both arrive already merged as LayerEnv. That is on
// purpose: which tier a declared value came from is answered by reading the
// case file, not by the argv assembler.
type Layer string

const (
	// LayerHarness is what the harness itself turns on for every node it
	// composes, regardless of chain or role. It is not a consensus-family
	// choice — the three knobs here (insecure unlock, the two legacy RPC
	// permissions) are what a test network needs and what the dialect then
	// filters by whether the binary has the flag at all.
	LayerHarness Layer = "harness"
	// LayerRole is derived from the node's own facts: where its data lives,
	// which ports it holds, which key files it opens, whether it mines.
	LayerRole Layer = "role"
	// LayerEnv is what a declaration asked for, after the chain-preset and the
	// case's own overrides have been merged into one document.
	LayerEnv Layer = "env.launch"
	// LayerCommand is what the invocation overrode, from the CLI or from MCP.
	// It is also where an override that names no layer lands, and it wins.
	LayerCommand Layer = "command"
)

// flagSpec is one dialect row: concrete spelling plus whether the flag is
// boolean (takes no value).
type flagSpec struct {
	name    string
	boolean bool
}

// Dialect is the flag vocabulary of one binary generation. It is the ONLY
// place that knows a binary's spelling.
type Dialect struct {
	// ID names the generation: "geth114" | "geth110-wemix".
	ID    string
	flags map[OptionKey]flagSpec
}

// Spelling returns the dialect's flag name for k, or ok=false when the
// generation does not have the knob.
func (d Dialect) Spelling(k OptionKey) (string, bool) {
	s, ok := d.flags[k]
	return s.name, ok
}

// IsBool reports whether k is a value-less flag in this dialect.
func (d Dialect) IsBool(k OptionKey) bool { return d.flags[k].boolean }

// geth114Common is the shared modern-geth surface (go-stablenet 179 flags /
// go-wbft 177 differ by two flags none of which chainbench sets; one table
// covers both — the measured result behind the two-dialect design).
func geth114Common() map[OptionKey]flagSpec {
	return map[OptionKey]flagSpec{
		KeyNodeKey:             {"--nodekey", false},
		KeyKeystore:            {"--keystore", false},
		KeyUnlock:              {"--unlock", false},
		KeyPassword:            {"--password", false},
		KeyAllowInsecureUnlock: {"--allow-insecure-unlock", true},
		KeyEtherbase:           {"--miner.etherbase", false},

		KeyDataDir:  {"--datadir", false},
		KeyConfig:   {"--config", false},
		KeySyncMode: {"--syncmode", false},
		KeyGCMode:   {"--gcmode", false},

		KeyPort:       {"--port", false},
		KeyBootnodes:  {"--bootnodes", false},
		KeyNoDiscover: {"--nodiscover", true},
		KeyMaxPeers:   {"--maxpeers", false},
		KeyNAT:        {"--nat", false},
		KeyNetworkID:  {"--networkid", false},

		KeyHTTP:           {"--http", true},
		KeyHTTPAddr:       {"--http.addr", false},
		KeyHTTPPort:       {"--http.port", false},
		KeyHTTPAPI:        {"--http.api", false},
		KeyHTTPVHosts:     {"--http.vhosts", false},
		KeyHTTPCorsDomain: {"--http.corsdomain", false},

		KeyWS:        {"--ws", true},
		KeyWSAddr:    {"--ws.addr", false},
		KeyWSPort:    {"--ws.port", false},
		KeyWSAPI:     {"--ws.api", false},
		KeyWSOrigins: {"--ws.origins", false},

		KeyAuthAddr:   {"--authrpc.addr", false},
		KeyAuthPort:   {"--authrpc.port", false},
		KeyIPCPath:    {"--ipcpath", false},
		KeyIPCDisable: {"--ipcdisable", true},

		KeyRPCDeprecatedPersonal: {"--rpc.enabledeprecatedpersonal", true},
		KeyRPCUnprotectedTxs:     {"--rpc.allow-unprotected-txs", true},
		KeyRPCGasCap:             {"--rpc.gascap", false},
		KeyRPCTxFeeCap:           {"--rpc.txfeecap", false},

		KeyMine:          {"--mine", true},
		KeyMinerGasLimit: {"--miner.gaslimit", false},
		KeyMinerGasPrice: {"--miner.gasprice", false},
		KeyMinerRecommit: {"--miner.recommit", false},

		KeyMetrics:     {"--metrics", true},
		KeyMetricsAddr: {"--metrics.addr", false},
		KeyMetricsPort: {"--metrics.port", false},

		// Spellings and value/boolean kinds below are read from the captured
		// binary surfaces, not from memory — docs/chain-analysis/*/cli-surface.txt.
		// TestDialectSpellingsExistInTheBinaries holds the table to them.
		KeyTxPoolLocals:       {"--txpool.locals", false},
		KeyTxPoolNoLocals:     {"--txpool.nolocals", true},
		KeyTxPoolJournal:      {"--txpool.journal", false},
		KeyTxPoolRejournal:    {"--txpool.rejournal", false},
		KeyTxPoolPriceLimit:   {"--txpool.pricelimit", false},
		KeyTxPoolPriceBump:    {"--txpool.pricebump", false},
		KeyTxPoolAccountSlots: {"--txpool.accountslots", false},
		KeyTxPoolGlobalSlots:  {"--txpool.globalslots", false},
		KeyTxPoolAccountQueue: {"--txpool.accountqueue", false},
		KeyTxPoolGlobalQueue:  {"--txpool.globalqueue", false},
		KeyTxPoolLifetime:     {"--txpool.lifetime", false},

		KeyCache:           {"--cache", false},
		KeyCacheDatabase:   {"--cache.database", false},
		KeyCacheTrie:       {"--cache.trie", false},
		KeyCacheGC:         {"--cache.gc", false},
		KeyCacheSnapshot:   {"--cache.snapshot", false},
		KeyCacheNoPrefetch: {"--cache.noprefetch", true},
		KeyCachePreimages:  {"--cache.preimages", true},

		KeyGPOBlocks:      {"--gpo.blocks", false},
		KeyGPOPercentile:  {"--gpo.percentile", false},
		KeyGPOMaxPrice:    {"--gpo.maxprice", false},
		KeyGPOIgnorePrice: {"--gpo.ignoreprice", false},

		KeySnapshot:       {"--snapshot", true},
		KeyDataDirAncient: {"--datadir.ancient", false},
		KeyTxLookupLimit:  {"--txlookuplimit", false},

		KeyMaxPendPeers: {"--maxpendpeers", false},
		KeyNetRestrict:  {"--netrestrict", false},
		KeyDiscoveryDNS: {"--discovery.dns", false},

		KeyDev:         {"--dev", true},
		KeyDevPeriod:   {"--dev.period", false},
		KeyDevGasLimit: {"--dev.gaslimit", false},

		KeyMinerExtraData: {"--miner.extradata", false},

		// Present in the geth114 generation only; removed for wemix below.
		KeyHistoryState:        {"--history.state", false},
		KeyHistoryTransactions: {"--history.transactions", false},
		KeyCacheBlockLogs:      {"--cache.blocklogs", false},
		KeyDiscoveryPort:       {"--discovery.port", false},
	}
}

// Geth114 is the dialect of go-stablenet and go-wbft.
func Geth114() Dialect {
	return Dialect{ID: "geth114", flags: geth114Common()}
}

// Geth110Wemix is the dialect of go-wemix: the modern surface minus the flags
// its older generation lacks, plus the wemix consensus knobs. Spellings are
// AST-verified against docs/chain-analysis/gwemix/cli-graph.md.
func Geth110Wemix() Dialect {
	f := geth114Common()
	// Not present in the go-wemix generation. Each deletion is a measured
	// absence from docs/chain-analysis/gwemix/cli-flags.txt, not a guess.
	delete(f, KeyRPCDeprecatedPersonal)
	delete(f, KeyHistoryState)
	delete(f, KeyHistoryTransactions)
	delete(f, KeyCacheBlockLogs)
	delete(f, KeyDiscoveryPort)
	// wemix consensus knobs (ChainExt module).
	f[KeyConsensusMethod] = flagSpec{"--consensusmethod", false}
	f[KeyBlocksPerTurn] = flagSpec{"--blocksperturn", false}
	f[KeyNonceLimit] = flagSpec{"--noncelimit", false}
	f[KeyMaxTxsPerBlock] = flagSpec{"--maxtxsperblock", false}
	f[KeyBlockInterval] = flagSpec{"--wemix.block.interval", false}
	f[KeyBlockTimeAdj] = flagSpec{"--wemix.block.timeadjblocks", false}
	f[KeyBlockMinBuildTime] = flagSpec{"--wemix.block.minbuildtime", false}
	f[KeyBlockMinBuildTxs] = flagSpec{"--wemix.block.minbuildtxs", false}
	f[KeyBlockTrailTime] = flagSpec{"--wemix.block.trailtime", false}
	f[KeyBootnodeCount] = flagSpec{"--wemix.bootnodecount", false}
	f[KeyMaxIdleBlock] = flagSpec{"--maxidleblockinterval", false}
	f[KeyFixedDifficulty] = flagSpec{"--fixeddifficulty", false}
	f[KeyFixedGasLimit] = flagSpec{"--fixedgaslimit", false}
	f[KeyMinerGasTarget] = flagSpec{"--miner.gastarget", false}
	return Dialect{ID: "geth110-wemix", flags: f}
}

// dialects are the flag vocabularies this build knows, by the id a chain
// manifest names. Which one a chain speaks is the chain's answer; what each one
// contains is this package's, and that split is the point: the mapping used to
// be an `if chainID == "wemix"` here, so a chain on the older generation under
// any other name silently got the modern vocabulary.
var dialects = map[string]func() Dialect{
	"geth114":       Geth114,
	"geth110-wemix": Geth110Wemix,
}

// DialectFor returns the flag vocabulary named by a manifest's dialect field.
//
// An unknown name is an error rather than a fallback. A fallback here is a node
// that launches with flags its binary does not have and dies at boot saying so
// about the flag, which tells nobody that a manifest named a vocabulary this
// build does not carry.
func DialectFor(name string) (Dialect, error) {
	if f, ok := dialects[name]; ok {
		return f(), nil
	}
	return Dialect{}, fmt.Errorf("launchopt: unknown dialect %q (this build has %s)", name, strings.Join(DialectNames(), ", "))
}

// DialectNames returns the sorted ids of the vocabularies this build carries.
func DialectNames() []string {
	names := make([]string, 0, len(dialects))
	for n := range dialects {
		names = append(names, n)
	}
	sort.Strings(names)
	return names
}

// perNode are the knobs whose value the assembler derives from ONE node's own
// facts: where its data lives, which ports it holds, which files it opens, which
// account it seals with.
//
// They are listed because one value cannot serve several nodes. Two nodes told
// the same p2p port do not both bind it, two told the same datadir write over
// each other, and two told the same keystore seal as one account. A knob set on
// a scope that covers more than one node is therefore refused rather than
// applied — see node.ScopeIndex for what "one node" means.
//
// The rest of what this layer sets is not here on purpose. --http and --ws turn
// an endpoint on; every node may have one, and saying so once is what a scope is
// for.
var perNode = map[OptionKey]bool{
	KeyPort:      true,
	KeyHTTPPort:  true,
	KeyWSPort:    true,
	KeyAuthPort:  true,
	KeyDataDir:   true,
	KeyConfig:    true,
	KeyNodeKey:   true,
	KeyKeystore:  true,
	KeyPassword:  true,
	KeyUnlock:    true,
	KeyEtherbase: true,
}

// IsPerNode reports whether k is a knob only one node can be told.
func IsPerNode(k OptionKey) bool { return perNode[k] }

// PerNodeKeys lists them, sorted, for a refusal that has to name what it means.
func PerNodeKeys() []string {
	out := make([]string, 0, len(perNode))
	for k := range perNode {
		out = append(out, string(k))
	}
	sort.Strings(out)
	return out
}
