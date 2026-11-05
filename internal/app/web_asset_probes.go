package app

import (
	"context"
	"debug/elf"
	"debug/macho"
	"encoding/json"
	"errors"
	"fmt"
	"os"
	"path/filepath"
	"runtime"
	"strings"

	"github.com/0xmhha/chainbench/internal/core/session"
	"github.com/0xmhha/chainbench/internal/resource"
)

// webInspectionServerRequired marks a binary this machine cannot run: it stays
// unusable until a server of its platform has answered for it.
const webInspectionServerRequired = "server-required"

// webInspectionOnServer marks a binary whose identity came from a server.
const webInspectionOnServer = "server-version-and-help"

// webBinaryProbes is what a binary answered on a server of its own platform.
// Every later check of the binary reads these answers, because this machine
// cannot ask it again. The bytes are pinned by checksum, so the answers hold.
type webBinaryProbes struct {
	OS           string                    `json:"os"`
	Architecture string                    `json:"architecture"`
	Answers      map[string]webProbeAnswer `json:"answers"`
}

type webProbeAnswer struct {
	Output []byte `json:"output"`
	Failed bool   `json:"failed,omitempty"`
}

func webProbeKey(args []string) string { return strings.Join(args, "\x00") }

// answer replays one probe. A question never asked of the binary fails, the
// same as a flag it does not know.
func (p webBinaryProbes) answer(args []string) ([]byte, error) {
	got, ok := p.Answers[webProbeKey(args)]
	if !ok || got.Failed {
		return got.Output, errors.New("binary probe has no successful answer")
	}
	return got.Output, nil
}

// webProbeSession asks a binary on a server and keeps what it answered.
type webProbeSession struct {
	OS, Architecture string
	ask              func(ctx context.Context, args []string) (resource.BinaryProbeResult, error)
	answers          map[string]webProbeAnswer
}

func (s *webProbeSession) probe(ctx context.Context, args []string) ([]byte, error) {
	key := webProbeKey(args)
	if got, ok := s.answers[key]; ok {
		return got.Output, got.failure()
	}
	result, err := s.ask(ctx, args)
	if err != nil {
		return nil, err
	}
	answer := webProbeAnswer{Output: result.Output, Failed: result.ExitCode != 0}
	s.answers[key] = answer
	return answer.Output, answer.failure()
}

func (a webProbeAnswer) failure() error {
	if a.Failed {
		return errors.New("binary probe exited with an error")
	}
	return nil
}

type webProbeSessionKey struct{}

func withWebProbeSession(ctx context.Context, s *webProbeSession) context.Context {
	return context.WithValue(ctx, webProbeSessionKey{}, s)
}

func webProbeSessionOf(ctx context.Context) *webProbeSession {
	s, _ := ctx.Value(webProbeSessionKey{}).(*webProbeSession)
	return s
}

// readWebBinaryProbes reads the answers recorded beside an asset's bytes. An
// asset this machine runs itself has none.
func readWebBinaryProbes(path string) (webBinaryProbes, bool, error) {
	var out webBinaryProbes
	raw, err := os.ReadFile(filepath.Join(filepath.Dir(path), session.AssetProbesFile))
	if errors.Is(err, os.ErrNotExist) {
		return out, false, nil
	}
	if err != nil {
		return out, false, err
	}
	if err = json.Unmarshal(raw, &out); err != nil {
		return out, false, err
	}
	return out, true, nil
}

// webBinaryPlatform says where a binary runs: the server being asked, the
// server that answered for it, or this machine.
func webBinaryPlatform(ctx context.Context, path string) (string, string, error) {
	if s := webProbeSessionOf(ctx); s != nil {
		return s.OS, s.Architecture, nil
	}
	recorded, ok, err := readWebBinaryProbes(path)
	if err != nil {
		return "", "", err
	}
	if ok {
		return recorded.OS, recorded.Architecture, nil
	}
	return runtime.GOOS, runtime.GOARCH, nil
}

