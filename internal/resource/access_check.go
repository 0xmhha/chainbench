package resource

import (
	"context"
	"errors"
	"net"
	"path/filepath"
	"strconv"
	"strings"

	"golang.org/x/crypto/ssh"

	"github.com/0xmhha/chainbench/internal/core/remote"
)

// AccessCheck distinguishes SSH authentication from target write permission.
type AccessCheck struct {
	Reachable         bool     `json:"reachable"`
	Authenticated     bool     `json:"authenticated"`
	HostIdentity      string   `json:"hostIdentity"`
	AllowedOperations []string `json:"allowedOperations"`
	Reason            string   `json:"reason,omitempty"`
}

// CheckAccess checks authentication and permission without creating node data.
// Neither credential bytes nor remote command output leave this boundary.
func CheckAccess(ctx context.Context, host Server, creds remote.Credentials, dataRoot string) (AccessCheck, error) {
	out := AccessCheck{AllowedOperations: []string{}}
	if err := ctx.Err(); err != nil {
		return out, err
	}
	if !filepath.IsAbs(dataRoot) || strings.ContainsAny(dataRoot, "\x00\r\n") {
		return out, errors.New("resource: access checks require an absolute data root")
	}
	policy := remote.HostKeyPolicy{KnownHostsFile: host.SSH.KnownHostsFile, InsecureHostKey: host.SSH.InsecureHostKey}
	cb, err := policy.Callback()
	if err != nil {
		out.Reason = "host-key verification unavailable"
		return out, nil
	}
	port := host.SSH.Port
	if port == 0 {
		port = remote.DefaultSSHPort
	}
	verify := func(hostname string, addr net.Addr, key ssh.PublicKey) error {
		if err := cb(hostname, addr, key); err != nil {
			return err
		}
		out.HostIdentity = net.JoinHostPort(host.Host, strconv.Itoa(port)) + "/" + ssh.FingerprintSHA256(key)
		return nil
	}
	_, closer, err := remote.DialTunnelClient(creds, verify)
	if err != nil {
		out.Reachable = out.HostIdentity != ""
		out.Reason = "SSH authentication or host access denied"
		return out, nil
	}
	_ = closer.Close()
	if err := ctx.Err(); err != nil {
		return out, err
	}
	out.Reachable = true
	out.Authenticated = true
	// Find the closest existing ancestor without creating or modifying node data.
	quote := "'" + strings.ReplaceAll(dataRoot, "'", "'\"'\"'") + "'"
	cmd := "p=" + quote + `; while [ ! -e "$p" ]; do p=${p%/*}; [ -n "$p" ] || p=/; done; test -d "$p" && test -w "$p" && test -x "$p"`
	res, err := remote.Exec(ctx, creds, verify, cmd)
	if err != nil || res.ExitCode != 0 {
		out.Reason = "data root or nearest existing ancestor is not writable/searchable"
	} else {
		out.AllowedOperations = []string{"deploy"}
	}
	return out, nil
}
