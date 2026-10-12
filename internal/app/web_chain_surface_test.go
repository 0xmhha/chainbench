package app

import (
	"context"
	"errors"
	"fmt"
	"os"
	"path/filepath"
	"regexp"
	"slices"
	"strings"
	"testing"
	"time"

	"github.com/0xmhha/chainbench/internal/core/chainsurface"
	"github.com/0xmhha/chainbench/internal/testengine"
)

var webSurfaceAnalysis = filepath.Join("..", "..", "docs", "chain-analysis")

func webSurfaceCatalog(t *testing.T, chain string) chainsurface.Catalog {
	t.Helper()
	c, err := LoadChainSurface(webSurfaceAnalysis, chain)
	if err != nil {
		t.Fatal(err)
	}
	return c
}

// webHelpOf renders flags the way a help page defines them, so a probe can
// be built from the capture and then changed one flag at a time.
func webHelpOf(flags []chainsurface.HelpFlag, drop string) []byte {
	var b strings.Builder
	b.WriteString("GLOBAL OPTIONS:\n")
	for _, f := range flags {
		if f.Spelling == drop {
			continue
		}
		value := " value"
		if f.Boolean {
			value = ""
		}
		fmt.Fprintf(&b, "    %s%s                        description\n", f.Spelling, value)
	}
	return []byte(b.String())
}

func webSurfaceProbe(c chainsurface.Catalog, commit, drop string) *webBinaryProbe {
	commands := map[string][]byte{}
	for _, cmd := range c.Commands {
		commands[strings.Join(cmd.Path, " ")] = webHelpOf(cmd.Flags, "")
	}
	hidden := map[string]bool{}
	for _, f := range c.Hidden {
		hidden[f] = true
	}
	return &webBinaryProbe{Help: webHelpOf(c.Global, drop), Commands: commands, Hidden: hidden,
		Fingerprint: WebBinaryFingerprint{AssetID: "asset-1", Checksum: strings.Repeat("a", 64), ChainID: "stablenet", Version: "Gstable", ChainCommit: commit, OS: "darwin", Architecture: "arm64", VerifiedAt: time.Unix(0, 0).UTC()}}
}

func webSurfaceOption(t *testing.T, s WebChainSurface, id string) WebChainOption {
	t.Helper()
	for _, o := range s.Options {
		if o.ID == id {
			return o
		}
	}
	t.Fatalf("option %s missing", id)
	return WebChainOption{}
}

func webSurfaceCommand(t *testing.T, s WebChainSurface, id string) WebChainCommand {
	t.Helper()
	for _, c := range s.Commands {
		if c.ID == id {
			return c
		}
	}
	t.Fatalf("command %s missing", id)
	return WebChainCommand{}
}

