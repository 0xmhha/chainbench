package app

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"os"
	"path/filepath"
	"strings"

	"github.com/0xmhha/chainbench/internal/chainsetup"
	"github.com/0xmhha/chainbench/internal/core/filestore"
	"github.com/0xmhha/chainbench/internal/core/registry"
	"github.com/0xmhha/chainbench/internal/core/session"
	"github.com/0xmhha/chainbench/internal/dsl"
	"github.com/0xmhha/chainbench/internal/resource"
	"github.com/0xmhha/chainbench/internal/testengine"
)

// Saved case bytes remain intact; the registered binary selection replaces
// the default executable through the existing engine request boundary.
// Registered finished genesis files use private immutable execution copies.
// Other file assets and mixed binaries need pinned contracts before use.
type webTestRun struct {
	Cases    []webExecutableCase `json:"cases"`
	Content  []json.RawMessage   `json:"content"`
	InputDir string              `json:"inputDir"`
	Plan     ComposePlan         `json:"plan"`
	Requests []resource.Request  `json:"requests"`
	Genesis  []webPresetGenesis  `json:"genesis,omitempty"`
}

func (e *WebChainEngine) prepareTestRun(ctx context.Context, p *webChainPayload) ([]resource.Request, error) {
	if p.Manifest.Source == "external" {
		return nil, errors.New("external test manifests require an execution projection; use the manifest setup adapter until that projection is available")
	}
	plugin, err := registry.Get(p.Binary.Chain)
	if err != nil {
		return nil, err
	}
	wc, err := deploymentWorkspace(p.Config.DeploymentDocumentInput)
	if err != nil {
		return nil, err
	}
	// Bootstrap derives native IPC names from the executable. A registered
	// asset's source filename is arbitrary, so provision the verified bytes
	// under the native manifest's name on either local or SSH targets.
	p.ExecutionBinary, err = wc.Resolve(resource.PurposeBinaries, p.Binary.SHA256+"/"+plugin.Manifest().Binary)
	if err != nil {
		return nil, err
	}
	cases, err := e.prepareTestCases(ctx, p.Binary.Chain, p.Arguments.CaseRefs)
	if err != nil {
		return nil, err
	}
	run := &webTestRun{Cases: cases, Content: []json.RawMessage{}}
	for _, c := range cases {
		spec, err := dsl.Parse(c.Content)
		if err != nil {
			return nil, err
		}
		// Registered finished files are projected below; all other file and
		// execution contracts still pass through the strict adapter guard.
		if spec.Chain.GenesisExisting != "" {
			if _, err := webAssetRefID(spec.Chain.GenesisExisting); err != nil {
				return nil, err
			}
			spec.Chain.GenesisExisting = ""
		}
		if err = validateWebTestInputs(spec); err != nil {
			return nil, fmt.Errorf("case %s: %w", c.Document.ID, err)
		}
		run.Content = append(run.Content, append(json.RawMessage(nil), c.Content...))
	}
	p.Keys, err = e.pinKeys(ctx)
	if err != nil {
		return nil, err
	}
	run.Content, run.Genesis, err = e.projectTestGenesis(ctx, *p, *run)
	if err != nil {
		return nil, err
	}
	pinned, err := json.Marshal(struct {
		Set, Config DeploymentDocument
		Content     []json.RawMessage
		Keys        webKeySnapshot
	}{p.Set, p.Config, run.Content, p.Keys})
	if err != nil {
		return nil, err
	}
	run.InputDir, err = filepath.Abs(filepath.Join(e.root, "test-inputs", manifestHash(pinned)))
	if err != nil {
		return nil, err
	}
	p.TestRun = run
	if err = writeWebTestInputs(ctx, p); err != nil {
		return nil, err
	}
	in := e.webSuiteInput(*p)
	// PlanSuite may materialize content-addressed overlays. Keep preparation
	// inside an owned input directory, away from the network or target servers.
	in.DataDir = filepath.Join(run.InputDir, "planning")
	layout, plan, err := testengine.PlanSuiteLayout(ctx, in)
	if err != nil {
		return nil, err
	}
	plan = normalizedWebTestPlan(plan, p.ControlDir)
	if plan.Nodes.AutoSize {
		return nil, errors.New("dynamic server sizing requires resolved inventory claims before test execution")
	}
	total := plan.Nodes.BP + plan.Nodes.EN + plan.Nodes.PN
	if plan.Nodes.BP < 1 || total < 1 || total > 128 {
		return nil, errors.New("test layout must have producers and at most 128 nodes")
	}
	if err = validateWebTestComposition(layout); err != nil {
		return nil, err
	}
	run.Plan = plan
	run.Requests = webPresetRequests(layout)
	p.Arguments.Validators = plan.Nodes.BP
	return run.Requests, nil
}

