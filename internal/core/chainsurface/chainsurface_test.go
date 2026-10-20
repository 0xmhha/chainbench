package chainsurface

import (
	"bufio"
	"os"
	"path/filepath"
	"slices"
	"testing"
	"time"
)

var analysis = filepath.Join("..", "..", "..", "docs", "chain-analysis")

func load(t *testing.T, chain string) Catalog {
	t.Helper()
	c, err := Load(filepath.Join(analysis, chain))
	if err != nil {
		t.Fatalf("load %s: %v", chain, err)
	}
	return c
}

// indexed reads cli-flags.txt, the capture's own list of every flag the help
// pages print, so the parser is held to the capture rather than to itself.
func indexed(t *testing.T, chain string) []string {
	t.Helper()
	f, err := os.Open(filepath.Join(analysis, chain, "cli-flags.txt"))
	if err != nil {
		t.Fatal(err)
	}
	defer func() { _ = f.Close() }()
	var out []string
	sc := bufio.NewScanner(f)
	for sc.Scan() {
		if line := sc.Text(); line != "" {
			out = append(out, line)
		}
	}
	if err = sc.Err(); err != nil {
		t.Fatal(err)
	}
	return out
}

// The index is a line-start grep of the help pages: every flag a page defines
// starts a line, but so does a flag named at the start of a wrapped
// description, and an alias after a comma does not. So every defined spelling
// must be indexed, and every indexed flag must be a defined spelling or alias.
func TestEveryIndexedFlagIsReadFromTheHelpPages(t *testing.T) {
	for _, chain := range []string{"gstable", "gwbft", "gwemix"} {
		c := load(t, chain)
		index := map[string]bool{}
		for _, f := range indexed(t, chain) {
			index[f] = true
		}
		defined := map[string]bool{}
		check := func(fs []HelpFlag) {
			for _, f := range fs {
				if !index[f.Spelling] {
					t.Errorf("%s: %s is defined but not indexed", chain, f.Spelling)
				}
				defined[f.Spelling] = true
				for _, a := range f.Aliases {
					defined[a] = true
				}
			}
		}
		check(c.Global)
		for _, cmd := range c.Commands {
			check(cmd.Flags)
		}
		for f := range index {
			if !defined[f] {
				t.Errorf("%s: indexed %s was not read as a definition", chain, f)
			}
		}
	}
}

func TestProvenanceComesFromTheCaptureHeader(t *testing.T) {
	c := load(t, "gstable")
	if c.Program != "gstable" || c.ChainCommit != "4d17eea" || !c.CapturedAt.Equal(time.Date(2026, 10, 17, 21, 51, 51, 0, time.UTC)) {
		t.Fatalf("provenance %q %q %v", c.Program, c.ChainCommit, c.CapturedAt)
	}
	kinds := map[string]bool{}
	for _, s := range c.Sources {
		if len(s.Digest) != 64 {
			t.Errorf("%s has no sha256 digest", s.Path)
		}
		kinds[s.Kind] = true
	}
	for _, k := range []string{"help", "flag-index", "ast-wiring", "rpc-metrics"} {
		if !kinds[k] {
			t.Errorf("missing %s source", k)
		}
	}
	if w := load(t, "gwemix"); w.ChainCommit != "4538a9f" {
		t.Errorf("gwemix commit %q", w.ChainCommit)
	}
}

func command(c Catalog, path ...string) (HelpCommand, bool) {
	for _, cmd := range c.Commands {
		if slices.Equal(cmd.Path, path) {
			return cmd, true
		}
	}
	return HelpCommand{}, false
}

func flag(fs []HelpFlag, spelling string) (HelpFlag, bool) {
	for _, f := range fs {
		if f.Spelling == spelling {
			return f, true
		}
	}
	return HelpFlag{}, false
}

func TestCommandsKeepPathsPositionalArgumentsAndLocalFlags(t *testing.T) {
	for _, chain := range []string{"gstable", "gwemix"} {
		c := load(t, chain)
		init, ok := command(c, "init")
		if !ok || !slices.Equal(init.Positional, []string{"<genesisPath>"}) || init.Summary == "" {
			t.Fatalf("%s init: %+v", chain, init)
		}
		if f, ok := flag(init.Flags, "--datadir"); !ok || f.Boolean {
			t.Errorf("%s init lacks its local --datadir value flag", chain)
		}
		if _, ok := command(c, "account", "import"); !ok {
			t.Errorf("%s lacks the account import subcommand", chain)
		}
		for _, cmd := range c.Commands {
			if len(cmd.Path) > 1 && cmd.Path[0] == "help" {
				t.Errorf("%s keeps help topic %v as a command", chain, cmd.Path)
			}
		}
	}
	if _, ok := command(load(t, "gwemix"), "wemix", "genesis"); !ok {
		t.Error("gwemix lacks wemix genesis")
	}
}

func TestGlobalFlagsKnowValuesAliasesHiddenAndRefusedFlags(t *testing.T) {
	c := load(t, "gstable")
	if f, ok := flag(c.Global, "--http"); !ok || !f.Boolean {
		t.Error("--http is a global boolean")
	}
	if f, ok := flag(c.Global, "--http.port"); !ok || f.Boolean {
		t.Error("--http.port takes a value")
	}
	if !slices.Contains(c.Hidden, "--vmodule") || !slices.Contains(c.Hidden, "--log.json") {
		t.Errorf("hidden %v", c.Hidden)
	}
	if !slices.Contains(c.NotAccepted, "--docroot") || slices.Contains(c.Hidden, "--docroot") {
		t.Errorf("not accepted %v", c.NotAccepted)
	}
	// gwbft prints --discv4 as an alias of --discovery.v4, so it is not hidden
	// even though the line-start index omits it.
	if w := load(t, "gwbft").Hidden; !slices.Contains(w, "--vmodule") || slices.Contains(w, "--discv4") {
		t.Errorf("gwbft hidden %v", w)
	}
	init, _ := command(c, "init")
	if f, ok := flag(init.Flags, "--header"); !ok || !slices.Contains(f.Aliases, "-H") {
		t.Errorf("--header alias: %+v", f)
	}
}

func TestLiveHelpIsReadWithTheSameRules(t *testing.T) {
	help := []byte("GLOBAL OPTIONS:\n   NETWORKING\n\n    \n    --port value                        (default: 30303)\n          Network listening port\n    \n    --nodiscover                        (default: false)\n          Disables the peer discovery\n")
	flags := ParseHelpFlags(help)
	if p, ok := flag(flags, "--port"); !ok || p.Boolean {
		t.Fatalf("%+v", flags)
	}
	if n, ok := flag(flags, "--nodiscover"); !ok || !n.Boolean {
		t.Fatalf("%+v", flags)
	}
	// urfave/cli v1, as go-wemix prints it, indents definitions by two spaces.
	v1 := ParseHelpFlags([]byte("ETHEREUM OPTIONS:\n  --datadir value                     Data directory\n  --nousb                             Disables USB\n"))
	if d, ok := flag(v1, "--datadir"); !ok || d.Boolean {
		t.Fatalf("%+v", v1)
	}
}

func TestMissingDocumentIsAnError(t *testing.T) {
	dir := t.TempDir()
	if _, err := Load(dir); err == nil {
		t.Fatal("an empty directory loaded")
	}
}
