package app

import (
	"bytes"
	"context"
	"encoding/json"
	"errors"
	"io"
	"os"
	"path/filepath"
	"reflect"
	"regexp"
	"strings"

	"github.com/0xmhha/chainbench/internal/core/filestore"
	"github.com/0xmhha/chainbench/internal/core/node"
	"github.com/0xmhha/chainbench/internal/core/registry"
	"github.com/0xmhha/chainbench/internal/resource"
)

// webNodeBinary pins a reviewed executable to one node and an owned target path.
// Multiple retained entries allow either the old or new recorded executable to
// remain verifiable when a stop or replacement launch fails.
type webNodeBinary struct {
	NodeID   string                 `json:"nodeId"`
	Evidence ManifestBinaryEvidence `json:"evidence"`
	Path     string                 `json:"path"`
}

type webBinaryReplacement struct {
	Evidence ManifestBinaryEvidence `json:"evidence"`
	Path     string                 `json:"path"`
}

func webNodeBinaryDeclarationsMatch(state State, ns node.Record) bool {
	return (state.BinaryChains[ns.Binary] == "" || state.BinaryChains[ns.Binary] == state.Chain) &&
		(state.GenesisPaths[ns.Binary] == "" || state.GenesisPaths[ns.Binary] == state.GenesisPath) &&
		state.GenesisConfigPaths[ns.Binary] == ""
}

func webNodeBinaryTarget(p webChainPayload, evidence ManifestBinaryEvidence) (string, error) {
	if evidence.ID == "" || evidence.Chain != p.Binary.Chain || evidence.OS != p.Target.OS || evidence.Architecture != p.Target.Architecture || !regexp.MustCompile(`^[0-9a-f]{64}$`).MatchString(evidence.SHA256) {
		return "", ErrDeploymentConflict
	}
	wc, err := deploymentWorkspace(p.Config.DeploymentDocumentInput)
	if err != nil {
		return "", err
	}
	plugin, err := registry.Get(evidence.Chain)
	if err != nil {
		return "", err
	}
	return wc.Resolve(resource.PurposeBinaries, evidence.SHA256+"/"+plugin.Manifest().Binary)
}

func readWebNodeBinaries(controlDir string) ([]webNodeBinary, string, error) {
	path := filepath.Join(controlDir, "web-node-binaries.json")
	info, err := os.Lstat(path)
	if os.IsNotExist(err) {
		return nil, manifestHash(nil), nil
	}
	if err != nil {
		return nil, "", err
	}
	if !info.Mode().IsRegular() {
		return nil, "", ErrDeploymentConflict
	}
	file, err := os.Open(path)
	if err != nil {
		return nil, "", err
	}
	defer func() { _ = file.Close() }()
	opened, err := file.Stat()
	if err != nil || !os.SameFile(info, opened) || !opened.Mode().IsRegular() {
		return nil, "", ErrDeploymentConflict
	}
	raw, err := io.ReadAll(io.LimitReader(file, (1<<20)+1))
	if err != nil || len(raw) > 1<<20 {
		return nil, "", ErrDeploymentConflict
	}
	var bindings []webNodeBinary
	dec := json.NewDecoder(bytes.NewReader(raw))
	dec.DisallowUnknownFields()
	if err = dec.Decode(&bindings); err != nil || dec.Decode(new(any)) != io.EOF {
		return nil, "", ErrDeploymentConflict
	}
	seen := map[string]bool{}
	for _, binding := range bindings {
		key := binding.NodeID + "\x00" + binding.Path
		if binding.NodeID == "" || binding.Path == "" || seen[key] {
			return nil, "", ErrDeploymentConflict
		}
		seen[key] = true
	}
	return bindings, manifestHash(raw), nil
}