func validateWebTestComposition(in ChainUpIn) error {
	if err := validateWebTableInputs(in); err != nil {
		return err
	}
	if in.GenesisExisting != "" {
		_, err := chainsetup.GenesisOptsFor(ChainGenesisIn{GenesisExisting: in.GenesisExisting, ChainID: in.ChainID, Set: in.GenesisSet, OverlayPath: in.OverlayPath})
		return err
	}
	return nil
}

func validateWebTestInputs(spec dsl.Spec) error {
	if spec.EnvAttach != nil {
		return errors.New("attached test execution requires the read-only attachment adapter")
	}
	if spec.Chain.ManifestPath != "" || spec.Chain.TemplatePath != "" || spec.Chain.GenesisExisting != "" || spec.Chain.Config != "" || spec.EnvBlueprint != "" || len(spec.Chain.GenesisPerBinary) > 0 || spec.EnvUpgrade != nil {
		return errors.New("test file references and mixed-binary upgrades require registered immutable assets")
	}
	if spec.EnvKeys != nil && (spec.EnvKeys.Source != "" && spec.EnvKeys.Source != "keyPreset" || spec.EnvKeys.Ref != "" && spec.EnvKeys.Ref != "presets/keys") {
		return errors.New("this test adapter uses the reviewed key preset; other key sources require their own pinned contract")
	}
	if len(spec.Chain.Binaries) > 1 {
		return errors.New("select registered assets for each named test binary before execution")
	}
	for name := range spec.Chain.Binaries {
		if name != dsl.BinaryDefault {
			return errors.New("only the reviewed default test binary is registered")
		}
	}
	for _, chain := range spec.Chain.BinaryChains {
		if chain != spec.Chain.Name {
			return errors.New("cross-chain binary assets are not resolved by this adapter")
		}
	}
	if len(spec.EnvAccounts) > 0 {
		return errors.New("declared test accounts require job-owned extensible key material before execution")
	}

	// An override may not move a process outside the reviewed paths or ports.
	// The remaining launch and config knobs still travel through the real engine.
	for _, knobs := range spec.EnvLaunch {
		for _, knob := range knobs {
			key, _, _ := strings.Cut(knob, "=")
			if webManagedLaunchInput(key) {
				return errors.New("test launch paths and ports must use resolved resource or asset references")
			}
		}
	}
	statements := append(append([]dsl.Statement{}, spec.Sequence...), webHookStatements(spec)...)
	for _, statement := range statements {
		if binary, ok := statement.Args["binary"]; ok && binary != "" && binary != dsl.BinaryDefault {
			return errors.New("test binary changes require a registered named asset")
		}
		for _, field := range []string{"keyFile", "abiFile", "bytecodeFile"} {
			if value, exists := statement.Args[field]; exists && value != "" {
				return errors.New("test step file references require immutable registered assets")
			}
		}
	}
	return nil
}

func webHookStatements(spec dsl.Spec) []dsl.Statement {
	out := []dsl.Statement{}
	for _, actions := range [][]map[string]any{spec.PreActions, spec.PostActions, spec.OnFailActions} {
		for _, action := range actions {
			name := dsl.ActionName(action)
			args, _ := action[name].(map[string]any)
			out = append(out, dsl.Statement{Do: name, Args: args})
		}
	}
	return out
}

