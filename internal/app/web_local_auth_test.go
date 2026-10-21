package app

import (
	"bytes"
	"context"
	"errors"
	"os"
	"path/filepath"
	"testing"
	"testing/synctest"
	"time"
)

func TestLocalWebIdentityPersistsOwnershipWithoutCreatingAccounts(t *testing.T) {
	root := t.TempDir()
	auth, err := OpenLocalWebAuth(root)
	if err != nil {
		t.Fatal(err)
	}
	if auth.AccountsEnabled() || auth.BootstrapRequired() || len(auth.Users()) != 0 {
		t.Fatal("local access requires registered accounts")
	}
	for _, name := range []string{"users.json", "setup.token"} {
		if _, err := os.Stat(filepath.Join(root, name)); !errors.Is(err, os.ErrNotExist) {
			t.Fatalf("local access created %s: %v", name, err)
		}
	}
	local, token, err := auth.LocalSession()
	if err != nil {
		t.Fatal(err)
	}
	if local.AccountsEnabled || local.User.ID == "" || local.User.Role != "administrator" || local.CSRFToken == "" {
		t.Fatal("missing local execution context")
	}
	actor := DeploymentActor{ID: local.User.ID, Role: "admin"}
	for _, role := range []string{"admin", "operator"} {
		if err := auth.AuthorizeJobActor(DeploymentActor{ID: actor.ID, Role: role}); err != nil {
			t.Fatal("local job/reconciliation refused", err)
		}
	}
	for _, rejected := range []DeploymentActor{{ID: "foreign", Role: "admin"}, {ID: actor.ID, Role: "viewer"}, {Role: "admin"}} {
		if !errors.Is(auth.AuthorizeJobActor(rejected), ErrDeploymentForbidden) {
			t.Fatal("foreign actor authorized")
		}
	}
	if _, _, err := auth.Bootstrap("admin", "long-admin-password", "unused"); !errors.Is(err, ErrDeploymentForbidden) {
		t.Fatal("local access bootstrapped an account", err)
	}
	if _, err := auth.CreateUser(actor, "another", "long-admin-password", "administrator"); !errors.Is(err, ErrDeploymentForbidden) {
		t.Fatal("local access created an account", err)
	}
	store, err := OpenDeploymentStore(root)
	if err != nil {
		t.Fatal(err)
	}
	store.SetAuthorizer(auth.AuthorizeJobActor)
	credential, err := store.SaveCredential(actor, DeploymentCredentialInput{Label: "local payer", Kind: "account-key", AccountKey: testAccountKey})
	if err != nil {
		t.Fatal(err)
	}
	restarted, err := OpenLocalWebAuth(root)
	if err != nil {
		t.Fatal(err)
	}
	if _, err := restarted.Session(token); !errors.Is(err, ErrDeploymentForbidden) {
		t.Fatal("browser token survived restart")
	}
	next, _, err := restarted.LocalSession()
	if err != nil || next.User.ID != actor.ID {
		t.Fatal("local ownership changed across restart", err)
	}
	store, err = OpenDeploymentStore(root)
	if err != nil {
		t.Fatal(err)
	}
	store.SetAuthorizer(restarted.AuthorizeJobActor)
	if len(store.Credentials(actor)) != 1 {
		t.Fatal("local credential disappeared")
	}
	if _, err := store.accountKey(context.Background(), actor, credential.ID); err != nil {
		t.Fatal("local credential no longer decrypts", err)
	}
}

