package app

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"os"
	"path/filepath"
	"strconv"
	"time"

	"go.yaml.in/yaml/v3"

	"github.com/0xmhha/chainbench/internal/core/filestore"
	"github.com/0xmhha/chainbench/internal/resource"
)

// composeThroughCLI runs `chainbench chain up` with args and reports each of
// steps, in order, as the CLI records it in the workspace at dir: running when
// it is the next one not done, succeeded once the record says so. It returns
// the step that failed, or "" when every step finished.
//
// The record is read through `chain status` every poll while the command runs,
// and once more when it ends, so a step the CLI finished just before exiting is
// not reported as the failure.
func composeThroughCLI(ctx context.Context, cli ChainCLI, dir string, steps, args []string, report func(WebJobPhase) error, poll time.Duration) (string, error) {
	if cli == nil {
		return "", errNoChainCLI
	}
	if poll <= 0 {
		poll = time.Millisecond
	}
	done := make(chan error, 1)
	go func() {
		_, err := cli.Run(ctx, args...)
		done <- err
	}()
	next, running := 0, -1
	var started time.Time
	advance := func() error {
		if _, err := os.Stat(filepath.Join(dir, "chain-record.json")); err != nil {
			return nil // Nothing recorded yet: the first step has not got far.
		}
		st, err := cli.ChainStatus(ctx, dir)
		if err != nil {
			return nil // A read that lost a race with a write is tried again.
		}
		for next < len(steps) {
			if running != next {
				started = time.Now().UTC()
				if err = report(WebJobPhase{Name: steps[next], State: "running", StartedAt: started}); err != nil {
					return err
				}
				running = next
			}
			if !st.Steps[steps[next]].Done {
				return nil
			}
			if err = report(WebJobPhase{Name: steps[next], State: "succeeded", StartedAt: started, FinishedAt: time.Now().UTC()}); err != nil {
				return err
			}
			next++
		}
		return nil
	}
	tick := time.NewTicker(poll)
	defer tick.Stop()
	for {
		select {
		case runErr := <-done:
			if err := advance(); err != nil {
				return stepAt(steps, next), err
			}
			if runErr == nil && next == len(steps) {
				return "", nil
			}
			if runErr == nil {
				runErr = fmt.Errorf("chainbench chain up ended without recording %s", steps[next])
			}
			failed := stepAt(steps, next)
			if running != next {
				started = time.Now().UTC()
			}
			phase := WebJobPhase{Name: failed, State: "failed", StartedAt: started, FinishedAt: time.Now().UTC(), Message: runErr.Error()}
			phase.Failure = recordedFailure(ctx, cli, dir, failed)
			if err := report(phase); err != nil {
				return failed, err
			}
			return failed, runErr
		case <-tick.C:
			if err := advance(); err != nil {
				return stepAt(steps, next), err
			}
		}
	}
}

func stepAt(steps []string, i int) string {
	if i < len(steps) {
		return steps[i]
	}
	return steps[len(steps)-1]
}

