package app

import (
	"context"
	"encoding/json"
	"errors"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/0xmhha/chainbench/internal/dsl"
	"github.com/0xmhha/chainbench/internal/resource"
)

func TestWebTestRunPersistsTargetBeforeAnyNodeLaunch(t *testing.T) {
	e, p := webRunPlanFixture(t, `{"bp":4}`)
	if _, err := e.prepareTestRun(context.Background(), &p); err != nil {
		t.Fatal(err)
	}
	p.Target = resource.Inspection{HostIdentity: "reviewed-target", DataPath: "/reviewed/data", Transport: "local"}
	stop := errors.New("stop before binary transfer or node launch")
	seen := false
	_, err := e.executeTestRun(context.Background(), DeploymentActor{ID: "operator", Role: "operator"}, p, func(phase WebJobPhase) error {
		if phase.Name != "test.binary" || phase.State != "running" {
			return nil
		}
		seen = true
		raw, readErr := os.ReadFile(filepath.Join(p.ControlDir, "web-target.json"))
		if readErr != nil {
			t.Errorf("target needed for restart observations was not persisted before effects: %v", readErr)
		} else {
			var target resource.Inspection
			if decodeErr := json.Unmarshal(raw, &target); decodeErr != nil || target != p.Target {
				t.Errorf("persisted target differs from accepted target: %+v %v", target, decodeErr)
			}
			info, statErr := os.Stat(filepath.Join(p.ControlDir, "web-target.json"))
			if statErr != nil || info.Mode().Perm() != 0600 {
				t.Error("target snapshot must be private")
			}
		}
		return stop
	})
	if !seen || !errors.Is(err, stop) {
		t.Fatalf("did not stop before execution: %v", err)
	}
	if _, err := os.Stat(filepath.Join(p.ControlDir, "chain-record.json")); !os.IsNotExist(err) {
		t.Fatal("fixture performed a native composition")
	}
}

func webRunPlanFixture(t *testing.T, layout string) (*WebChainEngine, webChainPayload) {
	t.Helper()
	e, _ := keySnapshotEngine(t)
	actor := DeploymentActor{ID: "operator", Role: "operator"}
	raw := json.RawMessage(strings.Replace(string(editorCase(`[{"expect":"blockNumber","compare":"GreaterOrEqual","is":0}]`)), `{"bp":4}`, layout, 1))
	c, err := e.documents.SaveDocument(actor, "", 0, DeploymentDocumentInput{Kind: "case", Name: "saved", ContractVersion: "2", Content: raw})
	if err != nil {
		t.Fatal(err)
	}
	set := deploymentTestSet()
	set.Content = json.RawMessage(strings.Replace(string(set.Content), `localhost.`, `127.0.0.1`, 1))
	config := deploymentTestConfig()
	config.Content = json.RawMessage(strings.Replace(string(config.Content), `/data/chainbench`, filepath.Join(e.root, "target"), 1))
	return e, webChainPayload{Arguments: webChainArguments{ServerRef: "ssh", CaseRefs: []DeploymentDocumentRef{{c.ID, c.Revision}}}, Manifest: ManagedManifest{Source: "builtin"}, Binary: ManifestBinaryEvidence{Chain: "stablenet", SHA256: strings.Repeat("b", 64)}, ExecutionBinary: filepath.Join(e.root, "binary"), ControlDir: filepath.Join(e.root, "networks", "owned"), Set: DeploymentDocument{DeploymentDocumentInput: set}, Config: DeploymentDocument{DeploymentDocumentInput: config}}
}

func TestWebTestRunPreservesDeclaredLayoutAndDefaultBinary(t *testing.T) {
	e, p := webRunPlanFixture(t, `{"bp":4,"pn":1,"en":2}`)
	requests, err := e.prepareTestRun(context.Background(), &p)
	if err != nil {
		t.Fatal(err)
	}
	if len(requests) != 7 || p.TestRun.Plan.Nodes.BP != 4 || p.TestRun.Plan.Nodes.EN != 2 || p.TestRun.Plan.Nodes.PN != 1 {
		t.Fatal("test topology was reduced to producer count")
	}
	if requests[4].Role != "en" || requests[6].Role != "pn" {
		t.Fatal("placement order differs from composer")
	}
	if filepath.Base(p.ExecutionBinary) != "gstable" {
		t.Fatal("renamed registered asset changed the native IPC socket name", p.ExecutionBinary)
	}
	if p.TestRun.Plan.Binary != p.ExecutionBinary || p.TestRun.Plan.Binaries[dsl.BinaryDefault] != p.ExecutionBinary {
		t.Fatal("named default bypassed selected asset")
	}
	if p.TestRun.Plan.Keys.Dir != webAcceptedKeyPath(e.root, p.Keys.SHA256) || p.TestRun.Plan.Workspace != p.ControlDir {
		t.Fatal("accepted keys/control path lost")
	}
	if _, err = os.Stat(p.ControlDir); !os.IsNotExist(err) {
		t.Fatal("preparing a test touched the retained network")
	}
	if !strings.Contains(string(p.TestRun.Cases[0].Content), `"gstable"`) {
		t.Fatal("execution projection rewrote the saved original")
	}
	in := e.webSuiteInput(p)
	if !in.KeepUp || !in.ReadOnlyKeys || in.BPCount != 0 || in.ArtifactRoot != filepath.Join(e.root, "sessions") {
		t.Fatal("test default retention/layout/evidence changed")
	}
	// A modified cached declaration must be refused rather than changing targets.
	if err = os.WriteFile(filepath.Join(p.TestRun.InputDir, "workspace-config.yaml"), []byte("redirected"), 0600); err != nil {
		t.Fatal(err)
	}
	if err = writeWebTestInputs(context.Background(), &p); err == nil {
		t.Fatal("changed cached input accepted")
	}
}

