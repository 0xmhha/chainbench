package remote

import (
	"bytes"
	"context"
	"errors"
	"fmt"
	"io"
	"net"
	"net/http"
	"os"
	"path/filepath"
	"strconv"
	"time"

	"golang.org/x/crypto/ssh"
	"golang.org/x/crypto/ssh/knownhosts"
)

// sshDialTimeout bounds the SSH handshake; the per-RPC context bounds the
// tunneled calls separately.
const sshDialTimeout = 15 * time.Second

const defaultSSHPort = 22

// HostKeyCallback is the SSH host-key verification policy (an alias for
// ssh.HostKeyCallback), re-exported so callers can name it without importing
// golang.org/x/crypto/ssh directly. Build one with ResolveHostKeyCallback.
type HostKeyCallback = ssh.HostKeyCallback

// Credentials carries the inputs for an SSH dial. Password is read by the caller
// from the env var named in the node's ssh-password auth; it builds the
// ssh.AuthMethod and is never logged or returned in errors.
type Credentials struct {
	User     string
	Host     string
	Port     int
	Password string
}

// DialTunnelClient opens an SSH connection and returns an *http.Client whose TCP
// dials are tunneled through it, plus the SSH connection as an io.Closer for the
// caller to release when done. The client is ready for rpc.DialWithClient with
// an RPC URL reachable from the remote host (e.g. http://127.0.0.1:8545).
// Errors never include the password.
func DialTunnelClient(creds Credentials, hostKey ssh.HostKeyCallback) (*http.Client, io.Closer, error) {
	sshClient, err := dialSSH(creds, hostKey)
	if err != nil {
		return nil, nil, err
	}
	return &http.Client{Transport: tunnelTransport(sshClient)}, sshClient, nil
}

// tunnelTransport builds an http.Transport whose TCP dials go through the SSH
// connection. The inner dial is not ctx-cancelable (x/crypto/ssh), but the SSH
// handshake timeout and the caller's per-request ctx bound it in practice.
func tunnelTransport(sshClient *ssh.Client) *http.Transport {
	return &http.Transport{
		DialContext: func(_ context.Context, network, dialAddr string) (net.Conn, error) {
			return sshClient.Dial(network, dialAddr)
		},
	}
}

// dialSSH validates credentials and opens an SSH connection with the given host
// key policy. Errors never echo the password.
func dialSSH(creds Credentials, hostKey ssh.HostKeyCallback) (*ssh.Client, error) {
	if creds.User == "" || creds.Host == "" {
		return nil, fmt.Errorf("remote: ssh user and host are required")
	}
	if creds.Password == "" {
		return nil, fmt.Errorf("remote: empty SSH password")
	}
	if hostKey == nil {
		return nil, fmt.Errorf("remote: nil host key callback")
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
		return nil, fmt.Errorf("remote: ssh dial %s@%s: %w", creds.User, addr, err)
	}
	return c, nil
}

// ExecResult captures a single remote command's output. A non-zero ExitCode is
// NOT an error from Exec's perspective (the connection worked) — the caller
// classifies it.
type ExecResult struct {
	Stdout   string
	Stderr   string
	ExitCode int
}

// Exec runs a single command on the remote host over SSH and captures its
// output. The SSH connection is opened per call and closed before returning.
// Only dial/session-open failures return an error; a command that exits non-zero
// returns a populated ExecResult with that code. Errors never include the
// password. ctx is accepted for symmetry; ssh.Session.Run is not ctx-cancelable,
// so the dial timeout bounds the connection.
func Exec(ctx context.Context, creds Credentials, hostKey ssh.HostKeyCallback, command string) (ExecResult, error) {
	_ = ctx
	c, err := dialSSH(creds, hostKey)
	if err != nil {
		return ExecResult{}, err
	}
	defer c.Close()

	sess, err := c.NewSession()
	if err != nil {
		return ExecResult{}, fmt.Errorf("remote: ssh session: %w", err)
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
		return res, fmt.Errorf("remote: ssh exec: %w", runErr)
	}
	return res, nil
}

// ResolveHostKeyCallback builds the SSH host key verifier per the security
// policy:
//
//   - CHAINBENCH_SSH_INSECURE_HOST_KEY=1 -> InsecureIgnoreHostKey (loud opt-in,
//     for test/sandbox use against ephemeral hosts).
//   - otherwise -> known_hosts verification using CHAINBENCH_SSH_KNOWN_HOSTS, or
//     ~/.ssh/known_hosts when unset. Unknown/mismatched hosts are rejected.
//
// env is injected for testability (production passes os.Getenv).
func ResolveHostKeyCallback(env func(string) string) (ssh.HostKeyCallback, error) {
	if env("CHAINBENCH_SSH_INSECURE_HOST_KEY") == "1" {
		return ssh.InsecureIgnoreHostKey(), nil
	}
	path := env("CHAINBENCH_SSH_KNOWN_HOSTS")
	if path == "" {
		home, err := os.UserHomeDir()
		if err != nil {
			return nil, fmt.Errorf("remote: cannot locate ~/.ssh/known_hosts: %w", err)
		}
		path = filepath.Join(home, ".ssh", "known_hosts")
	}
	cb, err := knownhosts.New(path)
	if err != nil {
		return nil, fmt.Errorf("remote: load known_hosts %q: %w "+
			"(set CHAINBENCH_SSH_KNOWN_HOSTS, or CHAINBENCH_SSH_INSECURE_HOST_KEY=1 to bypass)", path, err)
	}
	return cb, nil
}
