// Package launchopt is the single assembly point for a node's launch command
// line (background requirement #2, algorithm step 7). It replaces the five
// scattered argv sites measured in docs/dev/architecture/code-graph.md §3 with
// one Builder over ten concern modules.
//
// The design (docs/dev/chain-binary-flag-graph.md §3.3) rests on one measured
// fact: the three chain binaries expose two flag generations, not three, so a
// Dialect — the flag vocabulary of one binary generation — is the only place
// that knows a binary's spelling. Option modules stay chain-agnostic and speak
// typed Keys; the Dialect translates or reports the knob unsupported.
//
// Everything here is a pure function of its inputs: no I/O, no globals, fully
// unit-testable.
package launchopt

// Key is the chain-agnostic name of one launch knob. Typed so a knob is never
// a magic string; the Dialect maps it to (or refuses) a concrete flag.
type Key string

// The vocabulary. Grouped by owning module (see modules.go). Spellings on the
// right are the geth114 dialect; deviations live in the dialect tables.
const (
	// Identity
	KeyNodeKey             Key = "nodekey"               // --nodekey <file>
	KeyKeystore            Key = "keystore"              // --keystore <dir>
	KeyUnlock              Key = "unlock"                // --unlock <addr>
	KeyPassword            Key = "password"              // --password <file>
	KeyAllowInsecureUnlock Key = "allow-insecure-unlock" // --allow-insecure-unlock
	KeyEtherbase           Key = "miner.etherbase"       // --miner.etherbase <addr>

	// Storage
	KeyDataDir  Key = "datadir"  // --datadir <dir>
	KeyConfig   Key = "config"   // --config <file>
	KeySyncMode Key = "syncmode" // --syncmode full|snap
	KeyGCMode   Key = "gcmode"   // --gcmode full|archive

	// P2P
	KeyPort       Key = "port"       // --port <n>
	KeyBootnodes  Key = "bootnodes"  // --bootnodes <enodes>
	KeyNoDiscover Key = "nodiscover" // --nodiscover
	KeyMaxPeers   Key = "maxpeers"   // --maxpeers <n>
	KeyNAT        Key = "nat"        // --nat none|any|...
	KeyNetworkID  Key = "networkid"  // --networkid <n>

	// HTTPRPC
	KeyHTTP           Key = "http"            // --http
	KeyHTTPAddr       Key = "http.addr"       // --http.addr <ip>
	KeyHTTPPort       Key = "http.port"       // --http.port <n>
	KeyHTTPAPI        Key = "http.api"        // --http.api <list>
	KeyHTTPVHosts     Key = "http.vhosts"     // --http.vhosts <list>
	KeyHTTPCorsDomain Key = "http.corsdomain" // --http.corsdomain <list>

	// WSRPC
	KeyWS        Key = "ws"         // --ws
	KeyWSAddr    Key = "ws.addr"    // --ws.addr <ip>
	KeyWSPort    Key = "ws.port"    // --ws.port <n>
	KeyWSAPI     Key = "ws.api"     // --ws.api <list>
	KeyWSOrigins Key = "ws.origins" // --ws.origins <list>

	// AuthIPC
	KeyAuthAddr   Key = "authrpc.addr" // --authrpc.addr <ip>
	KeyAuthPort   Key = "authrpc.port" // --authrpc.port <n>
	KeyIPCPath    Key = "ipcpath"      // --ipcpath <path>
	KeyIPCDisable Key = "ipcdisable"   // --ipcdisable

	// RPCPolicy
	KeyRPCDeprecatedPersonal Key = "rpc.enabledeprecatedpersonal" // --rpc.enabledeprecatedpersonal
	KeyRPCUnprotectedTxs     Key = "rpc.allow-unprotected-txs"    // --rpc.allow-unprotected-txs
	KeyRPCGasCap             Key = "rpc.gascap"                   // --rpc.gascap <n>
	KeyRPCTxFeeCap           Key = "rpc.txfeecap"                 // --rpc.txfeecap <n>

	// Mining
	KeyMine          Key = "mine"           // --mine
	KeyMinerGasLimit Key = "miner.gaslimit" // --miner.gaslimit <n>
	KeyMinerGasPrice Key = "miner.gasprice" // --miner.gasprice <n>
	KeyMinerRecommit Key = "miner.recommit" // --miner.recommit <dur|nanos>

	// Metrics
	KeyMetrics     Key = "metrics"      // --metrics
	KeyMetricsAddr Key = "metrics.addr" // --metrics.addr <ip>
	KeyMetricsPort Key = "metrics.port" // --metrics.port <n>

	// Txpool — mempool admission and retention. These decide whether a
	// transaction a test submits is kept, replaced or dropped, so a test that
	// exercises nonce gaps or replacement needs to say what the pool does
	// rather than hope the default suits it.
	KeyTxPoolLocals       Key = "txpool.locals"       // --txpool.locals <addrs>
	KeyTxPoolNoLocals     Key = "txpool.nolocals"     // --txpool.nolocals
	KeyTxPoolJournal      Key = "txpool.journal"      // --txpool.journal <path>
	KeyTxPoolRejournal    Key = "txpool.rejournal"    // --txpool.rejournal <dur>
	KeyTxPoolPriceLimit   Key = "txpool.pricelimit"   // --txpool.pricelimit <n>
	KeyTxPoolPriceBump    Key = "txpool.pricebump"    // --txpool.pricebump <n>
	KeyTxPoolAccountSlots Key = "txpool.accountslots" // --txpool.accountslots <n>
	KeyTxPoolGlobalSlots  Key = "txpool.globalslots"  // --txpool.globalslots <n>
	KeyTxPoolAccountQueue Key = "txpool.accountqueue" // --txpool.accountqueue <n>
	KeyTxPoolGlobalQueue  Key = "txpool.globalqueue"  // --txpool.globalqueue <n>
	KeyTxPoolLifetime     Key = "txpool.lifetime"     // --txpool.lifetime <dur>

	// Cache — memory split across database, trie, pruning and snapshots. A
	// sync or pruning test that does not set these is measuring the default.
	KeyCache           Key = "cache"            // --cache <mb>
	KeyCacheDatabase   Key = "cache.database"   // --cache.database <pct>
	KeyCacheTrie       Key = "cache.trie"       // --cache.trie <pct>
	KeyCacheGC         Key = "cache.gc"         // --cache.gc <pct>
	KeyCacheSnapshot   Key = "cache.snapshot"   // --cache.snapshot <pct>
	KeyCacheNoPrefetch Key = "cache.noprefetch" // --cache.noprefetch
	KeyCachePreimages  Key = "cache.preimages"  // --cache.preimages

	// GPO — the gas price oracle behind eth_gasPrice and eth_maxPriorityFee.
	// The gas-policy suite asserts on those answers, and they come from here.
	KeyGPOBlocks      Key = "gpo.blocks"      // --gpo.blocks <n>
	KeyGPOPercentile  Key = "gpo.percentile"  // --gpo.percentile <n>
	KeyGPOMaxPrice    Key = "gpo.maxprice"    // --gpo.maxprice <wei>
	KeyGPOIgnorePrice Key = "gpo.ignoreprice" // --gpo.ignoreprice <wei>

	// State — what the node keeps and for how long. Archive-vs-pruned changes
	// which historical reads answer at all.
	KeySnapshot       Key = "snapshot"        // --snapshot
	KeyDataDirAncient Key = "datadir.ancient" // --datadir.ancient <path>
	KeyTxLookupLimit  Key = "txlookuplimit"   // --txlookuplimit <n>

	// Peering shape beyond the port: how many pending peers, which networks
	// may connect, where discovery looks.
	KeyMaxPendPeers  Key = "maxpendpeers"   // --maxpendpeers <n>
	KeyNetRestrict   Key = "netrestrict"    // --netrestrict <cidr>
	KeyDiscoveryDNS  Key = "discovery.dns"  // --discovery.dns <url>
	KeyDiscoveryPort Key = "discovery.port" // geth114 --discovery.port <n>

	// Dev mode — a single-node chain that seals on demand. Useful for tests
	// that need a chain and not a network.
	KeyDev         Key = "dev"          // --dev
	KeyDevPeriod   Key = "dev.period"   // --dev.period <s>
	KeyDevGasLimit Key = "dev.gaslimit" // --dev.gaslimit <n>

	// Miner extras beyond the ones the Mining module already owns.
	KeyMinerExtraData Key = "miner.extradata" // --miner.extradata <bytes>

	// History pruning (geth114 generation only).
	KeyHistoryState        Key = "history.state"        // --history.state <n>
	KeyHistoryTransactions Key = "history.transactions" // --history.transactions <n>
	KeyCacheBlockLogs      Key = "cache.blocklogs"      // --cache.blocklogs <n>

	// ChainExt — generation-specific consensus knobs. geth114 has none of
	// these; requesting one there is a classified error, never a silent skip.
	KeyConsensusMethod   Key = "chain.consensusmethod"    // gwemix --consensusmethod
	KeyBlocksPerTurn     Key = "chain.blocksperturn"      // gwemix --blocksperturn
	KeyNonceLimit        Key = "chain.noncelimit"         // gwemix --noncelimit
	KeyMaxTxsPerBlock    Key = "chain.maxtxsperblock"     // gwemix --maxtxsperblock
	KeyBlockInterval     Key = "chain.block.interval"     // gwemix --wemix.block.interval
	KeyBlockTimeAdj      Key = "chain.block.timeadj"      // gwemix --wemix.block.timeadjblocks
	KeyBlockMinBuildTime Key = "chain.block.minbuildtime" // gwemix --wemix.block.minbuildtime
	KeyBlockMinBuildTxs  Key = "chain.block.minbuildtxs"  // gwemix --wemix.block.minbuildtxs
	KeyBlockTrailTime    Key = "chain.block.trailtime"    // gwemix --wemix.block.trailtime
	KeyBootnodeCount     Key = "chain.bootnodecount"      // gwemix --wemix.bootnodecount
	KeyMaxIdleBlock      Key = "chain.maxidleblock"       // gwemix --maxidleblockinterval
	KeyFixedDifficulty   Key = "chain.fixeddifficulty"    // gwemix --fixeddifficulty
	KeyFixedGasLimit     Key = "chain.fixedgaslimit"      // gwemix --fixedgaslimit
	KeyMinerGasTarget    Key = "miner.gastarget"          // gwemix --miner.gastarget
)

