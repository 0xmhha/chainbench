package testengine

import (
	"encoding/json"
	"fmt"

	"github.com/0xmhha/chainbench/internal/core/session"
	"github.com/0xmhha/chainbench/internal/dsl"
	"github.com/0xmhha/chainbench/internal/dsl/interp"
)

type preparedCase struct {
	Environment session.Fingerprint
	Values      map[string]any
}

type preparedScope map[string]preparedCase

func (p preparedScope) inputs(spec *dsl.Spec, environment session.Fingerprint) error {
	spec.PreparedInputs, spec.PreparedOutputs = map[string]any{}, map[string]any{}
	if spec.Prepared == nil {
		return nil
	}
	if len(spec.Prepared.Exports) > 0 {
		if _, found := p[spec.ID]; found {
			return fmt.Errorf("prepared results already published by case %q", spec.ID)
		}
	}
	for name, ref := range spec.Prepared.Imports {
		producer, found := p[ref.Case]
		if !found {
			return fmt.Errorf("prepared input %q: case %q has no successful published results", name, ref.Case)
		}
		if producer.Environment != environment {
			return fmt.Errorf("prepared input %q belongs to a different environment", name)
		}
		value, found := producer.Values[ref.Name]
		if !found {
			return fmt.Errorf("prepared input %q: case %q did not export %q", name, ref.Case, ref.Name)
		}
		copied, err := interp.ClonePreparedValue(value)
		if err != nil {
			return err
		}
		spec.PreparedInputs[name] = copied
	}
	return nil
}

func (p preparedScope) publish(spec dsl.Spec, environment session.Fingerprint, rec session.TestRecord) error {
	if spec.Prepared == nil {
		return nil
	}
	values := map[string]any{}
	for name := range spec.Prepared.Exports {
		value, found := spec.PreparedOutputs[name]
		if !found {
			return fmt.Errorf("prepared result %q was not produced", name)
		}
		copied, err := interp.ClonePreparedValue(value)
		if err != nil {
			return err
		}
		values[name] = copied
	}
	if len(values) > 0 {
		p[spec.ID] = preparedCase{Environment: environment, Values: values}
	}
	data, _ := json.Marshal(map[string]any{"source": "ordered_case_results", "case": spec.ID, "environment": environment, "imports": spec.Prepared.Imports, "exports": spec.Prepared.Exports, "published": len(values)})
	rec.Observation("prepared-results.json", data)
	return nil
}
