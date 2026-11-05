package app

import (
	"bytes"
	"context"
	"debug/elf"
	"encoding/binary"
	"fmt"
	"runtime"
	"strings"
	"testing"

	"github.com/0xmhha/chainbench/internal/core/registry"
	"github.com/0xmhha/chainbench/internal/resource"
)

// foreignELF is a minimal executable header for a platform this host is not.
func foreignELF(t *testing.T) ([]byte, string) {
	t.Helper()
	machine, arch := elf.EM_AARCH64, "arm64"
	if runtime.GOOS == "linux" && runtime.GOARCH == "arm64" {
		machine, arch = elf.EM_X86_64, "amd64"
	}
	header := elf.Header64{Ident: [elf.EI_NIDENT]byte{0x7f, 'E', 'L', 'F', 2, 1, 1}, Type: uint16(elf.ET_EXEC), Machine: uint16(machine), Version: 1, Ehsize: 64, Phentsize: 56, Shentsize: 64}
	var raw bytes.Buffer
	if err := binary.Write(&raw, binary.LittleEndian, header); err != nil {
		t.Fatal(err)
	}
	return raw.Bytes(), arch
}

func stablenetProbeAnswers(t *testing.T, arch string) func(context.Context, []string) (resource.BinaryProbeResult, error) {
	t.Helper()
	plugin, err := registry.Get("stablenet")
	if err != nil {
		t.Fatal(err)
	}
	help := webHelpOf(webSurfaceCatalog(t, "stablenet").Global, "")
	version := fmt.Sprintf("%s\nGit Commit: %s\nOperating System: linux\nArchitecture: %s\n", plugin.Manifest().Build.MakeTarget, strings.Repeat("a", 40), arch)
	return func(_ context.Context, args []string) (resource.BinaryProbeResult, error) {
		switch strings.Join(args, " ") {
		case "version":
			return resource.BinaryProbeResult{Output: []byte(version)}, nil
		case "--help":
			return resource.BinaryProbeResult{Output: help}, nil
		}
		return resource.BinaryProbeResult{ExitCode: 2}, nil
	}
}

// A binary for the servers has to be registrable from a laptop of another
// platform, and stay unusable until a server has answered for it.
func TestForeignBinaryWaitsForAServer(t *testing.T) {
	store, err := OpenManifestStore(t.TempDir())
	if err != nil {
		t.Fatal(err)
	}
	raw, arch := foreignELF(t)
	actor := DeploymentActor{ID: "operator", Role: "operator"}
	uploaded, err := store.UploadAsset(context.Background(), actor, "binary", "gstable", bytes.NewReader(raw))
	if err != nil {
		t.Fatalf("foreign binary refused: %v", err)
	}
	if uploaded.Compatibility["inspection"] != webInspectionServerRequired || uploaded.Compatibility["architecture"] != arch {
		t.Fatalf("compatibility = %+v", uploaded.Compatibility)
	}
	if _, err = store.BinaryAsset(uploaded.ID); err == nil {
		t.Fatal("an unverified binary is usable")
	}

	receipt, path, err := store.assetReceipt(uploaded.ID)
	if err != nil {
		t.Fatal(err)
	}
	session := &webProbeSession{OS: "linux", Architecture: arch, answers: map[string]webProbeAnswer{}, ask: stablenetProbeAnswers(t, arch)}
	compatible, evidence, err := identifyWebBinary(withWebProbeSession(context.Background(), session), uploaded.ID, path, receipt.Checksum)
	if err != nil {
		t.Fatalf("identify on the server: %v", err)
	}
	if evidence.Chain != "stablenet" || evidence.OS != "linux" || evidence.Architecture != arch {
		t.Fatalf("evidence = %+v", evidence)
	}
	verified, err := store.recordServerVerification(actor, receipt, compatible, evidence, "server5", webBinaryProbes{OS: session.OS, Architecture: session.Architecture, Answers: session.answers})
	if err != nil {
		t.Fatal(err)
	}
	if verified.Compatibility["inspection"] != webInspectionOnServer || verified.Compatibility["server"] != "server5" {
		t.Fatalf("verified = %+v", verified.Compatibility)
	}

	// A later plan cannot ask the binary again; the recorded answers must do.
	asset, err := store.BinaryAsset(uploaded.ID)
	if err != nil {
		t.Fatalf("verified binary unusable: %v", err)
	}
	replayed, err := asset.Verify(context.Background(), "stablenet")
	if err != nil {
		t.Fatalf("replay: %v", err)
	}
	if replayed.HelpDigest != evidence.HelpDigest || replayed.OS != "linux" {
		t.Fatalf("replayed = %+v, want %+v", replayed, evidence)
	}
}

func TestRecordedProbesFailAQuestionNeverAsked(t *testing.T) {
	probes := webBinaryProbes{Answers: map[string]webProbeAnswer{webProbeKey([]string{"version"}): {Output: []byte("v")}, webProbeKey([]string{"bad"}): {Output: []byte("x"), Failed: true}}}
	if out, err := probes.answer([]string{"version"}); err != nil || string(out) != "v" {
		t.Fatalf("answer = %q, %v", out, err)
	}
	if _, err := probes.answer([]string{"--help"}); err == nil {
		t.Fatal("an unasked question was answered")
	}
	if _, err := probes.answer([]string{"bad"}); err == nil {
		t.Fatal("a failed probe read as success")
	}
}

func TestUploadRefusesAFileThatIsNoExecutable(t *testing.T) {
	store, err := OpenManifestStore(t.TempDir())
	if err != nil {
		t.Fatal(err)
	}
	if _, err = store.UploadAsset(context.Background(), DeploymentActor{ID: "operator", Role: "operator"}, "binary", "notes", strings.NewReader("not an executable")); err == nil {
		t.Fatal("a text file registered as a binary")
	}
}
