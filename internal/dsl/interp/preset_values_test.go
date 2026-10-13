package interp_test

import (
	"context"
	"reflect"
	"testing"

	"github.com/0xmhha/chainbench/internal/core/session"
	"github.com/0xmhha/chainbench/internal/dsl"
	"github.com/0xmhha/chainbench/internal/dsl/interp"
)

type initialValueAction func(*interp.ActionCtx) error

func (a initialValueAction) Do(_ context.Context, ac *interp.ActionCtx) error { return a(ac) }

func TestPresetValuesSeedEveryRunAndRemainIsolated(t *testing.T) {
	raw := []byte(`{"schemaVersion":"2","kind":"case","id":"initial","chainPreset":{"chain":"wbft","values":{"count":2,"enabled":true,"recipient":"0xabc","empty":null,"payload":{"items":[{"n":2}]}}},"hooks":{"pre":[{"do":"capture","phase":"pre","count":"$count","enabled":"${enabled}","empty":"$empty","payload":"$payload","text":"to=${recipient}"}],"post":[{"do":"capture","phase":"post","count":"$count"}]},"steps":[{"do":"mutate","payload":"$payload","save":"count"},{"expect":"check","is":"$count"}]}`)
	s, err := dsl.Parse(raw)
	if err != nil {
		t.Fatal(err)
	}
	reg := interp.NewRegistry()
	captures := 0
	reg.RegisterAction("capture", initialValueAction(func(ac *interp.ActionCtx) error {
		captures++
		if ac.Args["phase"] == "pre" {
			if ac.Args["count"] != float64(2) || ac.Args["enabled"] != true || ac.Args["empty"] != nil || ac.Args["text"] != "to=0xabc" {
				t.Errorf("initial types changed: %+v", ac.Args)
			}
			wanted := map[string]any{"items": []any{map[string]any{"n": float64(2)}}}
			if !reflect.DeepEqual(ac.Args["payload"], wanted) {
				t.Errorf("initial object leaked between runs: %+v", ac.Args["payload"])
			}
		} else if ac.Args["count"] != 5 {
			t.Errorf("saved result did not shadow initial value: %+v", ac.Args)
		}
		return nil
	}))
	reg.RegisterAction("mutate", initialValueAction(func(ac *interp.ActionCtx) error {
		ac.Args["payload"].(map[string]any)["items"].([]any)[0].(map[string]any)["n"] = 99
		ac.Value = 5
		return nil
	}))
	reg.RegisterAssertion("check", fakeAssertion{pass: true})
	if missing := interp.Unresolved(s, reg); len(missing) != 0 {
		t.Fatalf("initial values rejected offline: %v", missing)
	}
	runner := interp.NewInterpreter(interp.Deps{Actions: reg})
	for n := 0; n < 2; n++ {
		status, err := runner.Run(context.Background(), s, testEnv(t), &fakeRecord{})
		if err != nil || status != session.StatusPass {
			t.Fatal(status, err)
		}
	}
	if captures != 4 {
		t.Fatalf("hooks did not consume values: %d", captures)
	}
	other, err := dsl.Parse([]byte(`{"schemaVersion":"2","kind":"case","id":"other","chainPreset":{"chain":"wbft"},"steps":[{"expect":"check","is":"$count"}]}`))
	if err != nil {
		t.Fatal(err)
	}
	if len(interp.Unresolved(other, reg)) == 0 {
		t.Fatal("initial scope leaked into another case")
	}
	status, err := runner.Run(context.Background(), other, testEnv(t), &fakeRecord{})
	if err != nil || status != session.StatusFail {
		t.Fatalf("another case inherited count: %s %v", status, err)
	}
}