// composeChain composes the reviewed network with `chainbench chain up`. What
// the Web owns stays here and happens first: the record must still be vacant,
// a registered genesis and the binary must still be the accepted bytes, and the
// Web's own target notes are written. The composition itself, its order and
// its lock, belong to the CLI. It returns the step that failed.
func (e *WebChainEngine) composeChain(ctx context.Context, a DeploymentActor, p webChainPayload, asset ManifestBinary, keysDir string, servers []resource.Server, setPath, configPath string, steps []string, lookup *webTargetReader, report func(WebJobPhase) error) (string, error) {
	if err := e.allowed(a); err != nil {
		return steps[0], err
	}
	if err := recheckWebNetworkRecord(ctx, p, lookup, true); err != nil {
		return "new", err
	}
	if p.Preset != nil && p.Preset.Genesis != nil {
		g := p.Preset.Genesis
		path, err := e.manifests.assets.MaterializeFile(ctx, g.Asset.ID, g.Asset.Checksum, g.Fingerprint)
		if err != nil || path != g.Path {
			return "genesis", ErrDeploymentConflict
		}
	}
	files := filestore.Local{}
	set, err := ExportDeploymentDocument(p.Set.DeploymentDocumentInput, "yaml")
	if err != nil {
		return "new", err
	}
	if err = files.Write(ctx, setPath, set, 0o600); err != nil {
		return "new", err
	}
	config, err := e.composeWorkspaceConfig(ctx, p, asset)
	if err != nil {
		return "deploy", err
	}
	if err = files.Write(ctx, configPath, config, 0o600); err != nil {
		return "new", err
	}
	if err = writeWebTargetNotes(ctx, p); err != nil {
		return "new", err
	}
	chain := []string{"--chain", p.Manifest.ID}
	if p.Manifest.Source == "external" {
		manifest, template, err := e.manifests.files.Pin(p.ControlDir, p.Manifest.Manifest, []byte(p.Manifest.Template))
		if err != nil {
			return "new", err
		}
		chain = []string{"--manifest", manifest, "--genesis-template", template}
	}
	preset := ""
	if p.Preset != nil {
		preset = filepath.Join(p.ControlDir, "chain-preset.json")
		if err = files.Write(ctx, preset, p.Preset.Content, 0o600); err != nil {
			return "new", err
		}
	}
	args := chainUpArgs(p, steps[len(steps)-1], keysDir, setPath, configPath, chain, preset)
	// The job's logins live only as long as the command: a private directory
	// removed when it ends, so no secret outlives the job.
	logins, cleanup, err := e.jobLogins(ctx, a, p)
	if err != nil {
		return "new", err
	}
	args = append(args, "--server-logins", logins)
	failed, err := composeThroughCLI(ctx, e.cli, p.ControlDir, steps, args, func(phase WebJobPhase) error {
		if err := e.allowed(a); err != nil {
			return err
		}
		return report(phase)
	}, composePoll)
	return failed, errors.Join(err, cleanup())
}

// chainUpArgs is the `chain up` a reviewed composition runs: the files the Web
// wrote (keys, server set, workspace config, the pinned preset), the chain
// flags chosen for the manifest, and the plan arguments the jobs contract
// passes on.
func chainUpArgs(p webChainPayload, stage, keysDir, setPath, configPath string, chain []string, preset string) []string {
	args := []string{"chain", "up", "--workspace-dir", p.ControlDir, "--stage", stage, "--keys", keysDir,
		"--binary", p.ExecutionBinary, "--server-set", setPath, "--workspace-config", configPath}
	switch {
	case p.Arguments.AllServers:
		args = append(args, "--all-servers")
	case p.Arguments.ServerRef != "":
		args = append(args, "--server", p.Arguments.ServerRef)
	}
	args = append(args, chain...)
	if preset != "" {
		return append(args, "--chain-preset", preset)
	}
	args = append(args, "--bp", strconv.Itoa(p.Arguments.Validators))
	if p.Arguments.ChainID != 0 {
		args = append(args, "--chain-id", strconv.FormatInt(p.Arguments.ChainID, 10))
	}
	return args
}

// composePoll is how often a running composition's record is read for the
// steps it has finished.
const composePoll = 500 * time.Millisecond

// jobLogins writes the job's bound logins into a private directory and
// returns the server-set file naming them and the removal of that directory.
func (e *WebChainEngine) jobLogins(ctx context.Context, a DeploymentActor, p webChainPayload) (string, func() error, error) {
	root := filepath.Join(e.root, "job-logins")
	if err := os.Mkdir(root, 0o700); err != nil && !os.IsExist(err) {
		return "", nil, err
	}
	dir, err := os.MkdirTemp(root, "job-")
	if err != nil {
		return "", nil, err
	}
	cleanup := func() error { return os.RemoveAll(dir) }
	logins, err := e.documents.writeJobServerSet(ctx, a, p.Set.DeploymentDocumentInput, p.Input.CredentialBindings, filepath.Join(dir, "secrets"))
	if err != nil {
		return "", nil, errors.Join(err, cleanup())
	}
	return logins, cleanup, nil
}

// composeWorkspaceConfig writes the binary directory an SSH composition ships
// from and returns the workspace config it records: a link to the accepted
// binary under the name the nodes run it by, so the CLI ships those bytes to
// every machine and checks their sha256 there.
func (e *WebChainEngine) composeWorkspaceConfig(ctx context.Context, p webChainPayload, asset ManifestBinary) ([]byte, error) {
	if p.Target.Transport == "ssh" {
		b, err := os.ReadFile(asset.Path)
		if err != nil {
			return nil, err
		}
		if manifestHash(b) != p.Binary.SHA256 {
			return nil, ErrDeploymentConflict
		}
		dir := filepath.Join(p.ControlDir, "binaries")
		link := filepath.Join(dir, filepath.Base(p.ExecutionBinary))
		if err = (filestore.Local{}).Remove(ctx, link); err != nil {
			return nil, err
		}
		if err = os.Mkdir(dir, 0o700); err != nil && !os.IsExist(err) {
			return nil, err
		}
		if err = os.Symlink(asset.Path, link); err != nil {
			return nil, err
		}
	}
	return webRecordedWorkspaceConfig(p)
}