// Layer names one precedence level of the value stack. A later layer setting
// the same Key wins; Args records the winner so `net status` can show which
// layer produced each flag (mirrors config's flag > file > default rule).
type Layer string

const (
	LayerFamily Layer = "family"     // consensus-family defaults
	LayerRole   Layer = "role"       // role-derived (validator mines, ...)
	LayerEnv    Layer = "env.launch" // DSL environment launch block
	LayerCase   Layer = "case"       // per-test-case override — always wins
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
	flags map[Key]flagSpec
}

// Spelling returns the dialect's flag name for k, or ok=false when the
// generation does not have the knob.
func (d Dialect) Spelling(k Key) (string, bool) {
	s, ok := d.flags[k]
	return s.name, ok
}

// IsBool reports whether k is a value-less flag in this dialect.
func (d Dialect) IsBool(k Key) bool { return d.flags[k].boolean }

// geth114Common is the shared modern-geth surface (go-stablenet 179 flags /
// go-wbft 177 differ by two flags none of which chainbench sets; one table
// covers both — the measured result behind the two-dialect design).
func geth114Common() map[Key]flagSpec {
	return map[Key]flagSpec{
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

// DialectFor selects the dialect for a chain manifest's binary generation.
// The mapping is a measured fact (flag-graph §1.1): gstable and gwbft share
// one surface; gwemix is the older generation.
func DialectFor(chainID string) Dialect {
	if chainID == "wemix" {
		return Geth110Wemix()
	}
	return Geth114()
}
