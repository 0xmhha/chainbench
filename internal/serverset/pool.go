package serverset

import (
	"fmt"

	"github.com/0xmhha/chainbench/internal/core/netmap"
)

// validatePool checks a v2 (pooled) inventory. The pool's own consistency
// rules — spacing, duplicates — live in netmap, which owns the assignment;
// this only checks the file said what a pool needs said.
func (c *Config) validatePool() error {
	if len(c.Servers) > 0 {
		return fmt.Errorf("serverset: %s mixes a v2 pool with a v1 server list; pick one", c.path)
	}
	if c.Pool == nil {
		return fmt.Errorf("serverset: %s has version %d but no pool block", c.path, PoolVersion)
	}
	if err := c.nodePool().Validate(); err != nil {
		return fmt.Errorf("%v (in %s)", err, c.path)
	}
	return nil
}

// nodePool renders the file's pool block as netmap's type.
func (c *Config) nodePool() netmap.Pool {
	return netmap.Pool{Hosts: c.Pool.Hosts, PortBases: c.Pool.PortBases, DataRoot: c.DataRoot}
}

// NodePool returns the assignable pool of a v2 inventory. A v1 file has no
// pool — its servers drive the older placement path — and asking is an error
// that says how to migrate rather than an invented empty pool.
func (c *Config) NodePool() (netmap.Pool, error) {
	if c.Version != PoolVersion {
		return netmap.Pool{}, fmt.Errorf(
			"serverset: %s is a v%d per-server inventory; pooled assignment needs version: %d with a pool block",
			c.path, c.Version, PoolVersion)
	}
	return c.nodePool(), nil
}

// PoolSSH returns the v2 inventory's credential block (zero value when the
// file omitted it). Secrets in the environment still win, per the standing
// rule; sudo is carried for the bring-up procedure to act on.
func (c *Config) PoolSSH() SSH {
	if c.SSHConf == nil {
		return SSH{}
	}
	return *c.SSHConf
}
