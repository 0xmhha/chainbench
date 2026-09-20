package inspector

import (
	"context"
	"fmt"
	"os/exec"
	"strings"

	"github.com/0xmhha/chainbench/internal/core/process"
)

// OnPath reports where a bare command name resolves on the target's PATH, and
// whether it resolves at all.
//
// A name without a path separator is not a file at a known place: it is a
// request for whatever the target's PATH finds. [Paths] cannot answer that,
// because it asks the file store whether a path exists and a bare name is not
// one — it stats it relative to the working directory and reports a binary on
// PATH as missing. That is the wrong answer twice over, since the launch will
// then find it.
//
// run is nil for a local target, which resolves through this process's own
// PATH. A remote target resolves through its shell, so the answer is the
// target's PATH rather than ours.
func OnPath(ctx context.Context, run process.Runner, name string) (string, bool, error) {
	if strings.ContainsRune(name, '/') {
		return "", false, fmt.Errorf("inspector: %q is a path, not a command name", name)
	}
	if run == nil {
		path, err := exec.LookPath(name)
		if err != nil {
			return "", false, nil // not on PATH is an answer, not a failure
		}
		return path, true, nil
	}
	res, err := run(ctx, "command -v "+shellQuote(name))
	if err != nil {
		return "", false, fmt.Errorf("inspector: look up %q on the target: %w", name, err)
	}
	path := strings.TrimSpace(res.Stdout)
	if res.ExitCode != 0 || path == "" {
		return "", false, nil
	}
	return path, true, nil
}

// shellQuote wraps s for a POSIX shell. A command name should never need it,
// which is exactly why it is here: a name that does is not one, and quoting
// keeps it from becoming a second command.
func shellQuote(s string) string {
	return "'" + strings.ReplaceAll(s, "'", `'\''`) + "'"
}
