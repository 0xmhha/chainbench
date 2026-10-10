package app

import (
	"bytes"
	"crypto/rand"
	"encoding/json"
	"errors"
	"fmt"
	"strconv"
	"strings"
	"time"
)

type DocumentImportInput struct {
	Filename string `json:"filename"`
	Format   string `json:"format"`
	Source   string `json:"source"`
	Kind     string `json:"kind,omitempty"`
}
type DocumentValidationIssue struct {
	Path    string `json:"path"`
	Code    string `json:"code"`
	Message string `json:"message"`
}
type DocumentValidation struct {
	Valid           bool                      `json:"valid"`
	ContractVersion string                    `json:"contractVersion"`
	Errors          []DocumentValidationIssue `json:"errors"`
	Warnings        []string                  `json:"warnings"`
}
type DocumentImportPreview struct {
	PreviewID               string                    `json:"previewId"`
	Validation              DocumentValidation        `json:"validation"`
	RedactedDocuments       []DeploymentDocumentInput `json:"redactedDocuments"`
	Changes                 []string                  `json:"changes"`
	PrivateBindingsRequired []string                  `json:"privateBindingsRequired"`
	SourcePreserved         bool                      `json:"sourcePreserved"`
}
type DocumentImportCommit struct {
	PreviewID     string         `json:"previewId"`
	BaseRevisions map[string]int `json:"baseRevisions,omitempty"`
}
type deploymentImport struct {
	OwnerID            string                `json:"ownerId"`
	ExpiresAt          time.Time             `json:"expiresAt"`
	Preview            DocumentImportPreview `json:"preview"`
	Ciphertext         []byte                `json:"ciphertext"`
	Committed          bool                  `json:"committed"`
	QuarantinedPreview []byte                `json:"quarantinedPreview,omitempty"`
}

// PreviewImport keeps the original encrypted and returns only the validated
// public declaration. Secret paths/material require a separate personal binding.
func (s *DeploymentStore) PreviewImport(a DeploymentActor, in DocumentImportInput) (DocumentImportPreview, error) {
	if !a.canEdit() {
		return DocumentImportPreview{}, ErrDeploymentForbidden
	}
	if in.Filename == "" || len(in.Source) > 1<<20 {
		return DocumentImportPreview{}, errors.New("invalid import")
	}
	values, bundle, err := decodeImportSource(in)
	if err != nil {
		return DocumentImportPreview{}, err
	}
	preview := DocumentImportPreview{PreviewID: deploymentID(), Validation: DocumentValidation{ContractVersion: "2", Errors: []DocumentValidationIssue{}, Warnings: []string{}}, RedactedDocuments: []DeploymentDocumentInput{}, Changes: []string{}, PrivateBindingsRequired: []string{}, SourcePreserved: true}
	if !bundle {
		kind := in.Kind
		if declared, ok := values[0]["kind"].(string); ok {
			if kind != "" && kind != declared {
				return preview, errors.New("import kind mismatch")
			}
			kind = declared
		}
		if kind == "" {
			return preview, errors.New("select document kind")
		}
		name, _ := values[0]["id"].(string)
		if name == "" {
			name = in.Filename
		}
		importDocument(&preview, "", kind, name, values[0], nil)
	}
	seen := map[string]bool{}
	var presets map[string]bundlePreset
	if bundle {
		presets = bundlePresets(values)
	}
	for i, item := range values {
		if !bundle {
			break
		}
		prefix := fmt.Sprintf("/documents/%d", i)
		kind, name, content, issue := bundleDocument(item)
		if issue == "" && seen[kind+"\x00"+name] {
			issue = fmt.Sprintf("another document in this bundle is also %s %q", kind, name)
		}
		seen[kind+"\x00"+name] = true
		if issue != "" {
			preview.Validation.Errors = append(preview.Validation.Errors, DocumentValidationIssue{prefix, "invalid", issue})
			continue
		}
		importDocument(&preview, prefix, kind, name, content, presets)
	}
	preview.Validation.Valid = len(preview.Validation.Errors) == 0 && len(preview.RedactedDocuments) > 0
	if !preview.Validation.Valid {
		preview.RedactedDocuments = []DeploymentDocumentInput{}
	}
	// Encrypt even an invalid source; it is never returned in public responses.
	nonce := make([]byte, s.aead.NonceSize())
	if _, err = rand.Read(nonce); err != nil {
		return preview, err
	}
	ciphertext := s.aead.Seal(nonce, nonce, []byte(in.Source), []byte(preview.PreviewID+":"+a.ID))
	s.mu.Lock()
	defer s.mu.Unlock()
	err = s.commit(a, "document.import.preview", preview.PreviewID, func(next *deploymentState) {
		if next.Imports == nil {
			next.Imports = map[string]deploymentImport{}
		}
		for id, old := range next.Imports {
			if time.Now().After(old.ExpiresAt) {
				delete(next.Imports, id) // Expired previews can no longer be committed.
			}
		}
		next.Imports[preview.PreviewID] = deploymentImport{OwnerID: a.ID, ExpiresAt: time.Now().UTC().Add(15 * time.Minute), Preview: preview, Ciphertext: ciphertext}
	})
	return preview, err
}

