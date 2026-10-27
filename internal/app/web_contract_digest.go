package app

import (
	"encoding/json"
	"errors"

	"github.com/0xmhha/chainbench/internal/dsl"
)

// DocumentContractVersion is the version of the document format the engine
// contracts describe (the DSL's schemaVersion 2).
const DocumentContractVersion = webEngineContractVersion

// ErrContractChanged refuses a draft opened against an engine contract that
// has since changed; the editor validates it again and shows what differs.
// It is a conflict, but not a revision conflict, so it says only its own reason.
var ErrContractChanged error = contractChangedError{}

type contractChangedError struct{}

func (contractChangedError) Error() string {
	return "the engine contract changed since this draft was opened; validate it again before saving"
}
func (contractChangedError) Is(target error) bool { return target == ErrDeploymentConflict }

// ContractWithDigest returns a contract as a JSON object carrying its digest
// and document version. The digest is over the contract alone, so it changes
// exactly when the contract does.
func ContractWithDigest(contract any) (map[string]any, error) {
	raw, err := json.Marshal(contract)
	if err != nil {
		return nil, err
	}
	var out map[string]any
	if err = json.Unmarshal(raw, &out); err != nil {
		return nil, errors.New("contract is not a JSON object")
	}
	out["x-contractDigest"] = manifestHash(raw)
	out["x-contractVersion"] = DocumentContractVersion
	return out, nil
}

// ContractDigest is the digest of the contract a document of this kind is
// edited against: the chain's preset contract, the DSL grammar or the
// server-set and workspace-config contract.
func ContractDigest(kind string, content json.RawMessage) (string, error) {
	var contract any
	switch kind {
	case "chain-preset":
		var head struct {
			Chain string `json:"chain"`
		}
		_ = json.Unmarshal(content, &head)
		if head.Chain == "" {
			var schema map[string]any
			if err := json.Unmarshal(dsl.SchemaV2, &schema); err != nil {
				return "", err
			}
			contract = schema
			break
		}
		schema, err := ChainPresetContract(head.Chain)
		if err != nil {
			return "", err
		}
		contract = schema
	case "case":
		_, schema, err := DSLContract()
		if err != nil {
			return "", err
		}
		contract = schema
	case "server-set", "workspace-config":
		contract = DeploymentContract()
	default:
		return "", nil
	}
	raw, err := json.Marshal(contract)
	if err != nil {
		return "", err
	}
	return manifestHash(raw), nil
}

// CheckContractDigest refuses a draft whose contract digest is not current.
// A client that sends none is judged by validation alone.
func CheckContractDigest(in DeploymentDocumentInput) error {
	if in.ContractDigest == "" {
		return nil
	}
	current, err := ContractDigest(in.Kind, in.Content)
	if err != nil {
		return err
	}
	if current != "" && current != in.ContractDigest {
		return ErrContractChanged
	}
	return nil
}
