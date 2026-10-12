package app

import (
	"context"
	"errors"
	"os"
	"strings"
	"testing"

	"github.com/0xmhha/chainbench/internal/core/node"
)

func TestWebTestRunUsesResolvedTableOrderForClaims(t *testing.T) {
	e, p := webRunPlanFixture(t, `{"nodes":[{"index":3,"role":"bp"},{"index":1,"role":"en","sync":"archive"},{"index":2,"role":"bp"},{"index":5,"role":"bp"},{"index":4,"role":"bp"}]}`)
	requests, err := e.prepareTestRun(context.Background(), &p)
	if err != nil {
		t.Fatalf("engine-supported node table rejected: %v", err)
	}
	if len(requests) != 5 || requests[0].Role != node.RoleEN || requests[0].Label != "node1" || requests[1].Role != node.RoleBP || requests[1].Label != "node2" {
		t.Fatal("table was approximated by producer-first counts", requests)
	}
	if !p.TestRun.Plan.Nodes.Declared || p.TestRun.Plan.Nodes.BP != 4 || p.TestRun.Plan.Nodes.EN != 1 {
		t.Fatal("review differs from declared layout")
	}
	if _, err = os.Stat(p.ControlDir); !os.IsNotExist(err) {
		t.Fatal("table planning touched retained network", err)
	}
}

func TestWebTestRunRechecksReviewedTableBeforeLaunch(t *testing.T) {
	e, p := webRunPlanFixture(t, `{"nodes":[{"index":1,"role":"en"},{"index":2,"role":"bp"}]}`)
	if _, err := e.prepareTestRun(context.Background(), &p); err != nil {
		t.Fatal(err)
	}
	p.TestRun.Requests[0], p.TestRun.Requests[1] = p.TestRun.Requests[1], p.TestRun.Requests[0]
	_, err := e.executeTestRun(context.Background(), DeploymentActor{ID: "operator", Role: "operator"}, p, func(WebJobPhase) error { return nil })
	if !errors.Is(err, ErrDeploymentConflict) {
		t.Fatal("changed reviewed table reached launch", err)
	}
	if _, err = os.Stat(p.ControlDir); !os.IsNotExist(err) {
		t.Fatal("changed table touched retained network", err)
	}
}

func TestWebTestRunTableRefusesUnregisteredPerNodeInputs(t *testing.T) {
	for _, declaration := range []string{`"config":"/outside.toml"`, `"key":"/outside.key"`, `"key":"0x1234"`, `"binary":"unregistered"`} {
		t.Run(strings.Split(declaration, ":")[0], func(t *testing.T) {
			e, p := webRunPlanFixture(t, `{"nodes":[{"index":1,"role":"bp",`+declaration+`}]}`)
			if _, err := e.prepareTestRun(context.Background(), &p); err == nil {
				t.Fatal("unregistered node input was executable")
			}
			if _, err := os.Stat(p.ControlDir); !os.IsNotExist(err) {
				t.Fatal("rejected table touched retained network", err)
			}
		})
	}
}
