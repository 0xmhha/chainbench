// Package keygen generates a network's preset key set — per-node devp2p keys,
// their derived address and BLS public key/proof-of-possession (via the go-wbft
// bootnode tool), an encrypted keystore per node (via the accounts SDK, no node
// binary), and the metadata.json that keys.LoadPreset reads. It is the shared
// core behind `validator set` (the preset is defined by its validator set) and
// its MCP mirror; it lives here, not in the CLI, so both surfaces generate
// identically.
package keygen

import (
	"crypto/rand"
	"encoding/hex"
	"encoding/json"
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"regexp"
	"strings"
	"time"

	"github.com/0xmhha/accounts/keystore"
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
	ExtraData             string                    `json:"extraData"`
	SystemContractMembers string                    `json:"systemContractMembers"`
	SystemContractBLSKeys string                    `json:"systemContractBlsKeys"`
	Alloc                 map[string]map[string]any `json:"alloc"`
	Nodes                 []Node                    `json:"nodes"`
}

// PresetOpts configures preset generation.
type PresetOpts struct {
	Nodes      int
	Validators int
	Bootnode   string
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
	if _, err := exec.LookPath(opts.Bootnode); err != nil {
		if _, e2 := os.Stat(opts.Bootnode); e2 != nil {
			return Meta{}, fmt.Errorf("keygen: bootnode %q not found (build go-wbft/cmd/bootnode)", opts.Bootnode)
		}
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
		ExtraData:   "0x" + strings.Repeat("00", 32),
		Alloc:       map[string]map[string]any{},
	}
	for i := 1; i <= opts.Nodes; i++ {
		n, err := generateNode(i, opts.Bootnode, opts.Out, opts.Password, opts.BasePort)
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

	b, err := json.MarshalIndent(meta, "", "  ")
	if err != nil {
		return Meta{}, err
	}
	if err := os.WriteFile(filepath.Join(opts.Out, "metadata.json"), b, publicPerm); err != nil {
		return Meta{}, err
	}
	return meta, nil
}

var (
	bnHexRe  = regexp.MustCompile(`0x[0-9a-fA-F]+`)
	bnBareRe = regexp.MustCompile(`[0-9a-fA-F]{40,}`)
)

// generateNode creates node i's material: a random nodekey, its derived address
// + BLS pubkey/PoP + devp2p public key (via bootnode), and an encrypted keystore
// (via the accounts SDK keystore — no node binary). It writes the per-node dir
// and returns the node metadata.
func generateNode(i int, bootnode, out, password string, basePort int) (Node, error) {
	nodeDir := filepath.Join(out, fmt.Sprintf("node%d", i))
	if err := os.MkdirAll(nodeDir, dirPerm); err != nil {
		return Node{}, err
	}

	raw := make([]byte, 32)
	if _, err := rand.Read(raw); err != nil {
		return Node{}, err
	}
	nodekey := hex.EncodeToString(raw)
	if err := os.WriteFile(filepath.Join(nodeDir, "nodekey"), []byte(nodekey), secretPerm); err != nil {
		return Node{}, err
	}

	n, err := DeriveIdentity(bootnode, nodekey)
	if err != nil {
		return Node{}, err
	}
	n.Index = i
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

// DeriveIdentity runs the go-wbft bootnode over a devp2p/account private key
// (hex, with or without 0x) and returns the derived identity: address, devp2p
// public key, BLS public key and proof-of-possession. It is how a wbft-family
// validator gets its BLS material from its key.
func DeriveIdentity(bootnode, nodekeyHex string) (Node, error) {
	nodekeyHex = strings.TrimPrefix(strings.TrimSpace(nodekeyHex), "0x")
	res, err := exec.Command(bootnode, "-nodekeyhex", nodekeyHex, "-writeaddress").CombinedOutput()
	if err != nil {
		return Node{}, fmt.Errorf("bootnode writeaddress: %w: %s", err, res)
	}
	n, err := ParseBootnode(string(res))
	if err != nil {
		return Node{}, err
	}
	n.Nodekey = nodekeyHex
	return n, nil
}

// ParseBootnode extracts the address, devp2p public key, BLS public key and PoP
// from `bootnode -writeaddress` output.
func ParseBootnode(out string) (Node, error) {
	var n Node
	for _, line := range strings.Split(out, "\n") {
		low := strings.ToLower(strings.TrimSpace(line))
		switch {
		case strings.HasPrefix(low, "public key:"):
			n.PublicKey = strings.TrimPrefix(bnBareRe.FindString(line), "0x")
		case strings.HasPrefix(low, "address:"):
			n.Address = bnHexRe.FindString(line)
		case strings.Contains(low, "bls pop"), strings.Contains(low, "proof of possession"):
			n.BLSPoP = bnHexRe.FindString(line)
		case strings.Contains(low, "bls public key"):
			n.BLSPubKey = bnHexRe.FindString(line)
		}
	}
	if n.Address == "" || n.PublicKey == "" || n.BLSPubKey == "" || n.BLSPoP == "" {
		return n, fmt.Errorf("incomplete bootnode output:\n%s", out)
	}
	return n, nil
}