func writeWebTestInputs(ctx context.Context, p *webChainPayload) error {
	return writeWebDeploymentInputs(ctx, p.TestRun.InputDir, p.Set, p.Config)
}

func writeWebDeploymentInputs(ctx context.Context, dir string, set, config DeploymentDocument) error {
	for _, entry := range []struct {
		name string
		doc  DeploymentDocument
	}{{"server-set.yaml", set}, {"workspace-config.yaml", config}} {
		data, err := ExportDeploymentDocument(entry.doc.DeploymentDocumentInput, "yaml")
		if err != nil {
			return err
		}
		name := filepath.Join(dir, entry.name)
		existing, err := os.ReadFile(name)
		if err == nil {
			if string(existing) != string(data) {
				return ErrDeploymentConflict
			}
			continue
		}
		if !os.IsNotExist(err) {
			return err
		}
		if err = (filestore.Local{}).Write(ctx, name, data, 0600); err != nil {
			return err
		}
	}
	return ctx.Err()
}

func (e *WebChainEngine) webSuiteInput(p webChainPayload) RunSuiteIn {
	content := make([][]byte, len(p.TestRun.Content))
	for i, raw := range p.TestRun.Content {
		content[i] = raw
	}
	return RunSuiteIn{SpecContent: content, DataDir: p.ControlDir, Chain: p.Binary.Chain, Binary: p.ExecutionBinary, BinaryOverrides: map[string]string{dsl.BinaryDefault: p.ExecutionBinary}, KeysDir: webAcceptedKeyPath(e.root, p.Keys.SHA256), Server: resource.ServerRef{SetPath: filepath.Join(p.TestRun.InputDir, "server-set.yaml"), Name: p.Arguments.ServerRef}, WorkspaceConfigPath: filepath.Join(p.TestRun.InputDir, "workspace-config.yaml"), ArtifactRoot: webOwnedSessions(e.root), KeepUp: true}
}

func normalizedWebTestPlan(plan ComposePlan, control string) ComposePlan {
	plan.Workspace = control
	if plan.Genesis.Overlay != "" {
		plan.Genesis.Overlay = filepath.Join(control, filepath.Base(plan.Genesis.Overlay))
	}
	return plan
}

