package arch

import (
	"fmt"
	"os"
	"path/filepath"
	"regexp"
	"strings"
)

// layersDoc is the document that owns the architecture rules, relative to this
// package.
const layersDoc = "../../docs/dev/architecture/layers.md"

// modulePrefix is the import prefix of the packages under test, and moduleRoot
// is where the toolchain has to be invoked from to see all of them.
const (
	modulePrefix = "github.com/0xmhha/chainbench/internal/"
	moduleRoot   = "../.."
)

// section returns the part of the document under heading, up to the next
// heading of the same level.
func section(doc, heading string) (string, error) {
	i := strings.Index(doc, heading)
	if i < 0 {
		return "", fmt.Errorf("arch: %s has no %q section", layersDoc, heading)
	}
	rest := doc[i+len(heading):]
	level := strings.Repeat("#", strings.Count(strings.Fields(heading)[0], "#"))
	if j := strings.Index(rest, "\n"+level+" "); j >= 0 {
		rest = rest[:j]
	}
	return rest, nil
}

// readLayersDoc reads the architecture document.
func readLayersDoc() (string, error) {
	b, err := os.ReadFile(filepath.Clean(layersDoc))
	if err != nil {
		return "", fmt.Errorf("arch: read %s: %w", layersDoc, err)
	}
	return string(b), nil
}

var (
	layerHeading = regexp.MustCompile(`^### L(\d)`)
	backticked   = regexp.MustCompile("`([a-z][a-zA-Z0-9/_-]*)`")
	ruleRow      = regexp.MustCompile(`^\|`)
)

// firstCell returns a markdown row's first cell, or "" when the row is a header
// or a separator.
func firstCell(line string) string {
	if !ruleRow.MatchString(line) {
		return ""
	}
	parts := strings.Split(line, "|")
	if len(parts) < 2 {
		return ""
	}
	cell := parts[1]
	if strings.Contains(cell, "패키지") || strings.Trim(cell, "-: ") == "" {
		return ""
	}
	return cell
}

// packagesIn returns the package paths named in a table cell. Only the first
// cell of a row is read, so a description mentioning a type in backticks is not
// mistaken for a package.
func packagesIn(cell string) []string {
	var out []string
	for _, m := range backticked.FindAllStringSubmatch(cell, -1) {
		out = append(out, m[1])
	}
	return out
}