func TestChainSurfaceJoinsDocumentationAndEngineMapping(t *testing.T) {
	c := webSurfaceCatalog(t, "stablenet")
	s, err := JoinChainSurface("stablenet", c, nil)
	if err != nil {
		t.Fatal(err)
	}
	if s.Verification != "documented" || s.Binary != nil || s.Dialect != "geth114" || len(s.Fingerprint) != 64 || len(s.Sources) != 4 {
		t.Fatalf("surface %+v", s)
	}
	if s.Sources[0].Path != "docs/chain-analysis/gstable/cli-surface.txt" || s.Sources[0].ChainCommit != "0937ac5c9" || s.Sources[0].CapturedAt == nil {
		t.Fatalf("source %+v", s.Sources[0])
	}
	port := webSurfaceOption(t, s, "global:--http.port")
	if port.OptionKey == "" || port.Emission != "derived" || port.Support.Selectable {
		t.Errorf("--http.port is a per-node value the engine derives: %+v", port)
	}
	api := webSurfaceOption(t, s, "global:--http.api")
	if api.Emission != "user-input" || !api.Support.Selectable || api.SchemaRef == "" || api.ContractID != "chain-preset" {
		t.Errorf("--http.api is selectable user input: %+v", api)
	}
	if o := webSurfaceOption(t, s, "global:--password"); o.Emission != "private-binding" || o.Support.Selectable {
		t.Errorf("--password comes from private key material: %+v", o)
	}
	if o := webSurfaceOption(t, s, "global:--exitwhensynced"); o.Support.Status != "engine-unmapped" || o.Support.Selectable {
		t.Errorf("an unmapped flag is not selectable: %+v", o)
	}
	if o := webSurfaceOption(t, s, "global:--docroot"); o.Support.Status != "binary-absent" {
		t.Errorf("--docroot is refused by the binary: %+v", o)
	}
	if o := webSurfaceOption(t, s, "global:--vmodule"); !o.Hidden || o.Support.Selectable {
		t.Errorf("--vmodule is hidden and unmapped: %+v", o)
	}
	init := webSurfaceCommand(t, s, "command:init")
	if init.Support.Status != "supported" || !slices.Contains(init.EngineUsecases, "chain.setup") || !slices.Equal(init.Positional, []string{"<genesisPath>"}) {
		t.Errorf("init: %+v", init)
	}
	if o := webSurfaceOption(t, s, "local:init:--datadir"); o.Scope != "command-local" || o.Emission != "derived" || !slices.Equal(o.CommandPath, []string{"init"}) {
		t.Errorf("init --datadir: %+v", o)
	}
	if r := webSurfaceCommand(t, s, "command:removedb"); r.Support.Status != "engine-unmapped" || len(r.EngineUsecases) != 0 {
		t.Errorf("a documented command grants no execution: %+v", r)
	}
	if _, err := LoadChainSurface(webSurfaceAnalysis, "unknown"); err == nil {
		t.Error("an unknown chain loaded")
	}
}

func TestChainSurfaceJudgesTheSelectedBinary(t *testing.T) {
	c := webSurfaceCatalog(t, "stablenet")
	documented, _ := JoinChainSurface("stablenet", c, nil)
	verified, err := JoinChainSurface("stablenet", c, webSurfaceProbe(c, "0937ac5c93d4f56ded5b24899f381d3d1f208c02", ""))
	if err != nil {
		t.Fatal(err)
	}
	if verified.Verification != "verified" || verified.Binary == nil || verified.Fingerprint == documented.Fingerprint {
		t.Fatalf("verified %s binary=%v", verified.Verification, verified.Binary)
	}
	if o := webSurfaceOption(t, verified, "global:--http.api"); o.Support.Status != "supported" || !slices.Contains(o.Support.EvidenceRefs, "binary-probe") {
		t.Errorf("%+v", o)
	}
	stale, _ := JoinChainSurface("stablenet", c, webSurfaceProbe(c, strings.Repeat("1", 40), ""))
	if stale.Verification != "stale" || len(stale.Limitations) == 0 || !strings.Contains(stale.Limitations[0], "0937ac5c9") {
		t.Errorf("a different commit is stale: %s %v", stale.Verification, stale.Limitations)
	}
	broken, _ := JoinChainSurface("stablenet", c, webSurfaceProbe(c, "0937ac5c93d4f56ded5b24899f381d3d1f208c02", "--http.api"))
	if broken.Verification != "incompatible" || broken.Fingerprint == verified.Fingerprint {
		t.Errorf("a binary without a mapped flag is incompatible: %s", broken.Verification)
	}
	if o := webSurfaceOption(t, broken, "global:--http.api"); o.Support.Status != "binary-absent" || o.Support.Selectable {
		t.Errorf("%+v", o)
	}
	refusing := webSurfaceProbe(c, "0937ac5c93d4f56ded5b24899f381d3d1f208c02", "")
	refusing.Hidden["--vmodule"] = false
	if o := webSurfaceOption(t, mustJoin(t, c, refusing), "global:--vmodule"); o.Support.Status != "binary-absent" {
		t.Errorf("a hidden flag the binary rejects is absent: %+v", o)
	}
	noInit := webSurfaceProbe(c, "0937ac5c93d4f56ded5b24899f381d3d1f208c02", "")
	delete(noInit.Commands, "init")
	if s := mustJoin(t, c, noInit); s.Verification != "incompatible" || webSurfaceCommand(t, s, "command:init").Support.Status != "binary-absent" {
		t.Errorf("init help is required for the engine's init: %s", s.Verification)
	}
}

