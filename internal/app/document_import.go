package app

import (
	"bytes"
	"crypto/rand"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"strings"
	"time"

	"go.yaml.in/yaml/v3"
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
	OwnerID    string                `json:"ownerId"`
	ExpiresAt  time.Time             `json:"expiresAt"`
	Preview    DocumentImportPreview `json:"preview"`
	Ciphertext []byte                `json:"ciphertext"`
	Committed  bool                  `json:"committed"`
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
	var value map[string]any
	switch in.Format {
	case "json":
		dec := json.NewDecoder(strings.NewReader(in.Source))
		if err := dec.Decode(&value); err != nil {
			return DocumentImportPreview{}, errors.New("invalid JSON document")
		}
		if err := dec.Decode(new(any)); err != io.EOF {
			return DocumentImportPreview{}, errors.New("expected one document")
		}
	case "yaml":
		dec := yaml.NewDecoder(strings.NewReader(in.Source))
		if err := dec.Decode(&value); err != nil {
			return DocumentImportPreview{}, errors.New("invalid YAML document")
		}
		if err := dec.Decode(new(any)); err != io.EOF {
			return DocumentImportPreview{}, errors.New("expected one document")
		}
	default:
		return DocumentImportPreview{}, errors.New("unsupported import format")
	}
	if value == nil {
		return DocumentImportPreview{}, errors.New("expected object document")
	}
	preview := DocumentImportPreview{PreviewID: deploymentID(), Validation: DocumentValidation{ContractVersion: "2", Errors: []DocumentValidationIssue{}, Warnings: []string{}}, RedactedDocuments: []DeploymentDocumentInput{}, Changes: []string{}, PrivateBindingsRequired: []string{}, SourcePreserved: true}
	kind := in.Kind
	if declared, ok := value["kind"].(string); ok {
		if kind != "" && kind != declared {
			return preview, errors.New("import kind mismatch")
		}
		kind = declared
	}
	if kind == "" {
		return preview, errors.New("select document kind")
	}
	name, _ := value["id"].(string)
	if name == "" {
		name = in.Filename
	}
	if kind == "server-set" {
		if ssh, ok := value["ssh"].(map[string]any); ok {
			for _, key := range []string{"password", "password_file", "key_file", "key_passphrase_file"} {
				if _, exists := ssh[key]; exists {
					delete(ssh, key)
					preview.PrivateBindingsRequired = append(preview.PrivateBindingsRequired, "ssh."+key)
					preview.Changes = append(preview.Changes, "Moved ssh."+key+" out of shared declaration")
				}
			}
		}
	}
	raw, err := json.Marshal(value)
	if err != nil {
		return preview, errors.New("invalid declaration values")
	}
	if kind == "case" {
		prepared, prepareErr := PrepareTestCase(TestCaseInput{Content: raw})
		if prepareErr == nil {
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
		preview.Validation.Errors = append(preview.Validation.Errors, DocumentValidationIssue{"/content", "invalid", "Engine declaration validation failed; resolve unsupported fields and references"})
	} else {
		preview.Validation.Valid = true
		preview.RedactedDocuments = append(preview.RedactedDocuments, doc)
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
