package app

import (
	"bytes"
	"context"
	"debug/elf"
	"debug/macho"
	"errors"
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"runtime"
	"strings"
	"time"

	"github.com/0xmhha/chainbench/internal/core/nodeconfig"
	"github.com/0xmhha/chainbench/internal/core/registry"
)

// ManifestBinary is a server-provisioned asset with verified source identity.
// It is deliberately not accepted from a browser request.
type ManifestBinary struct {
	ID     string `json:"id"`
	Chain  string `json:"chain"`
	Path   string `json:"path"`
	SHA256 string `json:"sha256"`
	Commit string `json:"commit"`
}

type ManifestBinaryEvidence struct {
	ID           string `json:"id"`
	Chain        string `json:"chain"`
	SHA256       string `json:"sha256"`
	Version      string `json:"version"`
	HelpDigest   string `json:"helpDigest"`
	OS           string `json:"os"`
	Architecture string `json:"architecture"`
}

func (a ManifestBinary) Verify(ctx context.Context, chain string) (ManifestBinaryEvidence, error) {
	evidence := ManifestBinaryEvidence{ID: a.ID, Chain: a.Chain, OS: runtime.GOOS, Architecture: runtime.GOARCH}
	if a.Chain != chain || a.ID == "" || len(a.SHA256) != 64 || len(a.Commit) != 40 {
		return evidence, errors.New("binary identity is incompatible with selected chain")
	}
	raw, err := os.ReadFile(a.Path)
	if err != nil {
		return evidence, err
	}
	evidence.SHA256 = manifestHash(raw)
	if evidence.SHA256 != a.SHA256 {
		return evidence, errors.New("binary checksum changed; revalidate asset")
	}
	if err := verifyManifestNative(a.Path); err != nil {
		return evidence, err
	}
	probe, cancel := context.WithTimeout(ctx, 10*time.Second)
	defer cancel()
	version, err := manifestProbe(probe, a.Path, "version")
	if err != nil {
		return evidence, errors.New("binary version probe failed")
	}
	if !strings.Contains(string(version), "Git Commit: "+a.Commit) || !strings.Contains(string(version), "Operating System: "+runtime.GOOS) || !strings.Contains(string(version), "Architecture: "+runtime.GOARCH) {
		return evidence, errors.New("binary version identity mismatch")
	}
	evidence.Version = string(version)
	help, err := manifestProbe(probe, a.Path, "--help")
	if err != nil {
		return evidence, errors.New("binary help probe failed")
	}
	plugin, err := registry.Get(chain)
	if err != nil {
		return evidence, err
	}
	manifest := plugin.Manifest()
	if !strings.EqualFold(strings.TrimSpace(strings.Split(string(version), "\n")[0]), manifest.Build.MakeTarget) {
		return evidence, errors.New("binary program identity does not match chain build contract")
	}
	dialect, err := nodeconfig.DialectFor(manifest.Dialect)
	if err != nil {
		return evidence, err
	}
	// Distinguish the two gwemix builds by observed vocabulary, never filename.
	legacy := nodeconfig.Geth110Wemix()
	interval, _ := legacy.Spelling(nodeconfig.KeyBlockInterval)
	_, needsInterval := dialect.Spelling(nodeconfig.KeyBlockInterval)
	if strings.Contains(string(help), interval) != needsInterval {
		return evidence, errors.New("binary help dialect is incompatible with selected chain")
	}
	if personal, required := dialect.Spelling(nodeconfig.KeyRPCDeprecatedPersonal); required && !strings.Contains(string(help), personal) {
		return evidence, errors.New("binary help lacks required dialect mapping")
	}
	evidence.HelpDigest = manifestHash(help)
	return evidence, nil
}

type ManifestSetup struct {
	ID         string                 `json:"id"`
	ManifestID string                 `json:"manifestId"`
	Binary     ManifestBinaryEvidence `json:"binary"`
	State      State                  `json:"state"`
}

