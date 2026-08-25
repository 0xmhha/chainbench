package deploy

import (
	"fmt"
	"os"

	"go.yaml.in/yaml/v3"

	"github.com/0xmhha/chainbench/internal/core/remote"
)

// Credentials is the SSH auth for the cluster (the gitignored `credentials`
// file, copied from credentials.sample). Auth is a password and/or a private
// key (key_file); a per-server override map keys off the server index.
type Credentials struct {
	User      string                `yaml:"user"`
	Password  string                `yaml:"password"`
	KeyFile   string                `yaml:"key_file,omitempty"`
	Overrides map[int]CredsOverride `yaml:"overrides,omitempty"`
	// fromFile records that these values came from a credentials file. A file
	// is the single source of the cluster's login — the same rule the server
	// set enforces — so the environment is consulted only when there is no
	// file at all.
	fromFile bool
}

// CredsOverride is a per-server auth override.
type CredsOverride struct {
	User     string `yaml:"user,omitempty"`
	Password string `yaml:"password,omitempty"`
}

// LoadCredentials reads the credentials file. If path is empty it returns
// credentials that read the environment (CHAINBENCH_REMOTE_*) at For() time —
// the only case in which the environment is consulted.
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
	cr.fromFile = true
	return &cr, nil
}

// For builds the remote.Credentials for a server: user/password from the
// per-server override, else the file's global values. A loaded credentials
// file is the single source — a leftover CHAINBENCH_REMOTE_* export must not
// silently redirect a login, the same rule the server set enforces. Only when
// no file was given does the environment supply the values.
func (cr *Credentials) For(c *Cluster, s Server, env func(string) string) (remote.Credentials, error) {
	user := cr.User
	pass := cr.Password
	keyFile := cr.KeyFile
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
	var passphrase string
	if !cr.fromFile && env != nil {
		if v := env("CHAINBENCH_REMOTE_USER"); v != "" {
			user = v
		}
		if v := env("CHAINBENCH_REMOTE_PASS"); v != "" {
			pass = v
		}
		if v := env("CHAINBENCH_REMOTE_KEY_FILE"); v != "" {
			keyFile = v
		}
		passphrase = env("CHAINBENCH_REMOTE_KEY_PASSPHRASE")
	}
	if user == "" {
		return remote.Credentials{}, fmt.Errorf("deploy: no SSH user for server %d (set credentials.user, or CHAINBENCH_REMOTE_USER when no credentials file is used)", s.Index)
	}

	rc := remote.Credentials{User: user, Host: s.Host, Port: c.SSHPortFor(s), Password: pass}
	if keyFile != "" {
		key, err := remote.LoadPrivateKey(keyFile)
		if err != nil {
			return remote.Credentials{}, fmt.Errorf("deploy: server %d: %w", s.Index, err)
		}
		rc.PrivateKey = key
		rc.Passphrase = passphrase
	}
	if rc.Password == "" && len(rc.PrivateKey) == 0 {
		return remote.Credentials{}, fmt.Errorf("deploy: no SSH auth for server %d (set credentials.password/key_file in the credentials file, or CHAINBENCH_REMOTE_PASS/CHAINBENCH_REMOTE_KEY_FILE when no file is used)", s.Index)
	}
	return rc, nil
}
