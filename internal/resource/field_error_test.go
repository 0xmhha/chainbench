package resource

import (
	"errors"
	"reflect"
	"strings"
	"testing"
)

// A refusal names the declared field it is about, so an editor can show the
// reason beside that field. The message stays the sentence the CLI prints.
func TestWorkspaceConfigRefusalsNameTheirField(t *testing.T) {
	for _, c := range []struct {
		from, to string
		path     []string
		message  string
	}{
		{"version: 1", "version: 7", []string{"version"}, "version 7 is not supported"},
		{"dataRoot: /data", "dataRoot: data", []string{"dataRoot"}, "must be an absolute target path"},
		{"keyrings: keys", "keyrings: ../keys", []string{"paths", "keyrings"}, "must not contain a .. path segment"},
		{"gwemix: linux-amd64/gwemix", "gwemix: /abs/gwemix", []string{"binaryAliases", "gwemix"}, "must be relative"},
		{"artifactRoot: ~/.chainbench", "artifactRoot: ''", []string{"control", "artifactRoot"}, "control.artifactRoot is required"},
		{"mode: generated", "mode: guessed", []string{"inputs", "mode"}, `inputs.mode "guessed" is unknown`},
		{"mode: generated", "mode: existing", []string{"inputs", "name"}, "needs inputs.name"},
		{"chain: fresh", "chain: maybe", []string{"execution", "chain"}, `execution.chain "maybe" is unknown`},
		{"chain: fresh", "chain: fresh\nlimits:\n  minFreeDisk: lots", []string{"limits", "minFreeDisk"}, "limits.minFreeDisk"},
	} {
		_, err := ParseWorkspaceConfig([]byte(strings.Replace(validConfig, c.from, c.to, 1)))
		assertFieldError(t, c.to, err, c.path, c.message)
	}
}

func TestServerSetRefusalsNameTheirField(t *testing.T) {
	const valid = `
version: 2
pool:
  hosts:
    - {name: a, addr: 10.0.0.1}
    - {name: b, addr: 10.0.0.2}
  slots: 1
  ports:
    p2p: {base: 31000, step: 10}
    rpc: {base: 8600, step: 10}
`
	for _, c := range []struct {
		from, to string
		path     []string
		message  string
	}{
		{"version: 2", "version: 3", []string{"version"}, "has version 3"},
		{"{name: b, addr: 10.0.0.2}", "{name: b, addr: ''}", []string{"pool", "hosts", "1", "addr"}, "has no addr"},
		{"slots: 1", "slots: -1", []string{"pool", "slots"}, "want >= 1"},
		{"{name: b, addr: 10.0.0.2}", "{name: a, addr: 10.0.0.2}", []string{"pool", "hosts", "1", "name"}, `duplicate server name "a"`},
		{"rpc: {base: 8600, step: 10}", "rpc: {base: 8600, step: 1}", []string{"pool", "ports", "rpc", "step"}, "rpcStep is 1"},
		{"version: 2", "version: 2\nssh: {port: 70000}", []string{"ssh", "port"}, "ssh port must be"},
	} {
		_, err := ParseSet([]byte(strings.Replace(valid, c.from, c.to, 1)))
		assertFieldError(t, c.to, err, c.path, c.message)
	}
}

func assertFieldError(t *testing.T, input string, err error, path []string, message string) {
	t.Helper()
	var field *FieldError
	if !errors.As(err, &field) {
		t.Errorf("%s: %v, want a field error at %v", input, err, path)
		return
	}
	if !reflect.DeepEqual(field.Path, path) {
		t.Errorf("%s: path %v, want %v", input, field.Path, path)
	}
	if !strings.Contains(err.Error(), message) {
		t.Errorf("%s: message %q, want it to keep %q", input, err.Error(), message)
	}
}
