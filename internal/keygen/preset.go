// Package keygen generates a network's preset key set — per-node devp2p keys,
// their derived address and BLS public key/proof-of-possession, an encrypted
// keystore per node (via the accounts SDK), and the metadata.json that
// keys.LoadPreset reads. It is the shared core behind `validator set` (the
// preset is defined by its validator set) and its MCP mirror; it lives here,
// not in the CLI, so both surfaces generate identically.
//
// Generation runs entirely in process: identity derivation is
// [keyring.Derive], so no chain binary has to be built or on PATH. This
// package is being absorbed into core/keyring (worklist K3).
package keygen

import (
	"crypto/rand"
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"time"

	"github.com/0xmhha/accounts/keystore"

	"github.com/0xmhha/chainbench/internal/core/keyring"
)

// dirPerm and file perms for generated material.
const (
	dirPerm     os.FileMode = 0o755
	secretPerm  os.FileMode = 0o600
	publicPerm  os.FileMode = 0o644
	defaultPort             = 30301
)

// keystore scrypt parameters. Light (geth's LightScryptN/P) — a preset is a
// local test fixture, and lighter params keep multi-node generation fast while
// staying a standard, node-readable v3 keystore.
const (
	keystoreScryptN = 1 << 12
	keystoreScryptP = 6
)

// Node is one generated node's material.
type Node struct {
	Index     int    `json:"index"`
	Nodekey   string `json:"nodekey"`
	PublicKey string `json:"publicKey"`
	Address   string `json:"address"`
	BLSPubKey string `json:"blsPublicKey"`
	BLSPoP    string `json:"blsPoP"`
	Enode     string `json:"enode"`
}

// Meta mirrors keys/preset/metadata.json (the shape keys.LoadPreset reads).
type Meta struct {
	Description           string                    `json:"description"`
	Warning               string                    `json:"warning"`
	Password              string                    `json:"password"`
	Validators            []string                  `json:"validators"`
	BLSPublicKeys         []string                  `json:"blsPublicKeys"`
	SystemContractMembers string                    `json:"systemContractMembers"`
	SystemContractBLSKeys string                    `json:"systemContractBlsKeys"`
	Alloc                 map[string]map[string]any `json:"alloc"`
	Nodes                 []Node                    `json:"nodes"`
}

// PresetOpts configures preset generation.
type PresetOpts struct {
	Nodes      int
	Validators int
	Out        string
	Password   string
	BasePort   int
	Balance    string
}

// GeneratePreset generates an N-node preset into opts.Out and returns the
// metadata. progress, when non-nil, receives a line per node. Validators default
// to all nodes; BasePort defaults to 30301.
func GeneratePreset(opts PresetOpts, progress func(string)) (Meta, error) {
	if opts.Nodes < 1 {
		return Meta{}, fmt.Errorf("keygen: nodes must be >= 1")
	}
	if opts.Validators < 1 || opts.Validators > opts.Nodes {
		opts.Validators = opts.Nodes
	}
	if opts.BasePort == 0 {
		opts.BasePort = defaultPort
	}
	if err := os.MkdirAll(opts.Out, dirPerm); err != nil {
		return Meta{}, err
	}
	// The shared password file is what the node unlocks with at launch
	// (--password); the keystores below are encrypted with the same password.
	pwFile := filepath.Join(opts.Out, "password")
	if err := os.WriteFile(pwFile, []byte(opts.Password), secretPerm); err != nil {
		return Meta{}, err
	}

	meta := Meta{
		Description: fmt.Sprintf("Generated preset: %d nodes (%d validators). chainbench validator set.", opts.Nodes, opts.Validators),
		Warning:     "TEST FIXTURE ONLY — do not import to mainnet/testnet.",
		Password:    opts.Password,
		Alloc:       map[string]map[string]any{},
	}
	for i := 1; i <= opts.Nodes; i++ {
		n, err := generateNode(i, opts.Out, opts.Password, opts.BasePort)
		if err != nil {
			return Meta{}, fmt.Errorf("keygen: node %d: %w", i, err)
		}
		meta.Nodes = append(meta.Nodes, n)
		meta.Alloc[strings.TrimPrefix(n.Address, "0x")] = map[string]any{"balance": opts.Balance}
		if i <= opts.Validators {
			meta.Validators = append(meta.Validators, n.Address)
			meta.BLSPublicKeys = append(meta.BLSPublicKeys, n.BLSPubKey)
		}
		if progress != nil {
			progress(fmt.Sprintf("node %d  %s  bls=%s…", i, n.Address, n.BLSPubKey[:min(14, len(n.BLSPubKey))]))
		}
	}
	meta.SystemContractMembers = strings.Join(meta.Validators, ",")
	meta.SystemContractBLSKeys = strings.Join(meta.BLSPublicKeys, ",")

	// No extra-data is written. It encodes the validator set, so a stored copy
	// goes stale the moment a network uses a subset of these validators — and a
	// genesis whose extra-data disagrees with its validator set is accepted and
	// then fails in consensus. The wbft family derives it at genesis time.

	b, err := json.MarshalIndent(meta, "", "  ")
	if err != nil {
		return Meta{}, err
	}
	if err := os.WriteFile(filepath.Join(opts.Out, "metadata.json"), b, publicPerm); err != nil {
		return Meta{}, err
	}
	return meta, nil
}

