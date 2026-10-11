// Package chainsurface reads the captured command surface of a chain binary.
package chainsurface

import (
	"bufio"
	"bytes"
	"crypto/sha256"
	"encoding/hex"
	"errors"
	"fmt"
	"os"
	"path/filepath"
	"regexp"
	"slices"
	"strings"
	"time"
)

// Source is one captured document and its content digest.
type Source struct {
	Kind   string `json:"kind"`
	Path   string `json:"path"`
	Digest string `json:"digest"`
}

// HelpFlag is one flag a help page defines. Boolean flags take no value.
type HelpFlag struct {
	Spelling string   `json:"spelling"`
	Aliases  []string `json:"aliases"`
	Boolean  bool     `json:"boolean"`
}

// HelpCommand is one command or subcommand of the binary, with the positional
// arguments its usage line names and the flags its own help page lists.
type HelpCommand struct {
	Path       []string   `json:"path"`
	Summary    string     `json:"summary"`
	Positional []string   `json:"positional"`
	Flags      []HelpFlag `json:"flags"`
}

// Catalog is the documented surface of one binary at one chain commit. Global
// flags are those the default node command accepts. Hidden flags are accepted
// by the binary but absent from its help; NotAccepted flags are defined in the
// source but registered on no command.
type Catalog struct {
	Program     string        `json:"program"`
	ChainCommit string        `json:"chainCommit"`
	CapturedAt  time.Time     `json:"capturedAt"`
	Sources     []Source      `json:"sources"`
	Global      []HelpFlag    `json:"global"`
	Commands    []HelpCommand `json:"commands"`
	Hidden      []string      `json:"hidden"`
	NotAccepted []string      `json:"notAccepted"`
}

var documents = []struct{ kind, name string }{
	{"help", "cli-surface.txt"},
	{"flag-index", "cli-flags.txt"},
	{"ast-wiring", "cli-graph.md"},
	{"rpc-metrics", "rpc-metrics-graph.md"},
}

var (
	headerProgram  = regexp.MustCompile(`(?m)^# CLI surface of (\S+),`)
	headerCommit   = regexp.MustCompile(`(?m)^# chain commit: ([0-9a-f]{7,40})\b`)
	headerCaptured = regexp.MustCompile(`(?m)^# captured:\s+(\S+)`)
	// A definition line: indentation, the flag, an optional value placeholder
	// (urfave prints "value" or a named one such as "loadScript"), optional
	// aliases, then at least two spaces before the next column. A flag named
	// inside another flag's description has one space after it and is not a
	// definition. The capture script indents every page by four more spaces
	// than the binary prints, so a live page is read with less indentation.
	definition     = regexp.MustCompile(`^\s{4,}(--[A-Za-z0-9._-]+)( [A-Za-z]\w*)?((?:,\s+-{1,2}[A-Za-z0-9._-]+(?: [A-Za-z]\w*)?)*)(?:\s{2,}\S|\s+\(|\s*$)`)
	liveDefinition = regexp.MustCompile(`^\s{2,}(--[A-Za-z0-9._-]+)( [A-Za-z]\w*)?((?:,\s+-{1,2}[A-Za-z0-9._-]+(?: [A-Za-z]\w*)?)*)(?:\s{2,}\S|\s+\(|\s*$)`)
	alias          = regexp.MustCompile(`-{1,2}[A-Za-z0-9._-]+`)
	documented     = regexp.MustCompile("`(--[A-Za-z0-9._-]+)`")
	listing        = regexp.MustCompile(`^\s+([a-z][a-z0-9-]*)(?:,\s*[a-z]+)?\s{2,}(\S.*)$`)
	placeholder    = regexp.MustCompile(`<[^>]+>|\[arguments\.\.\.\]`)
)