func TestLocalWebJobsRunAndCancelWithTheSameActor(t *testing.T) {
	synctest.Test(t, func(t *testing.T) {
		auth, err := OpenLocalWebAuth(t.TempDir())
		if err != nil {
			t.Fatal(err)
		}
		local, _, err := auth.LocalSession()
		if err != nil {
			t.Fatal(err)
		}
		actor := DeploymentActor{ID: local.User.ID, Role: "admin"}
		engine := newBlockingJobEngine()
		jobs, err := OpenWebJobs(t.TempDir(), engine, nil, auth.AuthorizeJobActor)
		if err != nil {
			t.Fatal(err)
		}
		_, accepted := startFixtureJob(t, jobs, actor, WebPlanInput{WorkspaceID: "local-workspace", Operation: "chain.deploy"}, "local-start")
		synctest.Wait()
		if engine.starts.Load() != 1 {
			t.Fatal("local job did not execute")
		}
		if _, err := jobs.Cancel(actor, accepted.ID); err != nil {
			t.Fatal("local operator could not cancel its job", err)
		}
		synctest.Wait()
		job, err := jobs.Get(accepted.ID)
		if err != nil || job.State != "cancelled" || job.ActorID != local.User.ID {
			t.Fatal("local job failed to complete cancellation", err)
		}
	})
}

func TestLocalWebAccessRenewsExpiredBrowserContext(t *testing.T) {
	auth, err := OpenLocalWebAuth(t.TempDir())
	if err != nil {
		t.Fatal(err)
	}
	now := time.Date(2026, 10, 13, 0, 0, 0, 0, time.UTC)
	auth.clock = func() time.Time { return now }
	first, token, err := auth.LocalSession()
	if err != nil {
		t.Fatal(err)
	}
	now = first.ExpiresAt
	if _, err := auth.Session(token); !errors.Is(err, ErrDeploymentForbidden) {
		t.Fatal("expired local browser session accepted")
	}
	renewed, _, err := auth.LocalSession()
	if err != nil || renewed.User.ID != first.User.ID || renewed.CSRFToken == first.CSRFToken || !renewed.ExpiresAt.After(first.ExpiresAt) {
		t.Fatal("local browser context did not renew", err)
	}
}

func TestAccountAndLocalModesDoNotAdoptEachOthersIdentity(t *testing.T) {
	root := t.TempDir()
	accounts, err := OpenWebAuth(root, "personal")
	if err != nil {
		t.Fatal(err)
	}
	setup, err := os.ReadFile(filepath.Join(root, "setup.token"))
	if err != nil {
		t.Fatal(err)
	}
	registered, accountToken, err := accounts.Bootstrap("admin", "long-admin-password", string(setup))
	if err != nil {
		t.Fatal(err)
	}
	before, err := os.ReadFile(filepath.Join(root, "users.json"))
	if err != nil {
		t.Fatal(err)
	}
	if _, _, err := accounts.LocalSession(); !errors.Is(err, ErrDeploymentForbidden) {
		t.Fatal("account-enabled service provided passwordless access")
	}
	local, err := OpenLocalWebAuth(root)
	if err != nil {
		t.Fatal(err)
	}
	if _, err := local.Session(accountToken); !errors.Is(err, ErrDeploymentForbidden) {
		t.Fatal("local service adopted an account browser token")
	}
	if err := local.AuthorizeJobActor(DeploymentActor{ID: registered.User.ID, Role: "admin"}); !errors.Is(err, ErrDeploymentForbidden) {
		t.Fatal("local service adopted an account's jobs")
	}
	if _, _, err := local.Login("admin", "long-admin-password"); !errors.Is(err, ErrDeploymentForbidden) {
		t.Fatal("local service logged in an existing account")
	}
	localSession, localToken, err := local.LocalSession()
	if err != nil {
		t.Fatal(err)
	}
	after, err := os.ReadFile(filepath.Join(root, "users.json"))
	if err != nil || !bytes.Equal(before, after) {
		t.Fatal("switching to local access changed accounts", err)
	}
	restarted, err := OpenWebAuth(root, "personal")
	if err != nil {
		t.Fatal(err)
	}
	if _, err := restarted.Session(localToken); !errors.Is(err, ErrDeploymentForbidden) {
		t.Fatal("account service accepted a local token")
	}
	if err := restarted.AuthorizeJobActor(DeploymentActor{ID: localSession.User.ID, Role: "admin"}); !errors.Is(err, ErrDeploymentForbidden) {
		t.Fatal("account service adopted local jobs")
	}
	if _, _, err := restarted.Login("admin", "long-admin-password"); err != nil {
		t.Fatal("registered login was not preserved", err)
	}
}
