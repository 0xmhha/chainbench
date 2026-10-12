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
	// Detached work survives client disconnection. No raw argv enters this path.
	ctx = context.WithoutCancel(ctx)
	manifest, template := "", ""
	if item.Source == "external" {
		// These files belong to this composition, never a client-supplied path.
		if manifest, template, err = s.files.Pin(dir, item.Manifest, []byte(item.Template)); err != nil {
			return out, err
		}
	}
	if s.cli == nil {
		return out, errNoChainCLI
	}
	if _, err = s.cli.Run(ctx, manifestSetupArgs(dir, keys, asset.Path, plugin.Manifest().ID, manifest, template)...); err != nil {
		return out, fmt.Errorf("manifest setup %s: %w", out.ID, err)
	}
	out.State, err = s.cli.ChainStatus(ctx, dir)
	return out, err
}

// manifestSetupArgs is the `chain up` a manifest setup runs: four producers of
// the selected chain from the verified asset and the reviewed key preset,
// initialised and left stopped.
func manifestSetupArgs(dir, keys, binary, chain, manifest, template string) []string {
	args := []string{"chain", "up", "--workspace-dir", dir, "--stage", "init", "--bp", "4", "--keys", keys, "--binary", binary}
	if manifest != "" {
		return append(args, "--manifest", manifest, "--genesis-template", template)
	}
	return append(args, "--chain", chain)
}

// UseCLI gives the store the chainbench command its setups run.
func (s *ManifestStore) UseCLI(c ChainCLI) { s.cli = c }

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

// manifestProbe runs one inspection of a native binary in an empty working
// and home directory with a minimal environment. The server's own environment
// must not reach the binary: go-wbft prints a warning before its program name
// for every unknown GWEMIX_* variable, which made the identity line depend on
// how the server was started, and an inherited HOME would let a binary that
// ignored --help write into the operator's default data directory. Help
// prints defaults under HOME, so the fresh directory is written back as $HOME:
// the same binary must give the same output, or a plan that pins it could
// never start.
func manifestProbe(ctx context.Context, path string, args ...string) ([]byte, error) {
	home, err := os.MkdirTemp("", "chainbench-probe-")
	if err != nil {
		return nil, err
	}
	defer func() { _ = os.RemoveAll(home) }()
	var output manifestProbeOutput
	cmd := exec.CommandContext(ctx, path, args...)
	cmd.Dir, cmd.Env = home, []string{"HOME=" + home, "PATH=/usr/bin:/bin"}
	cmd.Stdout, cmd.Stderr = &output, &output
	err = cmd.Run()
	out := output.Bytes()
	// The resolved path (/private/var/... on macOS) contains the short one, so
	// it is replaced first.
	names := []string{home}
	if real, rerr := filepath.EvalSymlinks(home); rerr == nil && real != home {
		names = []string{real, home}
	}
	for _, name := range names {
		out = bytes.ReplaceAll(out, []byte(name), []byte("$HOME"))
	}
	return out, err
}
