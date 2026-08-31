package nodeconfig

import (
	"fmt"
	"slices"
	"strconv"
)

// Module is one launch concern. Pure: no I/O, no globals. A module emits only
// the knobs its fields set, checks only its own invariants, and stays
// chain-agnostic — generation differences live in the Dialect.
type Module interface {
	// Name identifies the module in errors.
	Name() string
	// Apply validates the module's own invariants and records its knobs.
	Apply(a *Args) error
}

// unixSocketPathMax is the kernel limit on a unix socket path; an ipcpath at or
// beyond it fails at bind time with a message that hides the cause (hit for
// real with a deeply nested session directory).
const unixSocketPathMax = 104

// Identity owns who the node is: its devp2p key and the account it signs with.
type Identity struct {
	NodeKeyFile         string // --nodekey
	KeystoreDir         string // --keystore
	Unlock              string // --unlock (0x address)
	PasswordFile        string // --password
	AllowInsecureUnlock bool
	Etherbase           string // --miner.etherbase
}

func (Identity) Name() string { return "identity" }

func (m Identity) Apply(a *Args) error {
	if m.Unlock != "" && m.PasswordFile == "" {
		return fmt.Errorf("identity: unlock %s needs a password file", m.Unlock)
	}
	if m.NodeKeyFile != "" {
		a.Set(KeyNodeKey, m.NodeKeyFile, LayerRole)
	}
	if m.KeystoreDir != "" {
		a.Set(KeyKeystore, m.KeystoreDir, LayerRole)
	}
	if m.Unlock != "" {
		a.Set(KeyUnlock, m.Unlock, LayerRole)
		a.Set(KeyPassword, m.PasswordFile, LayerRole)
	}
	if m.AllowInsecureUnlock {
		a.Enable(KeyAllowInsecureUnlock, LayerFamily)
	}
	if m.Etherbase != "" {
		a.Set(KeyEtherbase, m.Etherbase, LayerRole)
	}
	return nil
}

// Storage owns where the node keeps its state.
type Storage struct {
	DataDir    string // --datadir, required
	ConfigFile string // --config
	SyncMode   string // --syncmode
	GCMode     string // --gcmode
}

func (Storage) Name() string { return "storage" }

func (m Storage) Apply(a *Args) error {
	// DataDir may be relative: the engine roots node datadirs in the session
	// tree, which itself defaults to a CWD-relative artifact root. The binary
	// resolves it against its own CWD, which the driver controls.
	if m.DataDir == "" {
		return fmt.Errorf("storage: datadir is required")
	}
	a.Set(KeyDataDir, m.DataDir, LayerRole)
	if m.ConfigFile != "" {
		a.Set(KeyConfig, m.ConfigFile, LayerRole)
	}
	if m.SyncMode != "" {
		a.Set(KeySyncMode, m.SyncMode, LayerEnv)
	}
	if m.GCMode != "" {
		a.Set(KeyGCMode, m.GCMode, LayerEnv)
	}
	return nil
}

// P2P owns the devp2p surface.
type P2P struct {
	Port       int    // --port, required when the module is used
	Bootnodes  string // --bootnodes
	NoDiscover bool   // --nodiscover
	MaxPeers   int    // --maxpeers, 0 = binary default
	NAT        string // --nat
	NetworkID  int64  // --networkid, 0 = binary default/derived
}

func (P2P) Name() string { return "p2p" }

func (m P2P) Apply(a *Args) error {
	if m.Port <= 0 {
		return fmt.Errorf("p2p: port must be positive, got %d", m.Port)
	}
	a.Set(KeyPort, strconv.Itoa(m.Port), LayerRole)
	if m.Bootnodes != "" {
		a.Set(KeyBootnodes, m.Bootnodes, LayerEnv)
	}
	if m.NoDiscover {
		a.Enable(KeyNoDiscover, LayerEnv)
	}
	if m.MaxPeers > 0 {
		a.Set(KeyMaxPeers, strconv.Itoa(m.MaxPeers), LayerEnv)
	}
	if m.NAT != "" {
		a.Set(KeyNAT, m.NAT, LayerEnv)
	}
	if m.NetworkID != 0 {
		a.Set(KeyNetworkID, strconv.FormatInt(m.NetworkID, 10), LayerEnv)
	}
	return nil
}

// HTTPRPC owns the HTTP JSON-RPC endpoint.
type HTTPRPC struct {
	Enabled bool
	Addr    string // --http.addr
	Port    int    // --http.port
	API     string // --http.api
	VHosts  string // --http.vhosts
	Cors    string // --http.corsdomain
}

func (HTTPRPC) Name() string { return "httprpc" }

func (m HTTPRPC) Apply(a *Args) error {
	if !m.Enabled {
		if m.Port != 0 || m.API != "" || m.Addr != "" {
			return fmt.Errorf("httprpc: endpoint values set but the endpoint is disabled")
		}
		return nil
	}
	a.Enable(KeyHTTP, LayerRole)
	if m.Addr != "" {
		a.Set(KeyHTTPAddr, m.Addr, LayerEnv)
	}
	if m.Port != 0 {
		a.Set(KeyHTTPPort, strconv.Itoa(m.Port), LayerRole)
	}
	if m.API != "" {
		a.Set(KeyHTTPAPI, m.API, LayerEnv)
	}
	if m.VHosts != "" {
		a.Set(KeyHTTPVHosts, m.VHosts, LayerEnv)
	}
	if m.Cors != "" {
		a.Set(KeyHTTPCorsDomain, m.Cors, LayerEnv)
	}
	return nil
}

// WSRPC owns the WebSocket JSON-RPC endpoint.
type WSRPC struct {
	Enabled bool
	Addr    string
	Port    int
	API     string
	Origins string
}

