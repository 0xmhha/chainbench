package main

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

	"github.com/spf13/cobra"
)

// newKeysCmd is the `keys` command group. Today it generates preset key sets.
func newKeysCmd() *cobra.Command {
	c := &cobra.Command{Use: "keys", Short: "Generate/manage local preset key sets"}
	c.AddCommand(newKeysGenerateCmd())
	return c
}

// genNode is one generated node's material.
type genNode struct {
	Index     int    `json:"index"`
	Nodekey   string `json:"nodekey"`
	PublicKey string `json:"publicKey"`
	Address   string `json:"address"`
	BLSPubKey string `json:"blsPublicKey"`
	BLSPoP    string `json:"blsPoP"`
	Enode     string `json:"enode"`
}

// genMeta mirrors keys/preset/metadata.json (the shape keys.LoadPreset reads).
type genMeta struct {
	Description           string                    `json:"description"`
	Warning               string                    `json:"warning"`
	Password              string                    `json:"password"`
	Validators            []string                  `json:"validators"`
	BLSPublicKeys         []string                  `json:"blsPublicKeys"`
	ExtraData             string                    `json:"extraData"`
	SystemContractMembers string                    `json:"systemContractMembers"`
	SystemContractBLSKeys string                    `json:"systemContractBlsKeys"`
	Alloc                 map[string]map[string]any `json:"alloc"`
	Nodes                 []genNode                 `json:"nodes"`
}

func newKeysGenerateCmd() *cobra.Command {
	var (
		nodes      int
		validators int
		bootnode   string
		binary     string
		out        string
		password   string
		basePort   int
		balance    string
	)
	cmd := &cobra.Command{
		Use:   "generate",
		Short: "Generate an N-node preset key set (nodekeys, BLS, keystores, metadata)",
		Long: "Generates a preset key set the local harness consumes (keys.LoadPreset):\n" +
			"random nodekeys, their derived address + BLS public key/PoP (via the go-wbft\n" +
			"`bootnode -writeaddress` tool), an encrypted keystore per node (via the node\n" +
			"binary's `account import`), and a metadata.json. The croissant/WBFT validator\n" +
			"set lives in the genesis config (croissant.init), so extraData is a plain\n" +
			"32-byte vanity — no istanbul RLP encoding is needed. Use it to build networks\n" +
			"larger than the committed 5-node preset (e.g. the n=6 quorum cases).",
		RunE: func(cmd *cobra.Command, _ []string) error {
			if nodes < 1 {
				return fmt.Errorf("--nodes must be >= 1")
			}
			if validators < 1 || validators > nodes {
				validators = nodes
			}
			if _, err := exec.LookPath(bootnode); err != nil {
				if _, e2 := os.Stat(bootnode); e2 != nil {
					return fmt.Errorf("--bootnode %q not found (build go-wbft/cmd/bootnode)", bootnode)
				}
			}
			if _, err := os.Stat(binary); err != nil {
				return fmt.Errorf("--binary %q not found", binary)
			}
			if err := os.MkdirAll(out, 0o755); err != nil {
				return err
			}
			pwFile := filepath.Join(out, "password")
			if err := os.WriteFile(pwFile, []byte(password), 0o600); err != nil {
				return err
			}

			meta := genMeta{
				Description: fmt.Sprintf("Generated preset: %d nodes (%d validators). chainbench keys generate.", nodes, validators),
				Warning:     "TEST FIXTURE ONLY — do not import to mainnet/testnet.",
				Password:    password,
				ExtraData:   "0x" + strings.Repeat("00", 32),
				Alloc:       map[string]map[string]any{},
			}
			outw := cmd.OutOrStdout()
			for i := 1; i <= nodes; i++ {
				n, err := generateNode(i, bootnode, binary, out, pwFile, basePort)
				if err != nil {
					return fmt.Errorf("node %d: %w", i, err)
				}
				meta.Nodes = append(meta.Nodes, n)
				meta.Alloc[strings.TrimPrefix(n.Address, "0x")] = map[string]any{"balance": balance}
				if i <= validators {
					meta.Validators = append(meta.Validators, n.Address)
					meta.BLSPublicKeys = append(meta.BLSPublicKeys, n.BLSPubKey)
				}
				fmt.Fprintf(outw, "node %d  %s  bls=%s…\n", i, n.Address, n.BLSPubKey[:14])
			}
			meta.SystemContractMembers = strings.Join(meta.Validators, ",")
			meta.SystemContractBLSKeys = strings.Join(meta.BLSPublicKeys, ",")

			b, err := json.MarshalIndent(meta, "", "  ")
			if err != nil {
				return err
			}
			if err := os.WriteFile(filepath.Join(out, "metadata.json"), b, 0o644); err != nil {
				return err
			}
			fmt.Fprintf(outw, "wrote %d-node preset (%d validators) to %s\n", nodes, validators, out)
			return nil
		},
	}
	cmd.Flags().IntVar(&nodes, "nodes", 0, "total nodes to generate")
	cmd.Flags().IntVar(&validators, "validators", 0, "how many nodes are validators (default: all)")
	cmd.Flags().StringVar(&bootnode, "bootnode", "bootnode", "path to the go-wbft bootnode tool (address+BLS derivation)")
	cmd.Flags().StringVar(&binary, "binary", "", "path to the node binary (gwemix) for keystore `account import`")
	cmd.Flags().StringVar(&out, "out", "", "output preset directory")
	cmd.Flags().StringVar(&password, "password", "1", "keystore password")
	cmd.Flags().IntVar(&basePort, "base-p2p", 30301, "base p2p port for enode URLs")
	cmd.Flags().StringVar(&balance, "balance", "0x200000000000000000000000000000000000000000000000000000000000000", "genesis balance per node (0x-hex wei)")
	_ = cmd.MarkFlagRequired("nodes")
	_ = cmd.MarkFlagRequired("binary")
	_ = cmd.MarkFlagRequired("out")
	return cmd
}