func saveWebNodeBinary(ctx context.Context, controlDir, expectedDigest string, binding webNodeBinary) error {
	if err := ctx.Err(); err != nil {
		return err
	}
	bindings, digest, err := readWebNodeBinaries(controlDir)
	if err != nil {
		return err
	}
	if digest != expectedDigest {
		return ErrDeploymentConflict
	}
	for _, prior := range bindings {
		if prior.NodeID == binding.NodeID && prior.Path == binding.Path {
			if !reflect.DeepEqual(prior, binding) {
				return ErrDeploymentConflict
			}
			return nil
		}
	}
	bindings = append(bindings, binding)
	raw, err := json.Marshal(bindings)
	if err != nil {
		return err
	}
	if len(raw) > 1<<20 {
		return errors.New("reviewed executable history exceeds its storage limit")
	}
	return (filestore.Local{}).Write(ctx, filepath.Join(controlDir, "web-node-binaries.json"), raw, 0600)
}

// currentWebNodeBinary validates durable provenance separately from the base
// executable. Native source evidence and deployed bytes must both still match.
func (e *WebChainEngine) currentWebNodeBinary(ctx context.Context, state State, p webChainPayload, lookup resource.Lookup) (*webNodeBinary, string, error) {
	bindings, digest, err := readWebNodeBinaries(p.ControlDir)
	if err != nil {
		return nil, "", err
	}
	if err = webSelectedNode(state, p.Input.NodeIDs); err != nil {
		return nil, "", err
	}
	for _, ns := range state.Nodes {
		if string(ns.NodeLabel()) != p.Input.NodeIDs[0] {
			continue
		}
		if ns.Binary == "" {
			return nil, digest, nil
		}
		if !webNodeBinaryDeclarationsMatch(state, ns) {
			return nil, "", ErrDeploymentConflict
		}
		for _, binding := range bindings {
			if binding.NodeID != string(ns.NodeLabel()) || binding.Path != state.Binaries[ns.Binary] {
				continue
			}
			path, err := webNodeBinaryTarget(p, binding.Evidence)
			if err != nil || path != binding.Path {
				return nil, "", ErrDeploymentConflict
			}
			if err = inspectWebBinaryParent(ctx, p, path, lookup); err != nil {
				return nil, "", err
			}
			asset, err := e.binaryAsset(binding.Evidence.ID)
			if err != nil {
				return nil, "", ErrDeploymentConflict
			}
			observed, err := asset.Verify(ctx, p.Binary.Chain)
			if err != nil || !reflect.DeepEqual(observed, binding.Evidence) {
				return nil, "", ErrDeploymentConflict
			}
			access, err := (resource.Opener{Lookup: lookup}).Open(webNodeTarget(state, ns))
			if err != nil {
				return nil, "", err
			}
			if err = access.VerifyRegularFile(ctx, path); err != nil {
				return nil, "", ErrDeploymentConflict
			}
			checksum, err := access.Files.Checksum(ctx, path)
			if err != nil || checksum != "sha256:"+binding.Evidence.SHA256 {
				return nil, "", ErrDeploymentConflict
			}
			copy := binding
			return &copy, digest, nil
		}
		return nil, "", ErrDeploymentConflict
	}
	return nil, "", ErrDeploymentNotFound
}

func (e *WebChainEngine) prepareWebBinaryReplacement(ctx context.Context, state State, p *webChainPayload) error {
	if p.Arguments.ReplacementAssetID == "" {
		return nil
	}
	if p.Input.Operation != "node.swap" {
		return errors.New("a replacement executable requires a node replacement job")
	}
	asset, err := e.binaryAsset(p.Arguments.ReplacementAssetID)
	if err != nil {
		return err
	}
	evidence, err := asset.Verify(ctx, p.Binary.Chain)
	if err != nil {
		return err
	}
	path, err := webNodeBinaryTarget(*p, evidence)
	if err != nil {
		return err
	}
	current := p.Binary.SHA256
	if p.CurrentNodeBinary != nil {
		current = p.CurrentNodeBinary.Evidence.SHA256
	}
	if evidence.SHA256 == current {
		return errors.New("select a different verified executable")
	}
	help, err := manifestProbe(ctx, asset.Path, "--help")
	if err != nil {
		return err
	}
	for _, ns := range state.Nodes {
		if string(ns.NodeLabel()) != p.Input.NodeIDs[0] {
			continue
		}
		if err = verifyWebReplacementArguments(ns.Args, string(help)); err != nil {
			return err
		}
	}
	p.Replacement = &webBinaryReplacement{Evidence: evidence, Path: path}
	return nil
}

