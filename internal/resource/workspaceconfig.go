package resource

import (
	"bytes"
	"fmt"
	"os"
	"path"
	"path/filepath"
	"strings"

	"go.yaml.in/yaml/v3"
)

// WorkspaceConfig is the environment file (`--workspace-config`) that owns where
// a composition's files live ON THE TARGET: the dataRoot and the purpose
// directories under it, plus the local results root. It is the counterpart to
// the server-set (which owns servers and credentials) and the DSL (which owns
// test content and logical references). One DSL runs unchanged across local,
// docker, and remote targets by swapping the server-set and this file.
//
// This type and its resolver are pure: parsing and path arithmetic only, no
// I/O. Reading and writing files on the resolved target is the file-access
// layer's job (Access), so a resolved path always carries which machine it
// belongs to before anything opens it.
//
// The config is authored for a target that may not be this machine, so target
// paths are POSIX (path, not filepath). Only [WorkspaceConfig.ArtifactRoot] is
// local to the machine running chainbench.
type WorkspaceConfig struct {
	// Version is the format version. Only 1 is accepted; an unknown version is
	// refused rather than parsed on a best guess.
	Version int `yaml:"version"`
	// DataRoot is the absolute path on the target under which every purpose
	// directory sits. It is a target path, not resolved against this machine.
	DataRoot string `yaml:"dataRoot"`
	// Paths are the purpose directories, each relative to DataRoot.
	Paths WorkspacePaths `yaml:"paths"`
	// BinaryAliases maps a logical binary name a DSL references to a file under
	// paths.binaries, so an environment can carry a different file name without
	// editing the DSL. Optional; a name with no alias is used verbatim.
	BinaryAliases map[string]string `yaml:"binaryAliases,omitempty"`
	// Control holds the paths local to the machine running chainbench.
	Control Control `yaml:"control"`
	// Inputs decides whether inputs are prepared (found on the target) or
	// generated (built by the in-process Builders).
	Inputs Inputs `yaml:"inputs"`
	// Execution decides how a chain is used: a fresh isolated composition, a
	// matching one reused, or an already-running one attached to.
	Execution Execution `yaml:"execution"`
	// Presets are named bundles of prepared inputs (genesis, keyring, configs),
	// referenced by Inputs.Preset when Inputs.Mode is prepared.
	Presets map[string]InputPreset `yaml:"presets,omitempty"`

	// dir is the directory this config was read from, used to resolve a
	// relative local ArtifactRoot. It is not a YAML field.
	dir string
}

// WorkspacePaths are the dataRoot-relative purpose directories. Every field is
// required and must be a relative path with no traversal — the resolver joins
// them onto DataRoot, so an absolute or `..` value would escape the target root.
type WorkspacePaths struct {
	Binaries string `yaml:"binaries"`
	Configs  string `yaml:"configs"`
	Genesis  string `yaml:"genesis"`
	Keystore string `yaml:"keystore"`
	Keyrings string `yaml:"keyrings"`
	Nodes    string `yaml:"nodes"`
	Runtime  string `yaml:"runtime"`
	Logs     string `yaml:"logs"`
}

// Control holds machine-local paths — the only paths in the file that belong to
// the machine running chainbench rather than the target.
type Control struct {
	// ArtifactRoot is where run results and the final report land locally. A
	// leading ~ expands to the local home; a relative value is relative to the
	// config file's directory.
	ArtifactRoot string `yaml:"artifactRoot"`
}

// Inputs selects how a composition's inputs are prepared.
type Inputs struct {
	// Mode is prepared or generated.
	Mode InputMode `yaml:"mode"`
	// Preset names the entry in Presets to use; required when Mode is prepared,
	// forbidden when generated.
	Preset string `yaml:"preset,omitempty"`
}

// Execution selects how the chain is used across runs.
type Execution struct {
	// Chain is fresh, reuse-if-matching, or attach.
	Chain ChainMode `yaml:"chain"`
}