// webBinaryHeaderPlatform reads an executable's platform from its header,
// without running it.
func webBinaryHeaderPlatform(path string) (string, string, error) {
	if f, err := elf.Open(path); err == nil {
		defer func() { _ = f.Close() }()
		switch f.Machine {
		case elf.EM_AARCH64:
			return "linux", "arm64", nil
		case elf.EM_X86_64:
			return "linux", "amd64", nil
		}
		return "", "", errors.New("unsupported binary architecture")
	}
	if f, err := macho.Open(path); err == nil {
		defer func() { _ = f.Close() }()
		switch f.Cpu {
		case macho.CpuArm64:
			return "darwin", "arm64", nil
		case macho.CpuAmd64:
			return "darwin", "amd64", nil
		}
		return "", "", errors.New("unsupported binary architecture")
	}
	return "", "", errors.New("binary is neither ELF nor Mach-O")
}

// WebAssetVerifyIn names the server that answers for a registered binary.
type WebAssetVerifyIn struct {
	WorkspaceID string `json:"workspaceId"`
	ServerRef   string `json:"serverRef"`
}

// webAssetVerifier is the engine's side of WebJobs.VerifyAsset.
type webAssetVerifier interface {
	VerifyAssetOnServer(ctx context.Context, a DeploymentActor, id string, in WebAssetVerifyIn) (WebAsset, error)
}

// VerifyAsset asks a server of a binary's platform what the binary is.
func (s *WebJobs) VerifyAsset(ctx context.Context, a DeploymentActor, id string, in WebAssetVerifyIn) (WebAsset, error) {
	if err := s.allowed(a); err != nil {
		return WebAsset{}, err
	}
	verifier, ok := s.engine.(webAssetVerifier)
	if !ok {
		return WebAsset{}, errors.New("this server cannot verify binaries remotely")
	}
	return verifier.VerifyAssetOnServer(ctx, a, id, in)
}

// VerifyAssetOnServer ships a binary this machine cannot run to one server with
// the caller's own login, runs the same version and help checks there that a
// native binary gets here, and records the answers. It is a separate action
// because a plan only reads; this one writes one file to the chosen server.
func (e *WebChainEngine) VerifyAssetOnServer(ctx context.Context, a DeploymentActor, id string, in WebAssetVerifyIn) (WebAsset, error) {
	if err := e.allowed(a); err != nil {
		return WebAsset{}, err
	}
	receipt, path, err := e.manifests.assetReceipt(id)
	if err != nil {
		return WebAsset{}, err
	}
	if receipt.Kind != "binary" || receipt.Compatibility["inspection"] != webInspectionServerRequired {
		return WebAsset{}, webConflict("this asset needs no server verification")
	}
	p, server, err := e.assetProbePayload(a, in)
	if err != nil {
		return WebAsset{}, err
	}
	lookup, err := e.targetReader(ctx, a, p)
	if err != nil {
		return WebAsset{}, err
	}
	wc, err := deploymentWorkspace(p.Config.DeploymentDocumentInput)
	if err != nil {
		return WebAsset{}, err
	}
	spec := resource.TargetOf(server, wc.DataRoot)
	target, err := lookup.Inspect(ctx, spec)
	if err != nil {
		return WebAsset{}, err
	}
	wantOS, wantArch, err := webBinaryHeaderPlatform(path)
	if err != nil {
		return WebAsset{}, err
	}
	if target.OS != wantOS || target.Architecture != wantArch {
		return WebAsset{}, fmt.Errorf("binary runs on %s/%s but %s is %s/%s", wantOS, wantArch, in.ServerRef, target.OS, target.Architecture)
	}
	remotePath, err := wc.Resolve(resource.PurposeBinaries, receipt.Checksum+"/probe")
	if err != nil {
		return WebAsset{}, err
	}
	if err = e.uploadProbeBinary(ctx, a, p, path, receipt.Checksum, remotePath); err != nil {
		return WebAsset{}, err
	}
	probes := &webProbeSession{OS: target.OS, Architecture: target.Architecture, answers: map[string]webProbeAnswer{},
		ask: func(ctx context.Context, args []string) (resource.BinaryProbeResult, error) {
			return lookup.probe(ctx, resource.ProbeRequest{Target: spec, Path: remotePath, Args: args})
		}}
	probed := withWebProbeSession(ctx, probes)
	compatible, evidence, err := identifyWebBinary(probed, id, path, receipt.Checksum)
	if err != nil {
		return WebAsset{}, err
	}
	// The surface checks ask their own questions; replays need those too.
	catalog, err := LoadChainSurface(e.surfaces, evidence.Chain)
	if err != nil {
		return WebAsset{}, err
	}
	withPath := *compatible
	withPath.Path = path
	if _, err = e.judgeSurface(probed, catalog, withPath, evidence); err != nil {
		return WebAsset{}, err
	}
	return e.manifests.recordServerVerification(a, receipt, compatible, evidence, in.ServerRef, webBinaryProbes{OS: probes.OS, Architecture: probes.Architecture, Answers: probes.answers})
}

