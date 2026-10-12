package app

import (
	"context"
	"errors"
	"sync/atomic"
	"testing"

	"github.com/0xmhha/chainbench/internal/core/remote"
)

func leaseFixture(t *testing.T) (*DeploymentStore, DeploymentActor, DeploymentWorkspace, DeploymentCredential) {
	t.Helper()
	s, err := OpenDeploymentStore(t.TempDir())
	if err != nil {
		t.Fatal(err)
	}
	a := DeploymentActor{ID: "alice", Role: "operator"}
	set, err := s.SaveDocument(a, "", 0, deploymentTestSet())
	if err != nil {
		t.Fatal(err)
	}
	config, err := s.SaveDocument(a, "", 0, deploymentTestConfig())
	if err != nil {
		t.Fatal(err)
	}
	w, err := s.SaveWorkspace(a, "", 0, DeploymentWorkspaceInput{Name: "Lease", Documents: []DeploymentDocumentRef{{set.ID, set.Revision}, {config.ID, config.Revision}}})
	if err != nil {
		t.Fatal(err)
	}
	c, err := s.SaveCredential(a, DeploymentCredentialInput{Label: "private", Kind: "password", SSHUser: "ssh", Password: "lease-secret-password"})
	if err != nil {
		t.Fatal(err)
	}
	if err = s.BindCredential(a, w.ID, "ssh", c.ID); err != nil {
		t.Fatal(err)
	}
	return s, a, w, c
}

func TestCredentialLeaseRejectsRevocationChangesAndCancelledAccess(t *testing.T) {
	for _, scenario := range []string{"credential", "account", "workspace", "binding", "context", "unrelated"} {
		t.Run(scenario, func(t *testing.T) {
			s, a, w, c := leaseFixture(t)
			var revoked atomic.Bool
			s.SetAuthorizer(func(DeploymentActor) error {
				if revoked.Load() {
					return ErrDeploymentForbidden
				}
				return nil
			})
			ctx, cancel := context.WithCancel(context.Background())
			defer cancel()
			_, _, creds, err := s.credentialLease(ctx, a, w.ID, "ssh", c.ID)
			if err != nil || creds.BeforeDial == nil || creds.BeforeDial() != nil {
				t.Fatal("lease not established", err)
			}
			if _, _, _, err = s.credentialLease(ctx, DeploymentActor{ID: "bob", Role: "operator"}, w.ID, "ssh", c.ID); !errors.Is(err, ErrDeploymentNotFound) {
				t.Fatal("foreign credential exposed", err)
			}
			err = nil
			switch scenario {
			case "credential":
				err = s.RevokeCredential(a, c.ID)
			case "account":
				revoked.Store(true)
			case "workspace":
				in := w.DeploymentWorkspaceInput
				in.Name = "Changed"
				_, err = s.SaveWorkspace(a, w.ID, w.Revision, in)
			case "binding", "unrelated":
				var other DeploymentCredential
				other, err = s.SaveCredential(a, DeploymentCredentialInput{Label: "other", Kind: "password", SSHUser: "ssh", Password: "other-lease-password"})
				if err == nil {
					if scenario == "binding" {
						err = s.BindCredential(a, w.ID, "ssh", other.ID)
					} else {
						err = s.RevokeCredential(a, other.ID)
					}
				}
			case "context":
				cancel()
			}
			if err != nil {
				t.Fatal(err)
			}
			err = creds.BeforeDial()
			if scenario == "unrelated" {
				if err != nil {
					t.Fatal("unrelated revocation invalidated this actor's lease", err)
				}
			} else {
				if err == nil {
					t.Fatal("cached credential remains authorized after change")
				}
				// The low-level dial must refuse before attempting even a bad
				// address. No SSH server or network success is claimed here.
				if _, _, err = remote.DialTunnelClient(creds, nil); err == nil {
					t.Fatal("revoked lease reached a transport")
				}
			}
		})
	}
}

func TestJobCredentialLeasePinsAcceptedInputsWhileRecheckingRevocation(t *testing.T) {
	s, a, w, c := leaseFixture(t)
	set, err := s.DocumentRevision(w.Documents[0].ID, w.Documents[0].Revision)
	if err != nil {
		t.Fatal(err)
	}
	bindings := map[string]string{"ssh": c.ID}
	lookup, err := s.jobCredentialLookup(context.Background(), a, set.DeploymentDocumentInput, bindings)
	if err != nil {
		t.Fatal(err)
	}
	// The caller must not be able to mutate the accepted lease by changing
	// its map, nor may future shared configuration or binding edits redirect it.
	bindings["ssh"] = "foreign"
	other, err := s.SaveCredential(a, DeploymentCredentialInput{Label: "other", Kind: "password", SSHUser: "other-user", Password: "other-password"})
	if err != nil {
		t.Fatal(err)
	}
	if err = s.BindCredential(a, w.ID, "ssh", other.ID); err != nil {
		t.Fatal(err)
	}
	in := w.DeploymentWorkspaceInput
	in.Name = "Updated after acceptance"
	if _, err = s.SaveWorkspace(a, w.ID, w.Revision, in); err != nil {
		t.Fatal(err)
	}
	creds, err := lookup("ssh")
	if err != nil || creds.User != "ssh" || creds.Host != "localhost." || creds.BeforeDial() != nil {
		t.Fatal("accepted inputs redirected", creds.User, creds.Host, err)
	}
	foreign, err := s.jobCredentialLookup(context.Background(), DeploymentActor{ID: "bob", Role: "operator"}, set.DeploymentDocumentInput, map[string]string{"ssh": c.ID})
	if err != nil {
		t.Fatal(err)
	}
	if _, err = foreign("ssh"); !errors.Is(err, ErrDeploymentForbidden) {
		t.Fatal("foreign credential accepted", err)
	}
	if _, err = lookup("unknown"); err == nil {
		t.Fatal("unselected server accepted")
	}
	if err = s.RevokeCredential(a, c.ID); err != nil {
		t.Fatal(err)
	}
	if creds.BeforeDial() == nil {
		t.Fatal("cached lease survived revocation")
	}
	if _, err = lookup("ssh"); err == nil {
		t.Fatal("fresh lease survived revocation")
	}
}
