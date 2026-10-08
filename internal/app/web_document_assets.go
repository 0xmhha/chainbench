package app

import (
	"encoding/hex"
	"encoding/json"
	"errors"
	"strings"
)

func webAssetRefID(ref string) (string, error) {
	if !strings.HasPrefix(ref, "asset:") {
		return "", errors.New("select a registered asset ID; filesystem references are unavailable")
	}
	id := strings.TrimPrefix(ref, "asset:")
	if len(id) != 32 || strings.ToLower(id) != id {
		return "", errors.New("invalid asset ID")
	}
	if _, err := hex.DecodeString(id); err != nil {
		return "", errors.New("invalid asset ID")
	}
	return id, nil
}

// These references are parser-owned content fields. The outer list must match
// exactly, so import/export cannot lose or invent an execution dependency.
func webDocumentAssetRefs(kind string, raw json.RawMessage) ([]string, error) {
	refs := []string{}
	if kind == "case" {
		ref, err := webCaseGenesisRef(raw)
		if err != nil {
			return nil, err
		}
		if strings.HasPrefix(ref, "asset:") {
			id, err := webAssetRefID(ref)
			if err != nil {
				return nil, err
			}
			refs = append(refs, id)
		}
		return refs, nil
	}
	if kind != "chain-preset" {
		return refs, nil
	}
	var doc struct {
		Genesis struct {
			Ref string `json:"ref"`
		} `json:"genesis"`
	}
	if err := json.Unmarshal(raw, &doc); err != nil {
		return nil, errors.New("invalid asset reference declaration")
	}
	if strings.HasPrefix(doc.Genesis.Ref, "asset:") {
		id, err := webAssetRefID(doc.Genesis.Ref)
		if err != nil {
			return nil, err
		}
		refs = append(refs, id)
	}
	return refs, nil
}

func webCaseGenesisRef(raw json.RawMessage) (string, error) {
	var doc struct {
		SchemaVersion string          `json:"schemaVersion"`
		ChainPreset   json.RawMessage `json:"chainPreset"`
		Chain         struct {
			GenesisExisting string `json:"genesisExisting"`
		} `json:"chain"`
	}
	if err := json.Unmarshal(raw, &doc); err != nil {
		return "", errors.New("invalid case asset declaration")
	}
	if doc.SchemaVersion != "2" {
		return doc.Chain.GenesisExisting, nil
	}
	if !strings.HasPrefix(strings.TrimSpace(string(doc.ChainPreset)), "{") {
		return "", nil // Named declarations are resolved by the case parser.
	}
	var env struct {
		Genesis struct {
			Ref string `json:"ref"`
		} `json:"genesis"`
	}
	if err := json.Unmarshal(doc.ChainPreset, &env); err != nil {
		return "", errors.New("invalid case genesis declaration")
	}
	return env.Genesis.Ref, nil
}

func validateWebDocumentAssetRefs(in DeploymentDocumentInput) error {
	refs, err := webDocumentAssetRefs(in.Kind, in.Content)
	if err != nil {
		return err
	}
	if len(refs) != len(in.AssetRefs) {
		return errors.New("assetRefs must match the registered references in the declaration")
	}
	for i, id := range refs {
		if in.AssetRefs[i] != id {
			return errors.New("assetRefs must match the registered references in the declaration")
		}
	}
	return nil
}