func (WSRPC) Name() string { return "wsrpc" }

func (m WSRPC) Apply(a *Args) error {
	if !m.Enabled {
		if m.Port != 0 || m.API != "" || m.Addr != "" {
			return fmt.Errorf("wsrpc: endpoint values set but the endpoint is disabled")
		}
		return nil
	}
	a.Enable(KeyWS, LayerRole)
	if m.Addr != "" {
		a.Set(KeyWSAddr, m.Addr, LayerEnv)
	}
	if m.Port != 0 {
		a.Set(KeyWSPort, strconv.Itoa(m.Port), LayerRole)
	}
	if m.API != "" {
		a.Set(KeyWSAPI, m.API, LayerEnv)
	}
	if m.Origins != "" {
		a.Set(KeyWSOrigins, m.Origins, LayerEnv)
	}
	return nil
}

// AuthIPC owns the engine-API and IPC endpoints.
type AuthIPC struct {
	AuthAddr   string
	AuthPort   int
	IPCPath    string
	IPCDisable bool
}

func (AuthIPC) Name() string { return "authipc" }

func (m AuthIPC) Apply(a *Args) error {
	if len(m.IPCPath) >= unixSocketPathMax {
		return fmt.Errorf("authipc: ipcpath %q is %d chars; unix sockets cap at %d",
			m.IPCPath, len(m.IPCPath), unixSocketPathMax)
	}
	if m.AuthAddr != "" {
		a.Set(KeyAuthAddr, m.AuthAddr, LayerEnv)
	}
	if m.AuthPort != 0 {
		a.Set(KeyAuthPort, strconv.Itoa(m.AuthPort), LayerRole)
	}
	if m.IPCPath != "" {
		a.Set(KeyIPCPath, m.IPCPath, LayerEnv)
	}
	if m.IPCDisable {
		a.Enable(KeyIPCDisable, LayerEnv)
	}
	return nil
}

// RPCPolicy owns the RPC permissiveness knobs chainbench's dev flows rely on.
type RPCPolicy struct {
	DeprecatedPersonal bool // harmless absence on geth110-wemix (still present there)
	UnprotectedTxs     bool
	GasCap             string
	TxFeeCap           string
}

func (RPCPolicy) Name() string { return "rpcpolicy" }

func (m RPCPolicy) Apply(a *Args) error {
	if m.DeprecatedPersonal {
		// go-wemix's generation predates the personal-namespace deprecation, so
		// the flag's absence there is the "harmless absence" branch: the
		// behavior the flag would enable is already the default.
		a.EnableIfSupported(KeyRPCDeprecatedPersonal, LayerFamily)
	}
	if m.UnprotectedTxs {
		a.Enable(KeyRPCUnprotectedTxs, LayerFamily)
	}
	if m.GasCap != "" {
		a.Set(KeyRPCGasCap, m.GasCap, LayerEnv)
	}
	if m.TxFeeCap != "" {
		a.Set(KeyRPCTxFeeCap, m.TxFeeCap, LayerEnv)
	}
	return nil
}

// Mining owns block production.
type Mining struct {
	Mine     bool
	GasLimit string
	GasPrice string
	Recommit string
}

func (Mining) Name() string { return "mining" }

func (m Mining) Apply(a *Args) error {
	if m.Mine {
		a.Enable(KeyMine, LayerRole)
	}
	if m.GasLimit != "" {
		a.Set(KeyMinerGasLimit, m.GasLimit, LayerEnv)
	}
	if m.GasPrice != "" {
		a.Set(KeyMinerGasPrice, m.GasPrice, LayerEnv)
	}
	if m.Recommit != "" {
		a.Set(KeyMinerRecommit, m.Recommit, LayerEnv)
	}
	return nil
}

// Metrics owns the metrics endpoint.
type Metrics struct {
	Enabled bool
	Addr    string
	Port    int
}

func (Metrics) Name() string { return "metrics" }

func (m Metrics) Apply(a *Args) error {
	if !m.Enabled {
		if m.Port != 0 || m.Addr != "" {
			// The defect class flag-graph §2.1-3 documents: a metrics port
			// without --metrics silently serves nothing.
			return fmt.Errorf("metrics: addr/port set but metrics not enabled")
		}
		return nil
	}
	a.Enable(KeyMetrics, LayerEnv)
	if m.Addr != "" {
		a.Set(KeyMetricsAddr, m.Addr, LayerEnv)
	}
	if m.Port != 0 {
		a.Set(KeyMetricsPort, strconv.Itoa(m.Port), LayerEnv)
	}
	return nil
}

// ChainExt owns generation-specific consensus knobs. Requesting one on a
// dialect that lacks it is an explicit error — these change consensus
// behavior, so a silent skip would run a materially different chain.
type ChainExt struct {
	Values map[Key]string
}

func (ChainExt) Name() string { return "chainext" }

func (m ChainExt) Apply(a *Args) error {
	for _, k := range chainExtOrder {
		if v, ok := m.Values[k]; ok {
			a.Set(k, v, LayerEnv)
		}
	}
	for k := range m.Values {
		if !isChainExtKey(k) {
			return fmt.Errorf("chainext: %q is not a chain-extension knob", k)
		}
	}
	return nil
}

// chainExtOrder fixes emission order for deterministic argv.
var chainExtOrder = []Key{
	KeyConsensusMethod, KeyBlocksPerTurn, KeyNonceLimit, KeyMaxTxsPerBlock,
	KeyBlockInterval, KeyBlockTimeAdj, KeyBlockMinBuildTime, KeyBlockMinBuildTxs,
}

func isChainExtKey(k Key) bool { return slices.Contains(chainExtOrder, k) }
