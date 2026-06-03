package main

import (
	"context"
	"encoding/json"
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"regexp"
	"time"

	"github.com/0xmhha/chainbench/network/internal/events"
)

// lifecycleTimeout bounds init/start/restart wall-clock. These spawn genesis
// generation and node boot, which are slower than stop/status, so the bound is
// generous.
const lifecycleTimeout = 180 * time.Second

// cleanTimeout bounds network.clean (data removal only — fast).
const cleanTimeout = 30 * time.Second

// profileNamePattern mirrors the MCP lifecycle tool's profile validation so the
// wire boundary rejects the same malformed names before spawning bash.
var profileNamePattern = regexp.MustCompile(`^[a-zA-Z0-9_\-/]+$`)

// runLifecycleScript spawns chainbench.sh with the given args, mirroring the
// network.stop_all / network.status thin-wrapper pattern (VISION §5.12 M2):
// inherit the parent env, append CHAINBENCH_DIR, and forward the bash CLI's
// combined output as {stdout}. The lifecycle reroutes (init/start_all/restart/
// clean) delegate to the existing bash flow rather than reimplementing genesis
// generation and node boot natively in Go.
func runLifecycleScript(chainbenchDir string, timeout time.Duration, args ...string) (map[string]any, error) {
	ctx, cancel := context.WithTimeout(context.Background(), timeout)
	defer cancel()
	cmd := exec.CommandContext(ctx, filepath.Join(chainbenchDir, "chainbench.sh"), args...)
	cmd.Env = append(os.Environ(), "CHAINBENCH_DIR="+chainbenchDir)
	out, err := cmd.CombinedOutput()
	if err != nil {
		return nil, NewUpstream("chainbench "+args[0], fmt.Errorf("%w: %s", err, string(out)))
	}
	return map[string]any{"stdout": string(out)}, nil
}

// validateWireBinaryPath rejects a non-absolute binary_path at the wire
// boundary. Empty is allowed (the profile default is used).
func validateWireBinaryPath(p string) error {
	if p == "" {
		return nil
	}
	if !filepath.IsAbs(p) {
		return NewInvalidArgs(fmt.Sprintf("binary_path must be an absolute path: %q", p))
	}
	return nil
}

// newHandleNetworkInit returns the "network.init" handler: it initializes a
// local chain from a profile (genesis + TOML + datadir) by spawning
// `chainbench.sh init --profile <profile> --quiet [--binary-path <path>]`.
//
// Args: { "profile"?: "default", "binary_path"?: "/abs/path" }
//
// Result: { "stdout": "<bash output>" }
//
// Error mapping:
//
//	INVALID_ARGS    — malformed args JSON, bad profile name, or relative binary_path.
//	UPSTREAM_ERROR  — chainbench.sh non-zero exit or spawn failure.
func newHandleNetworkInit(stateDir, chainbenchDir string) Handler {
	return func(args json.RawMessage, _ *events.Bus) (map[string]any, error) {
		var req struct {
			Profile    string `json:"profile"`
			BinaryPath string `json:"binary_path"`
		}
		if len(args) > 0 {
			if err := json.Unmarshal(args, &req); err != nil {
				return nil, NewInvalidArgs(fmt.Sprintf("args: %v", err))
			}
		}
		if req.Profile == "" {
			req.Profile = "default"
		}
		if !profileNamePattern.MatchString(req.Profile) {
			return nil, NewInvalidArgs(fmt.Sprintf(
				"invalid profile name %q: only alphanumeric, dash, underscore, and slash are allowed", req.Profile,
			))
		}
		if err := validateWireBinaryPath(req.BinaryPath); err != nil {
			return nil, err
		}
		_ = stateDir

		cmdArgs := []string{"init", "--profile", req.Profile, "--quiet"}
		if req.BinaryPath != "" {
			cmdArgs = append(cmdArgs, "--binary-path", req.BinaryPath)
		}
		return runLifecycleScript(chainbenchDir, lifecycleTimeout, cmdArgs...)
	}
}

// newHandleNetworkStartAll returns the "network.start_all" handler: it starts
// all nodes of an initialized local chain by spawning
// `chainbench.sh start --quiet [--binary-path <path>]`.
//
// Args: { "binary_path"?: "/abs/path" }
//
// Result: { "stdout": "<bash output>" }
//
// Error mapping mirrors network.init.
func newHandleNetworkStartAll(stateDir, chainbenchDir string) Handler {
	return func(args json.RawMessage, _ *events.Bus) (map[string]any, error) {
		var req struct {
			BinaryPath string `json:"binary_path"`
		}
		if len(args) > 0 {
			if err := json.Unmarshal(args, &req); err != nil {
				return nil, NewInvalidArgs(fmt.Sprintf("args: %v", err))
			}
		}
		if err := validateWireBinaryPath(req.BinaryPath); err != nil {
			return nil, err
		}
		_ = stateDir

		cmdArgs := []string{"start", "--quiet"}
		if req.BinaryPath != "" {
			cmdArgs = append(cmdArgs, "--binary-path", req.BinaryPath)
		}
		return runLifecycleScript(chainbenchDir, lifecycleTimeout, cmdArgs...)
	}
}

// newHandleNetworkRestart returns the "network.restart" handler: stop, clean,
// re-init, and start with the same profile by spawning
// `chainbench.sh restart --quiet [--binary-path <path>]`.
//
// Args: { "binary_path"?: "/abs/path" }
//
// Result: { "stdout": "<bash output>" }
//
// Error mapping mirrors network.init.
func newHandleNetworkRestart(stateDir, chainbenchDir string) Handler {
	return func(args json.RawMessage, _ *events.Bus) (map[string]any, error) {
		var req struct {
			BinaryPath string `json:"binary_path"`
		}
		if len(args) > 0 {
			if err := json.Unmarshal(args, &req); err != nil {
				return nil, NewInvalidArgs(fmt.Sprintf("args: %v", err))
			}
		}
		if err := validateWireBinaryPath(req.BinaryPath); err != nil {
			return nil, err
		}
		_ = stateDir

		cmdArgs := []string{"restart", "--quiet"}
		if req.BinaryPath != "" {
			cmdArgs = append(cmdArgs, "--binary-path", req.BinaryPath)
		}
		return runLifecycleScript(chainbenchDir, lifecycleTimeout, cmdArgs...)
	}
}

// newHandleNetworkClean returns the "network.clean" handler: remove node data
// (keeps config/profiles) by spawning `chainbench.sh clean`.
//
// Args: {} (none)
//
// Result: { "stdout": "<bash output>" }
//
// Error mapping:
//
//	UPSTREAM_ERROR  — chainbench.sh non-zero exit or spawn failure.
func newHandleNetworkClean(stateDir, chainbenchDir string) Handler {
	return func(_ json.RawMessage, _ *events.Bus) (map[string]any, error) {
		_ = stateDir
		return runLifecycleScript(chainbenchDir, cleanTimeout, "clean")
	}
}
