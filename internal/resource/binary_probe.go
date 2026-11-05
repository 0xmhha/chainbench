package resource

import (
	"bytes"
	"context"
	"errors"
	"fmt"
	"path/filepath"
	"strconv"
	"strings"

	"github.com/0xmhha/chainbench/internal/core/remote"
)

const (
	maxProbeArgs    = 16
	maxProbeArgLen  = 256
	maxProbePathLen = 4096
	// maxProbeOutput matches what the control plane accepts from a native probe.
	maxProbeOutput = 1 << 20
	// probeTimeoutSeconds is the remote `timeout`, where the target has one.
	probeTimeoutSeconds = 10
)

// ProbeRequest runs one inspection of a binary on a target, in an empty working
// and home directory with a minimal environment. It runs a program, so it stays
// out of ReadRequest and every read-only surface; those surfaces are the security
// boundary. The path check below only keeps a mistaken request to the data root:
// a caller of this command already holds the login.
type ProbeRequest struct {
	Target Spec     `json:"target"`
	Path   string   `json:"path"`
	Args   []string `json:"args"`
}

// BinaryProbeResult is what one inspection printed, stdout and stderr together,
// with its home directory written back as $HOME so the same binary always gives
// the same answer.
type BinaryProbeResult struct {
	Output   []byte `json:"output"`
	ExitCode int    `json:"exitCode"`
}

func validateProbe(r ProbeRequest) error {
	if err := ValidateNamedRead(r.Target); err != nil {
		return err
	}
	clean := filepath.Clean(r.Path)
	root := filepath.Clean(r.Target.DataRoot)
	// env reads a word with = as a variable, and would run the next argument.
	if !filepath.IsAbs(root) || r.Path != clean || !strings.HasPrefix(clean, root+"/") || len(clean) > maxProbePathLen || strings.ContainsAny(clean, "=\x00\r\n") {
		return errors.New("resource: a probe runs only a file under the target's data root")
	}
	if len(r.Args) > maxProbeArgs {
		return errors.New("resource: too many probe arguments")
	}
	for _, a := range r.Args {
		if len(a) > maxProbeArgLen || strings.ContainsRune(a, 0) {
			return errors.New("resource: invalid probe argument")
		}
	}
	return nil
}

// Probe inspects a binary already on a remote target.
func (o Opener) Probe(ctx context.Context, r ProbeRequest) (BinaryProbeResult, error) {
	if err := validateProbe(r); err != nil {
		return BinaryProbeResult{}, err
	}
	access, err := o.Open(r.Target)
	if err != nil {
		return BinaryProbeResult{}, err
	}
	return runBinaryProbe(ctx, access, r)
}

// probeCommand runs the binary with no environment of the login's and in a fresh
// home: an inherited HOME would let a binary that ignored --help write into the
// account's own data directory. The home's path comes first and the exit status
// last, so the output between them can be anything.
func probeCommand(p ProbeRequest) string {
	var argv strings.Builder
	argv.WriteString(remote.ShellQuote(p.Path))
	for _, a := range p.Args {
		argv.WriteString(" " + remote.ShellQuote(a))
	}
	return `t=$(mktemp -d) || exit 1
trap 'rm -rf "$t"' EXIT
limit=$(command -v timeout || true)
echo "$t"
cd "$t"
env -i HOME="$t" PATH=/usr/bin:/bin ${limit:+$limit ` + strconv.Itoa(probeTimeoutSeconds) + `} ` + argv.String() + ` >.out 2>&1
c=$?
head -c ` + strconv.Itoa(maxProbeOutput) + ` .out
printf '\n%s\n' "$c"`
}

// parseProbe splits what probeCommand printed and writes the home back as $HOME.
func parseProbe(stdout string) (BinaryProbeResult, error) {
	home, rest, found := strings.Cut(stdout, "\n")
	trimmed := strings.TrimSuffix(rest, "\n")
	cut := strings.LastIndexByte(trimmed, '\n')
	if !found || home == "" || cut < 0 {
		return BinaryProbeResult{}, errors.New("resource: malformed binary probe output")
	}
	code, err := strconv.Atoi(trimmed[cut+1:])
	if err != nil {
		return BinaryProbeResult{}, errors.New("resource: malformed binary probe status")
	}
	output := bytes.ReplaceAll([]byte(trimmed[:cut]), []byte(home), []byte("$HOME"))
	return BinaryProbeResult{Output: output, ExitCode: code}, nil
}

// runBinaryProbe inspects a binary already on a remote target.
func runBinaryProbe(ctx context.Context, a *Access, p ProbeRequest) (BinaryProbeResult, error) {
	if a.Runner == nil {
		return BinaryProbeResult{}, errors.New("resource: a binary probe runs on a remote target")
	}
	res, err := a.Runner(ctx, "/bin/sh -c "+remote.ShellQuote(probeCommand(p)))
	if err != nil {
		return BinaryProbeResult{}, err
	}
	if res.ExitCode != 0 {
		return BinaryProbeResult{}, fmt.Errorf("resource: binary probe failed (exit %d)", res.ExitCode)
	}
	return parseProbe(res.Stdout)
}
