// This owned security fixture simulates data saved before node-key publication
// guards existed. It never calls a production API to bypass current validation.
package main

import (
	"crypto/aes"
	"crypto/cipher"
	"crypto/rand"
	"encoding/hex"
	"encoding/json"
	"errors"
	"fmt"
	"os"
	"path/filepath"
	"time"
)

func main() {
	if err := run(); err != nil {
		fmt.Fprintln(os.Stderr, err)
		os.Exit(1)
	}
}

func run() error {
	if len(os.Args) != 3 {
		return errors.New("requires owned store and private fixture paths")
	}
	root, fixturePath := os.Args[1], os.Args[2]
	if filepath.Clean(root) != filepath.Join(filepath.Dir(fixturePath), "store") {
		return errors.New("store must belong to the private fixture runtime")
	}
	var fixture map[string]any
	if err := readJSON(fixturePath, &fixture); err != nil {
		return err
	}
	nodeKey, keyOK := fixture["nodeKey"].(string)
	owner, ownerOK := fixture["legacyOwnerID"].(string)
	if !keyOK || len(nodeKey) != 66 || !ownerOK || owner == "" {
		return errors.New("private fixture identity required")
	}
	key, err := os.ReadFile(filepath.Join(root, "credential.key"))
	if err != nil {
		return err
	}
	block, err := aes.NewCipher(key)
	if err != nil {
		return err
	}
	aead, err := cipher.NewGCM(block)
	if err != nil {
		return err
	}
	state := map[string]any{"documents": map[string]any{}, "workspaces": map[string]any{}, "credentials": map[string]any{}, "bindings": map[string]any{}, "audit": []any{}}
	path := filepath.Join(root, "deployment.json")
	if err = readJSON(path, &state); err != nil && !os.IsNotExist(err) {
		return err
	}
	documents, ok := state["documents"].(map[string]any)
	if !ok {
		return errors.New("fixture store documents missing")
	}
	imports, _ := state["imports"].(map[string]any)
	if imports == nil {
		imports = map[string]any{}
		state["imports"] = imports
	}
	topology := map[string]any{"nodes": []any{map[string]any{"index": 1, "role": "bp", "key": nodeKey}}}
	declarations := []map[string]any{
		{"schemaVersion": "2", "kind": "chain-preset", "id": "legacy-preset", "chain": "stablenet", "topology": topology},
		{"schemaVersion": "2", "kind": "case", "id": "legacy-case-v2", "chainPreset": map[string]any{"chain": "stablenet", "topology": topology}, "steps": []any{map[string]any{"expect": "blockNumber", "is": 0}}},
		{"schemaVersion": "1", "id": "legacy-case-v1", "chain": map[string]any{"name": "stablenet", "binary": "gstable"}, "topology": topology, "assertions": []any{map[string]any{"assert": "blockNumber", "expected": 0}}},
	}
	docIDs, importIDs := []string{}, []string{}
	for i, declaration := range declarations {
		id, err := randomID()
		if err != nil {
			return err
		}
		previewID, err := randomID()
		if err != nil {
			return err
		}
		kind := "case"
		if i == 0 {
			kind = "chain-preset"
		}
		doc := map[string]any{"kind": kind, "name": declaration["id"], "contractVersion": "2", "content": declaration, "assetRefs": []any{}, "id": id, "revision": 1, "updatedAt": time.Now().UTC(), "updatedBy": owner}
		documents[id] = []any{doc}
		source, err := json.Marshal(declaration)
		if err != nil {
			return err
		}
		nonce := make([]byte, aead.NonceSize())
		if _, err = rand.Read(nonce); err != nil {
			return err
		}
		imports[previewID] = map[string]any{
			"ownerId": owner, "expiresAt": time.Now().UTC().Add(time.Hour), "committed": false,
			"ciphertext": aead.Seal(nonce, nonce, source, []byte(previewID+":"+owner)),
			"preview":    map[string]any{"previewId": previewID, "validation": map[string]any{"valid": true, "contractVersion": "2", "errors": []any{}, "warnings": []any{}}, "redactedDocuments": []any{map[string]any{"kind": kind, "name": declaration["id"], "contractVersion": "2", "content": declaration, "assetRefs": []any{}}}, "sourcePreserved": true},
		}
		docIDs = append(docIDs, id)
		importIDs = append(importIDs, previewID)
	}
	fixture["legacyDocumentIDs"], fixture["legacyImportIDs"] = docIDs, importIDs
	if err = writeJSON(path, state); err != nil {
		return err
	}
	return writeJSON(fixturePath, fixture)
}

func randomID() (string, error) {
	b := make([]byte, 16)
	if _, err := rand.Read(b); err != nil {
		return "", err
	}
	return hex.EncodeToString(b), nil
}

func readJSON(path string, out any) error {
	b, err := os.ReadFile(path)
	if err != nil {
		return err
	}
	return json.Unmarshal(b, out)
}

func writeJSON(path string, value any) error {
	b, err := json.Marshal(value)
	if err != nil {
		return err
	}
	return os.WriteFile(path, b, 0600)
}