// SetupManifest prepares an isolated local network using the existing engine steps.
// It does not launch a process; the returned state distinguishes init from start.
func (s *ManifestStore) SetupManifest(ctx context.Context, actor DeploymentActor, id string, asset ManifestBinary, keys string) (ManifestSetup, error) {
	out := ManifestSetup{ManifestID: id}
	if actor.ID == "" || (actor.Role != "admin" && actor.Role != "operator") {
		return out, ErrDeploymentForbidden
	}
	item, err := s.Get(id)
	if err != nil {
		return out, err
	}
	plugin, err := ValidateManifest(item.ManifestInput)
	if err != nil {
		return out, err
	}
	out.Binary, err = asset.Verify(ctx, plugin.Protocol().Name)
	if err != nil {
		return out, err
	}
	dir, err := s.files.NewSetupDir()
	if err != nil {
		return out, err
	}
	out.ID = filepath.Base(dir)
	deps := Deps{Command: "web manifest setup by " + actor.ID}
	// Detached work survives client disconnection. No raw argv enters this path.
	ctx = context.WithoutCancel(ctx)
	if _, err = s.ApplyManifest(ctx, deps, actor, id, dir, keys, asset.Path); err != nil {
		return out, err
	}
	steps := []func() error{
		func() error { _, e := ChainAllocate(ctx, deps, ChainAllocateIn{DataDir: dir, BPCount: 4}); return e },
		func() error { _, e := ChainKeys(ctx, deps, ChainKeysIn{DataDir: dir, Nodes: 4}); return e },
		func() error { _, e := ChainGenesis(ctx, deps, ChainGenesisIn{DataDir: dir}); return e },
		func() error { _, e := ChainConfig(ctx, deps, ChainConfigIn{DataDir: dir}); return e },
		func() error { _, e := ChainLaunchOpts(ctx, deps, ChainLaunchOptsIn{DataDir: dir}); return e },
		func() error { _, e := ChainProvision(ctx, deps, ChainProvisionIn{DataDir: dir}); return e },
		func() error { _, e := ChainInit(ctx, deps, ChainInitIn{DataDir: dir}); return e },
	}
	for i, step := range steps {
		if err = step(); err != nil {
			return out, fmt.Errorf("manifest setup %s phase %d: %w", out.ID, i, err)
		}
	}
	state, err := ChainStatus(ctx, deps, ChainStatusIn{DataDir: dir})
	out.State = state.State
	return out, err
}

func verifyManifestNative(path string) error {
	switch runtime.GOOS {
	case "darwin":
		f, err := macho.Open(path)
		if err != nil {
			return errors.New("binary is not a native Mach-O asset")
		}
		defer func() { _ = f.Close() }()
		if (runtime.GOARCH == "arm64" && f.Cpu != macho.CpuArm64) || (runtime.GOARCH == "amd64" && f.Cpu != macho.CpuAmd64) {
			return errors.New("binary architecture mismatch")
		}
	case "linux":
		f, err := elf.Open(path)
		if err != nil {
			return errors.New("binary is not a native ELF asset")
		}
		defer func() { _ = f.Close() }()
		if (runtime.GOARCH == "arm64" && f.Machine != elf.EM_AARCH64) || (runtime.GOARCH == "amd64" && f.Machine != elf.EM_X86_64) {
			return errors.New("binary architecture mismatch")
		}
	default:
		return errors.New("unsupported binary platform")
	}
	return nil
}

// Bounded output prevents an uploaded program from filling control-plane memory.
type manifestProbeOutput struct{ buffer bytes.Buffer }

func (b *manifestProbeOutput) Write(p []byte) (int, error) {
	if b.buffer.Len()+len(p) > 1<<20 {
		return 0, errors.New("native inspection output limit exceeded")
	}
	return b.buffer.Write(p)
}

func (b *manifestProbeOutput) Bytes() []byte { return b.buffer.Bytes() }
func manifestProbe(ctx context.Context, path string, args ...string) ([]byte, error) {
	var output manifestProbeOutput
	cmd := exec.CommandContext(ctx, path, args...)
	cmd.Stdout, cmd.Stderr = &output, &output
	err := cmd.Run()
	return output.Bytes(), err
}