// webRecordedWorkspaceConfig is the workspace config a Web composition records:
// the reviewed document, and on SSH targets control.binaries naming the
// directory the CLI ships the accepted binary from. Composing and every later
// check of the record use this one answer.
func webRecordedWorkspaceConfig(p webChainPayload) ([]byte, error) {
	config, err := ExportDeploymentDocument(p.Config.DeploymentDocumentInput, "yaml")
	if err != nil || p.Target.Transport != "ssh" {
		return config, err
	}
	var doc map[string]any
	if err = yaml.Unmarshal(config, &doc); err != nil {
		return nil, err
	}
	control, _ := doc["control"].(map[string]any)
	if control == nil {
		control = map[string]any{}
	}
	control["binaries"] = filepath.Join(p.ControlDir, "binaries")
	doc["control"] = control
	return yaml.Marshal(doc)
}

// recordedDeclarationsMatch refuses a network whose recorded server set or
// workspace config is no longer what the reviewed documents compose.
func recordedDeclarationsMatch(state State, p webChainPayload) error {
	set, err := ExportDeploymentDocument(p.Set.DeploymentDocumentInput, "yaml")
	if err != nil {
		return err
	}
	config, err := webRecordedWorkspaceConfig(p)
	if err != nil {
		return err
	}
	for path, expected := range map[string][]byte{state.WorkspaceConfig: config, state.ServerSet: set} {
		actual, err := os.ReadFile(path)
		if err != nil || manifestHash(actual) != manifestHash(expected) {
			return webConflict("the recorded declarations differ from the reviewed documents")
		}
	}
	return nil
}

// writeWebTargetNotes records, beside the composition, the machine the Web
// reviewed, so its owned processes can be observed after a restart even if the
// composition never finished. A spread network records every machine; readers
// that know only one target refuse its controls rather than guess.
func writeWebTargetNotes(ctx context.Context, p webChainPayload) error {
	files := filestore.Local{}
	metadata, err := json.Marshal(p.Target)
	if err != nil {
		return err
	}
	if err = files.Write(ctx, filepath.Join(p.ControlDir, "web-target.json"), metadata, 0o600); err != nil {
		return err
	}
	if len(p.Targets) == 0 {
		return nil
	}
	all, err := json.Marshal(webRecordedMachines(p))
	if err != nil {
		return err
	}
	return files.Write(ctx, filepath.Join(p.ControlDir, "web-targets.json"), all, 0o600)
}

// runControl runs one chainbench command against the network recorded at
// p.ControlDir, with the job's own logins in a private directory removed when
// the command ends.
func (e *WebChainEngine) runControl(ctx context.Context, a DeploymentActor, p webChainPayload, args ...string) error {
	if e.cli == nil {
		return errNoChainCLI
	}
	logins, cleanup, err := e.jobLogins(ctx, a, p)
	if err != nil {
		return err
	}
	_, err = e.cli.Run(ctx, append(args, "--workspace-dir", p.ControlDir, "--server-logins", logins)...)
	return errors.Join(err, cleanup())
}

// nodeSwapArgs is the `node swap` a reviewed replacement runs.
func nodeSwapArgs(index int, binary string, changes []string) []string {
	args := []string{"node", "swap", "--index", strconv.Itoa(index), "--purpose", "web-reviewed-config"}
	if binary != "" {
		args = append(args, "--binary", binary)
	}
	for _, change := range changes {
		args = append(args, "--config", change)
	}
	return args
}

// recordedFailure reads how the CLI classified the failure of step, when the
// record says; a step that failed before its stage could classify it has none.
func recordedFailure(ctx context.Context, cli ChainCLI, dir, step string) *WebPhaseFailure {
	if _, err := os.Stat(filepath.Join(dir, "chain-record.json")); err != nil {
		return nil
	}
	st, err := cli.ChainStatus(ctx, dir)
	if err != nil {
		return nil
	}
	recorded := st.Steps[step]
	if recorded.Failed == "" {
		return nil
	}
	return &WebPhaseFailure{State: recorded.Failed, Detail: recorded.Detail}
}
