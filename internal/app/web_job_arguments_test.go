package app

import (
	"context"
	"encoding/json"
	"testing"
)

// webArgumentSamples sets one argument the way a plan would.
var webArgumentSamples = map[string]func(*webChainArguments){
	"manifestId":                func(a *webChainArguments) { a.ManifestID = "stablenet" },
	"assetId":                   func(a *webChainArguments) { a.AssetID = "asset" },
	"serverRef":                 func(a *webChainArguments) { a.ServerRef = "ssh" },
	"allServers":                func(a *webChainArguments) { a.AllServers = true },
	"validators":                func(a *webChainArguments) { a.Validators = 7 },
	"chainPresetRef":            func(a *webChainArguments) { a.ChainPresetRef = &DeploymentDocumentRef{ID: "p", Revision: 2} },
	"replacementAssetId":        func(a *webChainArguments) { a.ReplacementAssetID = "next" },
	"configOverrides":           func(a *webChainArguments) { a.ConfigOverrides = map[string]string{"syncMode": "full"} },
	"configAssetId":             func(a *webChainArguments) { a.ConfigAssetID = "config" },
	"caseRefs":                  func(a *webChainArguments) { a.CaseRefs = []DeploymentDocumentRef{{ID: "c", Revision: 1}} },
	"runDefinitionRef":          func(a *webChainArguments) { a.RunDefinitionRef = &DeploymentDocumentRef{ID: "r", Revision: 1} },
	"binaryAssets":              func(a *webChainArguments) { a.BinaryAssets = map[string]string{"next": "next"} },
	"noSkips":                   func(a *webChainArguments) { a.NoSkips = true },
	"waitBlocks":                func(a *webChainArguments) { a.WaitBlocks = 12 },
	"nodeMonitorTimeoutSeconds": func(a *webChainArguments) { a.NodeMonitorTimeoutSeconds = 300 },
}

// webCommandLine is the chainbench command an operation runs for args, built
// by the same functions the jobs use. What the engine resolves from an
// argument (the staged binary of an asset, the file of a pinned preset) is
// stood in for by a fixed path.
func webCommandLine(t *testing.T, operation string, args webChainArguments) []string {
	t.Helper()
	p := webChainPayload{Arguments: args, ControlDir: "/control", ExecutionBinary: "/target/gstable"}
	switch operation {
	case "chain.setup", "chain.deploy":
		preset := ""
		if args.ChainPresetRef != nil {
			preset = "/control/chain-preset.json"
		}
		return chainUpArgs(p, "start", "/keys", "/control/server-set.yaml", "/control/workspace-config.yaml", []string{"--chain", "stablenet"}, preset)
	case "node.swap":
		binary, changes := "", []string(nil)
		if args.ReplacementAssetID != "" {
			binary = "/target/gstable-next"
		}
		for key, value := range args.ConfigOverrides {
			changes = append(changes, key+"="+value)
		}
		if args.ConfigAssetID != "" {
			changes = append(changes, "syncMode=full")
		}
		return nodeSwapArgs(1, binary, changes)
	case "test.run":
		p.TestRun = &webTestRun{Content: []json.RawMessage{json.RawMessage(`{}`)}, InputDir: "/inputs"}
		for name := range args.BinaryAssets {
			p.NamedBinaries = append(p.NamedBinaries, webNamedBinary{Name: name, ExecutionBinary: "/target/" + name})
		}
		in := (&WebChainEngine{root: "/root"}).webSuiteInput(p)
		argv, err := suiteCLIArgs(context.Background(), in, t.TempDir(), "", webRunFlags(args)...)
		if err != nil {
			t.Fatal(err)
		}
		return argv
	}
	return nil
}

// TestEveryContractFlagReachesTheCommandLine binds the jobs contract to the
// commands: an argument the contract says becomes a flag produces that flag,
// and leaving it out leaves the flag out.
func TestEveryContractFlagReachesTheCommandLine(t *testing.T) {
	checked := 0
	for _, op := range JobsContract().Operations {
		for _, a := range op.Arguments {
			if a.Flag == "" {
				continue
			}
			set, ok := webArgumentSamples[a.Name]
			if !ok {
				t.Fatalf("no sample for %s", a.Name)
			}
			var args webChainArguments
			set(&args)
			with := webCommandLine(t, op.ID, args)
			if with == nil {
				t.Fatalf("%s: no command line builder for an operation whose contract names flags", op.ID)
			}
			// Some flags also carry what the engine resolves itself (--binary is
			// the staged executable, --bp stands for an absent preset), so the
			// argument must add an occurrence rather than merely be present.
			without := webCommandLine(t, op.ID, webChainArguments{})
			if count(with, a.Flag) == 0 || count(with, a.Flag) <= count(without, a.Flag) && a.Flag != "--bp" && a.Flag != "--binary" {
				t.Errorf("%s %s: %v does not add %s to %v", op.ID, a.Name, with, a.Flag, without)
			}
			checked++
		}
	}
	if checked < 14 {
		t.Fatalf("only %d flags checked", checked)
	}
}

func count(args []string, flag string) int {
	n := 0
	for _, a := range args {
		if a == flag {
			n++
		}
	}
	return n
}

// A skipped case fails the job only when the job asked for no skips.
func TestNoSkipsFailsARunThatSkippedACase(t *testing.T) {
	var skipped RunSuiteOut
	skipped.Summary.Summary.Pass, skipped.Summary.Summary.Skip = 1, 1
	if err := webSuiteVerdict(skipped, false); err != nil {
		t.Errorf("a skip failed a job that allowed it: %v", err)
	}
	if err := webSuiteVerdict(skipped, true); err == nil {
		t.Error("a skip passed a job that asked for none")
	}
	var passed RunSuiteOut
	passed.Summary.Summary.Pass = 2
	if err := webSuiteVerdict(passed, true); err != nil {
		t.Errorf("a clean run failed: %v", err)
	}
}
