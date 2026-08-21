// Package deploy models a multi-server remote wemix+etcd deployment: the
// declarative server list (1..N), roles, and per-node resolution used to drive
// an SSH-based deploy of a go-wemix -> go-wbft (Croissant) hardfork chain onto a
// closed network. It ports the wemix4 env.conf / node_env.json config into an
// explicit, chainbench-native model (docs/REMOTE_WEMIX_DEPLOY_DESIGN.md).
//
// Phases 1-5 are here: the config + cluster model (this file), the remote key
// read (keys.go, credentials.go), the provision+launch orchestration (plan.go,
// orchestrate.go), the governance+etcd bootstrap on the boot producer
// (bootstrap.go, accounts.go), and the hardfork handoff confirmation
// (handoff.go). The remaining work is porting the wemix4 test cases.
package deploy

import (
	"fmt"
	"os"

	"go.yaml.in/yaml/v3"
)

// Role is a server's role in the cluster.
type Role string

const (
	RoleWemixBP  Role = "wemix_bp" // pre-fork wpoa producer (runs the wemix binary)
	RoleWbftBP   Role = "wbft_bp"  // post-fork WBFT validator (runs the wbft binary)
	RoleEndpoint Role = "en"       // non-producing RPC endpoint
	RoleBootnode Role = "pn"       // bootnode / peer node
)

func (r Role) valid() bool {
	switch r {
	case RoleWemixBP, RoleWbftBP, RoleEndpoint, RoleBootnode:
		return true
	}
	return false
}

// Server is one remote host in the cluster. Unlike wemix4 (which couples the
// server number to the IP's last octet), identity is explicit.
type Server struct {
	Index    int    `yaml:"index"`
	Host     string `yaml:"host"`
	SSHPort  int    `yaml:"ssh_port,omitempty"` // 0 -> cluster default
	User     string `yaml:"user,omitempty"`     // "" -> credentials default
	Role     Role   `yaml:"role"`
	Binary   string `yaml:"binary,omitempty"`    // "" -> derived from role
	SyncMode string `yaml:"sync_mode,omitempty"` // "" -> "full"
}

// Cluster is a full remote-deployment config (cluster.yaml, copied from
// cluster.yaml.sample). It is chainbench's equivalent of wemix4's env.conf +
// node_env.json. SSH auth (user/password/key) and account/keystore material live
// in separate, gitignored files — never here.
type Cluster struct {
	RPCPort          int         `yaml:"rpc_port"`
	WSPort           int         `yaml:"ws_port"`
	P2PPort          int         `yaml:"p2p_port"`  // devp2p port (default 30303)
	DataRoot         string      `yaml:"data_root"` // remote node datadir (default /data/go-wbft)
	CroissantBlock   int64       `yaml:"croissant_block"`
	EpochLength      int         `yaml:"epoch_length"`
	TargetValidators int         `yaml:"target_validators"`
	GenesisFile      string      `yaml:"genesis_file"` // remote path to the genesis
	WemixBinary      string      `yaml:"wemix_binary"` // remote path, pre-fork producer
	WbftBinary       string      `yaml:"wbft_binary"`  // remote path, post-fork validator
	SSHPort          int         `yaml:"ssh_port"`     // default SSH port for all servers
	RemotePaths      RemotePaths `yaml:"remote_paths"` // where keys live on each server
	Servers          []Server    `yaml:"servers"`
}

// RemotePaths are the fixed on-server locations the key-read reads from. Empty
// fields fall back to the wemix4 defaults (see DefaultRemotePaths).
//
// A `bootnode:` key in an existing cluster file is accepted and ignored: the
// servers no longer need that tool, because identity is derived here from the
// nodekey rather than by running a binary on the host.
type RemotePaths struct {
	Nodekey          string `yaml:"nodekey"`           // the node's devp2p private key
	CoinbaseKeystore string `yaml:"coinbase_keystore"` // validator coinbase keystore
	OperatorKeystore string `yaml:"operator_keystore"` // operator keystore
}

// DefaultRemotePaths mirrors the wemix4 closed-network layout.
func DefaultRemotePaths() RemotePaths {
	return RemotePaths{
		Nodekey:          "/data/go-wbft/conf/nodekey",
		CoinbaseKeystore: "/data/go-wbft/conf/keystore/coinbase",
		OperatorKeystore: "/data/go-wbft/conf/keystore/operator",
	}
}

