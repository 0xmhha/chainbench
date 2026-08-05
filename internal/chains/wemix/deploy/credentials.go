package deploy

import (
	"fmt"
	"os"

	"go.yaml.in/yaml/v3"

	"github.com/0xmhha/chainbench/internal/core/remote"
)

// Credentials is the SSH auth for the cluster (the gitignored `credentials`
// file, copied from credentials.sample). Password auth mirrors wemix4's
// .credentials; a per-server override map keys off the server index.
//
// key_file is reserved for a future phase — the underlying remote.Credentials is
// password-only today, so key-based auth is not yet wired.
type Credentials struct {
	User      string                `yaml:"user"`
	Password  string                `yaml:"password"`
	KeyFile   string                `yaml:"key_file,omitempty"`
	Overrides map[int]CredsOverride `yaml:"overrides,omitempty"`
}

// CredsOverride is a per-server auth override.
type CredsOverride struct {
	User     string `yaml:"user,omitempty"`
	Password string `yaml:"password,omitempty"`
}

// LoadCredentials reads the credentials file. If path is empty it returns
// credentials with only the environment fallback (CHAINBENCH_REMOTE_PASS /
// CHAINBENCH_REMOTE_USER) applied at For() time.
func LoadCredentials(path string) (*Credentials, error) {
	if path == "" {
		return &Credentials{}, nil
	}
	b, err := os.ReadFile(path)
	if err != nil {
		return nil, fmt.Errorf("deploy: read credentials: %w", err)
	}
	var cr Credentials
	if err := yaml.Unmarshal(b, &cr); err != nil {
		return nil, fmt.Errorf("deploy: parse credentials: %w", err)
	}
	if cr.KeyFile != "" {
		return nil, fmt.Errorf("deploy: key_file auth is not supported yet (password only); unset key_file")
	}
	return &cr, nil
}

// For builds the remote.Credentials for a server: user/password from the
// per-server override, else the global values, with env fallbacks
// (CHAINBENCH_REMOTE_USER / CHAINBENCH_REMOTE_PASS — the standard chainbench
// secret channel) taking precedence over a file-stored password when set.
func (cr *Credentials) For(c *Cluster, s Server, env func(string) string) (remote.Credentials, error) {
	user := cr.User
	pass := cr.Password
	if o, ok := cr.Overrides[s.Index]; ok {
		if o.User != "" {
			user = o.User
		}
		if o.Password != "" {
			pass = o.Password
		}
	}
	if s.User != "" { // a server-level user in cluster.yaml wins over the global
		user = s.User
	}
	if env != nil {
		if v := env("CHAINBENCH_REMOTE_USER"); v != "" {
			user = v
		}
		if v := env("CHAINBENCH_REMOTE_PASS"); v != "" {
			pass = v
		}
	}
	if user == "" {
		return remote.Credentials{}, fmt.Errorf("deploy: no SSH user for server %d (set credentials.user or CHAINBENCH_REMOTE_USER)", s.Index)
	}
	if pass == "" {
		return remote.Credentials{}, fmt.Errorf("deploy: no SSH password for server %d (set credentials.password or CHAINBENCH_REMOTE_PASS)", s.Index)
	}
	return remote.Credentials{User: user, Host: s.Host, Port: c.SSHPortFor(s), Password: pass}, nil
}
