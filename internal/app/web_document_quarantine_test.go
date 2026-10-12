package app

import (
	"bytes"
	"crypto/rand"
	"encoding/json"
	"errors"
	"os"
	"path/filepath"
	"strings"
	"testing"
	"time"
)

func legacyNodeKeyFixture(t *testing.T, documentIndex int) (string, []byte, []DeploymentDocument, string) {
	t.Helper()
	root := t.TempDir()
	store, err := OpenDeploymentStore(root)
	if err != nil {
		t.Fatal(err)
	}
	a := DeploymentActor{ID: "operator", Role: "operator"}
	if _, err = store.SaveDocument(a, "", 0, deploymentTestSet()); err != nil {
		t.Fatal(err)
	}
	key := "0x" + strings.Repeat("de", 32)
	public := nodeKeyDocuments(t, "keys/node1.key")[documentIndex]
	history := []DeploymentDocument{
		{DeploymentDocumentInput: public, ID: "legacy-preset", Revision: 1, UpdatedBy: a.ID},
		{DeploymentDocumentInput: nodeKeyDocuments(t, key)[documentIndex], ID: "legacy-preset", Revision: 2, UpdatedBy: a.ID},
	}
	store.state.Documents["legacy-preset"] = history
	// Preserve an old approval whose public preview still contained key bytes.
	preview, err := store.PreviewImport(a, DocumentImportInput{Filename: "preset.json", Format: "json", Kind: public.Kind, Source: string(public.Content)})
	if err != nil {
		t.Fatal(err)
	}
	entry := store.state.Imports[preview.PreviewID]
	entry.ExpiresAt = time.Now().Add(time.Hour)
	entry.Preview.RedactedDocuments = []DeploymentDocumentInput{history[1].DeploymentDocumentInput}
	nonce := make([]byte, store.aead.NonceSize())
	if _, err = rand.Read(nonce); err != nil {
		t.Fatal(err)
	}
	entry.Ciphertext = store.aead.Seal(nonce, nonce, history[1].Content, []byte(preview.PreviewID+":"+a.ID))
	store.state.Imports[preview.PreviewID] = entry
	raw, err := json.Marshal(store.state)
	if err != nil || !bytes.Contains(raw, []byte(key)) {
		t.Fatal("legacy fixture does not contain plaintext node key", err)
	}
	if err = store.storage.WriteDeployment(raw); err != nil {
		t.Fatal(err)
	}
	return root, raw, history, preview.PreviewID
}

func TestOpenDeploymentStoreQuarantinesLegacyNodeKeyPublication(t *testing.T) {
	root, _, history, previewID := legacyNodeKeyFixture(t, 0)
	store, err := OpenDeploymentStore(root)
	if err != nil {
		t.Fatal(err)
	}
	for _, revision := range []int{0, 1, 2} {
		if _, err = store.DocumentRevision(history[0].ID, revision); !errors.Is(err, ErrDeploymentNotFound) {
			t.Errorf("legacy node-key history publicly readable at revision %d", revision)
		}
	}
	if len(store.Documents("chain-preset")) != 0 || len(store.Documents("server-set")) != 1 {
		t.Fatal("unsafe document remained public or unaffected document disappeared")
	}
	entry := store.state.Imports[previewID]
	if entry.Preview.Validation.Valid || len(entry.Preview.RedactedDocuments) != 0 {
		t.Fatal("legacy key-containing import preview remained publishable")
	}
	if _, err = store.CommitImport(DeploymentActor{ID: "operator", Role: "operator"}, DocumentImportCommit{PreviewID: previewID}); err == nil {
		t.Fatal("quarantined approval committed")
	}
	raw, err := os.ReadFile(filepath.Join(root, "deployment.json"))
	if err != nil || bytes.Contains(raw, []byte("0x"+strings.Repeat("de", 32))) {
		t.Fatal("legacy key remained in plaintext storage", err)
	}
	if _, err = OpenDeploymentStore(root); err != nil {
		t.Fatal("quarantine did not survive restart", err)
	}
}

func TestQuarantinePreservesHistoryAndImportSourceWithoutRewritingSafeDocuments(t *testing.T) {
	for index, name := range []string{"preset", "case-v2", "case-v1"} {
		t.Run(name, func(t *testing.T) {
			root, raw, history, previewID := legacyNodeKeyFixture(t, index)
			var before deploymentState
			if err := json.Unmarshal(raw, &before); err != nil {
				t.Fatal(err)
			}
			store, err := OpenDeploymentStore(root)
			if err != nil {
				t.Fatal(err)
			}
			entry, ok := store.state.QuarantinedDocuments[history[0].ID]
			if !ok || entry.Version != 1 || entry.Revisions != 2 {
				t.Fatal("legacy history was not preserved")
			}
			n := store.aead.NonceSize()
			original, err := store.aead.Open(nil, entry.Ciphertext[:n], entry.Ciphertext[n:], documentQuarantineAAD(history[0].ID))
			want, _ := json.Marshal(history)
			if err != nil || !bytes.Equal(original, want) {
				t.Fatal("original revisions were changed or lost", err)
			}
			if _, err = store.aead.Open(nil, entry.Ciphertext[:n], entry.Ciphertext[n:], documentQuarantineAAD("another-document")); err == nil {
				t.Fatal("quarantine was not bound to its document")
			}
			ciphertext := store.state.Imports[previewID].Ciphertext
			if !bytes.Equal(ciphertext, before.Imports[previewID].Ciphertext) {
				t.Fatal("actor-private imported source was overwritten")
			}
			original, err = store.aead.Open(nil, ciphertext[:n], ciphertext[n:], []byte(previewID+":operator"))
			if err != nil || !bytes.Equal(original, history[1].Content) {
				t.Fatal("original imported key source was lost", err)
			}
			quarantinedPreview := store.state.Imports[previewID].QuarantinedPreview
			original, err = store.aead.Open(nil, quarantinedPreview[:n], quarantinedPreview[n:], importQuarantineAAD(previewID, "operator"))
			want, _ = json.Marshal(before.Imports[previewID].Preview)
			if err != nil || !bytes.Equal(original, want) {
				t.Fatal("legacy preview lost during quarantine", err)
			}
			if _, err = store.aead.Open(nil, quarantinedPreview[:n], quarantinedPreview[n:], importQuarantineAAD(previewID, "another-actor")); err == nil {
				t.Fatal("legacy preview not bound to its owner")
			}
			for id, revisions := range before.Documents {
				if id != history[0].ID {
					old, _ := json.Marshal(revisions)
					now, _ := json.Marshal(store.state.Documents[id])
					if !bytes.Equal(old, now) {
						t.Fatal("unaffected shared document changed")
					}
				}
			}
			persisted, err := os.ReadFile(filepath.Join(root, "deployment.json"))
			if err != nil || bytes.Contains(persisted, []byte("0x"+strings.Repeat("de", 32))) {
				t.Fatal("legacy material not encrypted", err)
			}
			if _, err = OpenDeploymentStore(root); err != nil {
				t.Fatal(err)
			}
			reopened, _ := os.ReadFile(filepath.Join(root, "deployment.json"))
			if !bytes.Equal(persisted, reopened) {
				t.Fatal("reopening quarantine rewrote state or audit")
			}
		})
	}
}

