package app

import (
	"bytes"
	"encoding/json"
	"os"
	"path/filepath"
	"strconv"
	"strings"
	"testing"

	"github.com/0xmhha/chainbench/internal/dsl"
)

func nodeKeyDocuments(t *testing.T, key string) []DeploymentDocumentInput {
	t.Helper()
	quoted, err := json.Marshal(key)
	if err != nil {
		t.Fatal(err)
	}
	topology := `{"nodes":[{"index":1,"role":"bp","key":` + string(quoted) + `}]}`
	return []DeploymentDocumentInput{
		{Kind: "chain-preset", Name: "preset", ContractVersion: "2", Content: json.RawMessage(`{"schemaVersion":"2","kind":"chain-preset","id":"private-node","chain":"stablenet","topology":` + topology + `}`)},
		{Kind: "case", Name: "case-v2", ContractVersion: "2", Content: json.RawMessage(`{"schemaVersion":"2","kind":"case","id":"private-node","chainPreset":{"chain":"stablenet","topology":` + topology + `},"steps":[{"expect":"blockNumber","is":0}]}`)},
		{Kind: "case", Name: "case-v1", ContractVersion: "2", Content: json.RawMessage(`{"schemaVersion":"1","id":"private-node","chain":{"name":"stablenet","binary":"gstable"},"topology":` + topology + `,"assertions":[{"assert":"blockNumber","expected":0}]}`)},
	}
}

func TestSharedDocumentsRefuseNodeKeyMaterial(t *testing.T) {
	for _, key := range []string{strings.Repeat("ab", 32), "0x" + strings.Repeat("ab", 32), " \n0X" + strings.Repeat("AB", 32) + "\t", strings.Repeat("a", 32), strings.Repeat("b", 66)} {
		for _, in := range nodeKeyDocuments(t, key) {
			t.Run(in.Name+"/"+strconv.Itoa(len(key)), func(t *testing.T) {
				// The DSL parser retains its existing grammar; the shared Web
				// boundary must reject material before publishing it.
				if in.Kind == "chain-preset" {
					if _, err := dsl.ParseChainPreset(in.Content); err != nil {
						t.Fatal("CLI grammar changed", err)
					}
				} else if _, err := dsl.Parse(in.Content); err != nil {
					t.Fatal("CLI grammar changed", err)
				}
				store, err := OpenDeploymentStore(t.TempDir())
				if err != nil {
					t.Fatal(err)
				}
				if _, err = store.SaveDocument(DeploymentActor{ID: "operator", Role: "operator"}, "", 0, in); err == nil {
					t.Fatal("node key material published in a shared document")
				} else if strings.Contains(err.Error(), strings.TrimSpace(key)) {
					t.Fatal("node key material exposed by validation error")
				}
				for _, format := range []string{"json", "yaml"} {
					if exported, err := ExportDeploymentDocument(in, format); err == nil || len(exported) != 0 {
						t.Fatal("node key material exported")
					}
				}
				if len(store.Documents("")) != 0 {
					t.Fatal("rejected document changed shared state")
				}
			})
		}
	}
}

func TestNodeKeyImportPreservesOnlyEncryptedSource(t *testing.T) {
	key := "0x" + strings.Repeat("ba", 32)
	for _, in := range nodeKeyDocuments(t, key) {
		t.Run(in.Name, func(t *testing.T) {
			root := t.TempDir()
			store, err := OpenDeploymentStore(root)
			if err != nil {
				t.Fatal(err)
			}
			a := DeploymentActor{ID: "operator", Role: "operator"}
			preview, err := store.PreviewImport(a, DocumentImportInput{Filename: "private.json", Format: "json", Kind: in.Kind, Source: string(in.Content)})
			if err != nil || preview.Validation.Valid || len(preview.RedactedDocuments) != 0 || !preview.SourcePreserved {
				t.Fatal("private key import was publishable", err)
			}
			public, _ := json.Marshal(preview)
			if bytes.Contains(public, []byte(key)) {
				t.Fatal("private key in public preview")
			}
			if _, err = store.CommitImport(a, DocumentImportCommit{PreviewID: preview.PreviewID}); err == nil {
				t.Fatal("private key import committed")
			}
			reopened, err := OpenDeploymentStore(root)
			if err != nil {
				t.Fatal(err)
			}
			entry := reopened.state.Imports[preview.PreviewID]
			n := reopened.aead.NonceSize()
			original, err := reopened.aead.Open(nil, entry.Ciphertext[:n], entry.Ciphertext[n:], []byte(preview.PreviewID+":"+a.ID))
			if err != nil || !bytes.Equal(original, in.Content) {
				t.Fatal("encrypted original lost", err)
			}
			persisted, err := os.ReadFile(filepath.Join(root, "deployment.json"))
			if err != nil || bytes.Contains(persisted, []byte(key)) || len(reopened.Documents("")) != 0 {
				t.Fatal("key persisted in public plaintext", err)
			}
		})
	}
}

