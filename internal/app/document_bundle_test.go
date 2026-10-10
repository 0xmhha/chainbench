package app

import (
	"encoding/json"
	"strings"
	"testing"
)

func bundleSource(t *testing.T, docs ...map[string]any) string {
	t.Helper()
	raw, err := json.Marshal(map[string]any{"documents": docs})
	if err != nil {
		t.Fatal(err)
	}
	return string(raw)
}

func bundleItem(in DeploymentDocumentInput, mutate func(map[string]any)) map[string]any {
	var content map[string]any
	_ = json.Unmarshal(in.Content, &content)
	if mutate != nil {
		mutate(content)
	}
	return map[string]any{"kind": in.Kind, "name": in.Name, "contractVersion": "2", "content": content}
}

func TestDocumentBundleImportPreviewsCommitsAndExportsRoundTrip(t *testing.T) {
	store, err := OpenDeploymentStore(t.TempDir())
	if err != nil {
		t.Fatal(err)
	}
	a := DeploymentActor{ID: "operator", Role: "operator"}
	withPassword := func(c map[string]any) { c["ssh"] = map[string]any{"port": 22, "password": "bundle-ssh-secret"} }
	source := bundleSource(t, bundleItem(deploymentTestSet(), withPassword), bundleItem(deploymentTestConfig(), nil))
	preview, err := store.PreviewImport(a, DocumentImportInput{Filename: "workspace.bundle.json", Format: "json", Source: source})
	if err != nil {
		t.Fatal(err)
	}
	if !preview.Validation.Valid || len(preview.RedactedDocuments) != 2 {
		t.Fatalf("bundle preview = %+v", preview)
	}
	if strings.Contains(string(preview.RedactedDocuments[0].Content), "bundle-ssh-secret") || len(preview.PrivateBindingsRequired) != 1 || preview.PrivateBindingsRequired[0] != "/documents/0/ssh.password" {
		t.Fatalf("bundle kept a private SSH value in the shared declaration: %+v", preview)
	}
	saved, err := store.CommitImport(a, DocumentImportCommit{PreviewID: preview.PreviewID})
	if err != nil || len(saved) != 2 || saved[0].Kind != "server-set" || saved[1].Kind != "workspace-config" {
		t.Fatalf("atomic bundle commit = %+v, %v", saved, err)
	}
	workspace, err := store.SaveWorkspace(a, "", 0, DeploymentWorkspaceInput{Name: "bundled", Documents: []DeploymentDocumentRef{{saved[0].ID, saved[0].Revision}, {saved[1].ID, saved[1].Revision}}})
	if err != nil {
		t.Fatal(err)
	}
	exported, err := store.ExportWorkspaceBundle(a, workspace.ID)
	if err != nil {
		t.Fatal(err)
	}
	if strings.Contains(string(exported), "bundle-ssh-secret") {
		t.Fatal("workspace export leaked a private value")
	}
	again, err := store.PreviewImport(a, DocumentImportInput{Filename: "exported.json", Format: "json", Source: string(exported)})
	if err != nil || !again.Validation.Valid || len(again.RedactedDocuments) != 2 {
		t.Fatalf("exported bundle does not import again: %+v %v", again, err)
	}
	for i, doc := range again.RedactedDocuments {
		var before, after any
		_ = json.Unmarshal(saved[i].Content, &before)
		_ = json.Unmarshal(doc.Content, &after)
		b1, _ := json.Marshal(before)
		b2, _ := json.Marshal(after)
		if doc.Kind != saved[i].Kind || doc.Name != saved[i].Name || string(b1) != string(b2) {
			t.Fatalf("round trip changed document %d: %s vs %s", i, b1, b2)
		}
	}
}

func TestDocumentBundleImportReportsConcreteErrorsPerDocument(t *testing.T) {
	store, err := OpenDeploymentStore(t.TempDir())
	if err != nil {
		t.Fatal(err)
	}
	a := DeploymentActor{ID: "operator", Role: "operator"}
	unknown := func(c map[string]any) { c["extension"] = map[string]any{"plugin": "unregistered"} }
	source := "kind: server-set\nname: Pool\ncontent:\n  version: 2\n  pool:\n    hosts: [{name: ssh, addr: localhost.}]\n    slots: 2\n---\n"
	items, _ := json.Marshal(bundleItem(deploymentTestConfig(), unknown))
	var config map[string]any
	_ = json.Unmarshal(items, &config)
	yamlConfig, _ := json.Marshal(config) // JSON is valid YAML for one stream document.
	preview, err := store.PreviewImport(a, DocumentImportInput{Filename: "bundle.yaml", Format: "yaml", Source: source + string(yamlConfig)})
	if err != nil {
		t.Fatal(err)
	}
	if preview.Validation.Valid || len(preview.Validation.Errors) != 1 {
		t.Fatalf("invalid bundle accepted or errors lost: %+v", preview.Validation)
	}
	issue := preview.Validation.Errors[0]
	if issue.Path != "/documents/1/content" || !strings.Contains(issue.Message, "extension") {
		t.Fatalf("error does not name the document and field: %+v", issue)
	}
	if _, err = store.CommitImport(a, DocumentImportCommit{PreviewID: preview.PreviewID}); err == nil {
		t.Fatal("a bundle with an invalid document was saved")
	}
	if _, err = store.PreviewImport(a, DocumentImportInput{Filename: "empty.json", Format: "json", Source: `{"documents":[]}`}); err == nil {
		t.Fatal("an empty bundle was accepted")
	}
}

func TestDocumentBundleRejectsOversizedAndDuplicateBundles(t *testing.T) {
	store, err := OpenDeploymentStore(t.TempDir())
	if err != nil {
		t.Fatal(err)
	}
	a := DeploymentActor{ID: "operator", Role: "operator"}
	many := []map[string]any{}
	for i := 0; i <= webBundleLimit; i++ {
		many = append(many, bundleItem(deploymentTestConfig(), nil))
	}
	if _, err = store.PreviewImport(a, DocumentImportInput{Filename: "many.json", Format: "json", Source: bundleSource(t, many...)}); err == nil || !strings.Contains(err.Error(), "at most") {
		t.Fatalf("oversized bundle accepted: %v", err)
	}
	twice := bundleSource(t, bundleItem(deploymentTestConfig(), nil), bundleItem(deploymentTestConfig(), nil))
	preview, err := store.PreviewImport(a, DocumentImportInput{Filename: "twice.json", Format: "json", Source: twice})
	if err != nil {
		t.Fatal(err)
	}
	if preview.Validation.Valid || len(preview.Validation.Errors) != 1 || preview.Validation.Errors[0].Path != "/documents/1" {
		t.Fatalf("duplicate kind and name not reported: %+v", preview.Validation)
	}
}
