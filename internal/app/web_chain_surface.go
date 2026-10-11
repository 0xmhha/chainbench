package app

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"path/filepath"
	"slices"
	"sort"
	"strings"
	"time"

	"github.com/0xmhha/chainbench/internal/core/chainsurface"
	"github.com/0xmhha/chainbench/internal/core/filestore"
	"github.com/0xmhha/chainbench/internal/core/nodeconfig"
	"github.com/0xmhha/chainbench/internal/core/registry"
)

// webSurfaceDirs names the analysis directory of each engine chain, as the
// specification's chain table does. wbft and wemix both build a binary named
// gwemix, so the chain id, never the basename, selects the directory.
var webSurfaceDirs = map[string]string{"stablenet": "gstable", "wbft": "gwbft", "wemix": "gwemix"}

// webEngineCommand is a chain binary command an existing use case runs, and
// the flags the engine itself passes to it. A command's presence in the
// documentation grants nothing; only these are linked to job phases.
type webEngineCommand struct {
	path     []string
	families []string
	usecases []string
	flags    []string
}

var webEngineCommands = []webEngineCommand{
	{path: []string{}, usecases: []string{"chain.deploy", "test.run", "node.start", "node.restart", "node.swap"}},
	{path: []string{"init"}, usecases: []string{"chain.setup", "chain.deploy", "test.run", "node.reset"}, flags: []string{"--datadir"}},
	{path: []string{"wemix", "genesis"}, families: []string{"poa"}, usecases: []string{"chain.setup", "chain.deploy", "test.run"}, flags: []string{"--data", "--genesis", "--out"}},
	{path: []string{"wemix", "deploy-governance"}, families: []string{"poa"}, usecases: []string{"chain.deploy", "test.run"}, flags: []string{"--url", "--password"}},
	{path: []string{"attach"}, families: []string{"poa"}, usecases: []string{"chain.deploy", "test.run"}, flags: []string{"--exec"}},
}

// webPrivateKeys are per-node launch inputs that carry account or node
// secrets; the engine fills them from the operator's bound key material.
var webPrivateKeys = map[string]bool{"password": true, "nodekey": true, "keystore": true, "unlock": true}

type WebSurfaceSupport struct {
	Status       string   `json:"status"`
	Selectable   bool     `json:"selectable"`
	Reason       string   `json:"reason"`
	EvidenceRefs []string `json:"evidenceRefs"`
}

type WebSurfaceSource struct {
	Path        string     `json:"path"`
	Digest      string     `json:"digest"`
	Kind        string     `json:"kind"`
	ChainCommit string     `json:"chainCommit,omitempty"`
	CapturedAt  *time.Time `json:"capturedAt,omitempty"`
}

type WebBinaryFingerprint struct {
	AssetID      string    `json:"assetId"`
	Checksum     string    `json:"checksum"`
	ChainID      string    `json:"chainId"`
	Version      string    `json:"version"`
	ChainCommit  string    `json:"chainCommit,omitempty"`
	OS           string    `json:"os"`
	Architecture string    `json:"architecture"`
	VerifiedAt   time.Time `json:"verifiedAt"`
}

type WebChainCommand struct {
	ID             string            `json:"id"`
	CommandPath    []string          `json:"commandPath"`
	Positional     []string          `json:"positional"`
	Summary        string            `json:"summary"`
	EngineUsecases []string          `json:"engineUsecases"`
	Support        WebSurfaceSupport `json:"support"`
}

type WebChainOption struct {
	ID          string            `json:"id"`
	OptionKey   string            `json:"optionKey,omitempty"`
	Spelling    string            `json:"spelling"`
	Aliases     []string          `json:"aliases"`
	CommandPath []string          `json:"commandPath"`
	Scope       string            `json:"scope"`
	Boolean     bool              `json:"boolean"`
	Hidden      bool              `json:"hidden"`
	SchemaRef   string            `json:"schemaRef,omitempty"`
	ContractID  string            `json:"contractId,omitempty"`
	Emission    string            `json:"emission"`
	Support     WebSurfaceSupport `json:"support"`
}