func mustJoin(t *testing.T, c chainsurface.Catalog, p *webBinaryProbe) WebChainSurface {
	t.Helper()
	s, err := JoinChainSurface("stablenet", c, p)
	if err != nil {
		t.Fatal(err)
	}
	return s
}

func TestChainSurfaceListsWemixEngineCommandsOnlyForThatFamily(t *testing.T) {
	w, err := JoinChainSurface("wemix", webSurfaceCatalog(t, "wemix"), nil)
	if err != nil {
		t.Fatal(err)
	}
	if g := webSurfaceCommand(t, w, "command:wemix genesis"); g.Support.Status != "supported" {
		t.Errorf("%+v", g)
	}
	if a := webSurfaceCommand(t, w, "command:attach"); a.Support.Status != "supported" {
		t.Errorf("%+v", a)
	}
	s, _ := JoinChainSurface("stablenet", webSurfaceCatalog(t, "stablenet"), nil)
	if a := webSurfaceCommand(t, s, "command:attach"); a.Support.Status != "engine-unmapped" {
		t.Errorf("stablenet never runs attach: %+v", a)
	}
}

// webSurfaceBinary writes an executable that answers help requests the way
// the captured binary does, minus one global flag when drop is set. It
// answers nothing else, so a probe that tried to start a node would fail.
func webSurfaceBinary(t *testing.T, c chainsurface.Catalog, drop string) string {
	t.Helper()
	dir := t.TempDir()
	write := func(name string, body []byte) string {
		path := filepath.Join(dir, name)
		if err := os.WriteFile(path, body, 0o600); err != nil {
			t.Fatal(err)
		}
		return path
	}
	var script strings.Builder
	script.WriteString("#!/bin/sh\ncase \"$*\" in\n")
	fmt.Fprintf(&script, "  --help) cat %q ;;\n", write("top", webHelpOf(c.Global, drop)))
	for i, cmd := range c.Commands {
		fmt.Fprintf(&script, "  %q) cat %q ;;\n", strings.Join(cmd.Path, " ")+" --help", write(fmt.Sprintf("c%d", i), webHelpOf(cmd.Flags, "")))
	}
	for _, f := range c.Hidden {
		fmt.Fprintf(&script, "  %q) echo usage ;;\n", f+" --help --help")
	}
	for _, f := range c.Global {
		if f.Spelling != drop {
			fmt.Fprintf(&script, "  %q) echo usage ;;\n", f.Spelling+" --help --help")
		}
	}
	script.WriteString("  *) echo \"flag provided but not defined\"; exit 2 ;;\nesac\n")
	path := filepath.Join(dir, "gstable")
	if err := os.WriteFile(path, []byte(script.String()), 0o700); err != nil {
		t.Fatal(err)
	}
	return path
}

func webSurfaceEngine(t *testing.T, binary string) (*WebChainEngine, webChainPayload) {
	t.Helper()
	asset := ManifestBinary{ID: "asset-1", Chain: "stablenet", Path: binary, SHA256: strings.Repeat("b", 64), Commit: "0937ac5c93d4f56ded5b24899f381d3d1f208c02"}
	e := &WebChainEngine{root: t.TempDir(), surfaces: webSurfaceAnalysis, assets: map[string]ManifestBinary{asset.ID: asset}}
	p := webChainPayload{Arguments: webChainArguments{AssetID: asset.ID}, Binary: ManifestBinaryEvidence{ID: asset.ID, Chain: "stablenet", SHA256: asset.SHA256, Version: "Gstable", OS: "darwin", Architecture: "arm64"}}
	return e, p
}