// CommitImport saves all previewed declarations atomically, with explicit
// revisions for replacements. A preview belongs to its initiating operator.
func (s *DeploymentStore) CommitImport(a DeploymentActor, in DocumentImportCommit) ([]DeploymentDocument, error) {
	if !a.canEdit() {
		return nil, ErrDeploymentForbidden
	}
	s.mu.Lock()
	defer s.mu.Unlock()
	entry, ok := s.state.Imports[in.PreviewID]
	if !ok || entry.OwnerID != a.ID {
		return nil, ErrDeploymentNotFound
	}
	if entry.Committed || time.Now().After(entry.ExpiresAt) {
		return nil, ErrDeploymentConflict
	}
	if !entry.Preview.Validation.Valid || len(entry.Preview.RedactedDocuments) == 0 {
		return nil, errors.New("invalid preview cannot be saved")
	}
	if len(in.BaseRevisions) > len(entry.Preview.RedactedDocuments) {
		return nil, ErrDeploymentConflict
	}
	used := map[string]bool{}
	documents := []DeploymentDocument{}
	for _, incoming := range entry.Preview.RedactedDocuments {
		if err := ValidateDeploymentDocument(incoming); err != nil && !errors.Is(err, errCasePresetsNeedStore) {
			return nil, errors.New("import declaration no longer satisfies shared document validation")
		}
		id, revision := deploymentID(), 0
		for candidate, expected := range in.BaseRevisions {
			history := s.state.Documents[candidate]
			if len(history) == 0 || history[len(history)-1].Revision != expected {
				return nil, ErrDeploymentConflict
			}
			last := history[len(history)-1]
			if last.Kind == incoming.Kind && last.Name == incoming.Name {
				if revision != 0 || used[candidate] {
					return nil, ErrDeploymentConflict
				}
				id, revision = candidate, expected
				used[candidate] = true
			}
		}
		documents = append(documents, DeploymentDocument{incoming, id, revision + 1, time.Now().UTC(), a.ID})
	}
	if len(used) != len(in.BaseRevisions) {
		return nil, fmt.Errorf("replacement references do not match preview")
	}
	// A case pinned to a preset of the same bundle now names that preset's
	// saved revision, and is validated against exactly those declarations.
	for i := range documents {
		if err := resolveBundlePresetRefs(documents, i); err != nil {
			return nil, err
		}
	}
	err := s.commit(a, "document.import.commit", in.PreviewID, func(next *deploymentState) {
		for _, d := range documents {
			next.Documents[d.ID] = append(next.Documents[d.ID], d)
		}
		i := next.Imports[in.PreviewID]
		i.Committed = true
		next.Imports[in.PreviewID] = i
	})
	// Do not let callers modify committed JSON slices through returned objects.
	b, _ := json.Marshal(documents)
	_ = json.NewDecoder(bytes.NewReader(b)).Decode(&documents)
	return documents, err
}

func resolveBundlePresetRefs(documents []DeploymentDocument, i int) error {
	d := &documents[i]
	if len(d.PresetRefs) == 0 {
		return nil
	}
	refs := make([]DeploymentDocumentRef, 0, len(d.PresetRefs))
	presets := map[string]json.RawMessage{}
	for _, ref := range d.PresetRefs {
		index, err := strconv.Atoi(strings.TrimPrefix(ref.ID, bundlePresetRef))
		if !strings.HasPrefix(ref.ID, bundlePresetRef) || err != nil || index < 0 || index >= len(documents) || documents[index].Kind != "chain-preset" {
			return errors.New("import declaration no longer satisfies shared document validation")
		}
		p := documents[index]
		var head struct {
			ID string `json:"id"`
		}
		if err = json.Unmarshal(p.Content, &head); err != nil || head.ID == "" {
			return errors.New("import declaration no longer satisfies shared document validation")
		}
		presets[head.ID] = p.Content
		refs = append(refs, DeploymentDocumentRef{ID: p.ID, Revision: p.Revision})
	}
	d.PresetRefs = refs
	if err := validateDeploymentDocument(d.DeploymentDocumentInput, presets); err != nil {
		return errors.New("import declaration no longer satisfies shared document validation")
	}
	return nil
}
