package app

import (
	"bytes"
	"encoding/json"
	"errors"
	"fmt"
	"strconv"
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
	// CaseRefs pin the shared case revisions a run definition lists, in its
	// order. Only a run definition carries them.
	CaseRefs []DeploymentDocumentRef `json:"caseRefs,omitempty"`
	// ContractDigest names the engine contract the draft was edited against.
	// It is checked on save and never stored.
	ContractDigest string `json:"contractDigest,omitempty"`
}

// ValidateDeploymentDocument delegates grammar, paths and placement to resource.
func ValidateDeploymentDocument(in DeploymentDocumentInput) error {
	return validateDeploymentDocument(in, nil)
}

// errCasePresetsNeedStore refuses to judge a case against presets nobody
// supplied: its pinned preset revisions live in the document store.
var errCasePresetsNeedStore = errors.New("a case with preset references is validated against the shared preset revisions it pins")

// WebFieldError names the part of a request that failed validation, so a
// refusal can point the editor at it. Path is a JSON pointer into the
// request. A content failure points at the field the engine names, or at
// /content as a whole when it names none.
type WebFieldError struct {
	Path string
	Err  error
}

func (e *WebFieldError) Error() string { return e.Err.Error() }
func (e *WebFieldError) Unwrap() error { return e.Err }

// WebFieldPath is the JSON pointer a refusal points at: the part a
// WebFieldError names, or else the place an engine error about a document
// starts with.
func WebFieldPath(err error) string {
	var field *WebFieldError
	if errors.As(err, &field) {
		return field.Path
	}
	return ContentErrorPath(err)
}

func webFieldError(path string, err error) error {
	var field *WebFieldError
	if err == nil || errors.As(err, &field) {
		return err
	}
	return &WebFieldError{Path: path, Err: err}
}

func validateDeploymentDocument(in DeploymentDocumentInput, presets map[string]json.RawMessage) error {
	if len(in.PresetRefs) > 0 && in.Kind != "case" {
		return webFieldError("/presetRefs", errors.New("only a test case pins chain preset references"))
	}
	if in.Kind == "case" && len(in.PresetRefs) > 0 && presets == nil {
		return webFieldError("/presetRefs", errCasePresetsNeedStore)
	}
	if len(in.CaseRefs) > 0 && in.Kind != "run-definition" {
		return webFieldError("/caseRefs", errors.New("only a run definition pins case references"))
	}
	if strings.TrimSpace(in.Name) == "" {
		return webFieldError("/name", errors.New("named deployment document with contractVersion 2 required"))
	}
	if in.ContractVersion != "2" {
		return webFieldError("/contractVersion", errors.New("named deployment document with contractVersion 2 required"))
	}
	if err := validateWebDocumentAssetRefs(in); err != nil {
		return webFieldError("/assetRefs", err)
	}
	err := validateDeploymentContent(in, presets)
	return webFieldError(contentErrorPath(in.Kind, err), err)
}

// contentErrorPath points a content refusal at the field it is about: the
// field the engine names for a server set or workspace config, or the place an
// engine error about a chain preset or case starts with. Anything else
// belongs to the content as a whole.
func contentErrorPath(kind string, err error) string {
	var field *resource.FieldError
	if errors.As(err, &field) {
		path := "/content"
		for _, key := range field.Path {
			path += "/" + strings.NewReplacer("~", "~0", "/", "~1").Replace(key)
		}
		return path
	}
	if err != nil && (kind == "chain-preset" || kind == "case") {
		return ContentErrorPath(err)
	}
	return "/content"
}

func contentFieldError(err error, path ...string) error {
	return &resource.FieldError{Path: path, Err: err}
}

func validateDeploymentContent(in DeploymentDocumentInput, presets map[string]json.RawMessage) error {
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
					return contentFieldError(errors.New("SSH secrets and secret paths must use a personal credential binding"), "ssh", field)
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
			return contentFieldError(errors.New("pool required"), "pool")
		}
		hosts, ok := pool["hosts"].([]any)
		if !ok {
			return contentFieldError(errors.New("pool.hosts required"), "pool", "hosts")
		}
		addresses := map[string]bool{}
		for i, h := range hosts {
			if obj, ok := h.(map[string]any); ok {
				for k := range obj {
					if k != "name" && k != "addr" {
						return contentFieldError(fmt.Errorf("unknown host field %s", k), "pool", "hosts", strconv.Itoa(i), k)
					}
				}
			}
		}
		for i, h := range set.PoolSpec.Hosts {
			identity := strings.ToLower(strings.TrimSuffix(h.Addr, "."))
			if addresses[identity] {
				return contentFieldError(errors.New("duplicate host address"), "pool", "hosts", strconv.Itoa(i), "addr")
			}
			addresses[identity] = true
			if strings.ContainsAny(h.Addr, " \t\r\n/;@") {
				return contentFieldError(errors.New("host must be an address or DNS name"), "pool", "hosts", strconv.Itoa(i), "addr")
			}
		}
		p, err := set.Pool(1, 0)
		if err != nil {
			return contentFieldError(err, "pool")
		}
		if p.Slots > 4096 || len(p.Hosts) > 4096 || p.Cap() > 4096 {
			return contentFieldError(errors.New("placement exceeds 4096 slots"), "pool", "slots")
		}
		if err := p.Validate(); err != nil {
			// The pool's own checks cover its hosts, slots and port bands together.
			return contentFieldError(err, "pool")
		}
		return nil
	case "workspace-config":
		_, err := resource.ParseWorkspaceConfig(b)
		return err
	case "run-definition":
		_, err := runDefinitionCaseIDs(in.Content)
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
