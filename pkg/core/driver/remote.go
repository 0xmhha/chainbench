package driver

import (
	"context"
	"encoding/base64"
	"fmt"
	"io/fs"
	"path"
	"strconv"
	"strings"

	"github.com/0xmhha/chainbench/pkg/core/remote"
)

// Runner runs a shell command on a remote host and returns its result. It is
// injected so a RemoteDriver can be built over any transport (the default is an
// SSH exec) and tested without a real host. Build the SSH-backed runner with
// SSHRunner.
type Runner func(ctx context.Context, command string) (remote.ExecResult, error)

// SSHRunner returns a Runner that executes each command over SSH using the given
// credentials and host-key policy (pkg/core/remote). Errors never include the
// password.
func SSHRunner(creds remote.Credentials, hostKey remote.HostKeyCallback) Runner {
	return func(ctx context.Context, command string) (remote.ExecResult, error) {
		return remote.Exec(ctx, creds, hostKey, command)
	}
}

// RemoteDriver provisions, launches, and stops nodes on a remote host by running
// shell commands over the injected Runner. It is the remote counterpart of
// LocalDriver, built on the SSH access absorbed into pkg/core/remote.
type RemoteDriver struct {
	run Runner
}

// NewRemoteDriver returns a RemoteDriver over run (e.g. driver.SSHRunner(...)).
func NewRemoteDriver(run Runner) *RemoteDriver { return &RemoteDriver{run: run} }

// sh runs command and returns its stdout, treating a non-zero exit as an error
// (with stderr) so callers get a single error path.
func (d *RemoteDriver) sh(ctx context.Context, command string) (string, error) {
	res, err := d.run(ctx, command)
	if err != nil {
		return "", err
	}
	if res.ExitCode != 0 {
		return "", fmt.Errorf("remote command exited %d: %s", res.ExitCode, strings.TrimSpace(res.Stderr))
	}
	return res.Stdout, nil
}

// Provision creates the data dir and writes the node config (base64-piped to
// avoid quoting issues) on the remote host.
func (d *RemoteDriver) Provision(ctx context.Context, spec NodeSpec) error {
	if _, err := d.sh(ctx, "mkdir -p "+shq(spec.DataDir)); err != nil {
		return fmt.Errorf("driver: remote mkdir datadir: %w", err)
	}
	if spec.ConfigPath != "" && len(spec.ConfigContent) > 0 {
		b64 := base64.StdEncoding.EncodeToString(spec.ConfigContent)
		cmd := "mkdir -p " + shq(path.Dir(spec.ConfigPath)) +
			" && printf %s " + shq(b64) + " | base64 -d > " + shq(spec.ConfigPath)
		if _, err := d.sh(ctx, cmd); err != nil {
			return fmt.Errorf("driver: remote write config: %w", err)
		}
	}
	return nil
}

// ProvisionFile writes content to remotePath on the remote host (base64-piped to
// avoid quoting issues) and chmods it, satisfying the FileProvisioner capability
// so setup can ship per-node identity files (nodekey, keystore, password).
func (d *RemoteDriver) ProvisionFile(ctx context.Context, remotePath string, content []byte, mode fs.FileMode) error {
	b64 := base64.StdEncoding.EncodeToString(content)
	cmd := "mkdir -p " + shq(path.Dir(remotePath)) +
		" && printf %s " + shq(b64) + " | base64 -d > " + shq(remotePath) +
		" && chmod " + fmt.Sprintf("%o", mode.Perm()) + " " + shq(remotePath)
	if _, err := d.sh(ctx, cmd); err != nil {
		return fmt.Errorf("driver: remote write file %s: %w", remotePath, err)
	}
	return nil
}

// InitDatadir ships the genesis to the remote host (base64-piped, like the
// config) and runs `<binary> init --datadir <dataDir> <genesis>` over SSH,
// satisfying the Initializer capability so a remote setup needs no local files.
func (d *RemoteDriver) InitDatadir(ctx context.Context, spec NodeSpec, genesis []byte) error {
	genesisPath := path.Join(spec.DataDir, "genesis.json")
	b64 := base64.StdEncoding.EncodeToString(genesis)
	ship := "mkdir -p " + shq(spec.DataDir) +
		" && printf %s " + shq(b64) + " | base64 -d > " + shq(genesisPath)
	if _, err := d.sh(ctx, ship); err != nil {
		return fmt.Errorf("driver: remote ship genesis node%d: %w", spec.Index, err)
	}
	initCmd := shq(spec.Binary) + " init --datadir " + shq(spec.DataDir) + " " + shq(genesisPath)
	if _, err := d.sh(ctx, initCmd); err != nil {
		return fmt.Errorf("driver: remote init node%d: %w", spec.Index, err)
	}
	return nil
}

// Launch starts the node in the background on the remote host, redirecting
// output to the log file, and returns its PID.
func (d *RemoteDriver) Launch(ctx context.Context, spec NodeSpec) (Handle, error) {
	parts := make([]string, 0, len(spec.Args)+1)
	parts = append(parts, shq(spec.Binary))
	for _, a := range spec.Args {
		parts = append(parts, shq(a))
	}
	cmd := "mkdir -p " + shq(path.Dir(spec.LogPath)) +
		" && nohup " + strings.Join(parts, " ") +
		" > " + shq(spec.LogPath) + " 2>&1 & echo $!"
	out, err := d.sh(ctx, cmd)
	if err != nil {
		return Handle{}, fmt.Errorf("driver: remote launch node%d: %w", spec.Index, err)
	}
	pid, err := strconv.Atoi(strings.TrimSpace(out))
	if err != nil || pid <= 0 {
		return Handle{}, fmt.Errorf("driver: remote launch node%d returned no pid (%q)", spec.Index, out)
	}
	return Handle{Index: spec.Index, PID: pid}, nil
}

// Stop terminates the remote node by PID. A process that has already exited is
// not an error.
func (d *RemoteDriver) Stop(ctx context.Context, h Handle) error {
	// `kill -0` first so an already-gone process is a clean no-op.
	res, err := d.run(ctx, "kill "+strconv.Itoa(h.PID)+" 2>/dev/null || true")
	if err != nil {
		return fmt.Errorf("driver: remote stop node%d (pid %d): %w", h.Index, h.PID, err)
	}
	_ = res
	return nil
}

// shq single-quotes s for safe use in a POSIX shell command.
func shq(s string) string {
	return "'" + strings.ReplaceAll(s, "'", `'\''`) + "'"
}
