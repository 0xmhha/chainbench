package app

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"os"
	"path/filepath"
	"strings"

	"go.yaml.in/yaml/v3"

	"github.com/0xmhha/chainbench/internal/core/filestore"
)

// sshPathFields are the ssh fields that name a file. A set written into a job
// directory is read from there, so a relative path is made absolute first to
// keep meaning what it meant to the dashboard.
var sshPathFields = []string{"password_file", "key_file", "key_passphrase_file", "known_hosts_file"}

// writeJobServerSet writes the pinned server set for one CLI run into dir, a
// private directory it creates. Each bound server gets its own ssh block whose
// secrets are private files beside the set; the set itself holds no secret.
// The caller removes dir when the run ends.
func (s *DeploymentStore) writeJobServerSet(ctx context.Context, a DeploymentActor, document DeploymentDocumentInput, bindings map[string]string, dir string) (string, error) {
	lookup, err := s.jobCredentialLookup(ctx, a, document, bindings)
	if err != nil {
		return "", err
	}
	set, err := deploymentSet(document)
	if err != nil {
		return "", err
	}
	var doc map[string]any
	if err = json.Unmarshal(document.Content, &doc); err != nil {
		return "", err
	}
	pool, _ := doc["pool"].(map[string]any)
	hosts, _ := pool["hosts"].([]any)
	if len(hosts) != len(set.Servers) {
		return "", errors.New("server set hosts do not match the parsed servers")
	}
	if top, ok := doc["ssh"].(map[string]any); ok {
		if err = absoluteSSHPaths(top); err != nil {
			return "", err
		}
	}
	// The directory is made here rather than by the file store, which would
	// make it world-readable; it must be fresh, so nothing else is in it.
	if err = os.Mkdir(dir, 0o700); err != nil {
		return "", err
	}
	secret := func(name string, value []byte) (string, error) {
		path := filepath.Join(dir, name)
		return path, (filestore.Local{}).Write(ctx, path, value, 0o600)
	}
	for i, server := range set.Servers {
		if _, bound := bindings[server.Name]; !bound {
			continue
		}
		c, err := lookup(server.Name)
		if err != nil {
			return "", err
		}
		ssh := map[string]any{"user": c.User}
		if c.Port != 0 {
			ssh["port"] = c.Port
		}
		prefix := fmt.Sprintf("host%d-", i+1)
		if c.Password != "" {
			if ssh["password_file"], err = secret(prefix+"password", []byte(c.Password)); err != nil {
				return "", err
			}
		}
		if len(c.PrivateKey) > 0 {
			if ssh["key_file"], err = secret(prefix+"key", c.PrivateKey); err != nil {
				return "", err
			}
			if c.Passphrase != "" {
				if ssh["key_passphrase_file"], err = secret(prefix+"passphrase", []byte(c.Passphrase)); err != nil {
					return "", err
				}
			}
		}
		if c.HostKey.KnownHostsFile != "" {
			ssh["known_hosts_file"] = c.HostKey.KnownHostsFile
		}
		if c.HostKey.InsecureHostKey {
			ssh["insecure_host_key"] = true
		}
		if c.Sudo {
			ssh["sudo"] = true
		}
		if err = absoluteSSHPaths(ssh); err != nil {
			return "", err
		}
		hosts[i] = map[string]any{"name": server.Name, "addr": server.Host, "ssh": ssh}
	}
	raw, err := yaml.Marshal(doc)
	if err != nil {
		return "", err
	}
	path := filepath.Join(dir, "server-set.yaml")
	return path, (filestore.Local{}).Write(ctx, path, raw, 0o600)
}

// absoluteSSHPaths makes every relative file path in an ssh block absolute
// against the dashboard's working directory, where it was resolved before.
func absoluteSSHPaths(ssh map[string]any) error {
	for _, field := range sshPathFields {
		p, _ := ssh[field].(string)
		if p == "" || filepath.IsAbs(p) || p == "~" || strings.HasPrefix(p, "~/") {
			continue
		}
		abs, err := filepath.Abs(p)
		if err != nil {
			return err
		}
		ssh[field] = abs
	}
	return nil
}
