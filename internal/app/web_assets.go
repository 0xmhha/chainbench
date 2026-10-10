package app

import (
	"bytes"
	"context"
	"encoding/json"
	"errors"
	"io"
	"os"
	"path/filepath"
	"regexp"
	"strings"
	"time"
	"unicode"

	"github.com/0xmhha/chainbench/internal/core/registry"
	"github.com/0xmhha/chainbench/internal/core/session"
	"go.yaml.in/yaml/v3"
)

const WebBinaryAssetLimit int64 = 256 << 20
const WebFileAssetLimit int64 = 16 << 20

// WebAsset exposes metadata, never a server path or file contents. Compatibility
// reports observed native identity; it does not claim a source-code audit.
type WebAsset struct {
	ID            string         `json:"id"`
	Kind          string         `json:"kind"`
	Name          string         `json:"name"`
	Checksum      string         `json:"checksum"`
	Bytes         int64          `json:"bytes"`
	Compatibility map[string]any `json:"compatibility"`
	UploaderID    string         `json:"uploaderId"`
}
type webAssetReceipt struct {
	WebAsset
	Binary *ManifestBinary `json:"binary,omitempty"`
}

func (s *ManifestStore) UploadAsset(ctx context.Context, a DeploymentActor, kind, name string, reader io.Reader) (WebAsset, error) {
	var out WebAsset
	if !a.canEdit() {
		return out, ErrDeploymentForbidden
	}
	if name == "" || len(name) > 200 || name == "." || name == ".." || strings.ContainsAny(name, "/\\") || strings.IndexFunc(name, unicode.IsControl) >= 0 {
		return out, errors.New("select a plain file name")
	}
	limit := WebFileAssetLimit
	switch kind {
	case "binary":
		limit = WebBinaryAssetLimit
	case "configuration", "template", "material":
	default:
		return out, errors.New("unsupported asset kind")
	}
	stage, err := s.assets.Stage(reader, limit)
	if err != nil {
		return out, errors.New("asset is empty, unreadable or exceeds its size limit")
	}
	defer s.assets.Discard(stage)
	id, err := session.NewAssetID()
	if err != nil {
		return out, err
	}
	out = WebAsset{ID: id, Kind: kind, Name: name, Checksum: stage.Checksum, Bytes: stage.Bytes, UploaderID: a.ID, Compatibility: map[string]any{}}
	receipt := webAssetReceipt{WebAsset: out}
	if kind == "binary" {
		// Verify the native header before granting execute permission or probing.
		if err = verifyManifestNative(stage.Path); err != nil {
			return WebAsset{}, err
		}
		if err = os.Chmod(stage.Path, 0700); err != nil {
			return WebAsset{}, errors.New("cannot prepare native asset inspection")
		}
		probe, cancel := context.WithTimeout(ctx, 10*time.Second)
		defer cancel()
		version, err := manifestProbe(probe, stage.Path, "version")
		if err != nil {
			return WebAsset{}, errors.New("binary version inspection failed")
		}
		match := regexp.MustCompile(`(?m)^Git Commit: ([0-9a-f]{40})\s*$`).FindSubmatch(version)
		if len(match) != 2 {
			return WebAsset{}, errors.New("binary lacks a valid source commit identity")
		}
		var compatible *ManifestBinary
		var evidence ManifestBinaryEvidence
		for _, chain := range registry.Names() {
			asset := ManifestBinary{ID: id, Chain: chain, Path: stage.Path, SHA256: stage.Checksum, Commit: string(match[1])}
			observed, verifyErr := asset.Verify(probe, chain)
			if verifyErr == nil {
				if compatible != nil {
					return WebAsset{}, errors.New("binary identity is ambiguous")
				}
				compatible, evidence = &asset, observed
			}
		}
		if compatible == nil {
			return WebAsset{}, errors.New("binary does not match a supported native chain and dialect")
		}
		// The persisted receipt uses a server-owned ID, not the temporary path.
		compatible.Path = ""
		receipt.Binary = compatible
		out.Compatibility = map[string]any{"chain": evidence.Chain, "version": evidence.Version, "commit": compatible.Commit, "helpDigest": evidence.HelpDigest, "os": evidence.OS, "architecture": evidence.Architecture, "inspection": "native-version-and-help"}
		receipt.WebAsset = out
	} else {
		data, err := os.ReadFile(stage.Path)
		if err != nil {
			return WebAsset{}, errors.New("cannot inspect asset")
		}
		if err = validateWebFileAsset(kind, name, data); err != nil {
			return WebAsset{}, err
		}
		out.Compatibility = map[string]any{"format": strings.TrimPrefix(strings.ToLower(filepath.Ext(name)), "."), "inspection": "file-format", "executionReady": false}
		receipt.WebAsset = out
	}
	if err = ctx.Err(); err != nil {
		return WebAsset{}, err
	}
	raw, err := json.Marshal(receipt)
	if err != nil {
		return WebAsset{}, err
	}
	if err = s.AuditManifest(a.ID, "asset.upload", id, 0); err != nil {
		return WebAsset{}, errors.New("asset audit storage unavailable")
	}
	if err = s.assets.Publish(stage, id, raw, kind == "binary"); err != nil {
		return WebAsset{}, errors.New("asset storage unavailable")
	}
	_ = s.AuditManifest(a.ID, "asset.upload", id, 201)
	return out, nil
}

