package mcp

import (
	"context"
	"encoding/json"
	"fmt"

	"github.com/0xmhha/chainbench/internal/app"
)

// A matrix failure still carries its machine-readable per-target verdicts.
type structuredToolError struct {
	text  string
	cause error
}

func (e *structuredToolError) Error() string { return e.cause.Error() }
func (e *structuredToolError) Unwrap() error { return e.cause }

func matrixStrings(args map[string]any, key string) ([]string, error) {
	value, exists := args[key]
	if !exists {
		return nil, nil
	}
	raw, err := json.Marshal(value)
	if err != nil {
		return nil, err
	}
	var strings []string
	if err := json.Unmarshal(raw, &strings); err != nil || strings == nil {
		return nil, fmt.Errorf("%s must be an array of strings", key)
	}
	return strings, nil
}

func runMatrixTool(ctx context.Context, args map[string]any) (string, error) {
	if value, supplied := args["baseDir"]; supplied {
		if _, ok := value.(string); !ok {
			return "", fmt.Errorf("baseDir must be a directory string")
		}
	}
	if argBool(args, "attach", false) {
		return "", fmt.Errorf("chainPresets cannot combine with attach")
	}
	if value, exists := args["rpc"]; exists && value != nil {
		return "", fmt.Errorf("chainPresets cannot combine with rpc")
	}
	presets, err := matrixStrings(args, "chainPresets")
	if err != nil {
		return "", err
	}
	specs, err := matrixStrings(args, "specs")
	if err != nil {
		return "", err
	}
	if value, exists := args["spec"]; exists {
		spec, ok := value.(string)
		if !ok || spec == "" {
			return "", fmt.Errorf("spec must be a nonempty JSON string")
		}
		specs = append(specs, spec)
	}
	raws := make([][]byte, 0, len(specs))
	for _, spec := range specs {
		raws = append(raws, []byte(spec))
	}
	in, err := composeInput(args, raws, argString(args, "dataDir", ""))
	if err != nil {
		return "", err
	}
	in.NoNodeControl = argBool(args, "noNodeControl", false)
	out, err := invokeFeature[app.RunMatrixIn, app.RunMatrixOut](ctx, app.Deps{}.WithLogins(argString(args, "serverLogins", "")), "test.run-matrix", app.RunMatrixIn{RunSuiteIn: in, Presets: presets, BaseDir: argString(args, "baseDir", ""), Plan: argBool(args, "plan", false)})
	if err != nil {
		return "", err
	}
	raw, err := json.Marshal(out)
	if err != nil {
		return "", err
	}
	if code := out.ExitCode(argBool(args, "noSkips", false)); code != 0 {
		return string(raw), &structuredToolError{text: string(raw), cause: fmt.Errorf("matrix verdict %d: %d setup error(s), %d failed, %d blocked, %d skipped", code, out.SetupErrors, out.Summary.Fail, out.Summary.Blocked, out.Summary.Skip)}
	}
	return string(raw), nil
}