// WebChainSurface joins a chain's captured documentation, the engine's
// launch mapping and, when a binary is selected, that binary's own help.
type WebChainSurface struct {
	ChainID         string                `json:"chainId"`
	Dialect         string                `json:"dialect"`
	ContractVersion string                `json:"contractVersion"`
	Fingerprint     string                `json:"fingerprint"`
	Verification    string                `json:"verification"`
	Sources         []WebSurfaceSource    `json:"sources"`
	Binary          *WebBinaryFingerprint `json:"binary,omitempty"`
	Commands        []WebChainCommand     `json:"commands"`
	Options         []WebChainOption      `json:"options"`
	Limitations     []string              `json:"limitations"`
}

// WebChainSurfaceRef pins the surface a plan was reviewed against.
type WebChainSurfaceRef struct {
	ChainID         string `json:"chainId"`
	AssetID         string `json:"assetId"`
	Fingerprint     string `json:"fingerprint"`
	ContractVersion string `json:"contractVersion"`
}

// webBinaryProbe is what a selected binary said about itself: its top-level
// help, the help of each engine command, and whether each documented hidden
// flag is accepted. Nothing here starts a node or opens a database.
type webBinaryProbe struct {
	Fingerprint WebBinaryFingerprint
	Help        []byte
	Commands    map[string][]byte
	Hidden      map[string]bool
}

var errUnknownChainSurface = errors.New("no captured surface for this chain")

// webEngineContractVersion is the version of the engine-owned document
// contracts a surface was joined with.
const webEngineContractVersion = "2"

// LoadChainSurface reads the captured documentation for one chain.
func LoadChainSurface(analysisRoot, chain string) (chainsurface.Catalog, error) {
	dir, ok := webSurfaceDirs[chain]
	if !ok || analysisRoot == "" {
		return chainsurface.Catalog{}, errUnknownChainSurface
	}
	c, err := chainsurface.Load(filepath.Join(analysisRoot, dir))
	if err != nil {
		return c, err
	}
	for i := range c.Sources {
		c.Sources[i].Path = filepath.ToSlash(filepath.Join("docs", "chain-analysis", dir, filepath.Base(c.Sources[i].Path)))
	}
	return c, nil
}

// CheckChainSurfaces loads the documentation of every chain that has one.
func CheckChainSurfaces(analysisRoot string) error {
	for chain := range webSurfaceDirs {
		if _, err := LoadChainSurface(analysisRoot, chain); err != nil {
			return fmt.Errorf("%s: %w", chain, err)
		}
	}
	return nil
}

