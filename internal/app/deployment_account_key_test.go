package app

import (
	"context"
	"encoding/json"
	"os"
	"path/filepath"
	"strings"
	"testing"
)

const testAccountKey = "0x8f2a55949038a9610f50fb23b5883af3b4ecb3c3bb792cbcefbd1542c692be63"

// An account key signs a case's transactions. It is a personal secret like an
// SSH login, stored encrypted and listed as metadata only, and it is never
// usable as one.
func TestAccountKeyCredentialStaysPrivateAndIsNoSSHLogin(t *testing.T) {
	s, err := OpenDeploymentStore(t.TempDir())
	if err != nil {
		t.Fatal(err)
	}
	alice := DeploymentActor{ID: "alice", Role: "operator"}
	for name, in := range map[string]DeploymentCredentialInput{
		"bad hex":    {Label: "payer", Kind: "account-key", AccountKey: "0x1234"},
		"ssh user":   {Label: "payer", Kind: "account-key", SSHUser: "ssh", AccountKey: testAccountKey},
		"ssh secret": {Label: "payer", Kind: "account-key", Password: "pw", AccountKey: testAccountKey},
		"key on ssh": {Label: "login", Kind: "password", SSHUser: "ssh", Password: "pw", AccountKey: testAccountKey},
	} {
		if _, err := s.SaveCredential(alice, in); err == nil {
			t.Errorf("%s accepted", name)
		}
	}
	c, err := s.SaveCredential(alice, DeploymentCredentialInput{Label: "payer", Kind: "account-key", AccountKey: testAccountKey})
	if err != nil {
		t.Fatalf("account key refused: %v", err)
	}
	listed, _ := json.Marshal(s.Credentials(alice))
	if strings.Contains(string(listed), strings.TrimPrefix(testAccountKey, "0x")) {
		t.Fatal("credential listing exposed the account key")
	}
	set, err := s.SaveDocument(alice, "", 0, deploymentTestSet())
	if err != nil {
		t.Fatal(err)
	}
	config, err := s.SaveDocument(alice, "", 0, deploymentTestConfig())
	if err != nil {
		t.Fatal(err)
	}
	w, err := s.SaveWorkspace(alice, "", 0, DeploymentWorkspaceInput{Name: "Team", Documents: []DeploymentDocumentRef{{set.ID, set.Revision}, {config.ID, config.Revision}}})
	if err != nil {
		t.Fatal(err)
	}
	if err = s.BindCredential(alice, w.ID, "ssh", c.ID); err == nil {
		t.Fatal("an account key was bound as an SSH login")
	}
	if key, err := s.accountKey(context.Background(), alice, c.ID); err != nil || key != testAccountKey {
		t.Fatalf("owner cannot read the account key for a run: %v", err)
	}
	if _, err := s.accountKey(context.Background(), DeploymentActor{ID: "bob", Role: "operator"}, c.ID); err == nil {
		t.Fatal("another user's run read the account key")
	}
	if err = s.RevokeCredential(alice, c.ID); err != nil {
		t.Fatal(err)
	}
	if _, err := s.accountKey(context.Background(), alice, c.ID); err == nil {
		t.Fatal("a revoked account key was read")
	}
}

// A key file a case names is bound per label to the caller's own account key;
// the run writes it to a private file and points the case there.
func TestWebAttachBindsKeyFileAccountsToPrivateCredentials(t *testing.T) {
	a := DeploymentActor{ID: "operator", Role: "operator"}
	e, p, state := attachFixture(t, attachCase(`,"accounts":{"payer":{"keyFile":"${GSTABLE_TESTNET_KEY_FILE}"}}`))
	key, err := e.documents.SaveCredential(a, DeploymentCredentialInput{Label: "payer", Kind: "account-key", AccountKey: testAccountKey})
	if err != nil {
		t.Fatal(err)
	}
	login, err := e.documents.SaveCredential(a, DeploymentCredentialInput{Label: "login", Kind: "password", SSHUser: "ssh", Password: "pw"})
	if err != nil {
		t.Fatal(err)
	}
	foreign, err := e.documents.SaveCredential(DeploymentActor{ID: "bob", Role: "operator"}, DeploymentCredentialInput{Label: "payer", Kind: "account-key", AccountKey: testAccountKey})
	if err != nil {
		t.Fatal(err)
	}
	for name, bindings := range map[string]map[string]string{
		"unbound":    nil,
		"ssh login":  {"payer": login.ID},
		"foreign":    {"payer": foreign.ID},
		"undeclared": {"payer": key.ID, "other": key.ID},
	} {
		p.Input.AccountBindings = bindings
		if err := e.prepareTestAttach(context.Background(), a, &p, state); err == nil {
			t.Errorf("%s account binding accepted", name)
		}
	}
	p.Input.AccountBindings = map[string]string{"payer": key.ID}
	if err := e.prepareTestAttach(context.Background(), a, &p, state); err != nil {
		t.Fatalf("own account key refused: %v", err)
	}
	dir := t.TempDir()
	contents, err := e.writeAccountKeys(context.Background(), a, p, dir)
	if err != nil {
		t.Fatal(err)
	}
	var spec struct {
		ChainPreset struct {
			Accounts map[string]struct {
				KeyFile string `json:"keyFile"`
			} `json:"accounts"`
		} `json:"chainPreset"`
	}
	if err = json.Unmarshal(contents[0], &spec); err != nil {
		t.Fatal(err)
	}
	path := spec.ChainPreset.Accounts["payer"].KeyFile
	if filepath.Dir(path) != dir {
		t.Fatalf("the case still names a server path: %q", path)
	}
	info, err := os.Stat(path)
	if err != nil || info.Mode().Perm() != 0600 {
		t.Fatalf("account key file is not private: %v %v", info, err)
	}
	if raw, _ := os.ReadFile(path); strings.TrimSpace(string(raw)) != testAccountKey {
		t.Fatal("the written key differs from the credential")
	}
}
