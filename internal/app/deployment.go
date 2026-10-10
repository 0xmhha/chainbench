package app

import (
	"bytes"
	"encoding/json"
	"errors"
	"fmt"
	"strings"

	"github.com/0xmhha/chainbench/internal/resource"
	"go.yaml.in/yaml/v3"
)

// DeploymentDocumentInput uses the engine's object spelling, without SSH secrets.
type DeploymentDocumentInput struct {
	Kind            string          `json:"kind"`
	Name            string          `json:"name"`
	ContractVersion string          `json:"contractVersion"`
	Content         json.RawMessage `json:"content"`
	AssetRefs       []string        `json:"assetRefs"`
	// PresetRefs pin the shared chain-preset revisions a case names by id or
	// extends, so the case keeps its reference and still runs one exact
	// declaration. Only a case carries them.
	PresetRefs []DeploymentDocumentRef `json:"presetRefs,omitempty"`
}

// ValidateDeploymentDocument delegates grammar, paths and placement to resource.
func ValidateDeploymentDocument(in DeploymentDocumentInput) error {
	return validateDeploymentDocument(in, nil)
}

// errCasePresetsNeedStore refuses to judge a case against presets nobody
// supplied: its pinned preset revisions live in the document store.
var errCasePresetsNeedStore = errors.New("a case with preset references is validated against the shared preset revisions it pins")

func validateDeploymentDocument(in DeploymentDocumentInput, presets map[string]json.RawMessage) error {
	if len(in.PresetRefs) > 0 && in.Kind != "case" {
		return errors.New("only a test case pins chain preset references")
	}
	if in.Kind == "case" && len(in.PresetRefs) > 0 && presets == nil {
		return errCasePresetsNeedStore
	}
	if strings.TrimSpace(in.Name) == "" || in.ContractVersion != "2" {
		return errors.New("named deployment document with contractVersion 2 required")
	}
	if err := validateWebDocumentAssetRefs(in); err != nil {
		return err
	}
	var content map[string]any
	dec := json.NewDecoder(bytes.NewReader(in.Content))
	dec.UseNumber()
	if err := dec.Decode(&content); err != nil || content == nil {
		return errors.New("content must be an object")
	}
	if err := validateWebDocumentSecrets(in.Kind, content); err != nil {
		return err
	}
	// JSON numbers must remain numbers when transcoded to the engine YAML parser.
	var plain any
	if err := json.Unmarshal(in.Content, &plain); err != nil {
		return errors.New("invalid content")
	}
	b, err := yaml.Marshal(plain)
	if err != nil {
		return errors.New("invalid content")
	}
	switch in.Kind {
	case "chain-preset":
		return ValidateChainPreset(in.Content)
	case "case":
		_, err := PrepareTestCase(TestCaseInput{Content: in.Content, Presets: presets})
		return err
	case "server-set":
		if sshBlock, ok := content["ssh"].(map[string]any); ok {
			for _, field := range []string{"password", "password_file", "key_file", "key_passphrase_file"} {
				if _, exists := sshBlock[field]; exists {
					return errors.New("SSH secrets and secret paths must use a personal credential binding")
				}
			}
		}
		set, err := resource.ParseSet(b)
		if err != nil {
			return err
		}
		// The custom engine host decoder accepts shorthand. Refuse unregistered
		// object fields here rather than silently changing an imported declaration.
		pool, ok := plain.(map[string]any)["pool"].(map[string]any)
		if !ok {
			return errors.New("pool required")
		}
		hosts, ok := pool["hosts"].([]any)
		if !ok {
			return errors.New("pool.hosts required")
		}
		addresses := map[string]bool{}
		for _, h := range hosts {
			if obj, ok := h.(map[string]any); ok {
				for k := range obj {
					if k != "name" && k != "addr" {
						return fmt.Errorf("unknown host field %s", k)
					}
				}
			}
		}
		for _, h := range set.PoolSpec.Hosts {
			identity := strings.ToLower(strings.TrimSuffix(h.Addr, "."))
			if addresses[identity] {
				return errors.New("duplicate host address")
			}
			addresses[identity] = true
			if strings.ContainsAny(h.Addr, " \t\r\n/;@") {
				return errors.New("host must be an address or DNS name")
			}
		}
		p, err := set.Pool(1, 0)
		if err != nil {
			return err
		}
		if p.Slots > 4096 || len(p.Hosts) > 4096 || p.Cap() > 4096 {
			return errors.New("placement exceeds 4096 slots")
		}
		return p.Validate()
	case "workspace-config":
		_, err := resource.ParseWorkspaceConfig(b)
		return err
	default:
		return errors.New("unsupported deployment document kind")
	}
}

func deploymentSet(in DeploymentDocumentInput) (*resource.Set, error) {
	var value any
	if err := json.Unmarshal(in.Content, &value); err != nil {
		return nil, err
	}
	b, err := yaml.Marshal(value)
	if err != nil {
		return nil, err
	}
	return resource.ParseSet(b)
}

func deploymentWorkspace(in DeploymentDocumentInput) (resource.WorkspaceConfig, error) {
	var value any
	if err := json.Unmarshal(in.Content, &value); err != nil {
		return resource.WorkspaceConfig{}, err
	}
	b, err := yaml.Marshal(value)
	if err != nil {
		return resource.WorkspaceConfig{}, err
	}
	return resource.ParseWorkspaceConfig(b)
}

// ExportDeploymentDocument emits the engine declaration, without Web metadata or private overlays.
func ExportDeploymentDocument(in DeploymentDocumentInput, format string) ([]byte, error) {
	// A stored case with pinned presets was validated against them when saved;
	// its declaration is exported as written, references included.
	if err := ValidateDeploymentDocument(in); err != nil && !errors.Is(err, errCasePresetsNeedStore) {
		return nil, err
	}
	if format == "" || format == "json" {
		return append([]byte(nil), in.Content...), nil
	}
	if format != "yaml" {
		return nil, errors.New("unsupported export format")
	}
	var value any
	if err := json.Unmarshal(in.Content, &value); err != nil {
		return nil, err
	}
	return yaml.Marshal(value)
}
