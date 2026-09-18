package testengine

import (
	"os"
	"path/filepath"
	"testing"
)

// TestPlanFile_SurvivesTheRunThatMadeIt is the point of writing it at all.
//
// Before this the plan was said once, on stderr, while the run was starting.
// The question it answers — who chose this value — is asked later, by someone
// looking at a workspace that has been sitting there for a week.
func TestPlanFile_SurvivesTheRunThatMadeIt(t *testing.T) {
	dir := t.TempDir()
	want := ComposePlan{
		Chain: "stablenet", Workspace: dir, Binary: "/b/gstable",
		From: map[PlanField]PlanSource{
			FieldBinary: SourceCommand,
			FieldTarget: SourceHarness,
		},
		Launch: map[string][]PlanKnob{
			"bp": {{Knob: "mine=true", From: SourceDeclaration}},
		},
	}
	if err := WritePlan(dir, want); err != nil {
		t.Fatal(err)
	}
	if _, err := os.Stat(filepath.Join(dir, PlanFile)); err != nil {
		t.Fatalf("the plan must sit beside the record: %v", err)
	}

	got, err := ReadPlan(dir)
	if err != nil {
		t.Fatal(err)
	}
	if got.From[FieldBinary] != SourceCommand || got.From[FieldTarget] != SourceHarness {
		t.Errorf("the sources did not survive the round trip: %v", got.From)
	}
	if k := got.Launch["bp"]; len(k) != 1 || k[0].From != SourceDeclaration {
		t.Errorf("a launch knob lost who asked for it: %v", k)
	}
	if got.Binary != want.Binary || got.Chain != want.Chain {
		t.Errorf("plan = %+v, want %+v", got, want)
	}
}

// TestPlanFile_MissingIsAnError, rather than an empty plan: a workspace with no
// plan file was composed by something that did not write one, and answering
// "nobody chose anything" would be a worse answer than saying so.
func TestPlanFile_MissingIsAnError(t *testing.T) {
	if _, err := ReadPlan(t.TempDir()); err == nil {
		t.Fatal("a workspace with no plan must say so")
	}
}