func TestPlanPinsTheSurfaceOfEachBinaryItRuns(t *testing.T) {
	c := webSurfaceCatalog(t, "stablenet")
	e, p := webSurfaceEngine(t, webSurfaceBinary(t, c, ""))
	p.Preset = &webPresetComposition{Plan: ComposePlan{Launch: map[string][]testengine.PlanKnob{"all": {{Knob: "http.api=eth,net"}}}}}
	if err := e.pinPlanSurfaces(context.Background(), &p); err != nil {
		t.Fatal(err)
	}
	if len(p.ChainSurfaces) != 1 || p.ChainSurfaces[0].AssetID != "asset-1" || len(p.ChainSurfaces[0].Fingerprint) != 64 || p.ChainSurfaces[0].ContractVersion == "" {
		t.Fatalf("pinned %+v", p.ChainSurfaces)
	}
	if err := e.recheckPlanSurfaces(context.Background(), p); err != nil {
		t.Fatalf("an unchanged surface was refused: %v", err)
	}
	if _, err := os.Stat(filepath.Join(e.root, "chain-surfaces", strings.Repeat("b", 64), "help.txt")); err != nil {
		t.Errorf("the probed help is not kept as evidence: %v", err)
	}
}

func TestPlanRefusesABinaryWithoutAnOptionItSets(t *testing.T) {
	c := webSurfaceCatalog(t, "stablenet")
	e, p := webSurfaceEngine(t, webSurfaceBinary(t, c, "--http.api"))
	p.Preset = &webPresetComposition{Plan: ComposePlan{Launch: map[string][]testengine.PlanKnob{"all": {{Knob: "http.api=eth"}}}}}
	if err := e.pinPlanSurfaces(context.Background(), &p); err == nil || !strings.Contains(err.Error(), "http.api") {
		t.Fatalf("a binary without --http.api was accepted: %v", err)
	}
	e, p = webSurfaceEngine(t, webSurfaceBinary(t, c, "--datadir"))
	if err := e.pinPlanSurfaces(context.Background(), &p); err == nil {
		t.Fatal("a binary without an engine-derived mapping was accepted")
	}
}

func TestExecutionRefusesAChangedSurface(t *testing.T) {
	c := webSurfaceCatalog(t, "stablenet")
	binary := webSurfaceBinary(t, c, "")
	e, p := webSurfaceEngine(t, binary)
	if err := e.pinPlanSurfaces(context.Background(), &p); err != nil {
		t.Fatal(err)
	}
	// The binary now prints one flag fewer than when the plan was reviewed.
	changed := webSurfaceBinary(t, c, "--exitwhensynced")
	asset := e.assets["asset-1"]
	asset.Path = changed
	e.assets["asset-1"] = asset
	if err := e.recheckPlanSurfaces(context.Background(), p); !errors.Is(err, ErrDeploymentConflict) {
		t.Fatalf("a changed binary surface was not refused: %v", err)
	}
	// The documentation changed under the plan.
	asset.Path = binary
	e.assets["asset-1"] = asset
	docs := t.TempDir()
	for _, dir := range []string{"gstable", "gwbft", "gwemix"} {
		if err := os.CopyFS(filepath.Join(docs, dir), os.DirFS(filepath.Join(webSurfaceAnalysis, dir))); err != nil {
			t.Fatal(err)
		}
	}
	graph := filepath.Join(docs, "gstable", "rpc-metrics-graph.md")
	if err := os.WriteFile(graph, []byte("edited\n"), 0o600); err != nil {
		t.Fatal(err)
	}
	e.surfaces = docs
	if err := e.recheckPlanSurfaces(context.Background(), p); !errors.Is(err, ErrDeploymentConflict) {
		t.Fatalf("changed documentation was not refused: %v", err)
	}
}