func TestQuarantineTamperingFailsBeforeOpeningStore(t *testing.T) {
	root, _, _, _ := legacyNodeKeyFixture(t, 0)
	store, err := OpenDeploymentStore(root)
	if err != nil {
		t.Fatal(err)
	}
	entry := store.state.QuarantinedDocuments["legacy-preset"]
	entry.Ciphertext[len(entry.Ciphertext)-1] ^= 1
	store.state.QuarantinedDocuments["legacy-preset"] = entry
	raw, _ := json.Marshal(store.state)
	if err = store.storage.WriteDeployment(raw); err != nil {
		t.Fatal(err)
	}
	if published, err := OpenDeploymentStore(root); err == nil || published != nil {
		t.Fatal("corrupt quarantine opened for public use")
	}
}

func TestQuarantineWriteFailureDoesNotPublishChangedState(t *testing.T) {
	root, original, _, _ := legacyNodeKeyFixture(t, 0)
	store, err := OpenDeploymentStore(root)
	if err != nil {
		t.Fatal(err)
	}
	var legacy deploymentState
	if err = json.Unmarshal(original, &legacy); err != nil {
		t.Fatal(err)
	}
	store.state = legacy
	path := filepath.Join(root, "deployment.json")
	if err = os.Remove(path); err != nil {
		t.Fatal(err)
	}
	if err = os.Mkdir(path, 0700); err != nil {
		t.Fatal(err)
	}
	if err = store.quarantineLegacyNodeKeys(); err == nil {
		t.Fatal("failed atomic publication reported success")
	}
	after, _ := json.Marshal(store.state)
	if !bytes.Equal(after, original) {
		t.Fatal("uncommitted quarantine changed memory")
	}
	if opened, err := OpenDeploymentStore(root); err == nil || opened != nil {
		t.Fatal("unreadable snapshot opened publicly")
	}
}

func TestQuarantinedNodeKeysAreRedactedFromHistoricalJSONAndSSE(t *testing.T) {
	root, _, _, _ := legacyNodeKeyFixture(t, 0)
	store, err := OpenDeploymentStore(root)
	if err != nil {
		t.Fatal(err)
	}
	key := "0x" + strings.Repeat("de", 32)
	for _, value := range []string{key, strings.TrimPrefix(key, "0x")} {
		for _, text := range []string{`{"message":"previous failure ` + value + `"}`, "data: {\"message\":\"" + value + "\"}\n\n", "old log: " + value} {
			if strings.Contains(store.RedactWeb(text), value) {
				t.Fatal("quarantined node key still readable in historical output")
			}
		}
	}
}

func TestImportOnlyNodeKeyQuarantineRedactsAcrossRestart(t *testing.T) {
	root, raw, history, _ := legacyNodeKeyFixture(t, 0)
	var state deploymentState
	if err := json.Unmarshal(raw, &state); err != nil {
		t.Fatal(err)
	}
	delete(state.Documents, history[0].ID)
	raw, _ = json.Marshal(state)
	if err := os.WriteFile(filepath.Join(root, "deployment.json"), raw, 0600); err != nil {
		t.Fatal(err)
	}
	for range 2 {
		store, err := OpenDeploymentStore(root)
		if err != nil {
			t.Fatal(err)
		}
		key := "0x" + strings.Repeat("de", 32)
		if strings.Contains(store.RedactWeb("previous import error: "+key), key) {
			t.Fatal("private key from a legacy import leaked after quarantine or restart")
		}
	}
}

func TestImportQuarantineTamperingFailsBeforeOpeningStore(t *testing.T) {
	root, _, _, previewID := legacyNodeKeyFixture(t, 0)
	store, err := OpenDeploymentStore(root)
	if err != nil {
		t.Fatal(err)
	}
	entry := store.state.Imports[previewID]
	entry.QuarantinedPreview[len(entry.QuarantinedPreview)-1] ^= 1
	store.state.Imports[previewID] = entry
	raw, _ := json.Marshal(store.state)
	if err = store.storage.WriteDeployment(raw); err != nil {
		t.Fatal(err)
	}
	if published, err := OpenDeploymentStore(root); err == nil || published != nil {
		t.Fatal("corrupt private import quarantine opened publicly")
	}
}
