package app

import (
	"encoding/json"
	"errors"
	"github.com/0xmhha/chainbench/internal/resource"
	"os"
	"path/filepath"
	"strings"
	"sync"
	"testing"
)

func deploymentTestSet() DeploymentDocumentInput {
	return DeploymentDocumentInput{Kind: "server-set", Name: "Pool", ContractVersion: "2", Content: json.RawMessage(`{"version":2,"pool":{"hosts":[{"name":"ssh","addr":"localhost."}],"slots":2,"ports":{"p2p":{"base":31000,"step":10},"rpc":{"base":8600,"step":10}}}}`)}
}
func deploymentTestConfig() DeploymentDocumentInput {
	return DeploymentDocumentInput{Kind: "workspace-config", Name: "Paths", ContractVersion: "2", Content: json.RawMessage(`{"version":1,"dataRoot":"/data/chainbench","paths":{"binaries":"binaries","configs":"configs","genesis":"genesis","keystore":"keystore","keyrings":"keyrings","nodes":"nodes","runtime":"runtime","logs":"logs"},"control":{"artifactRoot":"chainbench-out"},"inputs":{"mode":"generated"},"execution":{"chain":"fresh"}}`)}
}
func TestDeploymentValidationUsesResource(t *testing.T) {
	for _, in := range []DeploymentDocumentInput{deploymentTestSet(), deploymentTestConfig()} {
		if err := ValidateDeploymentDocument(in); err != nil {
			t.Fatal(err)
		}
	}
	for _, change := range []struct{ from, to string }{
		{`"step":10`, `"step":-1`}, {`"base":31000`, `"base":8600`}, {`"slots":2`, `"slots":999999999`},
		{`"addr":"localhost."`, `"addr":"localhost.","unknown":"value"`},
		{`"version":2`, `"version":2,"ssh":{"password":"secret"}`},
		{`"hosts":[{"name":"ssh","addr":"localhost."}]`, `"hosts":[{"name":"ssh","addr":"localhost."},{"name":"other","addr":"localhost."}]`},
	} {
		in := deploymentTestSet()
		in.Content = json.RawMessage(strings.Replace(string(in.Content), change.from, change.to, 1))
		if err := ValidateDeploymentDocument(in); err == nil {
			t.Errorf("accepted invalid set: %s", change.to)
		}
	}
	for _, change := range []struct{ from, to string }{{`/data/chainbench`, `relative`}, {`"nodes":"nodes"`, `"nodes":"../nodes"`}, {`"mode":"generated"`, `"mode":"unknown"`}, {`"chain":"fresh"`, `"chain":"unknown"`}} {
		in := deploymentTestConfig()
		in.Content = json.RawMessage(strings.Replace(string(in.Content), change.from, change.to, 1))
		if err := ValidateDeploymentDocument(in); err == nil {
			t.Errorf("accepted invalid config: %s", change.to)
		}
	}
}
func TestDeploymentRevisionsPrivateOverlayAndRestart(t *testing.T) {
	root := t.TempDir()
	s, err := OpenDeploymentStore(root)
	if err != nil {
		t.Fatal(err)
	}
	alice := DeploymentActor{ID: "alice", Role: "operator"}
	bob := DeploymentActor{ID: "bob", Role: "operator"}
	d, err := s.SaveDocument(alice, "", 0, deploymentTestSet())
	if err != nil {
		t.Fatal(err)
	}
	config, err := s.SaveDocument(bob, "", 0, deploymentTestConfig())
	if err != nil {
		t.Fatal(err)
	}
	in := DeploymentWorkspaceInput{Name: "Team", Documents: []DeploymentDocumentRef{{d.ID, d.Revision}, {config.ID, config.Revision}}}
	w, err := s.SaveWorkspace(alice, "", 0, in)
	if err != nil {
		t.Fatal(err)
	}
	c, err := s.SaveCredential(alice, DeploymentCredentialInput{Label: "private", Kind: "password", SSHUser: "ssh", Password: "unique-secret-marker"})
	if err != nil {
		t.Fatal(err)
	}
	if err = s.BindCredential(alice, w.ID, "ssh", c.ID); err != nil {
		t.Fatal(err)
	}
	if err = s.BindCredential(bob, w.ID, "ssh", c.ID); !errors.Is(err, ErrDeploymentNotFound) {
		t.Fatalf("foreign binding: %v", err)
	}
	if len(s.Credentials(bob)) != 0 || len(s.Bindings(bob, w.ID)) != 0 {
		t.Fatal("private overlay exposed")
	}
	b, err := os.ReadFile(filepath.Join(root, "deployment.json"))
	if err != nil {
		t.Fatal(err)
	}
	if strings.Contains(string(b), "unique-secret-marker") {
		t.Fatal("plaintext secret persisted")
	}
	_, err = s.SaveDocument(bob, d.ID, 1, deploymentTestSet())
	if err != nil {
		t.Fatal(err)
	}
	if _, err = s.SaveDocument(alice, d.ID, 1, deploymentTestSet()); !errors.Is(err, ErrDeploymentConflict) {
		t.Fatal("stale edit accepted")
	}
	// Existing workspaces keep immutable refs until an explicit shared update.
	s, err = OpenDeploymentStore(root)
	if err != nil {
		t.Fatal(err)
	}
	restored, err := s.Workspace(w.ID)
	if err != nil || restored.Documents[0].Revision != 1 {
		t.Fatal("pinned revision lost")
	}
	s.mu.Lock()
	_, _, secret, err := s.resolveAccess(alice, w.ID, "ssh", c.ID)
	s.mu.Unlock()
	if err != nil || secret.Password != "unique-secret-marker" {
		t.Fatal("encrypted credential did not survive restart")
	}
	if s.Bindings(alice, w.ID)["ssh"] != c.ID {
		t.Fatal("binding lost")
	}
}
func TestDeploymentConcurrentRevisionGuard(t *testing.T) {
	s, err := OpenDeploymentStore(t.TempDir())
	if err != nil {
		t.Fatal(err)
	}
	a := DeploymentActor{ID: "operator", Role: "operator"}
	d, err := s.SaveDocument(a, "", 0, deploymentTestSet())
	if err != nil {
		t.Fatal(err)
	}
	var wg sync.WaitGroup
	results := make(chan error, 2)
	for i := 0; i < 2; i++ {
		wg.Go(func() { _, err := s.SaveDocument(a, d.ID, 1, deploymentTestSet()); results <- err })
	}
	wg.Wait()
	close(results)
	successes, conflicts := 0, 0
	for err := range results {
		if err == nil {
			successes++
		} else if errors.Is(err, ErrDeploymentConflict) {
			conflicts++
		} else {
			t.Fatal(err)
		}
	}
	if successes != 1 || conflicts != 1 {
		t.Fatalf("success=%d conflict=%d", successes, conflicts)
	}
}
func TestDeploymentFailedPersistenceDoesNotPublish(t *testing.T) {
	root := t.TempDir()
	s, err := OpenDeploymentStore(root)
	if err != nil {
		t.Fatal(err)
	}
	a := DeploymentActor{ID: "operator", Role: "operator"}
	d, err := s.SaveDocument(a, "", 0, deploymentTestSet())
	if err != nil {
		t.Fatal(err)
	}
	if err = os.RemoveAll(root); err != nil {
		t.Fatal(err)
	}
	if _, err = s.SaveDocument(a, d.ID, 1, deploymentTestSet()); err == nil {
		t.Fatal("disk failure ignored")
	}
	read, err := s.Document(d.ID)
	if err != nil || read.Revision != 1 {
		t.Fatal("uncommitted state visible")
	}
}
func TestDeploymentContractContainsPublicEngineFields(t *testing.T) {
	contract := DeploymentContract()
	ssh := contract["server-set"].Properties["ssh"]
	for _, field := range []string{"password", "password_file", "key_file", "key_passphrase_file"} {
		if _, ok := ssh.Properties[field]; ok {
			t.Fatal("shared secret field exposed")
		}
	}
	for _, field := range []string{"version", "pool", "ssh"} {
		if _, ok := contract["server-set"].Properties[field]; !ok {
			t.Fatal("missing field", field)
		}
	}
	if len(contract["workspace-config"].Properties["paths"].Properties) != 8 {
		t.Fatal("path coverage changed")
	}
}

