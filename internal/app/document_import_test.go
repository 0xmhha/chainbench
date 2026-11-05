package app

import (
	"bytes"
	"encoding/json"
	"errors"
	"os"
	"path/filepath"
	"strings"
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
	// The moved password, and the remote server that needs the importer's
	// own SSH binding.
	if len(preview.PrivateBindingsRequired) != 2 || !strings.HasPrefix(preview.PrivateBindingsRequired[1], "ssh:") {
		t.Fatalf("missing private SSH binding requirement: %v", preview.PrivateBindingsRequired)
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

// A server set or workspace config written for the CLI declares no kind and the
// import screen sends none, so the document's own fields have to say what it is.
func TestPreviewImport_RecognizesAnUndeclaredCLIDeclaration(t *testing.T) {
	store, err := OpenDeploymentStore(t.TempDir())
	if err != nil {
		t.Fatal(err)
	}
	a := DeploymentActor{ID: "o", Role: "operator"}
	cases := []struct {
		name, filename, source, wantKind string
	}{
		{"server set", "server-set.yaml", "version: 2\npool:\n  hosts:\n    - name: server1\n      addr: 192.0.2.11\n  slots: 4\n  ports:\n    p2p: {base: 30301, step: 1}\n    rpc: {base: 8601, step: 1}\n    ws: {base: 8701, step: 1}\n    auth: {base: 8501, step: 1}\nssh:\n  user: test\n  password: imported-secret-sentinel\n", "server-set"},
		{"workspace config", "workspace-config.yaml", "version: 1\ndataRoot: /data/chainbench\npaths:\n  binaries: bin\n  configs: configs\n  genesis: genesis\n  keystore: keystore\n  keyrings: keyrings\n  nodes: node\n  runtime: runtime\n  logs: logs\ncontrol:\n  artifactRoot: ~/.chainbench\ninputs:\n  mode: generated\nexecution:\n  chain: fresh\n", "workspace-config"},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			preview, err := store.PreviewImport(a, DocumentImportInput{Filename: tc.filename, Format: "yaml", Source: tc.source})
			if err != nil {
				t.Fatalf("PreviewImport: %v", err)
			}
			if len(preview.RedactedDocuments) != 1 || preview.RedactedDocuments[0].Kind != tc.wantKind {
				t.Fatalf("documents = %+v, errors %+v, want one %s", preview.RedactedDocuments, preview.Validation.Errors, tc.wantKind)
			}
			if strings.Contains(string(preview.RedactedDocuments[0].Content), "imported-secret-sentinel") {
				t.Fatal("an inline ssh password reached the shared declaration")
			}
		})
	}
}

func TestPreviewImport_AnUnrecognizableDocumentNamesWhatIsMissing(t *testing.T) {
	store, err := OpenDeploymentStore(t.TempDir())
	if err != nil {
		t.Fatal(err)
	}
	a := DeploymentActor{ID: "o", Role: "operator"}
	_, err = store.PreviewImport(a, DocumentImportInput{Filename: "notes.yaml", Format: "yaml", Source: "hello: world\n"})
	if err == nil || !strings.Contains(err.Error(), "kind") {
		t.Fatalf("err = %v, want the missing kind named", err)
	}
}

// A localmap keys its ports by number. YAML keeps those keys as numbers, which
// JSON cannot hold, so a pool file with its localmap used to be refused.
func TestPreviewImport_KeepsALocalmapWrittenInYAML(t *testing.T) {
	store, err := OpenDeploymentStore(t.TempDir())
	if err != nil {
		t.Fatal(err)
	}
	source := "version: 2\npool:\n  hosts:\n    - name: server1\n      addr: 192.0.2.11\n  slots: 4\n  ports:\n    p2p: {base: 30301, step: 1}\n    rpc: {base: 8601, step: 1}\n    ws: {base: 8701, step: 1}\n    auth: {base: 8501, step: 1}\nssh:\n  user: test\n  port: 10022\nlocalmap:\n  hosts:\n    192.0.2.11:\n      host: 127.0.0.1\n      ports:\n        10022: 2201\n        8601: 18601\n"
	preview, err := store.PreviewImport(DeploymentActor{ID: "o", Role: "operator"}, DocumentImportInput{Filename: "pool.yaml", Format: "yaml", Source: source})
	if err != nil {
		t.Fatal(err)
	}
	if len(preview.RedactedDocuments) != 1 {
		t.Fatalf("errors = %+v", preview.Validation.Errors)
	}
	if !strings.Contains(string(preview.RedactedDocuments[0].Content), `"10022":2201`) {
		t.Fatalf("content = %s", preview.RedactedDocuments[0].Content)
	}
}

