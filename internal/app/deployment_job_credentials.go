package app

import (
	"context"
	"encoding/json"
	"errors"

	"github.com/0xmhha/chainbench/internal/core/remote"
	"github.com/0xmhha/chainbench/internal/resource"
)

// jobCredentialLookup uses the accepted job's declaration and credential IDs.
// Later workspace/binding edits cannot redirect an accepted operation. Account
// and credential revocation still invalidate every cached connection handle.
func (s *DeploymentStore) jobCredentialLookup(ctx context.Context, a DeploymentActor, document DeploymentDocumentInput, bindings map[string]string) (resource.Lookup, error) {
	set, err := deploymentSet(document)
	if err != nil {
		return nil, err
	}
	pinned := map[string]string{}
	for name, id := range bindings {
		pinned[name] = id
	}
	return func(name string) (remote.Credentials, error) {
		host, err := set.ByName(name)
		if err != nil {
			return remote.Credentials{}, ErrDeploymentNotFound
		}
		id := pinned[name]
		guard := func() error {
			if err := ctx.Err(); err != nil {
				return err
			}
			s.mu.Lock()
			authorize := s.authorize
			c, found := s.state.Credentials[id]
			s.mu.Unlock()
			if !a.canEdit() || !found || c.Metadata.OwnerID != a.ID || c.Metadata.Revoked {
				return ErrDeploymentForbidden
			}
			if authorize != nil && authorize(a) != nil {
				return ErrDeploymentForbidden
			}
			return nil
		}
		if err = guard(); err != nil {
			return remote.Credentials{}, err
		}
		s.mu.Lock()
		c := s.state.Credentials[id]
		var b []byte
		if len(c.Ciphertext) >= s.aead.NonceSize() {
			b, err = s.aead.Open(nil, c.Ciphertext[:s.aead.NonceSize()], c.Ciphertext[s.aead.NonceSize():], []byte(c.Metadata.ID+":"+c.Metadata.OwnerID))
		} else {
			err = errors.New("invalid encrypted credential")
		}
		s.mu.Unlock()
		if err != nil {
			return remote.Credentials{}, errors.New("cannot decrypt job credential")
		}
		var secret DeploymentCredentialInput
		if err = json.Unmarshal(b, &secret); err != nil {
			return remote.Credentials{}, errors.New("invalid job credential")
		}
		if err = guard(); err != nil {
			return remote.Credentials{}, err
		}
		return remote.Credentials{User: secret.SSHUser, Host: host.Host, Port: host.SSH.Port, Password: secret.Password, PrivateKey: []byte(secret.PrivateKey), Passphrase: secret.Passphrase, HostKey: remote.HostKeyPolicy{KnownHostsFile: host.SSH.KnownHostsFile, InsecureHostKey: host.SSH.InsecureHostKey}, Sudo: host.SSH.Sudo, BeforeDial: guard}, nil
	}, nil
}