func (e *WebChainEngine) executeTestRun(ctx context.Context, a DeploymentActor, p webChainPayload, report func(WebJobPhase) error) (WebJobResult, error) {
	result := WebJobResult{NodeDisposition: "retained"}
	if p.TestRun == nil || len(p.TestRun.Cases) == 0 {
		return result, errors.New("no reviewed test cases")
	}
	keys, err := e.materializeKeys(ctx, p.Keys)
	if err != nil {
		return result, err
	}
	lookup, err := e.documents.jobCredentialLookup(ctx, a, p.Set.DeploymentDocumentInput, p.Input.CredentialBindings)
	if err != nil {
		return result, err
	}
	in := e.webSuiteInput(p)
	in.KeysDir, err = filepath.Abs(keys)
	if err != nil {
		return result, err
	}
	err = e.phase(ctx, a, "test.inputs", report, func() error {
		if err := e.recheckTestGenesis(ctx, p); err != nil {
			return err
		}
		if err := writeWebTestInputs(ctx, &p); err != nil {
			return err
		}
		// Replan inside the private input tree, without mutating retained nodes.
		planning := in
		planning.DataDir = filepath.Join(p.TestRun.InputDir, "planning")
		layout, current, err := testengine.PlanSuiteLayout(ctx, planning)
		if err != nil {
			return err
		}
		before, _ := json.Marshal(p.TestRun.Plan)
		after, _ := json.Marshal(normalizedWebTestPlan(current, p.ControlDir))
		if string(before) != string(after) {
			return ErrDeploymentConflict
		}
		if err = validateWebTestComposition(layout); err != nil {
			return err
		}
		before, _ = json.Marshal(p.TestRun.Requests)
		after, _ = json.Marshal(webPresetRequests(layout))
		if string(before) != string(after) {
			return ErrDeploymentConflict
		}
		// Save the accepted target before provisioning or launching any nodes.
		// A daemon interrupted inside RunSuite must still be able to observe
		// its owned processes without waiting for a final test result.
		metadata, err := json.Marshal(p.Target)
		if err != nil {
			return err
		}
		return (filestore.Local{}).Write(ctx, filepath.Join(p.ControlDir, "web-target.json"), metadata, 0600)
	})
	if err != nil {
		return result, err
	}
	err = e.phase(ctx, a, "test.binary", report, func() error { return e.uploadWebBinary(ctx, p, lookup) })
	if err != nil {
		return result, err
	}
	result.PartialEffects = append(result.PartialEffects, "Verified test binary provisioned: "+p.ExecutionBinary+" · SHA-256 "+p.Binary.SHA256)
	var out RunSuiteOut
	err = e.phase(ctx, a, "test.run", report, func() error {
		if err := e.recheckTestGenesis(ctx, p); err != nil {
			return err
		}
		var runErr error
		out, runErr = RunSuite(ctx, Deps{Command: "web test.run by " + a.ID, ServerLookup: lookup}, in)
		if runErr == nil && out.Summary.Failed() {
			runErr = errors.New("one or more selected test cases failed or were blocked")
		}
		return runErr
	})
	if out.SessionRoot != "" {
		ref, refErr := webTestSessionRef(in.ArtifactRoot, out.SessionRoot)
		if refErr != nil {
			err = errors.Join(err, refErr)
		} else {
			result.RunIDs = []string{ref}
		}
	}
	result.PartialEffects = append(result.PartialEffects, out.SetupSteps...)
	if err != nil {
		result.UnresolvedResources = []string{p.ControlDir, p.Target.DataPath}
	}
	return result, err
}

func webTestSessionRef(root, dir string) (string, error) {
	ref := filepath.ToSlash(session.IDFor(root, dir))
	if ref == "." || ref == ".." || strings.HasPrefix(ref, "../") || filepath.IsAbs(ref) || strings.Contains(ref, `\`) || len(strings.Split(ref, "/")) > 2 {
		return "", errors.New("engine session is outside the owned artifact root")
	}
	return "web:" + ref, nil
}

func (e *WebChainEngine) uploadWebBinary(ctx context.Context, p webChainPayload, lookup resource.Lookup) error {
	asset, err := e.binaryAsset(p.Arguments.AssetID)
	if err != nil {
		return err
	}
	wc, err := deploymentWorkspace(p.Config.DeploymentDocumentInput)
	if err != nil {
		return err
	}
	set, err := deploymentSet(p.Set.DeploymentDocumentInput)
	if err != nil {
		return err
	}
	server, err := set.ByName(p.Arguments.ServerRef)
	if err != nil {
		return err
	}
	access, err := (resource.Opener{Lookup: lookup}).Open(resource.TargetOf(server, wc.DataRoot))
	if err != nil {
		return err
	}
	bytes, err := os.ReadFile(asset.Path)
	if err != nil {
		return err
	}
	if manifestHash(bytes) != p.Binary.SHA256 {
		return ErrDeploymentConflict
	}
	if err = access.Files.Write(ctx, p.ExecutionBinary, bytes, 0755); err != nil {
		return err
	}
	checksum, err := access.Files.Checksum(ctx, p.ExecutionBinary)
	if err != nil {
		return err
	}
	if checksum != "sha256:"+p.Binary.SHA256 {
		return errors.New("provisioned test binary differs from the accepted asset")
	}
	return nil
}

func webAcceptedKeyPath(root, digest string) string {
	name, _ := filepath.Abs(filepath.Join(root, "key-material", digest))
	return name
}

func webOwnedSessions(root string) string {
	name, _ := filepath.Abs(filepath.Join(root, "sessions"))
	return name
}