// The files an operator already has come apart: the pool, its localmap, the
// paths. Picking them together has to give one server set with its localmap.
func TestPreviewImport_CombinesTheFilesOfAPool(t *testing.T) {
	store, err := OpenDeploymentStore(t.TempDir())
	if err != nil {
		t.Fatal(err)
	}
	pool := "version: 2\npool:\n  hosts:\n    - name: server1\n      addr: 192.0.2.11\n  slots: 4\n  ports:\n    p2p: {base: 30301, step: 1}\n    rpc: {base: 8601, step: 1}\n    ws: {base: 8701, step: 1}\n    auth: {base: 8501, step: 1}\nssh:\n  user: test\n  port: 10022\n  password: imported-secret-sentinel\n"
	localmap := "hosts:\n  192.0.2.11:\n    host: 127.0.0.1\n    ports:\n      10022: 2201\n      8601: 18601\n"
	config := "version: 1\ndataRoot: /data/chainbench\npaths:\n  binaries: bin\n  configs: configs\n  genesis: genesis\n  keystore: keystore\n  keyrings: keyrings\n  nodes: node\n  runtime: runtime\n  logs: logs\ncontrol:\n  artifactRoot: ~/.chainbench\ninputs:\n  mode: generated\nexecution:\n  chain: fresh\n"
	preview, err := store.PreviewImport(DeploymentActor{ID: "o", Role: "operator"}, DocumentImportInput{Filename: "pool files", Files: []DocumentImportFile{
		{Filename: "localmap.yaml", Format: "yaml", Source: localmap},
		{Filename: "server-set.yaml", Format: "yaml", Source: pool},
		{Filename: "workspace-config.yaml", Format: "yaml", Source: config},
	}})
	if err != nil {
		t.Fatal(err)
	}
	if !preview.Validation.Valid || len(preview.RedactedDocuments) != 2 {
		t.Fatalf("preview = %+v", preview.Validation)
	}
	set := preview.RedactedDocuments[0]
	if set.Kind != "server-set" || !strings.Contains(string(set.Content), `"localmap"`) {
		t.Fatalf("set = %s %s", set.Kind, set.Content)
	}
	if strings.Contains(string(set.Content), "imported-secret-sentinel") {
		t.Fatal("an inline ssh password reached the shared declaration")
	}
}

func TestPreviewImport_ALocalmapAloneSaysWhatItNeeds(t *testing.T) {
	store, err := OpenDeploymentStore(t.TempDir())
	if err != nil {
		t.Fatal(err)
	}
	a := DeploymentActor{ID: "o", Role: "operator"}
	_, err = store.PreviewImport(a, DocumentImportInput{Filename: "localmap.yaml", Format: "yaml", Source: "hosts:\n  192.0.2.11:\n    host: 127.0.0.1\n"})
	if err == nil || !strings.Contains(err.Error(), "server-set") {
		t.Fatalf("err = %v", err)
	}
	_, err = store.PreviewImport(a, DocumentImportInput{Filename: "two", Files: []DocumentImportFile{
		{Filename: "a.yaml", Format: "yaml", Source: "hosts: {}\n"}, {Filename: "b.yaml", Format: "yaml", Source: "hosts: {}\n"},
	}})
	if err == nil {
		t.Fatal("two localmaps without a server set were accepted")
	}
}
