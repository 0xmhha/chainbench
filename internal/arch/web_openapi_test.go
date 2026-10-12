package arch

import (
	"os"
	"testing"

	"go.yaml.in/yaml/v3"
)

// TestWebOpenAPIParses keeps the Web HTTP contract machine-readable; prose
// edits to descriptions must not break the YAML document.
func TestWebOpenAPIParses(t *testing.T) {
	raw, err := os.ReadFile("../../docs/dev/web-ui.openapi.yaml")
	if err != nil {
		t.Fatal(err)
	}
	var doc struct {
		OpenAPI string         `yaml:"openapi"`
		Paths   map[string]any `yaml:"paths"`
	}
	if err = yaml.Unmarshal(raw, &doc); err != nil {
		t.Fatalf("web-ui.openapi.yaml does not parse: %v", err)
	}
	for _, path := range []string{"/api/v1/events", "/api/v1/snapshot", "/api/v1/networks/{networkId}/metrics", "/api/v1/nodes/{nodeId}/logs"} {
		if doc.Paths[path] == nil {
			t.Errorf("contract lost %s", path)
		}
	}
}