func validateWebFileAsset(kind, name string, data []byte) error {
	text := strings.ToUpper(string(data))
	if strings.Contains(text, "PRIVATE KEY") {
		return errors.New("private key material must use private credential storage")
	}
	ext := strings.ToLower(filepath.Ext(name))
	var value any
	switch ext {
	case ".json", ".abi":
		dec := json.NewDecoder(bytes.NewReader(data))
		dec.UseNumber()
		if dec.Decode(&value) != nil || dec.Decode(new(any)) != io.EOF {
			return errors.New("asset must contain one valid JSON value")
		}
	case ".yaml", ".yml":
		dec := yaml.NewDecoder(bytes.NewReader(data))
		if dec.Decode(&value) != nil || dec.Decode(new(any)) != io.EOF {
			return errors.New("asset must contain one valid YAML value")
		}
	case ".bin", ".hex":
		if kind != "material" {
			return errors.New("bytecode must use the material kind")
		}
		hex := strings.TrimPrefix(strings.TrimSpace(string(data)), "0x")
		if len(hex) == 0 || len(hex)%2 != 0 || strings.IndexFunc(hex, func(r rune) bool { return !strings.ContainsRune("0123456789abcdefABCDEF", r) }) >= 0 {
			return errors.New("material bytecode must be hexadecimal")
		}
		return nil
	default:
		return errors.New("use JSON or YAML declarations, JSON ABI, or hexadecimal bytecode")
	}
	if value == nil {
		return errors.New("asset declaration is empty")
	}
	var secret func(any) bool
	secret = func(v any) bool {
		switch x := v.(type) {
		case map[string]any:
			for k, v := range x {
				key := strings.ToLower(strings.NewReplacer("_", "", "-", "").Replace(k))
				switch key {
				case "password", "passwordfile", "privatekey", "keyfile", "passphrase", "keypassphrasefile", "secret", "mnemonic", "ciphertext", "nodekey", "blssecretkey":
					return true
				}
				if secret(v) {
					return true
				}
			}
		case []any:
			for _, v := range x {
				if secret(v) {
					return true
				}
			}
		case map[any]any:
			// Non-string YAML keys cannot be safely inspected as declarations.
			return true
		}
		return false
	}
	if secret(value) {
		return errors.New("credentials and private keys cannot be shared assets")
	}
	return nil
}

func (s *ManifestStore) assetReceipt(id string) (webAssetReceipt, string, error) {
	var item webAssetReceipt
	raw, path, checksum, size, err := s.assets.Read(id)
	if errors.Is(err, os.ErrNotExist) {
		return item, "", ErrDeploymentNotFound
	}
	if err != nil {
		return item, "", errors.New("registered asset is unavailable or changed")
	}
	if json.Unmarshal(raw, &item) != nil || item.ID != id || item.Checksum != checksum || item.Bytes != size {
		return item, "", errors.New("registered asset checksum or receipt changed")
	}
	if item.Binary != nil && (item.Kind != "binary" || item.Binary.ID != id || item.Binary.SHA256 != checksum || item.Binary.Path != "") {
		return item, "", errors.New("registered binary receipt changed")
	}
	return item, path, nil
}
func (s *ManifestStore) Asset(id string) (WebAsset, error) {
	item, _, err := s.assetReceipt(id)
	return item.WebAsset, err
}
func (s *ManifestStore) Assets() ([]WebAsset, error) {
	ids, err := s.assets.IDs()
	if err != nil {
		return nil, errors.New("asset storage unavailable")
	}
	items := []WebAsset{}
	for _, id := range ids {
		item, err := s.Asset(id)
		if err != nil {
			return nil, err
		}
		items = append(items, item)
	}
	return items, nil
}
func (s *ManifestStore) BinaryAsset(id string) (ManifestBinary, error) {
	item, path, err := s.assetReceipt(id)
	if err != nil {
		return ManifestBinary{}, err
	}
	if item.Binary == nil {
		return ManifestBinary{}, errors.New("selected asset is not a compatible executable")
	}
	asset := *item.Binary
	asset.Path = path
	return asset, nil
}
