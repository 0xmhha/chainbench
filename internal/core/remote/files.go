package remote

import (
	"context"
	"encoding/base64"
	"fmt"
	"strings"
)

// ReadFileCommand returns the shell command that emits path's bytes as base64.
// Base64 avoids binary and quoting damage on the way back through a shell.
//
// It is exported with [DecodeReadFile] so that a caller holding a command
// runner rather than credentials — the driver's file store — reads files over
// exactly the same wire format, instead of growing a second one beside it.
func ReadFileCommand(path string) string { return "base64 < " + shellQuote(path) }

// DecodeReadFile turns the result of [ReadFileCommand] into the file's bytes.
// A non-zero exit is the file being unreadable, not a transport failure.
func DecodeReadFile(path string, res ExecResult) ([]byte, error) {
	if res.ExitCode != 0 {
		return nil, fmt.Errorf("remote: read %s: exit %d: %s", path, res.ExitCode, strings.TrimSpace(res.Stderr))
	}
	return base64.StdEncoding.DecodeString(strings.TrimSpace(res.Stdout))
}

// ReadFile reads a remote file's bytes over SSH. It is the read counterpart of
// the driver's file shipping, shared by the deploy key-fetch and the
// `keys/account/validator --remote-import` sources.
func ReadFile(ctx context.Context, creds Credentials, hostKey HostKeyCallback, path string) ([]byte, error) {
	res, err := Exec(ctx, creds, hostKey, ReadFileCommand(path))
	if err != nil {
		return nil, err
	}
	return DecodeReadFile(path, res)
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