// Paths returns the cluster's remote paths with defaults applied for any empty
// field.
func (c *Cluster) Paths() RemotePaths {
	d := DefaultRemotePaths()
	p := c.RemotePaths
	if p.Nodekey == "" {
		p.Nodekey = d.Nodekey
	}
	if p.CoinbaseKeystore == "" {
		p.CoinbaseKeystore = d.CoinbaseKeystore
	}
	if p.OperatorKeystore == "" {
		p.OperatorKeystore = d.OperatorKeystore
	}
	return p
}

// LoadCluster reads and validates a cluster config from path.
func LoadCluster(path string) (*Cluster, error) {
	b, err := os.ReadFile(path)
	if err != nil {
		return nil, fmt.Errorf("deploy: read cluster config: %w", err)
	}
	c, err := ParseCluster(b)
	if err != nil {
		return nil, err
	}
	return c, nil
}

// ParseCluster decodes and validates a cluster config from YAML bytes.
func ParseCluster(b []byte) (*Cluster, error) {
	var c Cluster
	if err := yaml.Unmarshal(b, &c); err != nil {
		return nil, fmt.Errorf("deploy: parse cluster config: %w", err)
	}
	if err := c.validate(); err != nil {
		return nil, err
	}
	return &c, nil
}

func (c *Cluster) validate() error {
	if len(c.Servers) == 0 {
		return fmt.Errorf("deploy: cluster has no servers")
	}
	if c.RPCPort <= 0 {
		return fmt.Errorf("deploy: rpc_port must be set")
	}
	seen := map[int]bool{}
	for i, s := range c.Servers {
		if s.Index <= 0 {
			return fmt.Errorf("deploy: server[%d] missing/invalid index", i)
		}
		if seen[s.Index] {
			return fmt.Errorf("deploy: duplicate server index %d", s.Index)
		}
		seen[s.Index] = true
		if s.Host == "" {
			return fmt.Errorf("deploy: server index %d missing host", s.Index)
		}
		if !s.Role.valid() {
			return fmt.Errorf("deploy: server index %d has invalid role %q (want wemix_bp|wbft_bp|en|pn)", s.Index, s.Role)
		}
	}
	return nil
}

// SSHPortFor returns the effective SSH port for s (server override, else cluster
// default, else 22).
func (c *Cluster) SSHPortFor(s Server) int {
	if s.SSHPort > 0 {
		return s.SSHPort
	}
	if c.SSHPort > 0 {
		return c.SSHPort
	}
	return 22
}

// BinaryFor returns the remote node binary path for s: an explicit override,
// else the role default (wemix_bp -> WemixBinary, everything else -> WbftBinary).
func (c *Cluster) BinaryFor(s Server) string {
	if s.Binary != "" {
		return s.Binary
	}
	if s.Role == RoleWemixBP {
		return c.WemixBinary
	}
	return c.WbftBinary
}

// SyncModeFor returns s's sync mode (default "full").
func (c *Cluster) SyncModeFor(s Server) string {
	if s.SyncMode != "" {
		return s.SyncMode
	}
	return "full"
}

// RPCURL returns the JSON-RPC URL for a server.
func (c *Cluster) RPCURL(s Server) string {
	return fmt.Sprintf("http://%s:%d", s.Host, c.RPCPort)
}

// WSURL returns the WebSocket URL for a server.
func (c *Cluster) WSURL(s Server) string {
	return fmt.Sprintf("ws://%s:%d", s.Host, c.WSPort)
}

// ByRole returns the servers with the given role, in config order.
func (c *Cluster) ByRole(role Role) []Server {
	var out []Server
	for _, s := range c.Servers {
		if s.Role == role {
			out = append(out, s)
		}
	}
	return out
}

// Producers/Validators/Endpoints/Bootnodes are role shortcuts.
func (c *Cluster) Producers() []Server  { return c.ByRole(RoleWemixBP) }
func (c *Cluster) Validators() []Server { return c.ByRole(RoleWbftBP) }
func (c *Cluster) Endpoints() []Server  { return c.ByRole(RoleEndpoint) }
func (c *Cluster) Bootnodes() []Server  { return c.ByRole(RoleBootnode) }

// ServerByIndex returns the server with the given index, or false.
func (c *Cluster) ServerByIndex(index int) (Server, bool) {
	for _, s := range c.Servers {
		if s.Index == index {
			return s, true
		}
	}
	return Server{}, false
}

// LaunchOrder returns servers in the order to START them: endpoints and
// bootnodes before producers/validators, so producers find peers on boot (this
// mirrors wemix4, which launches ENs/bootnode first).
func (c *Cluster) LaunchOrder() []Server {
	var first, last []Server
	for _, s := range c.Servers {
		switch s.Role {
		case RoleWemixBP, RoleWbftBP:
			last = append(last, s)
		default:
			first = append(first, s)
		}
	}
	return append(first, last...)
}
