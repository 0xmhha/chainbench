package app

import (
	"context"
	"encoding/json"
	"errors"
	"os"
	"path/filepath"
	"slices"
	"strings"
	"sync/atomic"
	"testing"
	"time"

	"github.com/0xmhha/chainbench/internal/resource"
)

func unitTargetReader(t *testing.T, lookup resource.Lookup) *webTargetReader {
	t.Helper()
	return &webTargetReader{read: func(ctx context.Context, in ResourceReadIn) (ResourceReadOut, error) {
		return ResourceRead(ctx, Deps{ServerLookup: lookup}, in)
	}, check: func(context.Context, string) error { return nil }}
}

type targetTestCLI struct {
	run func(context.Context, []string) ([]byte, error)
}

func (c targetTestCLI) ChainStatus(context.Context, string) (State, error) { return State{}, nil }
func (c targetTestCLI) Run(ctx context.Context, args ...string) ([]byte, error) {
	return c.run(ctx, args)
}

func TestWebReadSecretsArePrivateAndRemovedAfterRevocation(t *testing.T) {
	s, a, w, c := leaseFixture(t)
	var set DeploymentDocument
	for _, ref := range w.Documents {
		d, err := s.DocumentRevision(ref.ID, ref.Revision)
		if err != nil {
			t.Fatal(err)
		}
		if d.Kind == "server-set" {
			set = d
		}
	}
	e := NewWebChainEngine(t.TempDir(), "", s, nil, nil, nil)
	p := webChainPayload{Set: set, Input: WebPlanInput{CredentialBindings: map[string]string{"ssh": c.ID}}}
	started := make(chan string, 1)
	var revoked atomic.Bool
	s.SetAuthorizer(func(DeploymentActor) error {
		if revoked.Load() {
			return ErrDeploymentForbidden
		}
		return nil
	})
	e.UseCLI(targetTestCLI{run: func(ctx context.Context, args []string) ([]byte, error) {
		if strings.Contains(strings.Join(args, " "), "lease-secret-password") {
			t.Error("secret reached command arguments")
		}
		path := flagValue(args, "--server-logins")
		set, err := resource.LoadSet(path)
		if err != nil {
			t.Error(err)
			return nil, err
		}
		host, err := set.ByName("ssh")
		if err != nil {
			t.Error(err)
			return nil, err
		}
		secret := host.SSH.PasswordFile
		for _, check := range []struct {
			path string
			mode os.FileMode
		}{{path, 0600}, {secret, 0600}, {filepath.Dir(path), 0700}, {filepath.Dir(filepath.Dir(path)), 0700}} {
			info, err := os.Stat(check.path)
			if err != nil || info.Mode().Perm() != check.mode {
				t.Error("private mode mismatch", check.path, err)
			}
		}
		bytes, err := os.ReadFile(secret)
		if err != nil || string(bytes) != "lease-secret-password" {
			t.Error("bound password did not reach private file", err)
		}
		started <- filepath.Dir(filepath.Dir(path))
		<-ctx.Done()
		return nil, ctx.Err()
	}})
	r, err := e.targetReader(context.Background(), a, p)
	if err != nil {
		t.Fatal(err)
	}
	done := make(chan error, 1)
	go func() {
		_, err := r.Inspect(context.Background(), resource.Spec{Server: "ssh", DataRoot: "/data/chainbench"})
		done <- err
	}()
	var privateDir string
	select {
	case privateDir = <-started:
	case <-time.After(5 * time.Second):
		t.Fatal("read never reached command")
	}
	revoked.Store(true)
	select {
	case err = <-done:
		if !errors.Is(err, ErrDeploymentForbidden) {
			t.Fatal("revocation was not preserved", err)
		}
	case <-time.After(5 * time.Second):
		t.Fatal("revocation did not cancel command")
	}
	if _, err = os.Stat(privateDir); !os.IsNotExist(err) {
		t.Fatal("private read inputs survived command exit", err)
	}
}

func TestWebReconciliationUsesCLIInsteadOfOpeningTarget(t *testing.T) {
	e, _ := keySnapshotEngine(t)
	state, p := webOwnedControlFixture(t)
	p.ControlDir = t.TempDir()
	raw, err := json.Marshal(state)
	if err != nil {
		t.Fatal(err)
	}
	if err = os.WriteFile(filepath.Join(p.ControlDir, "chain-record.json"), raw, 0600); err != nil {
		t.Fatal(err)
	}
	cli := &runCLI{err: errors.New("CLI target read refused")}
	e.UseCLI(cli)
	payload, err := json.Marshal(p)
	if err != nil {
		t.Fatal(err)
	}
	out, err := e.Reconcile(context.Background(), DeploymentActor{ID: "operator", Role: "operator"}, WebPreparedJob{Payload: payload})
	if err != nil {
		t.Fatal(err)
	}
	if len(cli.args) < 2 || cli.args[0] != "resource" || cli.args[1] != "inspect" || !slices.Contains(cli.args, "--request") {
		t.Fatalf("reconciliation bypassed CLI: %+v %v", out, cli.args)
	}
	if out.Outcome != "unverified" {
		t.Fatalf("failed CLI observation released claims: %+v", out)
	}
}