func TestSharedNodeKeyFileReferencesAndPublicHexRemainValid(t *testing.T) {
	for _, key := range []string{"keys/node1.key", "/private/keys/node1.hex", "keys/" + strings.Repeat("ab", 32) + ".key", "${NODE_KEY_FILE}"} {
		for _, in := range nodeKeyDocuments(t, key) {
			if err := ValidateDeploymentDocument(in); err != nil {
				t.Fatalf("file reference %s refused: %v", in.Name, err)
			}
		}
	}
	in := nodeKeyDocuments(t, "keys/node1.key")[0]
	var content map[string]any
	if err := json.Unmarshal(in.Content, &content); err != nil {
		t.Fatal(err)
	}
	content["description"] = "0x" + strings.Repeat("ab", 32)
	in.Content, _ = json.Marshal(content)
	if err := ValidateDeploymentDocument(in); err != nil {
		t.Fatal("public hex text was treated as a private key", err)
	}
}

func TestImportCommitRevalidatesPreviouslyApprovedDocuments(t *testing.T) {
	store, err := OpenDeploymentStore(t.TempDir())
	if err != nil {
		t.Fatal(err)
	}
	a := DeploymentActor{ID: "operator", Role: "operator"}
	public := nodeKeyDocuments(t, "keys/node1.key")[0]
	preview, err := store.PreviewImport(a, DocumentImportInput{Filename: "preset.json", Format: "json", Kind: public.Kind, Source: string(public.Content)})
	if err != nil || !preview.Validation.Valid {
		t.Fatal("public reference import invalid", err)
	}
	// Simulate an approval stored by an older validator. Commit must enforce
	// the current publication rules, rather than trusting the saved boolean.
	entry := store.state.Imports[preview.PreviewID]
	entry.Preview.RedactedDocuments = []DeploymentDocumentInput{nodeKeyDocuments(t, "0x"+strings.Repeat("ca", 32))[0]}
	store.state.Imports[preview.PreviewID] = entry
	if _, err = store.CommitImport(a, DocumentImportCommit{PreviewID: preview.PreviewID}); err == nil {
		t.Fatal("old approval bypassed current shared-document validation")
	}
	if len(store.Documents("")) != 0 {
		t.Fatal("invalid old approval changed shared state")
	}
}

func TestWebEditorRefusesPrivateReferencedPreset(t *testing.T) {
	key := "0x" + strings.Repeat("db", 32)
	input := TestCaseInput{
		Content: json.RawMessage(`{"schemaVersion":"2","kind":"case","id":"reference","chainPreset":"private-node","steps":[{"expect":"blockNumber","is":0}]}`),
		Presets: map[string]json.RawMessage{"private-node": nodeKeyDocuments(t, key)[0].Content},
	}
	if _, err := PrepareTestCase(input); err != nil {
		t.Fatal("existing declaration grammar changed", err)
	}
	if prepared, err := PrepareWebTestCase(input); err == nil || len(prepared.Content) != 0 || strings.Contains(err.Error(), key) {
		t.Fatal("private referenced preset approved by Web editor")
	}
	input.Presets["private-node"] = nodeKeyDocuments(t, "keys/node1.key")[0].Content
	if _, err := PrepareWebTestCase(input); err != nil {
		t.Fatal("referenced key file refused", err)
	}
}
