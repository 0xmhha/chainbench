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
	"strings"
	"time"

	"golang.org/x/crypto/ssh"
	"golang.org/x/crypto/ssh/knownhosts"
)

// sshDialTimeout bounds the SSH handshake; the per-RPC context bounds the
// tunneled calls separately.
const sshDialTimeout = 15 * time.Second

// DefaultSSHPort is the port an SSH dial assumes when none was named. It is
// exported so the layer applying a dial-time AddrMap can resolve the default
// BEFORE mapping — a map keyed on port 22 must match a dial that will use 22.
const DefaultSSHPort = 22

// HostKeyCallback is the SSH host-key verification policy (an alias for
// ssh.HostKeyCallback), re-exported so callers can name it without importing
// golang.org/x/crypto/ssh directly. Build one with ResolveHostKeyCallback.
type HostKeyCallback = ssh.HostKeyCallback

// Credentials carries the inputs for an SSH dial. Auth is a password and/or a
// private key; at least one is required. Secrets are read by the caller from the
// configured source; they build the ssh.AuthMethod and are never logged or
// returned in errors.
type Credentials struct {
	User     string
	Host     string
	Port     int
	Password string
	// PrivateKey is a PEM-encoded SSH private key, an alternative or addition to
	// Password. When both are set the key is tried first. Never logged.
	PrivateKey []byte
	// Passphrase decrypts an encrypted PrivateKey; empty for an unencrypted key.
	Passphrase string
	// HostKey is how a dial to this machine verifies the host. The zero value
	// defers to the caller's policy (the server set's, or the safe default).
	HostKey HostKeyPolicy
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
	if hostKey == nil {
		return nil, fmt.Errorf("remote: nil host key callback")
	}
	auth, err := authMethods(creds)
	if err != nil {
		return nil, err
	}
	port := creds.Port
	if port == 0 {
		port = DefaultSSHPort
	}
	cfg := &ssh.ClientConfig{
		User:            creds.User,
		Auth:            auth,
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

// insecureKeyPermMask flags any group/other permission bit on a private key
// file (0600 means only the owner may read/write).
const insecureKeyPermMask = 0o077

// authMethods builds the SSH auth methods from the credentials: a private key
// (tried first when present) and/or a password. At least one must be supplied.
// Key/password material never appears in the returned errors.
func authMethods(creds Credentials) ([]ssh.AuthMethod, error) {
	var methods []ssh.AuthMethod
	if len(creds.PrivateKey) > 0 {
		signer, err := parseSigner(creds.PrivateKey, creds.Passphrase)
		if err != nil {
			return nil, err
		}
		methods = append(methods, ssh.PublicKeys(signer))
	}
	if creds.Password != "" {
		methods = append(methods, ssh.Password(creds.Password))
	}
	if len(methods) == 0 {
		return nil, fmt.Errorf("remote: no SSH auth provided (set a password or a private key)")
	}
	return methods, nil
}

// parseSigner parses a PEM private key, using passphrase when the key is
// encrypted. Errors carry no key material.
func parseSigner(pemKey []byte, passphrase string) (ssh.Signer, error) {
	if passphrase != "" {
		signer, err := ssh.ParsePrivateKeyWithPassphrase(pemKey, []byte(passphrase))
		if err != nil {
			return nil, fmt.Errorf("remote: parse encrypted private key: %w", err)
		}
		return signer, nil
	}
	signer, err := ssh.ParsePrivateKey(pemKey)
	if err != nil {
		return nil, fmt.Errorf("remote: parse private key: %w", err)
	}
	return signer, nil
}

// KeyNeedsPassphrase reports whether pemKey is passphrase-protected. It lets a
// credential loader fail at resolve time with "this key needs a passphrase"
// instead of deferring to a cryptic dial-time parse error.
func KeyNeedsPassphrase(pemKey []byte) bool {
	_, err := ssh.ParsePrivateKey(pemKey)
	var missing *ssh.PassphraseMissingError
	return errors.As(err, &missing)
}

// InsecureFilePerm reports whether path is group- or world-accessible (want
// 0600). It is the same rule LoadPrivateKey enforces, exported so other
// secret-holding files (a server set's password_file) get the same check.
// The returned perm is for the error message; ok=false means the stat failed.
func InsecureFilePerm(path string) (insecure bool, perm os.FileMode, err error) {
	info, err := os.Stat(path)
	if err != nil {
		return false, 0, err
	}
	p := info.Mode().Perm()
	return p&insecureKeyPermMask != 0, p, nil
}

// LoadPrivateKey reads a PEM-encoded SSH private key file for
// Credentials.PrivateKey. It rejects a key file that is group- or
// world-accessible (want 0600), so an exposed key is a fail-fast error rather
// than a silent security hole.
func LoadPrivateKey(path string) ([]byte, error) {
	info, err := os.Stat(path)
	if err != nil {
		return nil, fmt.Errorf("remote: stat key file: %w", err)
	}
	if info.Mode().Perm()&insecureKeyPermMask != 0 {
		return nil, fmt.Errorf("remote: key file %s has insecure permissions %#o (want 0600)", path, info.Mode().Perm())
	}
	b, err := os.ReadFile(path)
	if err != nil {
		return nil, fmt.Errorf("remote: read key file: %w", err)
	}
	return b, nil
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
	return ExecWithInput(ctx, creds, hostKey, command, "")
}

// ExecWithInput is Exec with data supplied on the remote command's stdin.
// It exists for commands that read a secret interactively — sudo -S reads the
// password from stdin — where putting the value in the command line would
// leave it in the remote host's process list and shell history.
func ExecWithInput(ctx context.Context, creds Credentials, hostKey ssh.HostKeyCallback, command, stdin string) (ExecResult, error) {
	_ = ctx
	c, err := dialSSH(creds, hostKey)
	if err != nil {
		return ExecResult{}, err
	}
	defer func() { _ = c.Close() }()

	sess, err := c.NewSession()
	if err != nil {
		return ExecResult{}, fmt.Errorf("remote: ssh session: %w", err)
	}
	defer func() { _ = sess.Close() }()

	var stdout, stderr bytes.Buffer
	sess.Stdout = &stdout
	sess.Stderr = &stderr
	if stdin != "" {
		sess.Stdin = strings.NewReader(stdin)
	}

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

// HostKeyPolicy is how a dial verifies the host it reached. It is DATA, loaded
// from the server set at runtime like every other connection setting — no
// environment variable configures it. The zero value is the safe default:
// verify against ~/.ssh/known_hosts, reject unknown or changed hosts.
type HostKeyPolicy struct {
	// KnownHostsFile verifies against this file instead of the default
	// ~/.ssh/known_hosts.
	KnownHostsFile string
	// InsecureHostKey skips verification entirely. Closed-network, throwaway
	// servers only — the server set that declares it should say why.
	InsecureHostKey bool
}

// Callback builds the SSH verifier the policy describes.
func (p HostKeyPolicy) Callback() (ssh.HostKeyCallback, error) {
	if p.InsecureHostKey {
		if p.KnownHostsFile != "" {
			return nil, fmt.Errorf("remote: host-key policy sets both insecure_host_key and known_hosts_file — keep exactly one")
		}
		return ssh.InsecureIgnoreHostKey(), nil
	}
	path := p.KnownHostsFile
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
			"(name a known_hosts_file in the server set's ssh block, or insecure_host_key: true on a closed network)", path, err)
	}
	return cb, nil
}
