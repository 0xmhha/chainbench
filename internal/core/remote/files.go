package remote

import (
	"context"
	"encoding/base64"
	"fmt"
	"strings"
)

// ReadFile reads a remote file's bytes over SSH, base64-piped to avoid binary
// and quoting issues. It is the read counterpart of the driver's file shipping,
// shared by the deploy key-fetch and the `keys/account/validator --remote-import`
// sources.
func ReadFile(ctx context.Context, creds Credentials, hostKey HostKeyCallback, path string) ([]byte, error) {
	res, err := Exec(ctx, creds, hostKey, "base64 < "+shellQuote(path))
	if err != nil {
		return nil, err
	}
	if res.ExitCode != 0 {
		return nil, fmt.Errorf("remote: read %s: exit %d: %s", path, res.ExitCode, strings.TrimSpace(res.Stderr))
	}
	return base64.StdEncoding.DecodeString(strings.TrimSpace(res.Stdout))
}

// CredentialsFromEnv builds SSH credentials from the given user/host/port and the
// standard chainbench environment: CHAINBENCH_REMOTE_USER overrides the user;
// CHAINBENCH_REMOTE_PASS or CHAINBENCH_REMOTE_KEY_FILE (+ _KEY_PASSPHRASE) supply
// the auth. Secrets are read only from env, never captured in a config, and
// never echoed in errors. env is injected for testability (production passes
// os.Getenv).
func CredentialsFromEnv(user, host string, port int, env func(string) string) (Credentials, error) {
	if env == nil {
		env = func(string) string { return "" }
	}
	if v := env("CHAINBENCH_REMOTE_USER"); v != "" {
		user = v
	}
	creds := Credentials{User: user, Host: host, Port: port}
	if v := env("CHAINBENCH_REMOTE_PASS"); v != "" {
		creds.Password = v
	}
	if kf := env("CHAINBENCH_REMOTE_KEY_FILE"); kf != "" {
		key, err := LoadPrivateKey(kf)
		if err != nil {
			return Credentials{}, err
		}
		creds.PrivateKey = key
		creds.Passphrase = env("CHAINBENCH_REMOTE_KEY_PASSPHRASE")
	}
	if creds.User == "" {
		return Credentials{}, fmt.Errorf("remote: no SSH user (set --remote-user or CHAINBENCH_REMOTE_USER)")
	}
	if creds.Password == "" && len(creds.PrivateKey) == 0 {
		return Credentials{}, fmt.Errorf("remote: no SSH auth (set CHAINBENCH_REMOTE_PASS or CHAINBENCH_REMOTE_KEY_FILE)")
	}
	return creds, nil
}

// shellQuote single-quotes a path for safe remote shell use.
func shellQuote(s string) string {
	return "'" + strings.ReplaceAll(s, "'", `'\''`) + "'"
}