func verifyWebReplacementArguments(args []string, help string) error {
	supported := map[string]bool{}
	for _, match := range regexp.MustCompile(`(?m)^\s*(--[A-Za-z0-9][A-Za-z0-9.-]*)(?:[, \t]|$)`).FindAllStringSubmatch(help, -1) {
		supported[match[1]] = true
	}
	for _, arg := range args {
		if !strings.HasPrefix(arg, "--") {
			continue
		}
		flag, _, _ := strings.Cut(arg, "=")
		if !supported[flag] {
			return errors.New("replacement executable lacks a recorded launch option")
		}
	}
	return nil
}

func inspectWebBinaryParent(ctx context.Context, p webChainPayload, path string, lookup resource.Lookup) error {
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
	relative, err := filepath.Rel(wc.DataRoot, filepath.Dir(path))
	if err != nil || relative == ".." || strings.HasPrefix(relative, ".."+string(filepath.Separator)) {
		return ErrDeploymentConflict
	}
	spec := resource.TargetOf(server, filepath.Dir(path))
	actual, err := (resource.Opener{Lookup: lookup}).Inspect(ctx, spec)
	if err != nil {
		return err
	}
	if actual.HostIdentity != p.Target.HostIdentity || actual.DataPath != filepath.Join(p.Target.DataPath, relative) {
		return ErrDeploymentConflict
	}
	return nil
}

// stageWebBinaryReplacement never replaces a different file already occupying a
// content-addressed destination. The retained binding precedes stopping so both
// successful and failed replacement records can resolve their reviewed bytes.
func (e *WebChainEngine) stageWebBinaryReplacement(ctx context.Context, state State, p webChainPayload, lookup resource.Lookup) error {
	if p.Replacement == nil {
		return nil
	}
	if err := ctx.Err(); err != nil {
		return err
	}
	asset, err := e.binaryAsset(p.Replacement.Evidence.ID)
	if err != nil {
		return err
	}
	observed, err := asset.Verify(ctx, p.Binary.Chain)
	if err != nil || !reflect.DeepEqual(observed, p.Replacement.Evidence) {
		return ErrDeploymentConflict
	}
	expected, err := webNodeBinaryTarget(p, observed)
	if err != nil || expected != p.Replacement.Path {
		return ErrDeploymentConflict
	}
	if err = inspectWebBinaryParent(ctx, p, expected, lookup); err != nil {
		return err
	}
	access, err := (resource.Opener{Lookup: lookup}).Open(state.Target)
	if err != nil {
		return err
	}
	exists, err := access.Files.Exists(ctx, expected)
	if err != nil {
		return err
	}
	if !exists {
		raw, err := os.ReadFile(asset.Path)
		if err != nil || manifestHash(raw) != observed.SHA256 {
			return ErrDeploymentConflict
		}
		if err = access.Files.Write(ctx, expected, raw, 0755); err != nil {
			return err
		}
	}
	if err = access.VerifyRegularFile(ctx, expected); err != nil {
		return ErrDeploymentConflict
	}
	checksum, err := access.Files.Checksum(ctx, expected)
	if err != nil || checksum != "sha256:"+observed.SHA256 {
		return ErrDeploymentConflict
	}
	if err = inspectWebBinaryParent(ctx, p, expected, lookup); err != nil {
		return err
	}
	if err = ctx.Err(); err != nil {
		return err
	}
	return saveWebNodeBinary(ctx, p.ControlDir, p.NodeBindingsDigest, webNodeBinary{NodeID: p.Input.NodeIDs[0], Evidence: observed, Path: expected})
}