// JoinChainSurface builds the response. probe is nil when no binary is
// selected; the result is then documentation only and cannot authorize a plan.
func JoinChainSurface(chain string, catalog chainsurface.Catalog, probe *webBinaryProbe) (WebChainSurface, error) {
	plugin, err := registry.Get(chain)
	if err != nil {
		return WebChainSurface{}, err
	}
	manifest := plugin.Manifest()
	dialect, err := nodeconfig.DialectFor(manifest.Dialect)
	if err != nil {
		return WebChainSurface{}, err
	}
	out := WebChainSurface{ChainID: chain, Dialect: dialect.ID, ContractVersion: webEngineContractVersion, Verification: "documented", Commands: []WebChainCommand{}, Options: []WebChainOption{}, Limitations: []string{}}
	for _, s := range catalog.Sources {
		src := WebSurfaceSource{Path: s.Path, Digest: s.Digest, Kind: s.Kind}
		if s.Kind == "help" {
			at := catalog.CapturedAt
			src.ChainCommit, src.CapturedAt = catalog.ChainCommit, &at
		}
		out.Sources = append(out.Sources, src)
	}
	docRef := "help:" + catalog.Sources[0].Path
	live := map[string]chainsurface.HelpFlag{}
	if probe != nil {
		out.Binary = &probe.Fingerprint
		out.Sources = append(out.Sources, WebSurfaceSource{Path: "asset:" + probe.Fingerprint.AssetID + "/--help", Digest: manifestHash(probe.Help), Kind: "binary-probe", ChainCommit: probe.Fingerprint.ChainCommit})
		for _, f := range chainsurface.ParseHelpFlags(probe.Help) {
			live[f.Spelling] = f
		}
	}
	// The engine's launch mapping, by flag spelling.
	type mapped struct {
		key     string
		boolean bool
		perNode bool
	}
	mapping := map[string]mapped{}
	for key, raw := range dialect.OptionSchemas(true) {
		s := raw.(map[string]any)
		mapping[s["x-flag"].(string)] = mapped{key: key, boolean: s["type"] == "boolean", perNode: s["x-perNode"] == true}
	}
	documented := map[string]chainsurface.HelpFlag{}
	for _, f := range catalog.Global {
		documented[f.Spelling] = f
	}
	spellings := map[string]bool{}
	for s := range documented {
		spellings[s] = true
	}
	for s := range mapping {
		spellings[s] = true
	}
	for _, s := range catalog.Hidden {
		spellings[s] = true
	}
	for _, s := range catalog.NotAccepted {
		spellings[s] = true
	}
	for s := range live {
		spellings[s] = true
	}
	var global []string
	for s := range spellings {
		global = append(global, s)
	}
	sort.Strings(global)
	incompatible := []string{}
	for _, spelling := range global {
		doc, isDoc := documented[spelling]
		hidden := slices.Contains(catalog.Hidden, spelling)
		refused := slices.Contains(catalog.NotAccepted, spelling)
		m, isMapped := mapping[spelling]
		opt := WebChainOption{ID: "global:" + spelling, Spelling: spelling, Aliases: doc.Aliases, CommandPath: []string{}, Scope: "global", Boolean: doc.Boolean, Hidden: hidden, Emission: "unmapped"}
		if opt.Aliases == nil {
			opt.Aliases = []string{}
		}
		if f, ok := live[spelling]; ok && !isDoc {
			opt.Aliases, opt.Boolean = f.Aliases, f.Boolean
		}
		if isMapped {
			opt.OptionKey, opt.Boolean, opt.ContractID = m.key, m.boolean, "chain-preset"
			opt.SchemaRef = "#/$defs/envSpec/properties/launch/additionalProperties/properties/" + m.key
			switch {
			case webPrivateKeys[m.key]:
				opt.Emission = "private-binding"
			case m.perNode:
				opt.Emission = "derived"
			default:
				opt.Emission = "user-input"
				opt.SchemaRef = "#/$defs/envSpec/properties/launch/properties/all/properties/" + m.key
			}
		}
		opt.Support = webOptionSupport(opt, isDoc, hidden, refused, isMapped, m.boolean, probe, live, docRef)
		if probe != nil && isMapped && opt.Support.Status != "supported" {
			incompatible = append(incompatible, spelling)
		}
		out.Options = append(out.Options, opt)
	}
	// Commands, with the flags the engine passes to the ones it runs.
	root := WebChainCommand{ID: "command:", CommandPath: []string{}, Positional: []string{}, Summary: "Run a node (the default command)"}
	commands := append([]chainsurface.HelpCommand{{Path: []string{}}}, catalog.Commands...)
	for _, c := range commands {
		cmd := root
		if len(c.Path) > 0 {
			cmd = WebChainCommand{ID: "command:" + strings.Join(c.Path, " "), CommandPath: c.Path, Positional: c.Positional, Summary: c.Summary}
		}
		engine, used := webEngineCommandFor(c.Path, manifest.ConsensusFamily)
		cmd.EngineUsecases = []string{}
		cmd.Support = WebSurfaceSupport{Status: "engine-unmapped", Reason: "no existing use case runs this command; its presence grants no Web execution", EvidenceRefs: []string{docRef}}
		if used {
			cmd.EngineUsecases = engine.usecases
			cmd.Support = WebSurfaceSupport{Status: "supported", Reason: "run by existing use cases with engine-owned arguments", EvidenceRefs: []string{docRef}}
		}
		for _, f := range c.Flags {
			opt := WebChainOption{ID: "local:" + strings.Join(c.Path, " ") + ":" + f.Spelling, Spelling: f.Spelling, Aliases: f.Aliases, CommandPath: c.Path, Scope: "command-local", Boolean: f.Boolean, Emission: "unmapped",
				Support: WebSurfaceSupport{Status: "engine-unmapped", Reason: "the engine does not pass this flag to this command", EvidenceRefs: []string{docRef}}}
			if used && slices.Contains(engine.flags, f.Spelling) {
				opt.Emission = "derived"
				opt.Support = WebSurfaceSupport{Status: "supported", Reason: "the engine derives this value", EvidenceRefs: []string{docRef}}
			}
			out.Options = append(out.Options, opt)
		}
		if used && probe != nil && len(c.Path) > 0 {
			help, ok := probe.Commands[strings.Join(c.Path, " ")]
			liveLocal := map[string]bool{}
			for _, f := range chainsurface.ParseHelpFlags(help) {
				liveLocal[f.Spelling] = true
			}
			missing := []string{}
			for _, f := range engine.flags {
				if !liveLocal[f] {
					missing = append(missing, f)
				}
			}
			if !ok || len(missing) > 0 {
				cmd.Support = WebSurfaceSupport{Status: "binary-absent", Reason: fmt.Sprintf("the selected binary's help for this command lacks %s", strings.Join(missing, ", ")), EvidenceRefs: []string{"binary-probe"}}
				incompatible = append(incompatible, cmd.ID)
			} else {
				cmd.Support.EvidenceRefs = append(cmd.Support.EvidenceRefs, "binary-probe")
			}
		}
		out.Commands = append(out.Commands, cmd)
	}
	switch {
	case probe == nil:
		out.Limitations = append(out.Limitations, "Judged from the captured documentation only; deployment, execution and replacement plans require a verified binary.")
	case len(incompatible) > 0:
		out.Verification = "incompatible"
		out.Limitations = append(out.Limitations, "The selected binary lacks engine mappings: "+strings.Join(incompatible, ", "))
	case !strings.HasPrefix(probe.Fingerprint.ChainCommit, catalog.ChainCommit):
		out.Verification = "stale"
		out.Limitations = append(out.Limitations, fmt.Sprintf("The documentation was captured at chain commit %s but the binary reports %s; the engine mappings were checked against this binary's own help.", catalog.ChainCommit, probe.Fingerprint.ChainCommit))
	default:
		out.Verification = "verified"
	}
	if probe != nil && len(catalog.Hidden) > 0 {
		out.Limitations = append(out.Limitations, "Hidden flags were checked by asking the binary to parse each one with --help; no node or database was touched.")
	}
	fp, err := webSurfaceFingerprint(out)
	if err != nil {
		return out, err
	}
	out.Fingerprint = fp
	return out, nil
}

