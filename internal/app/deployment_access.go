package app

import (
	"context"
	"encoding/json"
	"errors"
	"net"
	"strconv"
	"strings"

	"github.com/0xmhha/chainbench/internal/core/remote"
	"github.com/0xmhha/chainbench/internal/resource"
	"golang.org/x/crypto/ssh"
)

func validateDeploymentKey(in DeploymentCredentialInput) error {
	if in.Kind != "private-key" {
		return nil
	}
	var err error
	if in.Passphrase != "" {
		_, err = ssh.ParsePrivateKeyWithPassphrase([]byte(in.PrivateKey), []byte(in.Passphrase))
	} else {
		_, err = ssh.ParsePrivateKey([]byte(in.PrivateKey))
	}
	if err != nil {
		return errors.New("invalid SSH private key or passphrase")
	}
	return nil
}

func (s *DeploymentStore) resolveAccess(a DeploymentActor, workspace, server, credential string) (resource.Server, resource.WorkspaceConfig, DeploymentCredentialInput, error) {
	var host resource.Server
	var config resource.WorkspaceConfig
	var secret DeploymentCredentialInput
	w, ok := s.state.Workspaces[workspace]
	if !ok {
		return host, config, secret, ErrDeploymentNotFound
	}
	setdoc, configdoc, err := s.workspaceDocuments(w.DeploymentWorkspaceInput)
	if err != nil {
		return host, config, secret, err
	}
	set, err := deploymentSet(setdoc.DeploymentDocumentInput)
	if err != nil {
		return host, config, secret, err
	}
	host, err = set.ByName(server)
	if err != nil {
		return host, config, secret, errors.New("server reference not in workspace")
	}
	config, err = deploymentWorkspace(configdoc.DeploymentDocumentInput)
	if err != nil {
		return host, config, secret, err
	}
	c, ok := s.state.Credentials[credential]
	if !ok || c.Metadata.OwnerID != a.ID || c.Metadata.Revoked {
		return host, config, secret, ErrDeploymentNotFound
	}
	encrypted := c.Ciphertext
	if len(encrypted) < s.aead.NonceSize() {
		return host, config, secret, errors.New("invalid encrypted credential")
	}
	b, err := s.aead.Open(nil, encrypted[:s.aead.NonceSize()], encrypted[s.aead.NonceSize():], []byte(c.Metadata.ID+":"+c.Metadata.OwnerID))
	if err != nil {
		return host, config, secret, errors.New("cannot decrypt credential")
	}
	if err = json.Unmarshal(b, &secret); err != nil {
		return host, config, secret, err
	}
	return host, config, secret, nil
}

// DeploymentAccessResult separates successful authentication from permission to write.
type DeploymentAccessResult struct {
	Reachable         bool     `json:"reachable"`
	Authenticated     bool     `json:"authenticated"`
	HostIdentity      string   `json:"hostIdentity"`
	AllowedOperations []string `json:"allowedOperations"`
	Reason            string   `json:"reason,omitempty"`
}

// CheckCredential performs a real SSH handshake and a read-only data-root permission probe.
// Neither credentials nor remote command output are included in the response or audit.
func (s *DeploymentStore) CheckCredential(ctx context.Context, a DeploymentActor, workspace, server, credential string) (DeploymentAccessResult, error) {
	out := DeploymentAccessResult{AllowedOperations: []string{}}
	if !a.canEdit() {
		return out, ErrDeploymentForbidden
	}
	host, config, creds, err := s.credentialLease(ctx, a, workspace, server, credential)
	if err != nil {
		return out, err
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
	out.Reachable = true
	out.Authenticated = true
	// Find the closest existing ancestor without creating or modifying node data.
	quote := "'" + strings.ReplaceAll(config.DataRoot, "'", "'\"'\"'") + "'"
	cmd := "p=" + quote + `; while [ ! -e "$p" ]; do p=${p%/*}; [ -n "$p" ] || p=/; done; test -d "$p" && test -w "$p" && test -x "$p"`
	res, err := remote.Exec(ctx, creds, verify, cmd)
	if err != nil || res.ExitCode != 0 {
		out.Reason = "data root or nearest existing ancestor is not writable/searchable"
	} else {
		out.AllowedOperations = []string{"deploy"}
	}
	s.mu.Lock()
	defer s.mu.Unlock()
	if err := s.commit(a, "credential.check", workspace+":"+server, func(*deploymentState) {}); err != nil {
		return out, err
	}
	return out, nil
}
