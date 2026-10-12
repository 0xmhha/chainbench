package app

import (
	"crypto/rand"
	"encoding/json"
	"errors"
	"strings"
	"time"
)

type deploymentDocumentQuarantine struct {
	Version    int       `json:"version"`
	Revisions  int       `json:"revisions"`
	CreatedAt  time.Time `json:"createdAt"`
	Ciphertext []byte    `json:"ciphertext"`
}

func documentQuarantineAAD(id string) []byte {
	return []byte("web-document-quarantine:v1:" + id)
}

func importQuarantineAAD(id, owner string) []byte {
	return []byte("web-import-preview-quarantine:v1:" + id + ":" + owner)
}

// quarantineLegacyNodeKeys runs before the store is returned to any caller.
// Entire histories leave the public namespace together; no revision is silently
// rewritten. Their originals remain authenticated encrypted records. Old import
// previews are invalidated while their actor-bound encrypted sources stay intact.
func (s *DeploymentStore) quarantineLegacyNodeKeys() error {
	keys := []string{}
	for id, entry := range s.state.QuarantinedDocuments {
		n := s.aead.NonceSize()
		if entry.Version != 1 || entry.Revisions < 1 || len(entry.Ciphertext) <= n {
			return errors.New("invalid encrypted document quarantine")
		}
		original, err := s.aead.Open(nil, entry.Ciphertext[:n], entry.Ciphertext[n:], documentQuarantineAAD(id))
		var history []DeploymentDocument
		if err != nil || json.Unmarshal(original, &history) != nil || len(history) != entry.Revisions {
			return errors.New("encrypted document quarantine integrity check failed")
		}
		for _, doc := range history {
			values, err := webDocumentNodeKeys(doc.DeploymentDocumentInput)
			if err != nil {
				return err
			}
			keys = append(keys, values...)
		}
	}
	documents := map[string]deploymentDocumentQuarantine{}
	imports := map[string]deploymentImport{}
	for id, history := range s.state.Documents {
		unsafe := false
		for _, doc := range history {
			values, err := webDocumentNodeKeys(doc.DeploymentDocumentInput)
			if err != nil {
				return err
			}
			unsafe = unsafe || len(values) > 0
			keys = append(keys, values...)
		}
		if !unsafe {
			continue
		}
		if _, exists := s.state.QuarantinedDocuments[id]; exists {
			return errors.New("document quarantine identity conflict")
		}
		original, err := json.Marshal(history)
		if err != nil {
			return errors.New("cannot preserve legacy document history")
		}
		nonce := make([]byte, s.aead.NonceSize())
		if _, err = rand.Read(nonce); err != nil {
			return err
		}
		documents[id] = deploymentDocumentQuarantine{Version: 1, Revisions: len(history), CreatedAt: time.Now().UTC(), Ciphertext: s.aead.Seal(nonce, nonce, original, documentQuarantineAAD(id))}
	}
	for id, entry := range s.state.Imports {
		if len(entry.QuarantinedPreview) > 0 {
			n := s.aead.NonceSize()
			if len(entry.QuarantinedPreview) <= n {
				return errors.New("invalid encrypted import quarantine")
			}
			original, err := s.aead.Open(nil, entry.QuarantinedPreview[:n], entry.QuarantinedPreview[n:], importQuarantineAAD(id, entry.OwnerID))
			var preview DocumentImportPreview
			if err != nil || json.Unmarshal(original, &preview) != nil {
				return errors.New("encrypted import quarantine integrity check failed")
			}
			for _, doc := range preview.RedactedDocuments {
				values, err := webDocumentNodeKeys(doc)
				if err != nil {
					return err
				}
				keys = append(keys, values...)
			}
		}
		unsafe := false
		for _, doc := range entry.Preview.RedactedDocuments {
			values, err := webDocumentNodeKeys(doc)
			if err != nil {
				return err
			}
			unsafe = unsafe || len(values) > 0
			keys = append(keys, values...)
		}
		if !unsafe {
			continue
		}
		if len(entry.QuarantinedPreview) > 0 {
			return errors.New("import quarantine identity conflict")
		}
		original, err := json.Marshal(entry.Preview)
		if err != nil {
			return errors.New("cannot preserve legacy import preview")
		}
		nonce := make([]byte, s.aead.NonceSize())
		if _, err = rand.Read(nonce); err != nil {
			return err
		}
		entry.QuarantinedPreview = s.aead.Seal(nonce, nonce, original, importQuarantineAAD(id, entry.OwnerID))
		entry.Preview = DocumentImportPreview{
			PreviewID:         id,
			Validation:        DocumentValidation{ContractVersion: "2", Errors: []DocumentValidationIssue{{Path: "/content", Code: "private-key-material", Message: "Legacy import contains private node key material and cannot be published"}}, Warnings: []string{}},
			RedactedDocuments: []DeploymentDocumentInput{}, Changes: []string{"Private node key declaration removed from public preview"}, PrivateBindingsRequired: []string{}, SourcePreserved: true,
		}
		imports[id] = entry
	}
	if len(documents) == 0 && len(imports) == 0 {
		s.quarantinedNodeKeys = nodeKeyRedactions(keys)
		return nil
	}
	if err := s.commit(DeploymentActor{ID: "system", Role: "admin"}, "document.quarantine", "legacy-node-key-material", func(next *deploymentState) {
		if next.QuarantinedDocuments == nil {
			next.QuarantinedDocuments = map[string]deploymentDocumentQuarantine{}
		}
		for id, entry := range documents {
			next.QuarantinedDocuments[id] = entry
			delete(next.Documents, id)
		}
		for id, entry := range imports {
			next.Imports[id] = entry
		}
	}); err != nil {
		return err
	}
	s.quarantinedNodeKeys = nodeKeyRedactions(keys)
	return nil
}

func webDocumentNodeKeys(in DeploymentDocumentInput) ([]string, error) {
	if in.Kind != "chain-preset" && in.Kind != "case" {
		return nil, nil
	}
	var content map[string]any
	if err := json.Unmarshal(in.Content, &content); err != nil {
		return nil, errors.New("invalid legacy shared declaration")
	}
	return webDocumentNodeKeyValues(in.Kind, content), nil
}

func nodeKeyRedactions(keys []string) []string {
	out := []string{}
	seen := map[string]bool{}
	for _, key := range keys {
		trimmed := strings.TrimSpace(key)
		bare := strings.TrimPrefix(strings.TrimPrefix(trimmed, "0x"), "0X")
		for _, value := range []string{key, trimmed, bare, strings.ToLower(bare), strings.ToUpper(bare)} {
			if value != "" && !seen[value] {
				out = append(out, value)
				seen[value] = true
			}
		}
	}
	return out
}