func TestDeploymentReturnedObjectsCannotChangePinnedState(t *testing.T) {
	s, err := OpenDeploymentStore(t.TempDir())
	if err != nil {
		t.Fatal(err)
	}
	a := DeploymentActor{ID: "operator", Role: "operator"}
	in := deploymentTestSet()
	d, err := s.SaveDocument(a, "", 0, in)
	if err != nil {
		t.Fatal(err)
	}
	d.Content[0] = 'x'
	in.Content[0] = 'x'
	read, err := s.DocumentRevision(d.ID, 1)
	if err != nil || !json.Valid(read.Content) {
		t.Fatal("caller modified immutable revision")
	}
	if _, err = s.DocumentRevision(d.ID, 2); !errors.Is(err, ErrDeploymentNotFound) {
		t.Fatal("nonexistent revision accepted")
	}
	if _, err = s.SaveDocument(a, d.ID, 1, deploymentTestSet()); err != nil {
		t.Fatal(err)
	}
	pinned, err := s.DocumentRevision(d.ID, 1)
	if err != nil || pinned.Revision != 1 {
		t.Fatal("historic revision lost")
	}
}

func TestDeploymentMissingKeyNeverReplacesExistingCiphertext(t *testing.T) {
	root := t.TempDir()
	s, err := OpenDeploymentStore(root)
	if err != nil {
		t.Fatal(err)
	}
	a := DeploymentActor{ID: "operator", Role: "operator"}
	if _, err = s.SaveCredential(a, DeploymentCredentialInput{Label: "private", Kind: "password", SSHUser: "ssh", Password: "secret"}); err != nil {
		t.Fatal(err)
	}
	if err = os.Remove(filepath.Join(root, "credential.key")); err != nil {
		t.Fatal(err)
	}
	if _, err = OpenDeploymentStore(root); err == nil {
		t.Fatal("lost encryption key silently replaced")
	}
	if _, err = os.Stat(filepath.Join(root, "credential.key")); !os.IsNotExist(err) {
		t.Fatal("a replacement key was created")
	}
}

func TestDeploymentExportsEngineDeclarationInBothFormats(t *testing.T) {
	for _, in := range []DeploymentDocumentInput{deploymentTestSet(), deploymentTestConfig()} {
		for _, format := range []string{"json", "yaml"} {
			b, err := ExportDeploymentDocument(in, format)
			if err != nil {
				t.Fatal(err)
			}
			if strings.Contains(string(b), "ownerId") || strings.Contains(string(b), "contractVersion") {
				t.Fatal("Web metadata in engine export")
			}
			if in.Kind == "server-set" {
				if _, err = resource.ParseSet(b); err != nil {
					t.Fatal(err)
				}
			} else {
				if _, err = resource.ParseWorkspaceConfig(b); err != nil {
					t.Fatal(err)
				}
			}
		}
	}
	if _, err := ExportDeploymentDocument(deploymentTestSet(), "unsupported"); err == nil {
		t.Fatal("unknown export format accepted")
	}
}
