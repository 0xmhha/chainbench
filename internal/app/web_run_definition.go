package app

import (
	"bytes"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"strconv"
	"strings"
)

// webRunDefinitionLimit bounds how many cases one run definition lists.
const webRunDefinitionLimit = 64

// runDefinitionCaseIDs reads a run definition's content: the declaration ids
// of the cases it runs, in order, and an optional description.
func runDefinitionCaseIDs(content json.RawMessage) ([]string, error) {
	var value struct {
		Cases       []string `json:"cases"`
		Description string   `json:"description"`
	}
	dec := json.NewDecoder(bytes.NewReader(content))
	dec.DisallowUnknownFields()
	if err := dec.Decode(&value); err != nil || dec.Decode(new(any)) != io.EOF {
		return nil, errors.New("a run definition holds cases and an optional description")
	}
	if len(value.Cases) == 0 || len(value.Cases) > webRunDefinitionLimit {
		return nil, fmt.Errorf("a run definition lists 1 to %d cases", webRunDefinitionLimit)
	}
	seen := map[string]bool{}
	for _, id := range value.Cases {
		if strings.TrimSpace(id) == "" || seen[id] {
			return nil, errors.New("a run definition lists each case once by its id")
		}
		seen[id] = true
	}
	return value.Cases, nil
}

func caseDeclarationID(content json.RawMessage) string {
	var head struct {
		ID string `json:"id"`
	}
	_ = json.Unmarshal(content, &head)
	return head.ID
}

// RunDefinitionCases checks that a run definition pins exactly the shared
// case revisions its content lists, in the same order.
func (s *DeploymentStore) RunDefinitionCases(in DeploymentDocumentInput) ([]DeploymentDocumentRef, error) {
	ids, err := runDefinitionCaseIDs(in.Content)
	if err != nil {
		return nil, webFieldError("/content", err)
	}
	if len(in.CaseRefs) != len(ids) {
		return nil, webFieldError("/caseRefs", errors.New("pin one shared case revision for each listed case, in order"))
	}
	for i, ref := range in.CaseRefs {
		doc, err := s.DocumentRevision(ref.ID, ref.Revision)
		if err != nil || ref.Revision < 1 {
			return nil, webFieldError("/caseRefs", fmt.Errorf("case reference %s revision %d: %w", ref.ID, ref.Revision, ErrDeploymentNotFound))
		}
		if doc.Kind != "case" || caseDeclarationID(doc.Content) != ids[i] {
			return nil, webFieldError("/caseRefs", fmt.Errorf("case reference %d is not the case %q", i+1, ids[i]))
		}
	}
	return append([]DeploymentDocumentRef(nil), in.CaseRefs...), nil
}

// bundleCases indexes the cases of a bundle by their declaration id.
func bundleCases(values []map[string]any) map[string]int {
	out := map[string]int{}
	for i, item := range values {
		kind, _, content, issue := bundleDocument(item)
		if issue != "" || kind != "case" {
			continue
		}
		id, _ := content["id"].(string)
		if id == "" {
			continue
		}
		if _, twice := out[id]; twice {
			out[id] = -1
			continue
		}
		out[id] = i
	}
	return out
}

// linkBundleCases pins a run definition to the cases of its bundle.
func linkBundleCases(raw []byte, cases map[string]int) ([]DeploymentDocumentRef, error) {
	ids, err := runDefinitionCaseIDs(raw)
	if err != nil {
		return nil, err
	}
	refs := []DeploymentDocumentRef{}
	for _, id := range ids {
		index, ok := cases[id]
		if !ok {
			return nil, fmt.Errorf("the run definition lists case %q; include that case in this bundle", id)
		}
		if index < 0 {
			return nil, fmt.Errorf("two cases in this bundle declare %q", id)
		}
		refs = append(refs, DeploymentDocumentRef{ID: bundlePresetRef + strconv.Itoa(index)})
	}
	return refs, nil
}

// resolveBundleCaseRefs replaces a run definition's bundle positions with the
// saved case revisions of the same import.
func resolveBundleCaseRefs(documents []DeploymentDocument, i int) error {
	d := &documents[i]
	if d.Kind != "run-definition" {
		return nil
	}
	ids, err := runDefinitionCaseIDs(d.Content)
	if err != nil || len(ids) != len(d.CaseRefs) {
		return errors.New("import declaration no longer satisfies shared document validation")
	}
	refs := make([]DeploymentDocumentRef, 0, len(d.CaseRefs))
	for n, ref := range d.CaseRefs {
		index, err := strconv.Atoi(strings.TrimPrefix(ref.ID, bundlePresetRef))
		if !strings.HasPrefix(ref.ID, bundlePresetRef) || err != nil || index < 0 || index >= len(documents) || documents[index].Kind != "case" || caseDeclarationID(documents[index].Content) != ids[n] {
			return errors.New("import declaration no longer satisfies shared document validation")
		}
		refs = append(refs, DeploymentDocumentRef{ID: documents[index].ID, Revision: documents[index].Revision})
	}
	d.CaseRefs = refs
	return nil
}

// resolveRunDefinition replaces a test job's run definition with the case
// revisions it pins, in its order. The reference stays in the plan so the
// reviewed definition is visible.
func (e *WebChainEngine) resolveRunDefinition(args *webChainArguments, operation string) error {
	if args.RunDefinitionRef == nil {
		return nil
	}
	if operation != "test.run" || len(args.CaseRefs) != 0 {
		return errors.New("a run definition replaces the case list of a test job")
	}
	def, err := e.documents.DocumentRevision(args.RunDefinitionRef.ID, args.RunDefinitionRef.Revision)
	if err != nil {
		return err
	}
	if def.Kind != "run-definition" || args.RunDefinitionRef.Revision < 1 {
		return errors.New("select a saved run definition revision")
	}
	args.CaseRefs = append([]DeploymentDocumentRef(nil), def.CaseRefs...)
	return nil
}