var (
	bnHexRe  = regexp.MustCompile(`0x[0-9a-fA-F]+`)
	bnBareRe = regexp.MustCompile(`[0-9a-fA-F]{40,}`)
)

// generateNode creates node i's material: a random nodekey, its derived address +
// BLS pubkey/PoP + devp2p public key (via bootnode), and an encrypted keystore
// (via the binary's account import). It writes the per-node dir and returns the
// node metadata.
func generateNode(i int, bootnode, binary, out, pwFile string, basePort int) (genNode, error) {
	nodeDir := filepath.Join(out, fmt.Sprintf("node%d", i))
	if err := os.MkdirAll(nodeDir, 0o755); err != nil {
		return genNode{}, err
	}

	// Random devp2p/account private key.
	raw := make([]byte, 32)
	if _, err := rand.Read(raw); err != nil {
		return genNode{}, err
	}
	nodekey := hex.EncodeToString(raw)
	if err := os.WriteFile(filepath.Join(nodeDir, "nodekey"), []byte(nodekey), 0o600); err != nil {
		return genNode{}, err
	}

	// Derive address + BLS material + devp2p public key.
	res, err := exec.Command(bootnode, "-nodekeyhex", nodekey, "-writeaddress").CombinedOutput()
	if err != nil {
		return genNode{}, fmt.Errorf("bootnode writeaddress: %w: %s", err, res)
	}
	n, err := parseBootnode(string(res))
	if err != nil {
		return genNode{}, err
	}
	n.Index = i
	n.Nodekey = nodekey
	n.Enode = fmt.Sprintf("enode://%s@127.0.0.1:%d?discport=0", n.PublicKey, basePort+i-1)

	// Import the key as an encrypted keystore account.
	keyFile := filepath.Join(nodeDir, "rawkey")
	if err := os.WriteFile(keyFile, []byte(nodekey), 0o600); err != nil {
		return genNode{}, err
	}
	defer os.Remove(keyFile)
	imp := exec.Command(binary, "account", "import", "--datadir", nodeDir, "--password", pwFile, keyFile)
	if b, err := imp.CombinedOutput(); err != nil {
		return genNode{}, fmt.Errorf("account import: %w: %s", err, b)
	}

	for name, val := range map[string]string{
		"address":    n.Address,
		"pubkey":     n.PublicKey,
		"bls_pubkey": n.BLSPubKey,
	} {
		if err := os.WriteFile(filepath.Join(nodeDir, name), []byte(val), 0o644); err != nil {
			return genNode{}, err
		}
	}
	return n, nil
}

// parseBootnode extracts the address, devp2p public key, BLS public key and PoP
// from `bootnode -writeaddress` output.
func parseBootnode(out string) (genNode, error) {
	var n genNode
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
