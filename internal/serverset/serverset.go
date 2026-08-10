// Package serverset loads a static inventory of remote SSH servers from a YAML
// config (remote-server-config.yaml) so fleet operations select a server by
// index — `--server 7` — instead of repeating host/user/port on every command.
//
// It is a control-plane concern: it never dials. It resolves a chosen server to
// remote.Credentials, layering server-level values over file-level defaults, and
// letting the environment override secrets (CHAINBENCH_REMOTE_*) so passwords
// need not live in the file. The file itself is gitignored (only the .sample is
// tracked); prefer the environment for the password on shared machines.
package serverset

import (
	"bytes"
	"fmt"
	"os"
	"sort"

	"go.yaml.in/yaml/v3"

	"github.com/0xmhha/chainbench/internal/core/remote"
)

// DefaultConfigFile is the config path used when --server-config is omitted.
const DefaultConfigFile = "remote-server-config.yaml"

// defaultSSHPort is the SSH port assumed when neither server nor defaults set it.
const defaultSSHPort = 22

// Server is one remote host in the inventory. Index is the selector (--server N).
// User/Port/Password/KeyFile are optional overrides of the file-level defaults.
type Server struct {
	Index    int    `yaml:"index"`
	Host     string `yaml:"host"`
	User     string `yaml:"user,omitempty"`
	Port     int    `yaml:"port,omitempty"`
	Password string `yaml:"password,omitempty"`
	KeyFile  string `yaml:"key_file,omitempty"`
}

// Config is the parsed remote-server-config.yaml: file-level SSH defaults plus
// the server inventory. A server inherits any field it omits from the defaults.
type Config struct {
	User     string   `yaml:"user,omitempty"`
	Port     int      `yaml:"port,omitempty"`
	Password string   `yaml:"password,omitempty"`
	KeyFile  string   `yaml:"key_file,omitempty"`
	Servers  []Server `yaml:"servers"`
}

// Load reads and validates the server inventory at path. It rejects unknown
// fields (typo protection), a missing or empty inventory, duplicate indexes, and
// a server with no host.
func Load(path string) (*Config, error) {
	b, err := os.ReadFile(path)
	if err != nil {
		if os.IsNotExist(err) {
			return nil, fmt.Errorf("serverset: config %s not found (copy %s.sample and fill it in)", path, DefaultConfigFile)
		}
		return nil, fmt.Errorf("serverset: read %s: %w", path, err)
	}
	dec := yaml.NewDecoder(bytes.NewReader(b))
	dec.KnownFields(true)
	var c Config
	if err := dec.Decode(&c); err != nil {
		return nil, fmt.Errorf("serverset: parse %s: %w", path, err)
	}
	if err := c.validate(); err != nil {
		return nil, err
	}
	return &c, nil
}

// validate enforces a non-empty inventory with unique indexes and set hosts.
func (c *Config) validate() error {
	if len(c.Servers) == 0 {
		return fmt.Errorf("serverset: no servers configured")
	}
	seen := make(map[int]bool, len(c.Servers))
	for _, s := range c.Servers {
		if s.Host == "" {
			return fmt.Errorf("serverset: server index %d has no host", s.Index)
		}
		if seen[s.Index] {
			return fmt.Errorf("serverset: duplicate server index %d", s.Index)
		}
		seen[s.Index] = true
	}
	return nil
}

// Server returns the server with the given index, its omitted fields filled from
// the file-level defaults. It errors, listing the available indexes, if none
// matches.
func (c *Config) Server(index int) (Server, error) {
	for _, s := range c.Servers {
		if s.Index != index {
			continue
		}
		if s.User == "" {
			s.User = c.User
		}
		if s.Port == 0 {
			s.Port = c.Port
		}
		if s.Password == "" {
			s.Password = c.Password
		}
		if s.KeyFile == "" {
			s.KeyFile = c.KeyFile
		}
		return s, nil
	}
	return Server{}, fmt.Errorf("serverset: no server with index %d (available: %v)", index, c.indexes())
}

// indexes lists configured server indexes in ascending order for error messages.
func (c *Config) indexes() []int {
	out := make([]int, len(c.Servers))
	for i, s := range c.Servers {
		out[i] = s.Index
	}
	sort.Ints(out)
	return out
}

// Credentials builds the SSH credentials for the server. The environment
// overrides file values (CHAINBENCH_REMOTE_USER / _PASS / _KEY_FILE /
// _KEY_PASSPHRASE) so secrets can stay out of the file. It errors if no user or
// no auth (password or key) is resolved. env is injected for testing.
func (s Server) Credentials(env func(string) string) (remote.Credentials, error) {
	if env == nil {
		env = func(string) string { return "" }
	}
	user := s.User
	pass := s.Password
	keyFile := s.KeyFile
	if v := env("CHAINBENCH_REMOTE_USER"); v != "" {
		user = v
	}
	if v := env("CHAINBENCH_REMOTE_PASS"); v != "" {
		pass = v
	}
	if v := env("CHAINBENCH_REMOTE_KEY_FILE"); v != "" {
		keyFile = v
	}
	if user == "" {
		return remote.Credentials{}, fmt.Errorf("serverset: server %d has no SSH user (set user in the config or CHAINBENCH_REMOTE_USER)", s.Index)
	}
	port := s.Port
	if port == 0 {
		port = defaultSSHPort
	}
	rc := remote.Credentials{User: user, Host: s.Host, Port: port, Password: pass}
	if keyFile != "" {
		key, err := remote.LoadPrivateKey(keyFile)
		if err != nil {
			return remote.Credentials{}, fmt.Errorf("serverset: server %d: %w", s.Index, err)
		}
		rc.PrivateKey = key
		rc.Passphrase = env("CHAINBENCH_REMOTE_KEY_PASSPHRASE")
	}
	if rc.Password == "" && len(rc.PrivateKey) == 0 {
		return remote.Credentials{}, fmt.Errorf("serverset: server %d has no SSH auth (set password/key_file in the config or CHAINBENCH_REMOTE_PASS/CHAINBENCH_REMOTE_KEY_FILE)", s.Index)
	}
	return rc, nil
}