// generateNode creates node i's material: a random nodekey, its derived address
// + devp2p public key + BLS pubkey/PoP, and an encrypted keystore (via the
// accounts SDK keystore). It writes the per-node dir and returns the node
// metadata. No external process is run.
func generateNode(i int, out, password string, basePort int) (Node, error) {
	nodeDir := filepath.Join(out, fmt.Sprintf("node%d", i))
	if err := os.MkdirAll(nodeDir, dirPerm); err != nil {
		return Node{}, err
	}

	key, err := keyring.NewNodekey(rand.Reader)
	if err != nil {
		return Node{}, err
	}
	nodekey := key.Hex()
	if err := os.WriteFile(filepath.Join(nodeDir, "nodekey"), []byte(nodekey), secretPerm); err != nil {
		return Node{}, err
	}

	n, err := DeriveIdentity(nodekey)
	if err != nil {
		return Node{}, err
	}
	n.Index = i
	raw := key.Bytes()
	n.Enode = fmt.Sprintf("enode://%s@127.0.0.1:%d?discport=0", n.PublicKey, basePort+i-1)

	// Encrypt the account key into a standard v3 keystore with the SDK and place
	// it where the node reads it (<datadir>/keystore), matching what the node's
	// own `account import` used to produce — but with no binary shell-out.
	keyjson, err := keystore.Encrypt(raw, password, keystoreScryptN, keystoreScryptP)
	if err != nil {
		return Node{}, fmt.Errorf("keystore encrypt: %w", err)
	}
	ksDir := filepath.Join(nodeDir, "keystore")
	if err := os.MkdirAll(ksDir, dirPerm); err != nil {
		return Node{}, err
	}
	if err := os.WriteFile(filepath.Join(ksDir, keystoreFilename(n.Address)), keyjson, secretPerm); err != nil {
		return Node{}, err
	}

	for name, val := range map[string]string{"address": n.Address, "pubkey": n.PublicKey, "bls_pubkey": n.BLSPubKey} {
		if err := os.WriteFile(filepath.Join(nodeDir, name), []byte(val), publicPerm); err != nil {
			return Node{}, err
		}
	}
	return n, nil
}

// keystoreFilename is the geth-convention keystore file name
// (UTC--<timestamp>--<address>), which the node's keystore reader accepts.
func keystoreFilename(address string) string {
	ts := time.Now().UTC().Format("2006-01-02T15-04-05.000000000Z")
	return "UTC--" + ts + "--" + strings.TrimPrefix(strings.ToLower(address), "0x")
}

// DeriveIdentity returns the identity a devp2p/account private key (hex, with
// or without 0x) implies: address, devp2p public key, BLS public key and
// proof-of-possession. It is how a wbft-family validator gets its BLS material
// from its key.
//
// It used to shell out to the go-wbft bootnode tool. It no longer does — the
// derivation is [keyring.Derive], verified byte for byte against the shipped
// preset — so a key set can be generated with no chain build present.
func DeriveIdentity(nodekeyHex string) (Node, error) {
	key, err := keyring.ParseNodekey(nodekeyHex)
	if err != nil {
		return Node{}, fmt.Errorf("keygen: %w", err)
	}
	id, err := keyring.Derive(key, keyring.WithBLS)
	if err != nil {
		return Node{}, fmt.Errorf("keygen: %w", err)
	}
	return Node{
		Nodekey:   key.Hex(),
		PublicKey: id.PublicKey,
		Address:   id.Address,
		BLSPubKey: id.BLS.PublicKey,
		BLSPoP:    id.BLS.PoP,
	}, nil
}