func TestWebTestRunRefusesUnclaimedPathsAndBinaries(t *testing.T) {
	for name, edit := range map[string]func(*dsl.Spec){
		"data-dir":     func(s *dsl.Spec) { s.EnvLaunch = map[string][]string{"node1": {"datadir=/other"}} },
		"port":         func(s *dsl.Spec) { s.EnvLaunch = map[string][]string{"node1": {"http.port=19999"}} },
		"metrics-port": func(s *dsl.Spec) { s.EnvLaunch = map[string][]string{"all": {"metrics.port=19999"}} },
		"key-file":     func(s *dsl.Spec) { s.EnvAccounts = map[string]dsl.AccountV2{"account": {KeyFile: "/private"}} },
		// A declared second binary is refused unless the job binds it to a
		// verified asset (TestWebTestRunRefusesMissingWrongChainAndUndeclaredNamedBinaries);
		// its declared path is never what runs.
		"step-binary": func(s *dsl.Spec) {
			s.Sequence = []dsl.Statement{{Do: "swapNode", Args: map[string]any{"binary": "/other"}}}
		},
		"manifest":    func(s *dsl.Spec) { s.Chain.ManifestPath = "/other" },
		"blueprint":   func(s *dsl.Spec) { s.EnvBlueprint = "/other" },
		"custom-keys": func(s *dsl.Spec) { s.EnvKeys = &dsl.KeySourceV2{Source: "keyPreset", Ref: "/other"} },
		"attach":      func(s *dsl.Spec) { s.EnvAttach = &dsl.AttachV2{RPC: []string{"http://example.invalid"}} },
	} {
		t.Run(name, func(t *testing.T) {
			s, err := dsl.Parse(editorCase(`[{"expect":"blockNumber","is":0}]`))
			if err != nil {
				t.Fatal(err)
			}
			edit(&s)
			if err = validateWebTestInputs(s); err == nil {
				t.Fatal("unclaimed input was executable")
			}
		})
	}
}

func TestWebTestSessionReferencesCannotEscapeOrCollideWithLegacy(t *testing.T) {
	root := t.TempDir()
	ref, err := webTestSessionRef(root, filepath.Join(root, "process", "run"))
	if err != nil || ref != "web:process/run" {
		t.Fatal(ref, err)
	}
	for _, dir := range []string{root, filepath.Dir(root), filepath.Join(root, "a", "b", "c")} {
		if _, err = webTestSessionRef(root, dir); err == nil {
			t.Fatal("outside or invalid engine session accepted")
		}
	}
	if historySessionID(ref) == historySessionID("process/run") {
		t.Fatal("web and legacy source namespaces collide")
	}
}

func TestWebTestBinaryProvisioningChecksBytesBeforeLocalWrite(t *testing.T) {
	e, p := webRunPlanFixture(t, `{"bp":4}`)
	source := filepath.Join(t.TempDir(), "arbitrary-source-name")
	bytes := []byte("accepted fixture file bytes")
	if err := os.WriteFile(source, bytes, 0600); err != nil {
		t.Fatal(err)
	}
	p.Arguments.AssetID = "verified"
	p.Binary.SHA256 = manifestHash(bytes)
	if _, err := e.prepareTestRun(context.Background(), &p); err != nil {
		t.Fatal(err)
	}
	e.assets["verified"] = ManifestBinary{ID: "verified", Path: source}
	if err := e.uploadWebBinary(context.Background(), p, nil); err != nil {
		t.Fatal(err)
	}
	actual, err := os.ReadFile(p.ExecutionBinary)
	if err != nil || string(actual) != string(bytes) {
		t.Fatal("selected source was not provisioned", err)
	}
	info, err := os.Stat(p.ExecutionBinary)
	if err != nil || info.Mode().Perm() != 0755 {
		t.Fatal("provisioned binary is not executable", err)
	}
	if err = os.WriteFile(source, []byte("changed source"), 0600); err != nil {
		t.Fatal(err)
	}
	if err = e.uploadWebBinary(context.Background(), p, nil); err == nil {
		t.Fatal("changed source was provisioned after acceptance")
	}
	actual, err = os.ReadFile(p.ExecutionBinary)
	if err != nil || string(actual) != string(bytes) {
		t.Fatal("rejected source changed target bytes", err)
	}
}
