package app

import (
	"os"
	"strings"
	"testing"
)

const webJobArgumentsDoc = "../../docs/dev/web-job-arguments.md"

// renderWebJobArguments is the table the document keeps, one section per
// operation: each argument it reads and what that becomes on the command line.
func renderWebJobArguments() string {
	var b strings.Builder
	for _, op := range JobsContract().Operations {
		b.WriteString("### `" + op.ID + "` " + op.Label + "\n\n")
		b.WriteString("명령: `" + op.Command + "`\n\n")
		b.WriteString("| 인자 | 이름 | 플래그 | 명령줄에서 |\n|---|---|---|---|\n")
		for _, a := range op.Arguments {
			flag := "없음"
			if a.Flag != "" {
				flag = "`" + a.Flag + "`"
			}
			b.WriteString("| `" + a.Name + "` | " + a.Title + " | " + flag + " | " + a.CLI + " |\n")
		}
		b.WriteString("\n")
	}
	return strings.TrimSuffix(b.String(), "\n")
}

// TestWebJobArgumentsDocumentMatchesTheContract keeps the document's table
// equal to the jobs contract. CHAINBENCH_UPDATE_DOCS=1 rewrites it.
func TestWebJobArgumentsDocumentMatchesTheContract(t *testing.T) {
	const begin, end = "<!-- generated from app.JobsContract: begin -->\n", "<!-- generated from app.JobsContract: end -->"
	raw, err := os.ReadFile(webJobArgumentsDoc)
	if err != nil {
		t.Fatal(err)
	}
	doc := string(raw)
	i, j := strings.Index(doc, begin), strings.Index(doc, end)
	if i < 0 || j < i {
		t.Fatalf("%s has no generated table markers", webJobArgumentsDoc)
	}
	want := renderWebJobArguments() + "\n"
	if doc[i+len(begin):j] == want {
		return
	}
	if os.Getenv("CHAINBENCH_UPDATE_DOCS") == "1" {
		if err := os.WriteFile(webJobArgumentsDoc, []byte(doc[:i+len(begin)]+want+doc[j:]), 0o644); err != nil {
			t.Fatal(err)
		}
		return
	}
	t.Fatalf("%s differs from the jobs contract; run with CHAINBENCH_UPDATE_DOCS=1 to rewrite it", webJobArgumentsDoc)
}