// Load reads the four captured documents of one chain directory.
func Load(dir string) (Catalog, error) {
	var c Catalog
	raw := map[string][]byte{}
	for _, d := range documents {
		path := filepath.Join(dir, d.name)
		b, err := os.ReadFile(path)
		if err != nil {
			return c, fmt.Errorf("chain surface: %w", err)
		}
		sum := sha256.Sum256(b)
		raw[d.kind] = b
		c.Sources = append(c.Sources, Source{Kind: d.kind, Path: path, Digest: hex.EncodeToString(sum[:])})
	}
	surface := raw["help"]
	program, commit, captured := headerProgram.FindSubmatch(surface), headerCommit.FindSubmatch(surface), headerCaptured.FindSubmatch(surface)
	if program == nil || commit == nil || captured == nil {
		return c, errors.New("chain surface: capture header lacks program, chain commit or capture time")
	}
	at, err := time.Parse(time.RFC3339, string(captured[1]))
	if err != nil {
		return c, fmt.Errorf("chain surface: capture time: %w", err)
	}
	c.Program, c.ChainCommit, c.CapturedAt = string(program[1]), string(commit[1]), at.UTC()
	sections := splitSections(surface)
	summaries := map[string]string{}
	for i, s := range sections {
		if s.name == "top level" {
			c.Global = flagsOf(s.body)
		}
		for _, line := range s.body {
			if m := listing.FindStringSubmatch(line); m != nil && !strings.HasPrefix(strings.TrimSpace(line), "-") {
				key := m[1]
				if s.name != "top level" {
					key = s.name + " " + m[1]
				}
				if _, ok := summaries[key]; !ok {
					summaries[key] = strings.TrimSpace(m[2])
				}
			}
		}
		// The first section is the version output, not the version command.
		if i == 0 || s.name == "top level" || strings.HasPrefix(s.name, "help ") {
			continue
		}
		c.Commands = append(c.Commands, HelpCommand{Path: strings.Fields(s.name), Positional: positional(s.name, s.body), Flags: flagsOf(s.body)})
	}
	for i := range c.Commands {
		name := strings.Join(c.Commands[i].Path, " ")
		c.Commands[i].Summary = nameSummary(c.Program, name, sections)
		if c.Commands[i].Summary == "" {
			c.Commands[i].Summary = summaries[name]
		}
	}
	// A flag is printed when a help page defines it, under its own spelling or
	// as an alias. The flag index is a line-start grep of the same pages, so it
	// misses aliases and also picks up flags named at the start of a
	// description line; the definitions are read instead.
	printed := map[string]bool{}
	mark := func(fs []HelpFlag) {
		for _, f := range fs {
			printed[f.Spelling] = true
			for _, a := range f.Aliases {
				printed[a] = true
			}
		}
	}
	mark(c.Global)
	for _, cmd := range c.Commands {
		mark(cmd.Flags)
	}
	graph := string(raw["ast-wiring"])
	c.NotAccepted = notAccepted(graph)
	for _, m := range documented.FindAllStringSubmatch(graph, -1) {
		f := m[1]
		if len(f) > 3 && !printed[f] && !slices.Contains(c.NotAccepted, f) && !slices.Contains(c.Hidden, f) {
			c.Hidden = append(c.Hidden, f)
		}
	}
	slices.Sort(c.Hidden)
	return c, nil
}

// ParseHelpFlags reads the flag definitions of a live help page with the rules
// used for the captured pages.
func ParseHelpFlags(help []byte) []HelpFlag {
	return flagsMatching(liveDefinition, strings.Split(string(help), "\n"))
}

type section struct {
	name string
	body []string
}

func splitSections(b []byte) []section {
	var out []section
	sc := bufio.NewScanner(bytes.NewReader(b))
	sc.Buffer(make([]byte, 0, 64*1024), 1024*1024)
	for sc.Scan() {
		line := sc.Text()
		if name, ok := strings.CutPrefix(line, "## "); ok {
			out = append(out, section{name: strings.TrimSpace(name)})
			continue
		}
		if len(out) > 0 {
			out[len(out)-1].body = append(out[len(out)-1].body, line)
		}
	}
	return out
}

func flagsOf(lines []string) []HelpFlag { return flagsMatching(definition, lines) }

func flagsMatching(definition *regexp.Regexp, lines []string) []HelpFlag {
	out := []HelpFlag{}
	seen := map[string]bool{}
	for _, line := range lines {
		m := definition.FindStringSubmatch(line)
		if m == nil || seen[m[1]] {
			continue
		}
		seen[m[1]] = true
		aliases := alias.FindAllString(m[3], -1)
		if aliases == nil {
			aliases = []string{}
		}
		out = append(out, HelpFlag{Spelling: m[1], Aliases: aliases, Boolean: m[2] == ""})
	}
	return out
}

// positional reads the usage line: urfave/cli v2 prints it under USAGE:, v1
// as the section's first line. Both start with the command path.
func positional(name string, body []string) []string {
	usage := ""
	for i, line := range body {
		if strings.TrimSpace(line) == "USAGE:" && i+1 < len(body) {
			usage = strings.TrimSpace(body[i+1])
			break
		}
	}
	if usage == "" {
		for _, line := range body {
			if t := strings.TrimSpace(line); t != "" {
				if strings.HasPrefix(t, name) {
					usage = t
				}
				break
			}
		}
	}
	out := placeholder.FindAllString(usage, -1)
	if out == nil {
		out = []string{}
	}
	return out
}

func nameSummary(program, name string, sections []section) string {
	for _, s := range sections {
		if s.name != name {
			continue
		}
		for i, line := range s.body {
			if strings.TrimSpace(line) == "NAME:" && i+1 < len(s.body) {
				if _, after, ok := strings.Cut(s.body[i+1], program+" "+name+" - "); ok {
					return strings.TrimSpace(after)
				}
			}
		}
	}
	return ""
}

func notAccepted(graph string) []string {
	out := []string{}
	in := false
	for _, line := range strings.Split(graph, "\n") {
		switch {
		case strings.TrimSpace(line) == "```not-accepted":
			in = true
		case in && strings.TrimSpace(line) == "```":
			in = false
		case in:
			if f := strings.Fields(line); len(f) > 0 && strings.HasPrefix(f[0], "--") {
				out = append(out, f[0])
			}
		}
	}
	slices.Sort(out)
	return out
}
