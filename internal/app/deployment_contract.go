package app

import (
	"reflect"
	"strings"

	"github.com/0xmhha/chainbench/internal/resource"
)

// DeploymentField describes engine-owned YAML fields for structured Web editing.
type DeploymentField struct {
	Type       string                     `json:"type"`
	Minimum    *int                       `json:"minimum,omitempty"`
	Maximum    *int                       `json:"maximum,omitempty"`
	Properties map[string]DeploymentField `json:"properties,omitempty"`
	Required   []string                   `json:"required,omitempty"`
	Items      *DeploymentField           `json:"items,omitempty"`
	Additional *DeploymentField           `json:"additionalProperties,omitempty"`
	Enum       []any                      `json:"enum,omitempty"`
}

// DeploymentContract derives field names and types from resource, then narrows
// shared SSH fields to public connection settings. Validation still runs in resource.
func DeploymentContract() map[string]DeploymentField {
	set := deploymentField(reflect.TypeFor[resource.Set]())
	delete(set.Properties, "dataRoot")
	ssh := set.Properties["ssh"]
	for _, name := range []string{"password", "password_file", "key_file", "key_passphrase_file"} {
		delete(ssh.Properties, name)
	}
	set.Properties["ssh"] = ssh
	version := set.Properties["version"]
	version.Enum = []any{resource.SupportedVersion}
	set.Properties["version"] = version
	config := deploymentField(reflect.TypeFor[resource.WorkspaceConfig]())
	version = config.Properties["version"]
	version.Enum = []any{resource.SupportedWorkspaceVersion}
	config.Properties["version"] = version
	inputs := config.Properties["inputs"]
	mode := inputs.Properties["mode"]
	mode.Enum = []any{resource.InputGenerated, resource.InputExisting}
	inputs.Properties["mode"] = mode
	config.Properties["inputs"] = inputs
	execution := config.Properties["execution"]
	chain := execution.Properties["chain"]
	chain.Enum = []any{resource.ChainFresh, resource.ChainReuseIfMatching, resource.ChainAttach}
	execution.Properties["chain"] = chain
	config.Properties["execution"] = execution
	return map[string]DeploymentField{"server-set": set, "workspace-config": config}
}
func deploymentField(t reflect.Type) DeploymentField {
	if t.Kind() == reflect.Pointer {
		return deploymentField(t.Elem())
	}
	switch t.Kind() {
	case reflect.Struct:
		out := DeploymentField{Type: "object", Properties: map[string]DeploymentField{}}
		for i := 0; i < t.NumField(); i++ {
			f := t.Field(i)
			tag := f.Tag.Get("yaml")
			name := strings.Split(tag, ",")[0]
			if f.PkgPath != "" || name == "" || name == "-" {
				continue
			}
			field := deploymentField(f.Type)
			if (t == reflect.TypeFor[resource.BandSpec]() && (name == "base" || name == "step")) || (t == reflect.TypeFor[resource.SSH]() && name == "port") {
				minimum, maximum := 0, resource.MaxListenPort
				field.Minimum = &minimum
				if name != "step" {
					field.Maximum = &maximum
				}
			}
			out.Properties[name] = field
			if !strings.Contains(tag, "omitempty") {
				out.Required = append(out.Required, name)
			}
		}
		return out
	case reflect.Map:
		field := deploymentField(t.Elem())
		return DeploymentField{Type: "object", Additional: &field}
	case reflect.Slice:
		field := deploymentField(t.Elem())
		return DeploymentField{Type: "array", Items: &field}
	case reflect.Bool:
		return DeploymentField{Type: "boolean"}
	case reflect.Int, reflect.Int64, reflect.Uint64:
		return DeploymentField{Type: "integer"}
	default:
		return DeploymentField{Type: "string"}
	}
}
