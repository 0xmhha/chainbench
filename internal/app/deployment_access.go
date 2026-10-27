package app

import (
	"context"
	"encoding/json"
	"errors"
	"os"
	"path/filepath"

	"github.com/0xmhha/chainbench/internal/core/keyring/derive"
	"github.com/0xmhha/chainbench/internal/resource"
	"golang.org/x/crypto/ssh"
)

// accountKeyCredential is a key a test case signs with, not an SSH login.
const accountKeyCredential = "account-key"

// validCredentialShape accepts exactly the fields each kind uses.
func validCredentialShape(in DeploymentCredentialInput) bool {
	if in.Label == "" || !knownCredentialKind(in.Kind) {
		return false
	}
	switch in.Kind {
	case "password":
		return in.SSHUser != "" && in.Password != "" && in.PrivateKey == "" && in.Passphrase == "" && in.AccountKey == ""
	case "private-key":
		return in.SSHUser != "" && in.PrivateKey != "" && in.Password == "" && in.AccountKey == ""
	case accountKeyCredential:
		return in.AccountKey != "" && in.SSHUser == "" && in.Password == "" && in.PrivateKey == "" && in.Passphrase == ""
	}
	return false
}

func validateDeploymentKey(in DeploymentCredentialInput) error {
	if in.Kind == accountKeyCredential {
		// The parser's error names the length only, never the key.
		if _, err := derive.ParsePrivateKey(in.AccountKey); err != nil {
			return errors.New("invalid account key: want 32 bytes of hex")
		}
		return nil
	}
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
	if c.Metadata.Kind == accountKeyCredential {
		return host, config, secret, errors.New("an account key is not an SSH login")
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
type DeploymentAccessResult = resource.AccessCheck

// CheckCredential performs a real SSH handshake and a read-only data-root permission probe.
// Neither credentials nor remote command output are included in the response or audit.
func (s *DeploymentStore) CheckCredential(ctx context.Context, a DeploymentActor, workspace, server, credential string) (out DeploymentAccessResult, err error) {
	out = DeploymentAccessResult{AllowedOperations: []string{}}
	if !a.canEdit() {
		return out, ErrDeploymentForbidden
	}
	_, config, creds, err := s.credentialLease(ctx, a, workspace, server, credential)
	if err != nil {
		return out, err
	}
	s.mu.Lock()
	cli := s.cli
	w := s.state.Workspaces[workspace]
	setdoc, _, err := s.workspaceDocuments(w.DeploymentWorkspaceInput)
	s.mu.Unlock()
	if err != nil {
		return out, err
	}
	if cli == nil {
		return out, errNoChainCLI
	}
	dir, err := os.MkdirTemp("", "chainbench-access-")
	if err != nil {
		return out, err
	}
	defer func() { err = errors.Join(err, os.RemoveAll(dir)) }()
	logins, err := s.writeJobServerSet(ctx, a, setdoc.DeploymentDocumentInput, map[string]string{server: credential}, filepath.Join(dir, "secrets"))
	if err != nil {
		return out, err
	}
	guard := func() error {
		if err := ctx.Err(); err != nil {
			return err
		}
		if creds.BeforeDial != nil {
			return creds.BeforeDial()
		}
		return nil
	}
	args := []string{"resource", "access", "--json", "--server-set", logins, "--server-logins", logins, "--server", server, "--data-root", config.DataRoot}
	if webDocker(setdoc.DeploymentDocumentInput) {
		args = append(args, "--docker")
	}
	raw, err := runCLIWithAccessGuard(ctx, cli, guard, args)
	if err != nil {
		return out, err
	}
	if err = json.Unmarshal(raw, &out); err != nil {
		return out, errors.New("chainbench access check: invalid output")
	}
	if !out.Authenticated {
		return out, nil
	}
	s.mu.Lock()
	defer s.mu.Unlock()
	err = s.commit(a, "credential.check", workspace+":"+server, func(*deploymentState) {})
	return out, err
}
