package app

import (
	"context"
	"errors"
	"os"
	"path/filepath"
	"reflect"
	"strings"
	"testing"

	"github.com/0xmhha/chainbench/internal/core/node"
	"github.com/0xmhha/chainbench/internal/resource"
)

func TestReviewedNodeBinaryHistoryPreservesFailedReplacementChoices(t *testing.T) {
	dir := t.TempDir()
	_, empty, err := readWebNodeBinaries(dir)
	if err != nil {
		t.Fatal(err)
	}
	old := webNodeBinary{NodeID: "node1", Path: "/owned/old", Evidence: ManifestBinaryEvidence{ID: "old", SHA256: strings.Repeat("a", 64)}}
	next := webNodeBinary{NodeID: "node1", Path: "/owned/new", Evidence: ManifestBinaryEvidence{ID: "new", SHA256: strings.Repeat("b", 64)}}
	if err = saveWebNodeBinary(context.Background(), dir, empty, old); err != nil {
		t.Fatal(err)
	}
	_, first, err := readWebNodeBinaries(dir)
	if err != nil {
		t.Fatal(err)
	}
	if err = saveWebNodeBinary(context.Background(), dir, first, next); err != nil {
		t.Fatal(err)
	}
	entries, current, err := readWebNodeBinaries(dir)
	if err != nil || !reflect.DeepEqual(entries, []webNodeBinary{old, next}) {
		t.Fatal("replacement removed prior binding", entries, err)
	}
	if err = saveWebNodeBinary(context.Background(), dir, first, old); !errors.Is(err, ErrDeploymentConflict) {
		t.Fatal("stale review modified binding history", err)
	}
	if err = saveWebNodeBinary(context.Background(), dir, current, next); err != nil {
		t.Fatal("identical retained binding cannot be reused", err)
	}
	changed := next
	changed.Evidence.SHA256 = strings.Repeat("c", 64)
	if err = saveWebNodeBinary(context.Background(), dir, current, changed); !errors.Is(err, ErrDeploymentConflict) {
		t.Fatal("conflicting path evidence replaced history", err)
	}
	after, _, err := readWebNodeBinaries(dir)
	if err != nil || !reflect.DeepEqual(after, entries) {
		t.Fatal("refusal changed binding history", err)
	}
	info, err := os.Stat(filepath.Join(dir, "web-node-binaries.json"))
	if err != nil || info.Mode().Perm() != 0600 {
		t.Fatal("bindings are not private", err)
	}
}

func TestReviewedNodeBinaryTargetRefusesWrongProtocolPlatformOrDigest(t *testing.T) {
	p := webChainPayload{Binary: ManifestBinaryEvidence{Chain: "wbft"}, Target: resource.Inspection{OS: "darwin", Architecture: "arm64"}, Config: DeploymentDocument{DeploymentDocumentInput: deploymentTestConfig()}}
	valid := ManifestBinaryEvidence{ID: "registered", Chain: "wbft", OS: "darwin", Architecture: "arm64", SHA256: strings.Repeat("a", 64)}
	path, err := webNodeBinaryTarget(p, valid)
	if err != nil || !strings.HasSuffix(path, "/"+valid.SHA256+"/gwbft") {
		t.Fatal("replacement target is not content addressed", path, err)
	}
	for _, field := range []string{"id", "chain", "os", "architecture", "digest"} {
		candidate := valid
		switch field {
		case "id":
			candidate.ID = ""
		case "chain":
			candidate.Chain = "wemix"
		case "os":
			candidate.OS = "linux"
		case "architecture":
			candidate.Architecture = "amd64"
		case "digest":
			candidate.SHA256 = "../outside"
		}
		if _, err = webNodeBinaryTarget(p, candidate); !errors.Is(err, ErrDeploymentConflict) {
			t.Fatal("unreviewed target accepted", field, err)
		}
	}
}

func TestReviewedNodeBinaryHistoryRejectsMalformedOrDuplicateEntries(t *testing.T) {
	for _, raw := range []string{`{"unexpected":true}`, `[{"nodeId":"node1","path":"/owned","evidence":{}},{"nodeId":"node1","path":"/owned","evidence":{}}]`, `[] trailing`, strings.Repeat(" ", (1<<20)+1)} {
		dir := t.TempDir()
		if err := os.WriteFile(filepath.Join(dir, "web-node-binaries.json"), []byte(raw), 0600); err != nil {
			t.Fatal(err)
		}
		if _, _, err := readWebNodeBinaries(dir); !errors.Is(err, ErrDeploymentConflict) {
			t.Fatal("untrusted history accepted", err)
		}
	}
}

func TestReviewedNodeBinaryHistoryRefusesSymlink(t *testing.T) {
	dir := t.TempDir()
	outside := filepath.Join(t.TempDir(), "history.json")
	if err := os.WriteFile(outside, []byte("[]"), 0600); err != nil {
		t.Fatal(err)
	}
	if err := os.Symlink(outside, filepath.Join(dir, "web-node-binaries.json")); err != nil {
		t.Fatal(err)
	}
	if _, _, err := readWebNodeBinaries(dir); !errors.Is(err, ErrDeploymentConflict) {
		t.Fatal("symbolic history authorized a reviewed executable", err)
	}
}

func TestReplacementRequiresEveryRecordedLaunchOption(t *testing.T) {
	help := "OPTIONS:\n   --config value\n   --http, -h\n   --http.addr value\n   --metrics.addr value\n"
	args := []string{"--config", "/owned/config.toml", "--http", "--http.addr=127.0.0.1", "--metrics.addr", "127.0.0.1"}
	if err := verifyWebReplacementArguments(args, help); err != nil {
		t.Fatal(err)
	}
	if err := verifyWebReplacementArguments(append(args, "--missing.option"), help); err == nil {
		t.Fatal("replacement lacking a recorded option was accepted")
	}
}

func TestBinaryOnlyReplacementPreservesGeneratedConfiguration(t *testing.T) {
	state := State{Nodes: []node.Record{{Index: 1, Label: "node1", Role: "en"}}}
	p := webChainPayload{Input: WebPlanInput{NodeIDs: []string{"node1"}}, Replacement: &webBinaryReplacement{Path: "/reviewed/new"}}
	changes, err := webConfigChanges(state, p)
	if err != nil || len(changes) != 0 {
		t.Fatal("binary-only replacement requires or changes generated config", changes, err)
	}
	state.Nodes[0].Binary = "node1"
	p.CurrentNodeBinary = &webNodeBinary{NodeID: "node1"}
	if _, err = webConfigChanges(state, p); err != nil {
		t.Fatal("reviewed per-node executable prevents later replacement", err)
	}
	p.Replacement = nil
	if _, err = webConfigChanges(state, p); err == nil {
		t.Fatal("empty replacement job accepted")
	}
}