// InputPreset is a named bundle of prepared inputs — genesis, keyring, and
// configs held together so a promised identity set is verified as a unit. The
// name is deliberately not "Preset": keyring already owns that word for a key
// SOURCE, and one concept keeps one name. Each reference is a server file
// reference the resource layer resolves; the shapes it accepts are validated
// where the reference is consumed, not here.
type InputPreset struct {
	Genesis string            `yaml:"genesis,omitempty"`
	Keyring string            `yaml:"keyring,omitempty"`
	Configs map[string]string `yaml:"configs,omitempty"`
}

// InputMode is how inputs are prepared.
type InputMode string

const (
	// InputPrepared uses files already present on the target; a missing input
	// is an error, never a silent fall back to generation.
	InputPrepared InputMode = "prepared"
	// InputGenerated builds test keys and genesis/config with the in-process
	// Builders. It does not build or download a chain binary.
	InputGenerated InputMode = "generated"
)

// ChainMode is how a chain is used across runs.
type ChainMode string

const (
	// ChainFresh is a new isolated composition; it does not disturb another
	// running network's processes, database, or shared inputs.
	ChainFresh ChainMode = "fresh"
	// ChainReuseIfMatching reuses an existing composition only when the
	// configuration, input content, placement, and running state all match.
	ChainReuseIfMatching ChainMode = "reuse-if-matching"
	// ChainAttach tests an already-running chain; it creates, deploys, and
	// initializes nothing.
	ChainAttach ChainMode = "attach"
)

// Purpose names a directory under dataRoot that a reference resolves into.
type Purpose string

const (
	PurposeBinaries Purpose = "binaries"
	PurposeConfigs  Purpose = "configs"
	PurposeGenesis  Purpose = "genesis"
	PurposeKeystore Purpose = "keystore"
	PurposeKeyrings Purpose = "keyrings"
	PurposeNodes    Purpose = "nodes"
	PurposeRuntime  Purpose = "runtime"
	PurposeLogs     Purpose = "logs"
)

// dirFor returns the configured directory for a purpose.
func (p WorkspacePaths) dirFor(purpose Purpose) (string, bool) {
	switch purpose {
	case PurposeBinaries:
		return p.Binaries, true
	case PurposeConfigs:
		return p.Configs, true
	case PurposeGenesis:
		return p.Genesis, true
	case PurposeKeystore:
		return p.Keystore, true
	case PurposeKeyrings:
		return p.Keyrings, true
	case PurposeNodes:
		return p.Nodes, true
	case PurposeRuntime:
		return p.Runtime, true
	case PurposeLogs:
		return p.Logs, true
	default:
		return "", false
	}
}

// LoadWorkspaceConfig reads and validates a workspace-config file. The file is
// read from the local machine; its dataRoot and paths describe the target.
func LoadWorkspaceConfig(pathToFile string) (WorkspaceConfig, error) {
	b, err := os.ReadFile(pathToFile)
	if err != nil {
		return WorkspaceConfig{}, fmt.Errorf("workspace-config: read %s: %w", pathToFile, err)
	}
	c, err := ParseWorkspaceConfig(b)
	if err != nil {
		return WorkspaceConfig{}, err
	}
	c.dir = filepath.Dir(pathToFile)
	return c, nil
}

// ParseWorkspaceConfig parses and validates config bytes. Unknown fields and
// duplicate keys are rejected so a typo is an error rather than a silently
// ignored intention.
func ParseWorkspaceConfig(b []byte) (WorkspaceConfig, error) {
	dec := yaml.NewDecoder(bytes.NewReader(b))
	dec.KnownFields(true)
	var c WorkspaceConfig
	if err := dec.Decode(&c); err != nil {
		return WorkspaceConfig{}, fmt.Errorf("workspace-config: parse: %w", err)
	}
	if err := c.validate(); err != nil {
		return WorkspaceConfig{}, err
	}
	return c, nil
}

// supportedVersion is the only workspace-config format version accepted.
const supportedVersion = 1

