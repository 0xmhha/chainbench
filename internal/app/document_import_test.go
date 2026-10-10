package app

import (
	"bytes"
	"encoding/json"
	"errors"
	"os"
	"path/filepath"
	"testing"
)

func TestImportPreservesEncryptedSourceAndRequiresOwnerAndRevision(t *testing.T) {
	root := t.TempDir()
	store, err := OpenDeploymentStore(root)
	if err != nil {
		t.Fatal(err)
	}
	actor := DeploymentActor{ID: "first", Role: "operator"}
	source := `version: 2
pool:
  hosts: [{name: target, addr: localhost.}]
  slots: 1
  ports:
    p2p: {base: 31000, step: 10}
    rpc: {base: 8600, step: 10}
ssh:
  user: test
  password: imported-secret-sentinel
`
	preview, err := store.PreviewImport(actor, DocumentImportInput{Filename: "server-set.yaml", Format: "yaml", Kind: "server-set", Source: source})
	if err != nil || !preview.Validation.Valid || !preview.SourcePreserved {
		t.Fatalf("%+v %v", preview, err)
	}
	if len(preview.PrivateBindingsRequired) != 1 {
		t.Fatal("missing private SSH binding requirement")
	}
	public, _ := json.Marshal(preview)
	if bytes.Contains(public, []byte("imported-secret-sentinel")) {
		t.Fatal("secret in public preview")
	}
	if _, err = store.CommitImport(DeploymentActor{ID: "second", Role: "admin"}, DocumentImportCommit{PreviewID: preview.PreviewID}); !errors.Is(err, ErrDeploymentNotFound) {
		t.Fatalf("other user committed import: %v", err)
	}
	created, err := store.CommitImport(actor, DocumentImportCommit{PreviewID: preview.PreviewID})
	if err != nil || len(created) != 1 {
		t.Fatalf("%v %v", created, err)
	}
	if _, err = store.CommitImport(actor, DocumentImportCommit{PreviewID: preview.PreviewID}); !errors.Is(err, ErrDeploymentConflict) {
		t.Fatal("preview replay allowed")
	}
	reopened, err := OpenDeploymentStore(root)
	if err != nil {
		t.Fatal(err)
	}
	entry := reopened.state.Imports[preview.PreviewID]
	ciphertext := entry.Ciphertext
	nonceSize := reopened.aead.NonceSize()
	original, err := reopened.aead.Open(nil, ciphertext[:nonceSize], ciphertext[nonceSize:], []byte(preview.PreviewID+":"+actor.ID))
	if err != nil || string(original) != source {
		t.Fatal("original source not preserved")
	}
	persisted, err := os.ReadFile(filepath.Join(root, "deployment.json"))
	if err != nil {
		t.Fatal(err)
	}
	if bytes.Contains(persisted, []byte("imported-secret-sentinel")) {
		t.Fatal("secret stored in plaintext")
	}
	second, err := reopened.PreviewImport(actor, DocumentImportInput{Filename: "server-set.yaml", Format: "yaml", Kind: "server-set", Source: source})
	if err != nil {
		t.Fatal(err)
	}
	if _, err = reopened.CommitImport(actor, DocumentImportCommit{PreviewID: second.PreviewID, BaseRevisions: map[string]int{created[0].ID: 2}}); !errors.Is(err, ErrDeploymentConflict) {
		t.Fatal("stale revision accepted")
	}
	updated, err := reopened.CommitImport(actor, DocumentImportCommit{PreviewID: second.PreviewID, BaseRevisions: map[string]int{created[0].ID: 1}})
	if err != nil || updated[0].ID != created[0].ID || updated[0].Revision != 2 {
		t.Fatalf("%+v %v", updated, err)
	}
}

func TestInvalidImportCannotBecomeSharedConfiguration(t *testing.T) {
	store, err := OpenDeploymentStore(t.TempDir())
	if err != nil {
		t.Fatal(err)
	}
	actor := DeploymentActor{ID: "operator", Role: "operator"}
	raw := editorCase(`[{"expect":"unknownExtension","is":1}]`)
	preview, err := store.PreviewImport(actor, DocumentImportInput{Filename: "invalid.json", Format: "json", Kind: "case", Source: string(raw)})
	if err != nil || preview.Validation.Valid || len(preview.RedactedDocuments) != 0 {
		t.Fatalf("%+v %v", preview, err)
	}
	if _, err = store.CommitImport(actor, DocumentImportCommit{PreviewID: preview.PreviewID}); err == nil {
		t.Fatal("invalid import saved")
	}
	if len(store.Documents("")) != 0 {
		t.Fatal("failed import modified documents")
	}
	if _, err = store.PreviewImport(DeploymentActor{ID: "reader", Role: "viewer"}, DocumentImportInput{Filename: "x", Format: "json", Source: string(raw), Kind: "case"}); !errors.Is(err, ErrDeploymentForbidden) {
		t.Fatal("viewer created preview")
	}
}

func TestSharedCaseAndChainPresetRoundTrip(t *testing.T) {
	store, err := OpenDeploymentStore(t.TempDir())
	if err != nil {
		t.Fatal(err)
	}
	a := DeploymentActor{ID: "operator", Role: "operator"}
	for kind, raw := range map[string][]byte{
		"case":         editorCase(`[{"do":"read","source":"blockNumber","save":"height"},{"do":"waitBlock","target":"${height}"},{"expect":"blockNumber","is":"$height"}]`),
		"chain-preset": []byte(`{"schemaVersion":"2","kind":"chain-preset","id":"shared","chain":"stablenet","topology":{"bp":4}}`),
	} {
		d, err := store.SaveDocument(a, "", 0, DeploymentDocumentInput{Kind: kind, Name: kind, ContractVersion: "2", Content: raw})
		if err != nil {
			t.Fatal(err)
		}
		exported, err := ExportDeploymentDocument(d.DeploymentDocumentInput, "json")
		if err != nil || !bytes.Equal(exported, raw) {
			t.Fatal("shared declaration changed", err)
		}
		if _, err = store.SaveDocument(a, d.ID, 2, d.DeploymentDocumentInput); !errors.Is(err, ErrDeploymentConflict) {
			t.Fatal("revision guard missing")
		}
	}
}
