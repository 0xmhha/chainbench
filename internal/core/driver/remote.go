package driver

import (
	"context"
	"encoding/base64"
	"fmt"
	"io/fs"
	"path"
	"strconv"
	"strings"

	"github.com/0xmhha/chainbench/internal/core/remote"
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

// SSHSudoRunner is SSHRunner with every command elevated through sudo, the
// way a real fleet's operator account works: the login is an ordinary user,
// and sudo asks for that user's password. The password is the credentials'
// own and travels on stdin (sudo -S), never in the command line — a command
// line is visible in the remote process list and shell history.
//
// The inventory's ssh.sudo says whether a server PERMITS elevation; whether a
// given step NEEDS it stays the step's decision, which is why this is a
// separate Runner rather than a mode on SSHRunner.
func SSHSudoRunner(creds remote.Credentials, hostKey remote.HostKeyCallback) Runner {
	return func(ctx context.Context, command string) (remote.ExecResult, error) {
		return remote.ExecWithInput(ctx, creds, hostKey, sudoWrap(command), creds.Password+"\n")
	}
}

// sudoWrap renders a shell line so it runs under sudo with the password read
// from stdin. -S takes the password from stdin, -k drops any cached
// credential first (every command re-authenticates, so behavior does not
// depend on what ran before), and -p ” silences the prompt that would
// otherwise interleave with the command's own stderr.
func sudoWrap(command string) string {
	return "sudo -S -k -p '' /bin/sh -c " + shq(command)
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
	out, err := d.sh(ctx, launchCommand(spec))
	if err != nil {
		return Handle{}, fmt.Errorf("driver: remote launch node%d: %w", spec.Index, err)
	}
	pid, err := strconv.Atoi(strings.TrimSpace(out))
	if err != nil || pid <= 0 {
		return Handle{}, fmt.Errorf("driver: remote launch node%d returned no pid (%q)", spec.Index, out)
	}
	return Handle{Index: spec.Index, PID: pid}, nil
}

// launchCommand renders the shell line that starts a node in the background
// and echoes its PID.
//
// The grammar is load-bearing. `mkdir && nohup CMD … &` backgrounds the WHOLE
// `mkdir && nohup CMD` list: the shell forks a subshell, the redirections bind
// only to CMD inside it, and the subshell — still holding the SSH session's
// output pipes — waits on CMD in its foreground. The session then never sees
// EOF and Launch never returns. It went unnoticed while launched nodes crashed
// at startup (a dead subshell closes the pipes); the first node that lived
// hung the harness. `mkdir || exit 1; nohup CMD … &` backgrounds only CMD.
// stdin is redirected away so the node never holds the session's stdin either.
func launchCommand(spec NodeSpec) string {
	parts := make([]string, 0, len(spec.Args)+1)
	parts = append(parts, shq(spec.Binary))
	for _, a := range spec.Args {
		parts = append(parts, shq(a))
	}
	return "mkdir -p " + shq(path.Dir(spec.LogPath)) +
		" || exit 1; nohup " + strings.Join(parts, " ") +
		" > " + shq(spec.LogPath) + " 2>&1 < /dev/null & echo $!"
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