// validate checks the config is internally consistent before anything consumes
// it: a supported version, an absolute target dataRoot, every purpose directory
// present and traversal-free, and a coherent inputs/execution selection.
func (c WorkspaceConfig) validate() error {
	if c.Version != supportedVersion {
		return fmt.Errorf("workspace-config: version %d is not supported (want %d)", c.Version, supportedVersion)
	}
	if err := validateTargetRoot(c.DataRoot); err != nil {
		return err
	}
	for _, purpose := range []Purpose{
		PurposeBinaries, PurposeConfigs, PurposeGenesis, PurposeKeystore,
		PurposeKeyrings, PurposeNodes, PurposeRuntime, PurposeLogs,
	} {
		dir, _ := c.Paths.dirFor(purpose)
		if err := validateRelative(fmt.Sprintf("paths.%s", purpose), dir); err != nil {
			return err
		}
	}
	for name, alias := range c.BinaryAliases {
		if err := validateRelative(fmt.Sprintf("binaryAliases.%s", name), alias); err != nil {
			return err
		}
	}
	if strings.TrimSpace(c.Control.ArtifactRoot) == "" {
		return fmt.Errorf("workspace-config: control.artifactRoot is required")
	}
	if err := c.Inputs.validate(len(c.Presets) > 0); err != nil {
		return err
	}
	if _, ok := c.Inputs.presetName(); ok {
		if _, found := c.Presets[c.Inputs.Preset]; !found {
			return fmt.Errorf("workspace-config: inputs.preset %q has no entry in presets", c.Inputs.Preset)
		}
	}
	return c.Execution.validate()
}

func (in Inputs) validate(hasPresets bool) error {
	switch in.Mode {
	case InputPrepared:
		if in.Preset == "" {
			return fmt.Errorf("workspace-config: inputs.mode prepared needs inputs.preset")
		}
	case InputGenerated:
		if in.Preset != "" {
			return fmt.Errorf("workspace-config: inputs.preset is set but inputs.mode is generated — a generated run declares no preset")
		}
	default:
		return fmt.Errorf("workspace-config: inputs.mode %q is unknown (want %s or %s)", in.Mode, InputPrepared, InputGenerated)
	}
	return nil
}

// presetName returns the preset to use, if inputs declares one.
func (in Inputs) presetName() (string, bool) {
	if in.Mode == InputPrepared && in.Preset != "" {
		return in.Preset, true
	}
	return "", false
}

func (e Execution) validate() error {
	switch e.Chain {
	case ChainFresh, ChainReuseIfMatching, ChainAttach:
		return nil
	default:
		return fmt.Errorf("workspace-config: execution.chain %q is unknown (want %s, %s, or %s)",
			e.Chain, ChainFresh, ChainReuseIfMatching, ChainAttach)
	}
}

// Resolve joins a purpose directory and a portable reference onto the target
// dataRoot, POSIX-style. The reference must be relative and traversal-free —
// an absolute path, `..`, `~`, an environment variable, or an empty value is
// refused, because the whole point is that the same reference resolves under
// whatever root the environment names. An absolute source is expressed as a
// server file reference elsewhere, not here.
func (c WorkspaceConfig) Resolve(purpose Purpose, ref string) (string, error) {
	dir, ok := c.Paths.dirFor(purpose)
	if !ok {
		return "", fmt.Errorf("workspace-config: unknown purpose %q", purpose)
	}
	if err := validateRelative(fmt.Sprintf("%s reference", purpose), ref); err != nil {
		return "", err
	}
	return joinTarget(c.DataRoot, dir, ref)
}

