package app

import (
	"encoding/json"
	"errors"
	"fmt"

	"github.com/0xmhha/chainbench/internal/dsl"
)

// ValidateDocument validates a document as SaveDocument would, resolving the
// chain preset revisions a case pins.
func (s *DeploymentStore) ValidateDocument(in DeploymentDocumentInput) error {
	if in.Kind != "case" || len(in.PresetRefs) == 0 {
		return validateDeploymentDocument(in, nil)
	}
	presets, err := s.CasePresets(in)
	if err != nil {
		return err
	}
	return validateDeploymentDocument(in, presets)
}

// CasePresets returns the declarations a case's preset references pin, keyed
// by the id the case names them with. Every reference must be a chain preset
// the case actually names: an unused one would be a pinned revision with no
// effect on what runs.
func (s *DeploymentStore) CasePresets(in DeploymentDocumentInput) (map[string]json.RawMessage, error) {
	presets := map[string]json.RawMessage{}
	for _, ref := range in.PresetRefs {
		doc, err := s.DocumentRevision(ref.ID, ref.Revision)
		if err != nil || ref.Revision < 1 {
			return nil, fmt.Errorf("preset reference %s revision %d: %w", ref.ID, ref.Revision, ErrDeploymentNotFound)
		}
		if doc.Kind != "chain-preset" {
			return nil, fmt.Errorf("preset reference %s is a %s, not a chain preset", ref.ID, doc.Kind)
		}
		var head struct {
			ID string `json:"id"`
		}
		if err := json.Unmarshal(doc.Content, &head); err != nil || head.ID == "" {
			return nil, fmt.Errorf("preset reference %s has no declaration id", ref.ID)
		}
		if _, twice := presets[head.ID]; twice {
			return nil, fmt.Errorf("two preset references declare %q", head.ID)
		}
		presets[head.ID] = doc.Content
	}
	used := map[string]bool{}
	if _, err := dsl.InlineChainPreset(in.Content, func(id string) ([]byte, error) {
		used[id] = true
		raw, ok := presets[id]
		if !ok {
			return nil, fmt.Errorf("missing preset %q: pin a shared chain preset revision for it", id)
		}
		return raw, nil
	}); err != nil {
		return nil, err
	}
	for id := range presets {
		if !used[id] {
			return nil, errors.New("preset reference " + id + " is not named by the case")
		}
	}
	return presets, nil
}

// ExecutableCaseContent is the case with its pinned presets inlined, the one
// declaration the engine runs.
func (s *DeploymentStore) ExecutableCaseContent(in DeploymentDocumentInput) (json.RawMessage, error) {
	if len(in.PresetRefs) == 0 {
		return in.Content, nil
	}
	presets, err := s.CasePresets(in)
	if err != nil {
		return nil, err
	}
	return dsl.InlineChainPreset(in.Content, func(id string) ([]byte, error) { return presets[id], nil })
}
