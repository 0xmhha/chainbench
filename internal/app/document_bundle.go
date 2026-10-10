package app

import (
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"strings"

	"go.yaml.in/yaml/v3"

	"github.com/0xmhha/chainbench/internal/dsl"
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

// bundlePresetRef marks a case's link to a chain preset that arrives in the
// same bundle, by that preset's position; commit replaces it with the saved
// document's id and revision.
const bundlePresetRef = "bundle:"

// bundlePresets indexes the chain presets of a bundle by the declaration id a
// case extends them with.
func bundlePresets(values []map[string]any) map[string]bundlePreset {
	out := map[string]bundlePreset{}
	for i, item := range values {
		kind, _, content, issue := bundleDocument(item)
		if issue != "" || kind != "chain-preset" {
			continue
		}
		id, _ := content["id"].(string)
		raw, err := json.Marshal(content)
		if id == "" || err != nil {
			continue
		}
		if _, twice := out[id]; twice {
			out[id] = bundlePreset{index: -1}
			continue
		}
		out[id] = bundlePreset{index: i, raw: raw}
	}
	return out
}

type bundlePreset struct {
	index int
	raw   json.RawMessage
}

// linkBundlePresets pins a case to the bundle presets it extends. A preset the
// bundle does not carry is named, so the reader knows which file to add.
func linkBundlePresets(raw []byte, presets map[string]bundlePreset) ([]DeploymentDocumentRef, map[string]json.RawMessage, error) {
	refs, used := []DeploymentDocumentRef{}, map[string]json.RawMessage{}
	_, err := dsl.InlineChainPreset(raw, func(id string) ([]byte, error) {
		p, ok := presets[id]
		if !ok {
			return nil, fmt.Errorf("the case extends chain preset %q; include that preset in this bundle", id)
		}
		if p.index < 0 {
			return nil, fmt.Errorf("two chain presets in this bundle declare %q", id)
		}
		if _, seen := used[id]; !seen {
			refs = append(refs, DeploymentDocumentRef{ID: fmt.Sprintf("%s%d", bundlePresetRef, p.index)})
		}
		used[id] = p.raw
		return p.raw, nil
	})
	return refs, used, err
}

// importDocument separates private SSH values, prepares cases and validates
// one declaration against the engine contract, recording the concrete reason.
func importDocument(preview *DocumentImportPreview, prefix, kind, name string, value map[string]any, presets map[string]bundlePreset) {
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
	var refs []DeploymentDocumentRef
	var pinned map[string]json.RawMessage
	if kind == "case" && presets != nil {
		if refs, pinned, err = linkBundlePresets(raw, presets); err != nil {
			preview.Validation.Errors = append(preview.Validation.Errors, DocumentValidationIssue{prefix + "/content", "invalid", err.Error()})
			return
		}
		if len(refs) == 0 {
			refs, pinned = nil, nil
		}
	}
	if kind == "case" {
		if prepared, prepareErr := PrepareTestCase(TestCaseInput{Content: raw, Presets: pinned}); prepareErr == nil {
			raw = prepared.Content
			if prepared.Migrated {
				preview.Changes = append(preview.Changes, "Migrated v1 with unchanged executable fingerprint")
			}
		}
	}
	doc := DeploymentDocumentInput{Kind: kind, Name: name, ContractVersion: "2", Content: raw, AssetRefs: []string{}, PresetRefs: refs}
	if assets, refErr := webDocumentAssetRefs(kind, raw); refErr == nil {
		doc.AssetRefs = assets
	}
	if err := validateDeploymentDocument(doc, pinned); err != nil {
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

// ExportDocumentBundle writes one shared document as a bundle; a case brings
// the chain preset revisions it pins, so the bundle imports back runnable.
func (s *DeploymentStore) ExportDocumentBundle(a DeploymentActor, id string) ([]byte, error) {
	if a.ID == "" {
		return nil, ErrDeploymentForbidden
	}
	d, err := s.DocumentRevision(id, 0)
	if err != nil {
		return nil, err
	}
	docs := []DeploymentDocument{}
	for _, ref := range d.PresetRefs {
		p, err := s.DocumentRevision(ref.ID, ref.Revision)
		if err != nil {
			return nil, err
		}
		docs = append(docs, p)
	}
	documents := []map[string]any{}
	for _, doc := range append(docs, d) {
		var content any
		if err = json.Unmarshal(doc.Content, &content); err != nil {
			return nil, err
		}
		documents = append(documents, map[string]any{"kind": doc.Kind, "name": doc.Name, "contractVersion": doc.ContractVersion, "assetRefs": doc.AssetRefs, "content": content})
	}
	return json.MarshalIndent(map[string]any{"format": webBundleFormat, "documents": documents}, "", "  ")
}
