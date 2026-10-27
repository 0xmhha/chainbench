package app

import (
	"encoding/json"
	"errors"
	"testing"
)

func TestDocumentValidationNamesTheFieldThatFailed(t *testing.T) {
	for want, in := range map[string]DeploymentDocumentInput{
		"/name":            {Kind: "server-set", Name: " ", ContractVersion: "2", Content: json.RawMessage(`{}`)},
		"/contractVersion": {Kind: "server-set", Name: "x", ContractVersion: "1", Content: json.RawMessage(`{}`)},
		"/presetRefs":      {Kind: "server-set", Name: "x", ContractVersion: "2", Content: json.RawMessage(`{}`), PresetRefs: []DeploymentDocumentRef{{ID: "p", Revision: 1}}},
		"/content":         {Kind: "server-set", Name: "x", ContractVersion: "2", Content: json.RawMessage(`[]`)},
	} {
		err := ValidateDeploymentDocument(in)
		var field *WebFieldError
		if !errors.As(err, &field) || field.Path != want {
			t.Errorf("%s: %v", want, err)
		}
	}
}

// A refusal about one field of the content points at that field, so the
// editor can show the reason beside it rather than above the whole form.
func TestDocumentValidationPointsInsideTheContent(t *testing.T) {
	const ports = `"ports":{"p2p":{"base":31000,"step":10},"rpc":{"base":8600,"step":10}}`
	set := func(body string) DeploymentDocumentInput {
		return DeploymentDocumentInput{Kind: "server-set", Name: "x", ContractVersion: "2", AssetRefs: []string{}, Content: json.RawMessage(body)}
	}
	config := func(change string) DeploymentDocumentInput {
		body := `{"version":1,"dataRoot":"/data","paths":{"binaries":"b","configs":"c","genesis":"g","keystore":"k","keyrings":"r","nodes":"n","runtime":"u","logs":"l"},"control":{"artifactRoot":"out"},"inputs":{"mode":"generated"},"execution":{"chain":"fresh"}` + change + `}`
		return DeploymentDocumentInput{Kind: "workspace-config", Name: "x", ContractVersion: "2", AssetRefs: []string{}, Content: json.RawMessage(body)}
	}
	for want, in := range map[string]DeploymentDocumentInput{
		"/content/pool/hosts":               set(`{"version":2,"pool":{"hosts":[],"slots":1,` + ports + `}}`),
		"/content/pool/hosts/1/addr":        set(`{"version":2,"pool":{"hosts":[{"name":"a","addr":"10.0.0.1"},{"name":"b","addr":"10.0.0.1."}],"slots":1,` + ports + `}}`),
		"/content/pool/hosts/0/port":        set(`{"version":2,"pool":{"hosts":[{"name":"a","addr":"10.0.0.1","port":22}],"slots":1,` + ports + `}}`),
		"/content/pool/ports/rpc/step":      set(`{"version":2,"pool":{"hosts":["10.0.0.1"],"slots":1,"ports":{"p2p":{"base":31000,"step":10},"rpc":{"base":8600,"step":1}}}}`),
		"/content/ssh/password":             set(`{"version":2,"pool":{"hosts":["10.0.0.1"],"slots":1,` + ports + `},"ssh":{"password":"x"}}`),
		"/content/ssh/port":                 set(`{"version":2,"pool":{"hosts":["10.0.0.1"],"slots":1,` + ports + `},"ssh":{"port":70000}}`),
		"/content/dataRoot":                 config(`,"dataRoot":"data"`),
		"/content/binaryAliases/linux~1amd": config(`,"binaryAliases":{"linux/amd":"/abs"}`),
		"/content/execution/chain":          config(`,"execution":{"chain":"maybe"}`),
	} {
		err := ValidateDeploymentDocument(in)
		var field *WebFieldError
		if !errors.As(err, &field) || field.Path != want {
			t.Errorf("%s: %v", want, err)
			if field != nil {
				t.Errorf("  got path %s", field.Path)
			}
		}
	}
}
