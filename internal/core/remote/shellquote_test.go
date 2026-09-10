package remote_test

import (
	"os/exec"
	"strings"
	"testing"

	"github.com/0xmhha/chainbench/internal/core/remote"
)

// TestShellQuote_RoundTripsThroughARealShell is the test the four duplicated
// copies never had. Quoting is where a path becomes shell syntax, so the
// property that matters is not the output's spelling but that a real shell
// hands the byte sequence back unchanged: whatever went in comes out of
// `printf %s` verbatim, with no word splitting, expansion, or command
// substitution along the way.
func TestShellQuote_RoundTripsThroughARealShell(t *testing.T) {
	cases := map[string]string{
		"plain":               "/data/chainbench/bin/gstable",
		"space":               "/data/my chain/node 1",
		"single quote":        "it's-a-path",
		"double quote":        `say "hi"`,
		"dollar":              "$HOME/x",
		"backtick":            "a`whoami`b",
		"semicolon":           "a;rm -rf /",
		"pipe":                "a|b",
		"ampersand":           "a&b",
		"newline":             "line1\nline2",
		"tab":                 "a\tb",
		"glob":                "/data/*",
		"backslash":           `a\b`,
		"subshell":            "$(id)",
		"only a single quote": "'",
		"two single quotes":   "''",
		"quote then command":  `';id;'`,
		"empty":               "",
	}
	for name, in := range cases {
		t.Run(name, func(t *testing.T) {
			// printf %s emits its argument with no trailing newline, so any
			// difference is the quoting's fault and not the shell's.
			out, err := exec.Command("/bin/sh", "-c", "printf %s "+remote.ShellQuote(in)).Output()
			if err != nil {
				t.Fatalf("shell rejected the quoted form of %q: %v", in, err)
			}
			if string(out) != in {
				t.Errorf("round trip changed the value\n in: %q\nout: %q", in, string(out))
			}
		})
	}
}

// TestShellQuote_AlwaysSingleQuotes pins the form, because a caller that
// concatenates (…+ShellQuote(p)+…) depends on the result being one shell word
// that needs no separator of its own.
func TestShellQuote_AlwaysSingleQuotes(t *testing.T) {
	for _, in := range []string{"", "x", "a b", "'"} {
		got := remote.ShellQuote(in)
		if !strings.HasPrefix(got, "'") || !strings.HasSuffix(got, "'") {
			t.Errorf("ShellQuote(%q) = %q, want it wrapped in single quotes", in, got)
		}
	}
}

// TestShellQuote_UsedByReadFileCommand keeps the package's own caller honest:
// a path with a quote in it must not be able to close ReadFileCommand's
// redirect and append a second command.
func TestShellQuote_UsedByReadFileCommand(t *testing.T) {
	cmd := remote.ReadFileCommand("/tmp/a';id;'b")
	out, err := exec.Command("/bin/sh", "-c", "printf %s "+remote.ShellQuote(cmd)).Output()
	if err != nil {
		t.Fatalf("quoting the command itself failed: %v", err)
	}
	if strings.Count(string(out), ";id;") != 1 {
		t.Errorf("expected the injected text to survive as literal data, got %q", out)
	}
}
