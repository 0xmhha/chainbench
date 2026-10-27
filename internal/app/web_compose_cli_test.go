package app

import (
	"context"
	"errors"
	"os"
	"path/filepath"
	"slices"
	"strings"
	"sync"
	"testing"
	"time"

	"github.com/0xmhha/chainbench/internal/chainsetup"
)

// composeCLI stands in for `chainbench chain up`: each status read reports one
// more finished step, and the command ends once the last one is reported, or
// fails at failAt.
type composeCLI struct {
	mu     sync.Mutex
	steps  []string
	done   int
	failAt string
	// failed is the step record the CLI leaves for failAt, when set.
	failed chainsetup.Step
	args   []string
	ended  chan struct{}
}

func (c *composeCLI) ChainStatus(context.Context, string) (State, error) {
	c.mu.Lock()
	defer c.mu.Unlock()
	st := State{Steps: map[string]chainsetup.Step{}}
	for i, name := range c.steps {
		if i >= c.done || name == c.failAt {
			break
		}
		st.Steps[name] = chainsetup.Step{Done: true}
	}
	if c.done < len(c.steps) && c.steps[c.done] == c.failAt && c.failed != (chainsetup.Step{}) {
		st.Steps[c.failAt] = c.failed
	}
	if c.done < len(c.steps) && c.steps[c.done] != c.failAt {
		c.done++
	}
	if c.done == len(c.steps) || c.steps[c.done] == c.failAt {
		select {
		case <-c.ended:
		default:
			close(c.ended)
		}
	}
	return st, nil
}

func (c *composeCLI) Run(_ context.Context, args ...string) ([]byte, error) {
	c.mu.Lock()
	c.args = args
	c.mu.Unlock()
	<-c.ended
	if c.failAt != "" {
		return nil, errors.New("chainbench chain up: " + c.failAt + " failed")
	}
	return nil, nil
}

func phaseLog(t *testing.T, cli *composeCLI, steps []string, args []string) ([]WebJobPhase, string, error) {
	t.Helper()
	var got []WebJobPhase
	dir := t.TempDir()
	if err := os.WriteFile(filepath.Join(dir, "chain-record.json"), []byte("{}"), 0o600); err != nil {
		t.Fatal(err)
	}
	failed, err := composeThroughCLI(context.Background(), cli, dir, steps, args, func(p WebJobPhase) error {
		got = append(got, p)
		return nil
	}, 0)
	return got, failed, err
}

// TestComposeThroughCLIReportsEachStepAsTheCLIRecordsIt: the job shows the
// composition steps in order, each running then succeeded, as the CLI records
// them while `chain up` runs.
func TestComposeThroughCLIReportsEachStepAsTheCLIRecordsIt(t *testing.T) {
	steps := []string{"new", "place", "keys", "init"}
	cli := &composeCLI{steps: steps, ended: make(chan struct{})}
	got, failed, err := phaseLog(t, cli, steps, []string{"chain", "up", "--stage", "init"})
	if err != nil || failed != "" {
		t.Fatalf("compose = %q %v", failed, err)
	}
	var order []string
	for _, p := range got {
		order = append(order, p.Name+":"+p.State)
	}
	want := []string{"new:running", "new:succeeded", "place:running", "place:succeeded", "keys:running", "keys:succeeded", "init:running", "init:succeeded"}
	if !slices.Equal(order, want) {
		t.Fatalf("phases = %v, want %v", order, want)
	}
	if !slices.Equal(cli.args, []string{"chain", "up", "--stage", "init"}) {
		t.Fatalf("args = %v", cli.args)
	}
}

// TestComposeThroughCLINamesTheStepThatFailed: a failed `chain up` marks the
// first unfinished step failed with the CLI's reason, which decides what the
// job says about the target.
func TestComposeThroughCLINamesTheStepThatFailed(t *testing.T) {
	steps := []string{"new", "place", "keys", "deploy", "init"}
	cli := &composeCLI{steps: steps, failAt: "deploy", ended: make(chan struct{})}
	got, failed, err := phaseLog(t, cli, steps, []string{"chain", "up"})
	if err == nil || failed != "deploy" {
		t.Fatalf("compose = %q %v, want a failure at deploy", failed, err)
	}
	last := got[len(got)-1]
	if last.Name != "deploy" || last.State != "failed" || last.Message == "" {
		t.Fatalf("last phase = %+v, want deploy failed with the CLI's reason", last)
	}
	for _, p := range got {
		if p.Name == "init" {
			t.Fatalf("a step after the failure was reported: %+v", p)
		}
	}
	if last.Failure != nil {
		t.Fatalf("an unclassified failure carries a classification: %+v", last.Failure)
	}
}

