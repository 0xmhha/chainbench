package app

import (
	"context"
	"encoding/json"

	"github.com/0xmhha/chainbench/internal/core/remote"
	"github.com/0xmhha/chainbench/internal/resource"
)

// credentialLease keeps decrypted material in memory and rechecks revocation,
// account permission and the workspace declaration at each SSH connection.
// Shared server-set files never receive private material or a secret path.
func (s *DeploymentStore) credentialLease(ctx context.Context, a DeploymentActor, workspace, server, credential string) (resource.Server, resource.WorkspaceConfig, remote.Credentials, error) {
	s.mu.Lock()
	host, config, secret, err := s.resolveAccess(a, workspace, server, credential)
	w := s.state.Workspaces[workspace]
	binding := s.state.Bindings[a.ID][workspace][server]
	declaration, _ := json.Marshal(w)
	s.mu.Unlock()
	if err != nil {
		return host, config, remote.Credentials{}, err
	}
	guard := func() error {
		if err := ctx.Err(); err != nil {
			return err
		}
		s.mu.Lock()
		authorize := s.authorize
		s.mu.Unlock()
		if authorize != nil {
			if err := authorize(a); err != nil {
				return ErrDeploymentForbidden
			}
		}
		s.mu.Lock()
		defer s.mu.Unlock()
		c, found := s.state.Credentials[credential]
		if !a.canEdit() || !found || c.Metadata.OwnerID != a.ID || c.Metadata.Revoked {
			return ErrDeploymentForbidden
		}
		current, _ := json.Marshal(s.state.Workspaces[workspace])
		if string(current) != string(declaration) {
			return ErrDeploymentConflict
		}
		if s.state.Bindings[a.ID][workspace][server] != binding {
			return ErrDeploymentConflict
		}
		return nil
	}
	if err = guard(); err != nil {
		return host, config, remote.Credentials{}, err
	}
	creds := remote.Credentials{User: secret.SSHUser, Host: host.Host, Port: host.SSH.Port, Password: secret.Password, PrivateKey: []byte(secret.PrivateKey), Passphrase: secret.Passphrase, HostKey: remote.HostKeyPolicy{KnownHostsFile: host.SSH.KnownHostsFile, InsecureHostKey: host.SSH.InsecureHostKey}, Sudo: host.SSH.Sudo, BeforeDial: guard}
	return host, config, creds, nil
}