func webOptionSupport(opt WebChainOption, isDoc, hidden, refused, isMapped, boolean bool, probe *webBinaryProbe, live map[string]chainsurface.HelpFlag, docRef string) WebSurfaceSupport {
	if refused {
		return WebSurfaceSupport{Status: "binary-absent", Reason: "defined in the source but registered on no command; the binary rejects it", EvidenceRefs: []string{"ast-wiring"}}
	}
	if probe == nil {
		switch {
		case !isMapped:
			return WebSurfaceSupport{Status: "engine-unmapped", Reason: "no current engine OptionKey mapping", EvidenceRefs: []string{docRef}}
		case !isDoc && !hidden:
			return WebSurfaceSupport{Status: "binary-absent", Reason: "the engine maps this flag but the captured help does not define it", EvidenceRefs: []string{docRef}}
		case hidden:
			return WebSurfaceSupport{Status: "unverified", Reason: "hidden flag; accepted only after the selected binary is asked directly", EvidenceRefs: []string{"ast-wiring"}}
		}
		return WebSurfaceSupport{Status: "supported", Selectable: opt.Emission == "user-input", Reason: "documented and mapped by the engine", EvidenceRefs: []string{docRef}}
	}
	f, printed := live[opt.Spelling]
	switch {
	case printed && isMapped && f.Boolean != boolean:
		return WebSurfaceSupport{Status: "binary-absent", Reason: "the binary's flag and the engine mapping disagree on taking a value", EvidenceRefs: []string{"binary-probe"}}
	case printed && isMapped:
		return WebSurfaceSupport{Status: "supported", Selectable: opt.Emission == "user-input", Reason: "the selected binary's help defines this mapped flag", EvidenceRefs: []string{"binary-probe"}}
	case !printed && probe.Hidden[opt.Spelling] && isMapped:
		return WebSurfaceSupport{Status: "supported", Selectable: opt.Emission == "user-input", Reason: "absent from the top-level help but accepted when the selected binary is asked directly", EvidenceRefs: []string{"binary-probe:direct"}}
	case !isMapped && (printed || probe.Hidden[opt.Spelling]):
		return WebSurfaceSupport{Status: "engine-unmapped", Reason: "no current engine OptionKey mapping", EvidenceRefs: []string{"binary-probe"}}
	case !printed && hidden && !probe.Hidden[opt.Spelling]:
		return WebSurfaceSupport{Status: "binary-absent", Reason: "documented as hidden but the selected binary rejects it", EvidenceRefs: []string{"binary-probe:direct"}}
	}
	return WebSurfaceSupport{Status: "binary-absent", Reason: "the selected binary's help does not define this flag", EvidenceRefs: []string{"binary-probe"}}
}

