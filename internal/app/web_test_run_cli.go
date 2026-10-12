package app

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"path/filepath"
	"slices"

	"github.com/0xmhha/chainbench/internal/core/filestore"
	"github.com/0xmhha/chainbench/internal/dsl"
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
// that is what the command reads; serverSet, when set, is the job's own server
// set carrying the bound credentials and replaces in.Server.SetPath.
//
// A run whose tests failed exits non-zero with a complete document: that is a
// verdict, returned without an error. A run that never produced a session
// returns the reason the CLI gave.
func runSuiteThroughCLI(ctx context.Context, cli ChainCLI, in RunSuiteIn, dir, serverSet string) (RunSuiteOut, error) {
	if cli == nil {
		return RunSuiteOut{}, errNoChainCLI
	}
	args := []string{"run"}
	for i, raw := range in.SpecContent {
		name := filepath.Join(dir, "cases", fmt.Sprintf("%03d.json", i+1))
		if err := (filestore.Local{}).Write(ctx, name, raw, 0o600); err != nil {
			return RunSuiteOut{}, err
		}
		args = append(args, name)
	}
	if serverSet == "" {
		serverSet = in.Server.SetPath
	}
	// --suite: the reviewed cases share one network and one session, as the
	// plan the operator approved composed them.
	args = append(args, "--json", "--suite", "--workspace-dir", in.DataDir)
	for _, f := range []struct{ flag, value string }{
		{"--chain", in.Chain}, {"--binary", in.Binary}, {"--keys", in.KeysDir}, {"--artifact-root", in.ArtifactRoot},
		{"--workspace-config", in.WorkspaceConfigPath}, {"--server-set", serverSet}, {"--server", in.Server.Name},
	} {
		if f.value != "" {
			args = append(args, f.flag, f.value)
		}
	}
	names := make([]string, 0, len(in.BinaryOverrides))
	for name := range in.BinaryOverrides {
		if name != dsl.BinaryDefault {
			names = append(names, name)
		}
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
