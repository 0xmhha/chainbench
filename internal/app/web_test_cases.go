package app

import (
	"context"
	"encoding/json"
	"errors"

	"github.com/0xmhha/chainbench/internal/dsl"
)

type webExecutableCase struct {
	Document DeploymentDocument `json:"document"`
	Content  json.RawMessage    `json:"content"`
}

func (e *WebChainEngine) prepareTestCases(ctx context.Context, chain string, refs []DeploymentDocumentRef) ([]webExecutableCase, error) {
	if err := ctx.Err(); err != nil {
		return nil, err
	}
	if len(refs) == 0 || len(refs) > 256 {
		return nil, errors.New("select between one and 256 saved test cases")
	}
	cases := make([]webExecutableCase, 0, len(refs))
	seen := map[string]bool{}
	total := 0
	for _, ref := range refs {
		if err := ctx.Err(); err != nil {
			return nil, err
		}
		if ref.Revision < 1 || seen[ref.ID] {
			return nil, errors.New("select each test case once with an explicit saved revision")
		}
		seen[ref.ID] = true
		document, err := e.documents.DocumentRevision(ref.ID, ref.Revision)
		if err != nil {
			return nil, err
		}
		latest, err := e.documents.DocumentRevision(ref.ID, 0)
		if err != nil {
			return nil, err
		}
		if latest.Revision != ref.Revision {
			return nil, ErrDeploymentConflict
		}
		if document.Kind != "case" {
			return nil, errors.New("selected document is not a test case")
		}
		total += len(document.Content)
		if total > 16<<20 {
			return nil, errors.New("selected test cases exceed the execution input limit")
		}
		presets := map[string]json.RawMessage{}
		if len(document.PresetRefs) > 0 {
			if presets, err = e.documents.CasePresets(document.DeploymentDocumentInput); err != nil {
				return nil, err
			}
		}
		prepared, err := PrepareTestCase(TestCaseInput{Content: document.Content, Presets: presets})
		if err != nil {
			return nil, err
		}
		// The engine runs the pinned preset revision, inlined; the stored
		// declaration keeps its reference.
		executable, err := e.documents.ExecutableCaseContent(DeploymentDocumentInput{Kind: "case", Content: prepared.Content, PresetRefs: document.PresetRefs})
		if err != nil {
			return nil, err
		}
		prepared.Content = executable
		spec, err := dsl.Parse(prepared.Content)
		if err != nil {
			return nil, err
		}
		if spec.Chain.Name != chain {
			return nil, errors.New("selected case belongs to a different chain")
		}
		cases = append(cases, webExecutableCase{Document: document, Content: prepared.Content})
	}
	return cases, nil
}