func webEngineCommandFor(path []string, family string) (webEngineCommand, bool) {
	for _, c := range webEngineCommands {
		if slices.Equal(c.path, path) && (len(c.families) == 0 || slices.Contains(c.families, family)) {
			return c, true
		}
	}
	return webEngineCommand{}, false
}

func webSurfaceFingerprint(s WebChainSurface) (string, error) {
	type row struct{ ID, Status string }
	rows := []row{}
	for _, o := range s.Options {
		rows = append(rows, row{o.ID, o.Support.Status})
	}
	for _, c := range s.Commands {
		rows = append(rows, row{c.ID, c.Support.Status})
	}
	var binary string
	if s.Binary != nil {
		binary = s.Binary.Checksum
	}
	raw, err := json.Marshal(struct {
		Chain, Dialect, Contract, Verification, Binary string
		Sources                                        []WebSurfaceSource
		Rows                                           []row
	}{s.ChainID, s.Dialect, s.ContractVersion, s.Verification, binary, s.Sources, rows})
	if err != nil {
		return "", err
	}
	return manifestHash(raw), nil
}

// probeWebBinary asks a verified binary for its help pages and whether it
// accepts each documented hidden flag and each engine-mapped flag its
// top-level help does not print. A hidden flag is asked as
// "<flag> --help --help": a flag that takes a value consumes the first --help,
// and the second still prints help, so no form of the request reaches the
// node command. Each run also gets an empty working and home directory that
// is removed afterwards, so even a binary that ignored --help could not write
// into the operator's default data directory.
func probeWebBinary(ctx context.Context, evidenceRoot string, asset ManifestBinary, evidence ManifestBinaryEvidence, catalog chainsurface.Catalog, family string, mapped []string) (*webBinaryProbe, error) {
	help, ok := isolatedWebProbe(ctx, asset.Path, "--help")
	if !ok {
		return nil, errors.New("binary help probe failed")
	}
	out := &webBinaryProbe{Help: help, Commands: map[string][]byte{}, Hidden: map[string]bool{},
		Fingerprint: WebBinaryFingerprint{AssetID: asset.ID, Checksum: evidence.SHA256, ChainID: evidence.Chain, Version: strings.TrimSpace(evidence.Version), ChainCommit: asset.Commit, OS: evidence.OS, Architecture: evidence.Architecture, VerifiedAt: time.Now().UTC()}}
	for _, c := range webEngineCommands {
		if len(c.path) == 0 || (len(c.families) > 0 && !slices.Contains(c.families, family)) {
			continue
		}
		if page, ok := isolatedWebProbe(ctx, asset.Path, append(slices.Clone(c.path), "--help")...); ok {
			out.Commands[strings.Join(c.path, " ")] = page
		}
	}
	printed := map[string]bool{}
	for _, f := range chainsurface.ParseHelpFlags(help) {
		printed[f.Spelling] = true
	}
	ask := slices.Clone(catalog.Hidden)
	for _, f := range mapped {
		if !printed[f] && !slices.Contains(ask, f) {
			ask = append(ask, f)
		}
	}
	for _, f := range ask {
		page, ok := isolatedWebProbe(ctx, asset.Path, f, "--help", "--help")
		text := strings.ToLower(string(page))
		// Only "not defined" is a refusal. A numeric flag rejects --help as
		// its value ("invalid value ... for flag"), which proves it exists.
		out.Hidden[f] = !strings.Contains(text, "not defined") && (ok || strings.Contains(text, "invalid value"))
	}
	if evidenceRoot != "" {
		path := filepath.Join(evidenceRoot, "chain-surfaces", evidence.SHA256, "help.txt")
		files := filestore.Local{}
		if kept, err := files.Exists(ctx, path); err == nil && !kept {
			if err = files.Write(ctx, path, help, 0o600); err != nil {
				return nil, errors.New("cannot keep binary help evidence")
			}
		}
	}
	return out, nil
}

