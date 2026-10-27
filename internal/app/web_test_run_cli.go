package app

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"path/filepath"
	"slices"
	"strconv"

	"github.com/0xmhha/chainbench/internal/core/filestore"
)

// cliRunReport is the document `chainbench run --json` prints, on success and
// on a run that could not compose alike.
type cliRunReport struct {
	Session    string   `json:"session"`
	Error      string   `json:"error"`
	FailedAt   string   `json:"failedAt"`
	SetupSteps []string `json:"setupSteps"`
	RunSummary
}

// runSuiteThroughCLI runs the reviewed suite with `chainbench run` instead of
// the engine in this process. The cases are written into dir as files, since
// that is what the command reads; logins, when set, is the job's own file of
// bound credentials, named apart from the server set the network records.
//
// A run whose tests failed exits non-zero with a complete document: that is a
// verdict, returned without an error. A run that never produced a session
// returns the reason the CLI gave.
func runSuiteThroughCLI(ctx context.Context, cli ChainCLI, in RunSuiteIn, dir, logins string, flags ...string) (RunSuiteOut, error) {
	if cli == nil {
		return RunSuiteOut{}, errNoChainCLI
	}
	args, err := suiteCLIArgs(ctx, in, dir, logins, flags...)
	if err != nil {
		return RunSuiteOut{}, err
	}
	return runCLIDocument(ctx, cli, args)
}

func suiteCLIArgs(ctx context.Context, in RunSuiteIn, dir, logins string, flags ...string) ([]string, error) {
	args, err := writeCLICases(ctx, in.SpecContent, dir)
	if err != nil {
		return nil, err
	}
	// --suite: the reviewed cases share one network and one session, as the
	// plan the operator approved composed them.
	args = append(args, "--json", "--suite", "--workspace-dir", in.DataDir)
	for _, f := range []struct{ flag, value string }{
		{"--chain", in.Chain}, {"--binary", in.Binary}, {"--keys", in.KeysDir}, {"--artifact-root", in.ArtifactRoot},
		{"--workspace-config", in.WorkspaceConfigPath}, {"--server-set", in.Server.SetPath}, {"--server", in.Server.Name},
		{"--server-logins", logins},
	} {
		if f.value != "" {
			args = append(args, f.flag, f.value)
		}
	}
	names := make([]string, 0, len(in.BinaryOverrides))
	for name := range in.BinaryOverrides {
		names = append(names, name)
	}
	slices.Sort(names)
	for _, name := range names {
		args = append(args, "--binary-for", name+"="+in.BinaryOverrides[name])
	}
	if in.ReadOnlyKeys {
		args = append(args, "--read-only-keys")
	}
	if in.KeepUp {
		args = append(args, "--keep-up")
	}
	if in.Docker {
		args = append(args, "--docker")
	}
	return append(args, flags...), nil
}

// webRunFlags are the test.run arguments the contract passes to chainbench
// run as they are.
func webRunFlags(args webChainArguments) []string {
	var flags []string
	if args.NoSkips {
		flags = append(flags, "--no-skips")
	}
	if args.WaitBlocks != 0 {
		flags = append(flags, "--wait-blocks", strconv.FormatUint(args.WaitBlocks, 10))
	}
	if args.NodeMonitorTimeoutSeconds != 0 {
		flags = append(flags, "--node-monitor-timeout", strconv.Itoa(args.NodeMonitorTimeoutSeconds)+"s")
	}
	return flags
}

// attachThroughCLI runs the reviewed cases against the network recorded at
// control with `chainbench run --attach`, for reading only: no step may act on
// a node and the accepted key set is not written to. logins supplies the job's
// own logins for the recorded hosts.
func attachThroughCLI(ctx context.Context, cli ChainCLI, cases [][]byte, control, artifactRoot, dir, logins string) (RunSuiteOut, error) {
	if cli == nil {
		return RunSuiteOut{}, errNoChainCLI
	}
	args, err := writeCLICases(ctx, cases, dir)
	if err != nil {
		return RunSuiteOut{}, err
	}
	args = append(args, "--json", "--attach", "--workspace-dir", control, "--no-node-control", "--read-only-keys", "--artifact-root", artifactRoot)
	if logins != "" {
		args = append(args, "--server-logins", logins)
	}
	return runCLIDocument(ctx, cli, args)
}

// writeCLICases writes each case into dir as the file the CLI reads, and
// returns `run` followed by their paths.
func writeCLICases(ctx context.Context, cases [][]byte, dir string) ([]string, error) {
	args := []string{"run"}
	for i, raw := range cases {
		name := filepath.Join(dir, "cases", fmt.Sprintf("%03d.json", i+1))
		if err := (filestore.Local{}).Write(ctx, name, raw, 0o600); err != nil {
			return nil, err
		}
		args = append(args, name)
	}
	return args, nil
}

// runCLIDocument runs `chainbench run --json` and reads its document. A run
// whose tests failed exits non-zero with a complete document: that is a
// verdict, returned without an error. A run that never produced a session
// returns the reason the CLI gave.
func runCLIDocument(ctx context.Context, cli ChainCLI, args []string) (RunSuiteOut, error) {
	stdout, runErr := cli.Run(ctx, args...)
	var report cliRunReport
	if err := json.Unmarshal(stdout, &report); err != nil {
		if runErr != nil {
			return RunSuiteOut{}, runErr
		}
		return RunSuiteOut{}, fmt.Errorf("chainbench run: unreadable document: %w", err)
	}
	out := RunSuiteOut{SessionRoot: report.Session, SetupSteps: report.SetupSteps, Summary: report.RunSummary}
	if report.Error != "" {
		return out, errors.New(report.Error)
	}
	if report.Session == "" && runErr != nil {
		return out, runErr
	}
	return out, nil
}