// assetProbePayload resolves the workspace's pinned documents and the caller's
// login for the one server they chose.
func (e *WebChainEngine) assetProbePayload(a DeploymentActor, in WebAssetVerifyIn) (webChainPayload, resource.Server, error) {
	var p webChainPayload
	var server resource.Server
	workspace, err := e.documents.Workspace(in.WorkspaceID)
	if err != nil {
		return p, server, err
	}
	p.Input.WorkspaceID = in.WorkspaceID
	p.Arguments.ServerRef = in.ServerRef
	for _, ref := range workspace.Documents {
		d, err := e.documents.DocumentRevision(ref.ID, ref.Revision)
		if err != nil {
			return p, server, err
		}
		switch d.Kind {
		case "server-set":
			p.Set = d
		case "workspace-config":
			p.Config = d
		}
	}
	set, err := deploymentSet(p.Set.DeploymentDocumentInput)
	if err != nil {
		return p, server, err
	}
	if server, err = set.ByName(in.ServerRef); err != nil {
		return p, server, err
	}
	binding := e.documents.Bindings(a, in.WorkspaceID)[in.ServerRef]
	if binding == "" {
		return p, server, &WebAccessError{Missing: []string{"ssh:" + in.ServerRef}}
	}
	p.Input.CredentialBindings = map[string]string{in.ServerRef: binding}
	return p, server, nil
}

// uploadProbeBinary ships the bytes through the CLI, which checks their sha256
// on the server. The server set and logins live only as long as the command.
func (e *WebChainEngine) uploadProbeBinary(ctx context.Context, a DeploymentActor, p webChainPayload, path, checksum, remotePath string) (err error) {
	if e.cli == nil {
		return errNoChainCLI
	}
	dir, err := os.MkdirTemp(e.root, "asset-probe-")
	if err != nil {
		return err
	}
	defer func() { err = errors.Join(err, os.RemoveAll(dir)) }()
	if err = writeWebDeploymentInputs(ctx, dir, p.Set, p.Config); err != nil {
		return err
	}
	logins, cleanup, err := e.jobLogins(ctx, a, p)
	if err != nil {
		return err
	}
	defer func() { err = errors.Join(err, cleanup()) }()
	args := []string{"file", "upload", path, "--remote", remotePath, "--force-upload", "--sha256", checksum, "--mode", "0755",
		"--server", p.Arguments.ServerRef, "--server-set", filepath.Join(dir, "server-set.yaml"),
		"--workspace-config", filepath.Join(dir, "workspace-config.yaml"), "--server-logins", logins}
	if webDocker(p.Set.DeploymentDocumentInput) {
		args = append(args, "--docker")
	}
	_, err = e.cli.Run(ctx, args...)
	return err
}

// recordServerVerification turns a waiting asset into a usable one.
func (s *ManifestStore) recordServerVerification(a DeploymentActor, receipt webAssetReceipt, compatible *ManifestBinary, evidence ManifestBinaryEvidence, server string, probes webBinaryProbes) (WebAsset, error) {
	stored := *compatible
	stored.Path = ""
	receipt.Binary = &stored
	receipt.Compatibility = map[string]any{"chain": evidence.Chain, "version": evidence.Version, "commit": stored.Commit, "helpDigest": evidence.HelpDigest,
		"os": evidence.OS, "architecture": evidence.Architecture, "inspection": webInspectionOnServer, "server": server}
	rawProbes, err := json.Marshal(probes)
	if err != nil {
		return WebAsset{}, err
	}
	raw, err := json.Marshal(receipt)
	if err != nil {
		return WebAsset{}, err
	}
	if err = s.AuditManifest(a.ID, "asset.verify", receipt.ID, 0); err != nil {
		return WebAsset{}, errors.New("asset audit storage unavailable")
	}
	if err = s.assets.Amend(receipt.ID, raw, rawProbes); err != nil {
		return WebAsset{}, errors.New("asset storage unavailable")
	}
	_ = s.AuditManifest(a.ID, "asset.verify", receipt.ID, 200)
	return receipt.WebAsset, nil
}
