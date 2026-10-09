package app

import (
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"strings"

	"go.yaml.in/yaml/v3"
)

// webBundleFormat names the workspace bundle a Web export writes and an import
// reads back: the pinned shared declarations, with no private material.
const webBundleFormat = "chainbench-web-bundle"

// webBundleLimit bounds one import so a preview stays cheap to validate and store.
const webBundleLimit = 64

// decodeImportSource reads one declaration, or a bundle of declarations: a JSON
// or YAML object with a "documents" list, or a YAML stream of documents that
// each name their kind, name and content.
func decodeImportSource(in DocumentImportInput) ([]map[string]any, bool, error) {
	var values []map[string]any
	switch in.Format {
	case "json":
		var value map[string]any
		dec := json.NewDecoder(strings.NewReader(in.Source))
		if err := dec.Decode(&value); err != nil {
			return nil, false, errors.New("invalid JSON document")
		}
		if err := dec.Decode(new(any)); err != io.EOF {
			return nil, false, errors.New("expected one JSON document; use a documents list for a bundle")
		}
		values = []map[string]any{value}
	case "yaml":
		dec := yaml.NewDecoder(strings.NewReader(in.Source))
		for {
			var value map[string]any
			err := dec.Decode(&value)
			if err == io.EOF {
				break
			}
			if err != nil {
				return nil, false, errors.New("invalid YAML document")
			}
			if value != nil {
				values = append(values, value)
			}
			if len(values) > webBundleLimit {
				return nil, false, fmt.Errorf("a bundle holds at most %d documents", webBundleLimit)
			}
		}
		if len(values) > 1 {
			return values, true, nil
		}
	default:
		return nil, false, errors.New("unsupported import format")
	}
	if len(values) == 0 || values[0] == nil {
		return nil, false, errors.New("expected object document")
	}
	list, isBundle := values[0]["documents"]
	if !isBundle || in.Kind != "" {
		return values, false, nil
	}
	for key := range values[0] {
		if key != "documents" && key != "format" && key != "workspace" {
			return nil, false, fmt.Errorf("unsupported bundle field %q", key)
		}
	}
	items, _ := list.([]any)
	if len(items) == 0 {
		return nil, false, errors.New("a bundle needs at least one document")
	}
	if len(items) > webBundleLimit {
		return nil, false, fmt.Errorf("a bundle holds at most %d documents", webBundleLimit)
	}
	documents := []map[string]any{}
	for i, item := range items {
		document, ok := item.(map[string]any)
		if !ok {
			return nil, false, fmt.Errorf("bundle document %d is not an object", i)
		}
		documents = append(documents, document)
	}
	return documents, true, nil
}

// bundleDocument reads one bundle entry; an issue explains why it is unusable.
func bundleDocument(item map[string]any) (string, string, map[string]any, string) {
	for key := range item {
		switch key {
		case "kind", "name", "contractVersion", "assetRefs", "content":
		default:
			return "", "", nil, fmt.Sprintf("unsupported bundle document field %q", key)
		}
	}
	kind, _ := item["kind"].(string)
	name, _ := item["name"].(string)
	content, _ := item["content"].(map[string]any)
	if version, ok := item["contractVersion"]; ok && version != "2" {
		return "", "", nil, "unsupported contract version"
	}
	if kind == "" || content == nil {
		return "", "", nil, "a bundle document needs a kind and an object content"
	}
	if name == "" {
		name = kind
	}
	return kind, name, content, ""
}

// importDocument separates private SSH values, prepares cases and validates
// one declaration against the engine contract, recording the concrete reason.
func importDocument(preview *DocumentImportPreview, prefix, kind, name string, value map[string]any) {
	if kind == "server-set" {
		if ssh, ok := value["ssh"].(map[string]any); ok {
			for _, key := range []string{"password", "password_file", "key_file", "key_passphrase_file"} {
				if _, exists := ssh[key]; exists {
					delete(ssh, key)
					binding := "ssh." + key
					if prefix != "" {
						binding = prefix + "/" + binding
					}
					preview.PrivateBindingsRequired = append(preview.PrivateBindingsRequired, binding)
					preview.Changes = append(preview.Changes, "Moved "+binding+" out of shared declaration")
				}
			}
		}
	}
	raw, err := json.Marshal(value)
	if err != nil {
		preview.Validation.Errors = append(preview.Validation.Errors, DocumentValidationIssue{prefix + "/content", "invalid", "invalid declaration values"})
		return
	}
	if kind == "case" {
		if prepared, prepareErr := PrepareTestCase(TestCaseInput{Content: raw}); prepareErr == nil {
			raw = prepared.Content
			if prepared.Migrated {
				preview.Changes = append(preview.Changes, "Migrated v1 with unchanged executable fingerprint")
			}
		}
	}
	doc := DeploymentDocumentInput{Kind: kind, Name: name, ContractVersion: "2", Content: raw, AssetRefs: []string{}}
	if refs, refErr := webDocumentAssetRefs(kind, raw); refErr == nil {
		doc.AssetRefs = refs
	}
	if err := ValidateDeploymentDocument(doc); err != nil {
		preview.Validation.Errors = append(preview.Validation.Errors, DocumentValidationIssue{prefix + "/content", "invalid", err.Error()})
		return
	}
	preview.RedactedDocuments = append(preview.RedactedDocuments, doc)
}

// ExportWorkspaceBundle writes the workspace's pinned shared declarations as a
// bundle that imports back unchanged. Shared documents hold no private values.
func (s *DeploymentStore) ExportWorkspaceBundle(a DeploymentActor, id string) ([]byte, error) {
	if a.ID == "" {
		return nil, ErrDeploymentForbidden
	}
	workspace, err := s.Workspace(id)
	if err != nil {
		return nil, err
	}
	documents := []map[string]any{}
	for _, ref := range workspace.Documents {
		d, err := s.DocumentRevision(ref.ID, ref.Revision)
		if err != nil {
			return nil, err
		}
		var content any
		if err = json.Unmarshal(d.Content, &content); err != nil {
			return nil, err
		}
		documents = append(documents, map[string]any{"kind": d.Kind, "name": d.Name, "contractVersion": d.ContractVersion, "assetRefs": d.AssetRefs, "content": content})
	}
	return json.MarshalIndent(map[string]any{"format": webBundleFormat, "workspace": workspace.Name, "documents": documents}, "", "  ")
}
