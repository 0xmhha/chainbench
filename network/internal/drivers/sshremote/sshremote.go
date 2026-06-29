package sshremote

import (
	"bytes"
	"context"
	"errors"
	"fmt"
	"net"
	"net/http"
	"os"
	"path/filepath"
	"strconv"
	"time"

	"golang.org/x/crypto/ssh"
	"golang.org/x/crypto/ssh/knownhosts"

	"github.com/0xmhha/chainbench/network/internal/drivers/remote"
)

// sshDialTimeout bounds the SSH handshake. The per-RPC timeout (handler ctx)
// bounds the tunneled calls separately.
const sshDialTimeout = 15 * time.Second

const defaultSSHPort = 22

// Credentials carries the inputs for an SSH dial. Password is read by the
// caller from the env var named in the node's ssh-password auth; it is used
// only to build the ssh.AuthMethod and is never logged or returned in errors.
type Credentials struct {
	User     string
	Host     string
	Port     int
	Password string
}

// Dial establishes an SSH connection and returns a remote.Client whose RPC
// traffic is tunneled through it. The returned client's Close() also closes the
// SSH connection (wired via remote.DialOptions.Closer), so callers treat an
// SSH-tunneled client exactly like a plain HTTP one.
//
// rpcURL is the node's RPC endpoint as reachable *from the remote host* (e.g.
// http://127.0.0.1:8545 for an RPC bound to the node's loopback). hostKey
// verifies the server identity; build it with ResolveHostKeyCallback.
//
// Errors never include creds.Password — only the host and (upstream) cause.
func Dial(ctx context.Context, creds Credentials, rpcURL string, hostKey ssh.HostKeyCallback) (*remote.Client, error) {
	sshClient, err := dialSSH(creds, hostKey)
	if err != nil {
		return nil, err
	}

	// Tunnel every RPC TCP connection through the SSH session. The inner Dial
	// is not ctx-cancelable (x/crypto/ssh), but the SSH handshake timeout above
	// and the handler's per-call ctx on the RPC round-trip bound it in practice.
	transport := &http.Transport{
		DialContext: func(_ context.Context, network, dialAddr string) (net.Conn, error) {
			return sshClient.Dial(network, dialAddr)
		},
	}

	client, err := remote.DialWithOptions(ctx, rpcURL, remote.DialOptions{
		Transport: transport,
		Closer:    sshClient,
	})
	if err != nil {
		// remote does not take ownership of Closer on error — close the SSH
		// connection here to avoid leaking it.
		_ = sshClient.Close()
		return nil, err
	}
	return client, nil
}

// dialSSH validates credentials and opens an SSH connection with the given host
// key policy. Errors never echo the password (ssh.Dial wraps auth/host-key
// failures by message only). Shared by Dial (tunnel) and Exec (command).
func dialSSH(creds Credentials, hostKey ssh.HostKeyCallback) (*ssh.Client, error) {
	if creds.User == "" || creds.Host == "" {
		return nil, fmt.Errorf("sshremote: user and host are required")
	}
	if creds.Password == "" {
		return nil, fmt.Errorf("sshremote: empty SSH password")
	}
	if hostKey == nil {
		return nil, fmt.Errorf("sshremote: nil host key callback")
	}
	port := creds.Port
	if port == 0 {
		port = defaultSSHPort
	}
	cfg := &ssh.ClientConfig{
		User:            creds.User,
		Auth:            []ssh.AuthMethod{ssh.Password(creds.Password)},
		HostKeyCallback: hostKey,
		Timeout:         sshDialTimeout,
	}
	addr := net.JoinHostPort(creds.Host, strconv.Itoa(port))
	c, err := ssh.Dial("tcp", addr, cfg)
	if err != nil {
		return nil, fmt.Errorf("ssh dial %s@%s: %w", creds.User, addr, err)
	}
	return c, nil
}

// ExecResult captures the output of a single remote command. ExitCode carries
// the command's exit status; a non-zero ExitCode is NOT an error from Exec's
// perspective (the connection worked) — the caller classifies it.
type ExecResult struct {
	Stdout   string
	Stderr   string
	ExitCode int
}

// Exec runs a single command on the remote host over SSH and captures its
// output (Sprint 5b.2 — powers ssh-remote process/fs handlers). The SSH
// connection is opened per call and closed before returning, matching the
// spawn-per-call model. Only dial / session-open failures return an error; a
// command that exits non-zero returns a populated ExecResult with that code.
//
// ctx is accepted for API symmetry; x/crypto/ssh's Session.Run is not
// ctx-cancelable, so the dial timeout bounds the connection. Errors never
// include the password.
func Exec(ctx context.Context, creds Credentials, hostKey ssh.HostKeyCallback, command string) (ExecResult, error) {
	_ = ctx
	c, err := dialSSH(creds, hostKey)
	if err != nil {
		return ExecResult{}, err
	}
	defer c.Close()

	sess, err := c.NewSession()
	if err != nil {
		return ExecResult{}, fmt.Errorf("ssh session: %w", err)
	}
	defer sess.Close()

	var stdout, stderr bytes.Buffer
	sess.Stdout = &stdout
	sess.Stderr = &stderr

	runErr := sess.Run(command)
	res := ExecResult{Stdout: stdout.String(), Stderr: stderr.String()}
	if runErr != nil {
		var exitErr *ssh.ExitError
		if errors.As(runErr, &exitErr) {
			res.ExitCode = exitErr.ExitStatus()
			return res, nil // non-zero exit is a command result, not a transport error
		}
		return res, fmt.Errorf("ssh exec: %w", runErr)
	}
	return res, nil
}

// ResolveHostKeyCallback builds the SSH host key verifier per the security
// policy (Sprint 5b.1 D3):
//
//   - CHAINBENCH_SSH_INSECURE_HOST_KEY=1 → InsecureIgnoreHostKey (loud opt-in,
//     for test/sandbox use against ephemeral hosts).
//   - otherwise → known_hosts verification using CHAINBENCH_SSH_KNOWN_HOSTS, or
//     ~/.ssh/known_hosts when unset. Unknown/mismatched hosts are rejected.
//
// env is injected for testability; production callers pass os.Getenv.
func ResolveHostKeyCallback(env func(string) string) (ssh.HostKeyCallback, error) {
	if env("CHAINBENCH_SSH_INSECURE_HOST_KEY") == "1" {
		return ssh.InsecureIgnoreHostKey(), nil
	}
	path := env("CHAINBENCH_SSH_KNOWN_HOSTS")
	if path == "" {
		home, err := os.UserHomeDir()
		if err != nil {
			return nil, fmt.Errorf("sshremote: cannot locate ~/.ssh/known_hosts: %w", err)
		}
		path = filepath.Join(home, ".ssh", "known_hosts")
	}
	cb, err := knownhosts.New(path)
	if err != nil {
		return nil, fmt.Errorf("sshremote: load known_hosts %q: %w "+
			"(set CHAINBENCH_SSH_KNOWN_HOSTS, or CHAINBENCH_SSH_INSECURE_HOST_KEY=1 to bypass)", path, err)
	}
	return cb, nil
}
