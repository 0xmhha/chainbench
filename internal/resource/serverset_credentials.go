package resource

import (
	"fmt"
	"os"
	"strings"

	"github.com/0xmhha/chainbench/internal/core/remote"
)

// Turning a declared login into the credential a dial uses.
//
// A secret can be written in the file or pointed at by path, and the path form
// is the one a secret manager renders to disk — so the set stays readable and
// the secret stays out of it. The environment is never consulted: a variable
// left over from another environment must not be able to redirect a dial.

func (s Server) Credentials() (remote.Credentials, error) {
	if !s.IsRemote() {
		return remote.Credentials{}, fmt.Errorf("serverset: server %s is local — it has no SSH credentials", s.label(0))
	}
	if s.SSH.User == "" {
		return remote.Credentials{}, fmt.Errorf(
			"serverset: server %s has no SSH user (set ssh.user in the server set)", s.label(0))
	}
	pass := s.SSH.Password
	if s.SSH.PasswordFile != "" {
		if pass != "" {
			return remote.Credentials{}, fmt.Errorf(
				"serverset: server %s sets both ssh.password and ssh.password_file — keep exactly one", s.label(0))
		}
		v, err := readSecretFile(s.SSH.PasswordFile)
		if err != nil {
			return remote.Credentials{}, fmt.Errorf("serverset: server %s: %w", s.label(0), err)
		}
		pass = v
	}
	port := s.SSH.Port
	if port == 0 {
		port = defaultSSHPort
	}
	rc := remote.Credentials{
		User: s.SSH.User, Host: s.Host, Port: port, Password: pass,
		HostKey: remote.HostKeyPolicy{
			KnownHostsFile:  s.SSH.KnownHostsFile,
			InsecureHostKey: s.SSH.InsecureHostKey,
		},
		Sudo: s.SSH.Sudo,
	}
	if s.SSH.KeyFile != "" {
		key, err := remote.LoadPrivateKey(s.SSH.KeyFile)
		if err != nil {
			return remote.Credentials{}, fmt.Errorf("serverset: server %s: %w", s.label(0), err)
		}
		rc.PrivateKey = key
		if s.SSH.KeyPassphraseFile != "" {
			v, err := readSecretFile(s.SSH.KeyPassphraseFile)
			if err != nil {
				return remote.Credentials{}, fmt.Errorf("serverset: server %s: %w", s.label(0), err)
			}
			rc.Passphrase = v
		}
		// An encrypted key with no passphrase would only fail at dial time
		// with a bare parse error; name the missing field here instead.
		if rc.Passphrase == "" && remote.KeyNeedsPassphrase(key) {
			return remote.Credentials{}, fmt.Errorf(
				"serverset: server %s: key %s is passphrase-protected — set ssh.key_passphrase_file", s.label(0), s.SSH.KeyFile)
		}
	}
	if rc.Password == "" && len(rc.PrivateKey) == 0 {
		return remote.Credentials{}, fmt.Errorf(
			"serverset: server %s has no SSH auth (set ssh.password, ssh.password_file, or ssh.key_file in the server set)", s.label(0))
	}
	return rc, nil
}

// readSecretFile reads a one-line secret referenced from the server set,
// trimming the trailing newline an editor leaves. It enforces the same 0600
// rule the key-file path does — the plaintext password deserves no weaker a
// check than the key. The value never appears in an error: a failure names
// the path, not the content.
func readSecretFile(path string) (string, error) {
	insecure, perm, err := remote.InsecureFilePerm(path)
	if err != nil {
		return "", fmt.Errorf("stat secret file: %w", err)
	}
	if insecure {
		return "", fmt.Errorf("secret file %s has insecure permissions %#o (want 0600)", path, perm)
	}
	b, err := os.ReadFile(path)
	if err != nil {
		return "", fmt.Errorf("read secret file: %w", err)
	}
	v := strings.TrimRight(string(b), "\r\n")
	if v == "" {
		return "", fmt.Errorf("secret file %s is empty", path)
	}
	return v, nil
}

// SetLookup returns a Lookup backed by the server set at
// path (empty uses DefaultSetFile). It is how an srv://<name>/path target
// gets its host, port and credentials without any of those appearing in a
// command line, a spec file, or a persisted workspace.
//
// The file is opened on each lookup rather than cached: a lookup happens once
// per target, and an operator editing the server set mid-session should not have
// to reason about which copy is in effect.
func SetLookup(path string) Lookup {
	return func(name string) (remote.Credentials, error) {
		if path == "" {
			path = DefaultSetFile
		}
		cfg, err := LoadSet(path)
		if err != nil {
			return remote.Credentials{}, fmt.Errorf("serverset: %q needs the server set: %w", name, err)
		}
		s, err := cfg.ByName(name)
		if err != nil {
			return remote.Credentials{}, err
		}
		return s.Credentials()
	}
}

// SetPolicy loads the set-level host-key policy from the server set at path
// (empty uses the default location). It is what covers a DIRECTLY named host
// (user@host:/path): that form has no set entry to carry a policy, but the
// operator's set still says how this site verifies hosts. A missing set is
// the zero policy — the safe known_hosts default — not an error, because the
// direct form must keep working with no set at all.
func SetPolicy(path string) remote.HostKeyPolicy {
	if path == "" {
		path = DefaultSetFile
	}
	cfg, err := LoadSet(path)
	if err != nil {
		return remote.HostKeyPolicy{}
	}
	return remote.HostKeyPolicy{
		KnownHostsFile:  cfg.SSH.KnownHostsFile,
		InsecureHostKey: cfg.SSH.InsecureHostKey,
	}
}