// TestChainSurfaceJudgesTheSelectedNativeBinaries asks real chain binaries
// for their help pages. Each variable names a binary built at the commit its
// documentation was captured from, so each surface must be verified.
func TestChainSurfaceJudgesTheSelectedNativeBinaries(t *testing.T) {
	ran := 0
	for _, c := range []struct{ env, chain string }{{"GSTABLE_BIN", "stablenet"}, {"GWBFT_BIN", "wbft"}, {"GWEMIX_BIN", "wemix"}} {
		path := os.Getenv(c.env)
		if path == "" {
			continue
		}
		ran++
		raw, err := os.ReadFile(path)
		if err != nil {
			t.Fatal(err)
		}
		version, err := manifestProbe(context.Background(), path, "version")
		if err != nil {
			t.Fatal(err)
		}
		commit := regexp.MustCompile(`Git Commit: ([0-9a-f]{40})`).FindSubmatch(version)
		if commit == nil {
			t.Fatalf("%s reports no commit", c.env)
		}
		asset := ManifestBinary{ID: "native-" + c.chain, Chain: c.chain, Path: path, SHA256: manifestHash(raw), Commit: string(commit[1])}
		evidence, err := asset.Verify(context.Background(), c.chain)
		if err != nil {
			t.Fatalf("%s: %v", c.chain, err)
		}
		e := &WebChainEngine{root: t.TempDir(), surfaces: webSurfaceAnalysis, assets: map[string]ManifestBinary{asset.ID: asset}}
		s, err := e.ChainSurface(context.Background(), DeploymentActor{ID: "operator", Role: "operator"}, c.chain, asset.ID)
		if err != nil {
			t.Fatalf("%s: %v", c.chain, err)
		}
		if s.Verification != "verified" || s.Binary == nil || s.Binary.Checksum != evidence.SHA256 {
			t.Errorf("%s: %s %v", c.chain, s.Verification, s.Limitations)
		}
		for _, o := range s.Options {
			if o.OptionKey != "" && o.Scope == "global" && o.Support.Status != "supported" {
				t.Errorf("%s: mapped %s is %s: %s", c.chain, o.Spelling, o.Support.Status, o.Support.Reason)
			}
		}
		for _, cmd := range s.Commands {
			if len(cmd.EngineUsecases) > 0 && cmd.Support.Status != "supported" {
				t.Errorf("%s: engine command %v is %s: %s", c.chain, cmd.CommandPath, cmd.Support.Status, cmd.Support.Reason)
			}
		}
	}
	if ran == 0 {
		t.Skip("set GSTABLE_BIN, GWBFT_BIN or GWEMIX_BIN to a selected native binary")
	}
}

// A plan's fingerprint includes each pinned surface, and starting the plan
// prepares it again, so judging the same binary twice must give one answer.
func TestChainSurfaceFingerprintIsStableAcrossJudgements(t *testing.T) {
	for _, chain := range []string{"stablenet", "wbft", "wemix"} {
		c := webSurfaceCatalog(t, chain)
		first, err := JoinChainSurface(chain, c, nil)
		if err != nil {
			t.Fatal(err)
		}
		for i := 0; i < 20; i++ {
			again, err := JoinChainSurface(chain, c, nil)
			if err != nil {
				t.Fatal(err)
			}
			if again.Fingerprint != first.Fingerprint {
				t.Fatalf("%s: fingerprint changed between identical judgements", chain)
			}
		}
	}
	path := os.Getenv("GWBFT_BIN")
	if path == "" {
		t.Skip("GWBFT_BIN is not set; the native judgement was not repeated")
	}
	raw, err := os.ReadFile(path)
	if err != nil {
		t.Fatal(err)
	}
	version, err := manifestProbe(context.Background(), path, "version")
	if err != nil {
		t.Fatal(err)
	}
	commit := regexp.MustCompile(`Git Commit: ([0-9a-f]{40})`).FindSubmatch(version)
	asset := ManifestBinary{ID: "native-wbft", Chain: "wbft", Path: path, SHA256: manifestHash(raw), Commit: string(commit[1])}
	e := &WebChainEngine{root: t.TempDir(), surfaces: webSurfaceAnalysis, assets: map[string]ManifestBinary{asset.ID: asset}}
	a := DeploymentActor{ID: "operator", Role: "operator"}
	first, err := e.ChainSurface(context.Background(), a, "wbft", asset.ID)
	if err != nil {
		t.Fatal(err)
	}
	for i := 0; i < 3; i++ {
		again, err := e.ChainSurface(context.Background(), a, "wbft", asset.ID)
		if err != nil {
			t.Fatal(err)
		}
		if again.Fingerprint != first.Fingerprint {
			t.Fatal("native wbft fingerprint changed between identical judgements")
		}
	}
}
