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
		"/content":         {Kind: "server-set", Name: "x", ContractVersion: "2", Content: json.RawMessage(`{"version":2,"pool":{"hosts":[],"slots":2}}`)},
	} {
		err := ValidateDeploymentDocument(in)
		var field *WebFieldError
		if !errors.As(err, &field) || field.Path != want {
			t.Errorf("%s: %v", want, err)
		}
	}
}