// TestComposeThroughCLICarriesTheClassifiedFailure: when the CLI recorded what
// kind of failure stopped a step, the failed phase carries that state and its
// detail, so a screen need not read the reason out of the message.
func TestComposeThroughCLICarriesTheClassifiedFailure(t *testing.T) {
	steps := []string{"new", "place", "deploy", "init"}
	cli := &composeCLI{steps: steps, failAt: "deploy", ended: make(chan struct{}),
		failed: chainsetup.Step{Err: "listen tcp :30303: address already in use", Failed: "ChainLaunchNodesFailPortBusy", Detail: "node2 p2p port 30303 is in use"}}
	got, failed, err := phaseLog(t, cli, steps, []string{"chain", "up"})
	if err == nil || failed != "deploy" {
		t.Fatalf("compose = %q %v", failed, err)
	}
	last := got[len(got)-1]
	want := &WebPhaseFailure{State: "ChainLaunchNodesFailPortBusy", Detail: "node2 p2p port 30303 is in use"}
	if last.Failure == nil || *last.Failure != *want {
		t.Fatalf("failed phase failure = %+v, want %+v", last.Failure, want)
	}
}

// TestRunControlRunsOneCommandWithTheJobsLogins: a node control is one
// chainbench command on the recorded network, with logins that do not outlive it.
func TestRunControlRunsOneCommandWithTheJobsLogins(t *testing.T) {
	e, p := webRunPlanFixture(t, `{"bp":4}`)
	cli := &uploadsCLI{}
	e.UseCLI(cli)
	if err := e.runControl(context.Background(), DeploymentActor{ID: "operator", Role: "operator"}, p, nodeSwapArgs(2, "/bin/next", []string{"syncmode=full"})...); err != nil {
		t.Fatal(err)
	}
	args := cli.calls[0]
	want := []string{"node", "swap", "--index", "2", "--purpose", "web-reviewed-config", "--binary", "/bin/next", "--config", "syncmode=full", "--workspace-dir", p.ControlDir}
	if !slices.Equal(args[:len(want)], want) {
		t.Fatalf("args = %v, want %v then logins", args, want)
	}
	logins := flagValue(args, "--server-logins")
	if logins == "" {
		t.Fatal("no logins were named")
	}
	if _, err := os.Stat(logins); !os.IsNotExist(err) {
		t.Fatalf("the job's logins outlived the command: %v", err)
	}
}

// TestWebJobsCleanAPhaseFailureDetail: a classified failure's detail is cleaned
// like the phase message before the job records it.
func TestWebJobsCleanAPhaseFailureDetail(t *testing.T) {
	jobs, err := OpenWebJobs(t.TempDir(), nil, nil, nil)
	if err != nil {
		t.Fatal(err)
	}
	jobs.state.Plans["plan"] = savedWebPlan{Prepared: WebPreparedJob{Phases: []string{"deploy"}}}
	jobs.state.Jobs["job"] = WebJob{ID: "job", ActorID: "operator", PlanID: "plan", State: "running", Phases: []WebJobPhase{}}
	pem := "-----BEGIN PRIVATE KEY-----\nMIIEsecret\n-----END PRIVATE KEY-----"
	phase := WebJobPhase{Name: "deploy", State: "failed", StartedAt: time.Now().UTC(), Failure: &WebPhaseFailure{State: "ChainLaunchNodesFailPortBusy", Detail: "key file held " + pem}}
	if err = jobs.progress("job", phase); err != nil {
		t.Fatal(err)
	}
	got := jobs.state.Jobs["job"].Phases[0].Failure
	if got == nil || strings.Contains(got.Detail, "MIIEsecret") || !strings.Contains(got.Detail, "[REDACTED]") {
		t.Fatalf("recorded failure = %+v", got)
	}
	if strings.Contains(phase.Failure.Detail, "[REDACTED]") {
		t.Fatal("cleaning changed the caller's phase")
	}
}
