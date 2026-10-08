package app

import (
	"encoding/json"
	"errors"
	"strings"
)

// PrepareWebTestCase applies the Web publication rules before returning editor
// import/export content. CLI and MCP preparation retain their grammar contract.
func PrepareWebTestCase(in TestCaseInput) (TestCasePrepared, error) {
	var content map[string]any
	if err := json.Unmarshal(in.Content, &content); err != nil {
		return TestCasePrepared{}, errors.New("invalid case declaration")
	}
	if err := validateWebDocumentSecrets("case", content); err != nil {
		return TestCasePrepared{}, err
	}
	for _, raw := range in.Presets {
		var preset map[string]any
		if err := json.Unmarshal(raw, &preset); err != nil {
			return TestCasePrepared{}, errors.New("invalid referenced preset declaration")
		}
		if err := validateWebDocumentSecrets("chain-preset", preset); err != nil {
			return TestCasePrepared{}, err
		}
	}
	return PrepareTestCase(in)
}

// validateWebDocumentSecrets allows a private node-key file reference, but never
// its bytes. Inspect only the parser-owned key reference fields: public hashes,
// addresses and bytecode elsewhere in the declaration are not private keys.
func validateWebDocumentSecrets(kind string, content map[string]any) error {
	if kind != "chain-preset" && kind != "case" {
		return nil
	}
	env := content
	if kind == "case" {
		if inline, ok := content["chainPreset"].(map[string]any); ok {
			env = inline
		}
	}
	topology, _ := env["topology"].(map[string]any)
	nodes, _ := topology["nodes"].([]any)
	for _, value := range nodes {
		n, _ := value.(map[string]any)
		key, _ := n["key"].(string)
		if webNodeKeyMaterial(key) {
			return errors.New("shared node key references must name a private file; inline key material is forbidden")
		}
	}
	return nil
}

// webNodeKeyMaterial matches conservative execution-boundary recognition of key
// material, including malformed long hex. Values are never included in errors.
func webNodeKeyMaterial(ref string) bool {
	h := strings.TrimSpace(ref)
	if strings.HasPrefix(h, "0x") || strings.HasPrefix(h, "0X") {
		h = h[2:]
	}
	if len(h) < 32 {
		return false
	}
	for _, c := range h {
		switch {
		case c >= '0' && c <= '9', c >= 'a' && c <= 'f', c >= 'A' && c <= 'F':
		default:
			return false
		}
	}
	return true
}