// AdoptDataRoot folds this config's data root into a compose target.
//
// The workspace-config is the single owner of the data root, and that is the
// whole rule: a target that already names a different one is two answers to
// where the data plane lives, so it is a conflict rather than a silent
// override. The same value, or none, folds in quietly. The target's locality
// (local vs remote/SSH) is not touched — that comes from the server set.
//
// It lives here because both callers that fold a config into a target — the
// step-form surfaces through app, and the run path through testengine — were
// each carrying their own copy of the comparison and their own wording for the
// refusal. Two copies of a single-owner rule is one copy too many.
//
// origin names where the target came from, for the message; empty says "the
// target".
func (c WorkspaceConfig) AdoptDataRoot(target Spec, origin string) (Spec, error) {
	if origin == "" {
		origin = "the target"
	}
	if target.DataRoot != "" && target.DataRoot != c.DataRoot {
		return target, fmt.Errorf(
			"data root conflict: %s says %q but --workspace-config says %q — put the data root in workspace-config alone",
			origin, target.DataRoot, c.DataRoot)
	}
	target.DataRoot = c.DataRoot
	return target, nil
}

// BinaryPath resolves a binary reference to its target path, applying a
// binaryAlias when one is declared for the name.
func (c WorkspaceConfig) BinaryPath(ref string) (string, error) {
	if err := validateRelative("binary reference", ref); err != nil {
		return "", err
	}
	name := ref
	if alias, ok := c.BinaryAliases[ref]; ok {
		name = alias
	}
	return joinTarget(c.DataRoot, c.Paths.Binaries, name)
}

// ArtifactRoot resolves the local results root: ~ expands to the local home,
// and a relative value is relative to the config file's directory. This is the
// one path in the file that is local rather than on the target.
func (c WorkspaceConfig) ArtifactRoot() (string, error) {
	root := c.Control.ArtifactRoot
	if strings.HasPrefix(root, "~") {
		home, err := os.UserHomeDir()
		if err != nil {
			return "", fmt.Errorf("workspace-config: expand ~ in artifactRoot: %w", err)
		}
		return filepath.Join(home, strings.TrimPrefix(root, "~")), nil
	}
	if filepath.IsAbs(root) {
		return root, nil
	}
	return filepath.Join(c.dir, root), nil
}

// validateTargetRoot checks dataRoot is an absolute POSIX target path with no
// local expansion tokens. It is not resolved against this machine.
func validateTargetRoot(root string) error {
	if strings.TrimSpace(root) == "" {
		return fmt.Errorf("workspace-config: dataRoot is required")
	}
	if !path.IsAbs(root) {
		return fmt.Errorf("workspace-config: dataRoot %q must be an absolute target path", root)
	}
	if strings.HasPrefix(root, "~") || strings.ContainsAny(root, "$") {
		return fmt.Errorf("workspace-config: dataRoot %q must not use ~ or environment variables", root)
	}
	return nil
}

// validateRelative refuses a reference that is empty, absolute, escapes its
// root, or carries a local expansion token. It is the guard that keeps a
// portable reference portable.
func validateRelative(what, ref string) error {
	if strings.TrimSpace(ref) == "" {
		return fmt.Errorf("workspace-config: %s is empty", what)
	}
	if path.IsAbs(ref) || filepath.IsAbs(ref) {
		return fmt.Errorf("workspace-config: %s %q must be relative", what, ref)
	}
	if strings.HasPrefix(ref, "~") || strings.ContainsAny(ref, "$") {
		return fmt.Errorf("workspace-config: %s %q must not use ~ or environment variables", what, ref)
	}
	// Normalize and reject anything that climbs above the root.
	clean := path.Clean("/" + strings.ReplaceAll(ref, "\\", "/"))
	if clean == "/" || strings.HasPrefix(clean, "/../") || strings.Contains(ref, "..") {
		return fmt.Errorf("workspace-config: %s %q must not contain a .. path segment", what, ref)
	}
	return nil
}

// joinTarget composes a target path POSIX-style and confirms the result stays
// under the root after normalization, so a reference cannot climb out even
// through a directory value.
func joinTarget(root, dir, ref string) (string, error) {
	base := path.Join(root, dir)
	full := path.Join(base, ref)
	if full != base && !strings.HasPrefix(full, base+"/") {
		return "", fmt.Errorf("workspace-config: %q escapes %q", ref, base)
	}
	return full, nil
}