// isolatedWebProbe runs one help request with a ten-second limit through
// manifestProbe's isolation. It reports false when the binary exits with an
// error, which is how a binary refuses a flag it does not define.
func isolatedWebProbe(ctx context.Context, path string, args ...string) ([]byte, bool) {
	probe, cancel := context.WithTimeout(ctx, 10*time.Second)
	defer cancel()
	out, err := manifestProbe(probe, path, args...)
	return out, err == nil
}

// UseChainSurfaces points plans at the captured chain documentation. When it
// is set, every plan pins the surface of each binary it runs and execution
// refuses a plan whose surface changed.
func (e *WebChainEngine) UseChainSurfaces(analysisRoot string) { e.surfaces = analysisRoot }

// ChainSurface reads one chain's surface, judged against a registered binary
// when assetID names one. Running a binary's help is part of planning, so
// only roles that may plan can ask for it.
func (e *WebChainEngine) ChainSurface(ctx context.Context, a DeploymentActor, chain, assetID string) (WebChainSurface, error) {
	if assetID != "" && !a.canEdit() {
		return WebChainSurface{}, ErrDeploymentForbidden
	}
	catalog, err := LoadChainSurface(e.surfaces, chain)
	if errors.Is(err, errUnknownChainSurface) {
		return WebChainSurface{}, ErrDeploymentNotFound
	}
	if err != nil {
		return WebChainSurface{}, err
	}
	if assetID == "" {
		return JoinChainSurface(chain, catalog, nil)
	}
	asset, err := e.binaryAsset(assetID)
	if err != nil {
		return WebChainSurface{}, err
	}
	evidence, err := e.verify(ctx, asset, chain)
	if err != nil {
		return WebChainSurface{}, err
	}
	return e.judgeSurface(ctx, catalog, asset, evidence)
}

func (e *WebChainEngine) judgeSurface(ctx context.Context, catalog chainsurface.Catalog, asset ManifestBinary, evidence ManifestBinaryEvidence) (WebChainSurface, error) {
	plugin, err := registry.Get(evidence.Chain)
	if err != nil {
		return WebChainSurface{}, err
	}
	dialect, err := nodeconfig.DialectFor(plugin.Manifest().Dialect)
	if err != nil {
		return WebChainSurface{}, err
	}
	var mapped []string
	for _, raw := range dialect.OptionSchemas(true) {
		mapped = append(mapped, raw.(map[string]any)["x-flag"].(string))
	}
	sort.Strings(mapped)
	probe, err := probeWebBinary(ctx, e.root, asset, evidence, catalog, plugin.Manifest().ConsensusFamily, mapped)
	if err != nil {
		return WebChainSurface{}, err
	}
	return JoinChainSurface(evidence.Chain, catalog, probe)
}

// pinSurface judges one binary a plan runs. The binary must carry every
// engine mapping, and every launch option the plan sets must be supported
// by that binary, whether or not the documentation is current for it.
func (e *WebChainEngine) pinSurface(ctx context.Context, asset ManifestBinary, evidence ManifestBinaryEvidence, used []string) (WebChainSurfaceRef, error) {
	catalog, err := LoadChainSurface(e.surfaces, evidence.Chain)
	if err != nil {
		return WebChainSurfaceRef{}, err
	}
	s, err := e.judgeSurface(ctx, catalog, asset, evidence)
	if err != nil {
		return WebChainSurfaceRef{}, err
	}
	if s.Verification == "incompatible" {
		return WebChainSurfaceRef{}, fmt.Errorf("binary %s cannot run this plan: %s", asset.ID, strings.Join(s.Limitations, " "))
	}
	for _, key := range used {
		ok := false
		for _, o := range s.Options {
			if o.OptionKey == key && o.Scope == "global" && o.Support.Status == "supported" {
				ok = true
				break
			}
		}
		if !ok {
			return WebChainSurfaceRef{}, fmt.Errorf("binary %s does not support launch option %s", asset.ID, key)
		}
	}
	return WebChainSurfaceRef{ChainID: s.ChainID, AssetID: asset.ID, Fingerprint: s.Fingerprint, ContractVersion: s.ContractVersion}, nil
}

// webPlanSurfaceAssets lists each binary a plan runs with its verified
// evidence, so the plan can pin the surface of every one.
func (e *WebChainEngine) webPlanSurfaceAssets(p webChainPayload) ([]ManifestBinary, []ManifestBinaryEvidence, error) {
	var assets []ManifestBinary
	var evidence []ManifestBinaryEvidence
	add := func(id string, ev ManifestBinaryEvidence) error {
		if id == "" {
			return nil
		}
		for _, a := range assets {
			if a.ID == id {
				return nil
			}
		}
		asset, err := e.binaryAsset(id)
		if err != nil {
			return err
		}
		assets, evidence = append(assets, asset), append(evidence, ev)
		return nil
	}
	if err := add(p.Arguments.AssetID, p.Binary); err != nil {
		return nil, nil, err
	}
	for _, named := range p.NamedBinaries {
		if err := add(named.AssetID, named.Binary); err != nil {
			return nil, nil, err
		}
	}
	if p.Replacement != nil {
		if err := add(p.Arguments.ReplacementAssetID, p.Replacement.Evidence); err != nil {
			return nil, nil, err
		}
	}
	return assets, evidence, nil
}

// webPlanLaunchKeys lists the launch OptionKeys the plan's declarations set.
func webPlanLaunchKeys(p webChainPayload) []string {
	seen := map[string]bool{}
	var plans []ComposePlan
	if p.Preset != nil {
		plans = append(plans, p.Preset.Plan)
	}
	if p.TestRun != nil {
		plans = append(plans, p.TestRun.Plan)
	}
	for _, plan := range plans {
		for _, knobs := range plan.Launch {
			for _, k := range knobs {
				key, _, _ := strings.Cut(k.Knob, "=")
				seen[strings.TrimSpace(key)] = true
			}
		}
	}
	out := make([]string, 0, len(seen))
	for k := range seen {
		out = append(out, k)
	}
	sort.Strings(out)
	return out
}

// pinPlanSurfaces records the surface of every binary the plan runs.
func (e *WebChainEngine) pinPlanSurfaces(ctx context.Context, p *webChainPayload) error {
	if e.surfaces == "" {
		return nil
	}
	refs, err := e.planSurfaces(ctx, *p)
	if err != nil {
		return err
	}
	p.ChainSurfaces = refs
	return nil
}

func (e *WebChainEngine) planSurfaces(ctx context.Context, p webChainPayload) ([]WebChainSurfaceRef, error) {
	assets, evidence, err := e.webPlanSurfaceAssets(p)
	if err != nil {
		return nil, err
	}
	used := webPlanLaunchKeys(p)
	refs := []WebChainSurfaceRef{}
	for i, asset := range assets {
		ref, err := e.pinSurface(ctx, asset, evidence[i], used)
		if err != nil {
			return nil, err
		}
		refs = append(refs, ref)
	}
	return refs, nil
}

// recheckPlanSurfaces refuses execution when a pinned surface changed: the
// documentation, the engine mapping or what the binary itself reports.
func (e *WebChainEngine) recheckPlanSurfaces(ctx context.Context, p webChainPayload) error {
	if e.surfaces == "" {
		return nil
	}
	now, err := e.planSurfaces(ctx, p)
	if err != nil || len(now) != len(p.ChainSurfaces) {
		return ErrDeploymentConflict
	}
	for i := range now {
		if now[i] != p.ChainSurfaces[i] {
			return ErrDeploymentConflict
		}
	}
	return nil
}
