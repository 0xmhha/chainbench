// Package stablenet is the adapter for the stablenet chain type.
//
// ExtraStartFlags + ConsensusRpcNamespace mirror the bash adapter
// (lib/adapters/stablenet.sh) byte-for-byte. GenerateGenesis and
// GenerateToml port the Python genesis + per-node TOML builders
// from the same file. Semantic equivalence is locked via golden-file
// contract tests under testdata/golden/.
package stablenet

import (
	"context"
	"encoding/json"
	"fmt"
	"maps"
	"os"
	"path/filepath"
	"strings"

	"github.com/0xmhha/chainbench/network/internal/adapters/spec"
)

// Adapter is the stablenet implementation of spec.Adapter.
type Adapter struct{}

// New returns a zero-configuration stablenet adapter.
func New() *Adapter { return &Adapter{} }

// create2ProxyAddress is Arachnid's deterministic deployment proxy; preserved
// in mixed case to match the bash adapter's output.
const create2ProxyAddress = "4e59b44847b379578588920cA78FbF26c0B4956C"

// create2ProxyCode is the EVM bytecode Arachnid's proxy deploys.
const create2ProxyCode = "0x7fffffffffffffffffffffffffffffffffffffffffffffffffffffffffffffffe03601600081602082378035828234f58015156014578182fd5b80825250506014600cf3"

// defaultAllocBalance is the balance bash assigns to each validator/metadata
// node when no user override is present. Stays in sync with stablenet.sh:74.
const defaultAllocBalance = "0x84595161401484a000000"

// GenerateGenesis ports lib/adapters/stablenet.sh's Python genesis builder.
// Precedence of fallbacks: profile.chain.chain_id (default 8283),
// overrides.wbft.* (default 2/1/140/0/null), metadata.extraData (falls back
// to overrides.extraData, then "0x00"). Writes the rendered genesis to
// input.OutputPath after confirming it parses as JSON.
func (*Adapter) GenerateGenesis(_ context.Context, in spec.GenesisInput) error {
	if len(in.TemplateBytes) == 0 {
		return fmt.Errorf("stablenet.GenerateGenesis: template bytes required")
	}
	if in.NumValidators <= 0 {
		return fmt.Errorf("stablenet.GenerateGenesis: NumValidators must be > 0")
	}
	if in.OutputPath == "" {
		return fmt.Errorf("stablenet.GenerateGenesis: OutputPath required")
	}

	chain := getMap(in.Profile, "chain")
	genesis := getMap(in.Profile, "genesis")
	overrides := getMap(genesis, "overrides")
	wbft := getMap(overrides, "wbft")

	chainID := getInt(chain, "chain_id", 8283)
	reqTimeout := getInt(wbft, "requestTimeoutSeconds", 2)
	blockPeriod := getInt(wbft, "blockPeriodSeconds", 1)
	epochLength := getInt(wbft, "epochLength", 140)
	proposerPolicy := getInt(wbft, "proposerPolicy", 0)
	maxReqTimeout, hasMaxReq := wbft["maxRequestTimeoutSeconds"]

	nodes, err := metadataNodes(in.Metadata)
	if err != nil {
		return err
	}
	if in.NumValidators > len(nodes) {
		return fmt.Errorf("stablenet.GenerateGenesis: NumValidators=%d exceeds metadata.nodes (%d)",
			in.NumValidators, len(nodes))
	}
	validatorNodes := nodes[:in.NumValidators]
	validators := make([]string, len(validatorNodes))
	blsKeys := make([]string, len(validatorNodes))
	for i, n := range validatorNodes {
		validators[i] = getString(n, "address", "")
		blsKeys[i] = getString(n, "blsPublicKey", "")
	}
	membersCSV := strings.Join(validators, ",")
	blsCSV := strings.Join(blsKeys, ",")

	sysContracts := cloneMap(getMap(overrides, "systemContracts"))
	if gv := getMap(sysContracts, "govValidator"); gv != nil {
		params := ensureParams(gv)
		params["validators"] = membersCSV
		params["members"] = membersCSV
		params["blsPublicKeys"] = blsCSV
	}
	for _, name := range []string{"govMinter", "govMasterMinter", "govCouncil"} {
		if m := getMap(sysContracts, name); m != nil {
			ensureParams(m)["members"] = membersCSV
		}
	}

	alloc := buildAlloc(nodes, getMap(overrides, "alloc"))

	extraData := pickExtraData(in.Metadata, overrides)

	validatorsJSON, _ := json.Marshal(validators)
	blsJSON, _ := json.Marshal(blsKeys)
	sysJSON, err := json.MarshalIndent(sysContracts, "", "    ")
	if err != nil {
		return fmt.Errorf("stablenet.GenerateGenesis: marshal systemContracts: %w", err)
	}
	allocJSON, err := json.MarshalIndent(alloc, "", "    ")
	if err != nil {
		return fmt.Errorf("stablenet.GenerateGenesis: marshal alloc: %w", err)
	}

	maxReqStr := "null"
	if hasMaxReq && maxReqTimeout != nil {
		maxReqStr = fmt.Sprintf("%v", maxReqTimeout)
	}

	result := string(in.TemplateBytes)
	replacements := []struct{ k, v string }{
		{"__CHAIN_ID__", fmt.Sprintf("%d", chainID)},
		{"__REQUEST_TIMEOUT_SECONDS__", fmt.Sprintf("%d", reqTimeout)},
		{"__BLOCK_PERIOD_SECONDS__", fmt.Sprintf("%d", blockPeriod)},
		{"__EPOCH_LENGTH__", fmt.Sprintf("%d", epochLength)},
		{"__PROPOSER_POLICY__", fmt.Sprintf("%d", proposerPolicy)},
		{"__MAX_REQUEST_TIMEOUT_SECONDS__", maxReqStr},
	}
	for _, r := range replacements {
		result = strings.ReplaceAll(result, r.k, r.v)
	}
	// JSON-array + object placeholders: both quoted and bare forms, quoted first.
	for _, r := range []struct{ k, v string }{
		{`"__VALIDATORS_JSON__"`, string(validatorsJSON)},
		{"__VALIDATORS_JSON__", string(validatorsJSON)},
		{`"__BLS_PUBLIC_KEYS_JSON__"`, string(blsJSON)},
		{"__BLS_PUBLIC_KEYS_JSON__", string(blsJSON)},
		{`"__SYSTEM_CONTRACTS_JSON__"`, string(sysJSON)},
		{"__SYSTEM_CONTRACTS_JSON__", string(sysJSON)},
		{`"__ALLOC_JSON__"`, string(allocJSON)},
		{"__ALLOC_JSON__", string(allocJSON)},
		{`"__EXTRA_DATA__"`, fmt.Sprintf("%q", extraData)},
		{"__EXTRA_DATA__", extraData},
	} {
		result = strings.ReplaceAll(result, r.k, r.v)
	}

	var probe any
	if err := json.Unmarshal([]byte(result), &probe); err != nil {
		return fmt.Errorf("stablenet.GenerateGenesis: rendered template is not valid JSON: %w", err)
	}

	if err := os.MkdirAll(filepath.Dir(in.OutputPath), 0o755); err != nil {
		return fmt.Errorf("stablenet.GenerateGenesis: mkdir: %w", err)
	}
	if err := os.WriteFile(in.OutputPath, []byte(result), 0o644); err != nil {
		return fmt.Errorf("stablenet.GenerateGenesis: write: %w", err)
	}
	return nil
}

// GenerateToml ports lib/adapters/stablenet.sh's per-node TOML emitter. For
// each of the first in.TotalNodes metadata entries it builds a shared static-
// nodes block (enodes with monotonic p2p ports), then renders a per-node
// config_node<N>.toml into in.WriteDir with sequential port offsets, a
// validator-only miner section, and per-node keystore + ethstats strings.
func (*Adapter) GenerateToml(_ context.Context, in spec.TomlInput) error {
	if len(in.TemplateBytes) == 0 {
		return fmt.Errorf("stablenet.GenerateToml: template bytes required")
	}
	if in.TotalNodes <= 0 {
		return fmt.Errorf("stablenet.GenerateToml: TotalNodes must be > 0")
	}
	if in.WriteDir == "" {
		return fmt.Errorf("stablenet.GenerateToml: WriteDir required")
	}

	nodes, err := metadataNodes(in.Metadata)
	if err != nil {
		return err
	}
	if in.TotalNodes > len(nodes) {
		return fmt.Errorf("stablenet.GenerateToml: TotalNodes=%d exceeds metadata.nodes (%d)",
			in.TotalNodes, len(nodes))
	}
	active := nodes[:in.TotalNodes]

	// Shared static-nodes block: every node embeds the same enode list.
	staticEntries := make([]string, len(active))
	for i, n := range active {
		idx := getInt(n, "index", int64(i+1))
		port := int64(in.BaseP2P) + idx - 1
		enode := fmt.Sprintf("enode://%s@127.0.0.1:%d?discport=0", getString(n, "publicKey", ""), port)
		staticEntries[i] = fmt.Sprintf("    %q", enode)
	}
	discoverySection := "NoDiscovery = true\nStaticNodes = [\n" + strings.Join(staticEntries, ",\n") + "\n]"

	if err := os.MkdirAll(in.WriteDir, 0o755); err != nil {
		return fmt.Errorf("stablenet.GenerateToml: mkdir: %w", err)
	}

	template := string(in.TemplateBytes)
	for i := 1; i <= in.TotalNodes; i++ {
		offset := i - 1
		minerSection := ""
		if i <= in.NumValidators {
			minerSection = "[Eth.Miner]\nRecommit = \"2s\""
		}
		keystoreDir := filepath.Join(in.WriteDir, "keystores", fmt.Sprintf("node%d", i))
		ethstatsURL := fmt.Sprintf("node%d:local@localhost:3000", i)

		result := template
		for _, r := range []struct{ k, v string }{
			{"__SYNC_MODE__", "full"},
			{"__MINER_SECTION__", minerSection},
			{"__KEYSTORE_DIR__", keystoreDir},
			{"__AUTH_PORT__", fmt.Sprintf("%d", in.BaseAuth+offset)},
			{"__HTTP_HOST__", "0.0.0.0"},
			{"__HTTP_PORT__", fmt.Sprintf("%d", in.BaseHTTP+offset)},
			{"__P2P_PORT__", fmt.Sprintf("%d", in.BaseP2P+offset)},
			{"__DISCOVERY_SECTION__", discoverySection},
			{"__ETHSTATS_URL__", ethstatsURL},
			{"__METRICS_ENABLED__", "true"},
			{"__METRICS_HTTP__", "127.0.0.1"},
			{"__METRICS_PORT__", fmt.Sprintf("%d", in.BaseMetrics+offset)},
		} {
			result = strings.ReplaceAll(result, r.k, r.v)
		}

		outPath := filepath.Join(in.WriteDir, fmt.Sprintf("config_node%d.toml", i))
		if err := os.WriteFile(outPath, []byte(result), 0o644); err != nil {
			return fmt.Errorf("stablenet.GenerateToml: write %s: %w", outPath, err)
		}
	}
	return nil
}

// ExtraStartFlags returns the CLI flags appended to the stablenet client
// start command. Validators additionally get `--mine` so they participate
// in block production.
func (*Adapter) ExtraStartFlags(role spec.Role) string {
	flags := "--allow-insecure-unlock --rpc.enabledeprecatedpersonal --rpc.allow-unprotected-txs"
	if role == spec.RoleValidator {
		flags += " --mine"
	}
	return flags
}

// ConsensusRpcNamespace returns the consensus RPC module name exposed by
// stablenet (WBFT consensus uses the `istanbul` namespace).
func (*Adapter) ConsensusRpcNamespace() string { return "istanbul" }

// --- helpers -----------------------------------------------------------------

func getMap(m map[string]any, key string) map[string]any {
	if m == nil {
		return nil
	}
	if v, ok := m[key].(map[string]any); ok {
		return v
	}
	return nil
}

func getString(m map[string]any, key, def string) string {
	if v, ok := m[key].(string); ok {
		return v
	}
	return def
}

// getInt coerces JSON-decoded numeric values (float64) or integer-typed
// values to int64, falling back to def.
func getInt(m map[string]any, key string, def int64) int64 {
	if m == nil {
		return def
	}
	switch v := m[key].(type) {
	case float64:
		return int64(v)
	case int:
		return int64(v)
	case int64:
		return v
	case json.Number:
		n, err := v.Int64()
		if err != nil {
			return def
		}
		return n
	}
	return def
}

func cloneMap(src map[string]any) map[string]any {
	if src == nil {
		return map[string]any{}
	}
	return maps.Clone(src)
}

func ensureParams(contract map[string]any) map[string]any {
	if p, ok := contract["params"].(map[string]any); ok {
		return p
	}
	p := map[string]any{}
	contract["params"] = p
	return p
}

// metadataNodes drills into metadata["nodes"] and returns the slice coerced
// to []map[string]any. Returns error when the key is missing or not an array.
func metadataNodes(meta map[string]any) ([]map[string]any, error) {
	raw, ok := meta["nodes"].([]any)
	if !ok {
		return nil, fmt.Errorf("metadata.nodes missing or not an array")
	}
	out := make([]map[string]any, 0, len(raw))
	for i, entry := range raw {
		m, ok := entry.(map[string]any)
		if !ok {
			return nil, fmt.Errorf("metadata.nodes[%d] is not an object", i)
		}
		out = append(out, m)
	}
	return out, nil
}

// buildAlloc composes the alloc map: default balances keyed by stripped
// (0x-removed, case preserved) addresses for every metadata node, user
// overrides layered on top (with quote-stripping to match the Python source),
// and the CREATE2 proxy appended when not already present (case-insensitive
// comparison).
func buildAlloc(nodes []map[string]any, overrideAlloc map[string]any) map[string]any {
	alloc := make(map[string]any, len(nodes)+len(overrideAlloc)+1)
	for _, n := range nodes {
		addr := stripHexPrefix(getString(n, "address", ""))
		if addr == "" {
			continue
		}
		alloc[addr] = map[string]any{"balance": defaultAllocBalance}
	}
	for addr, balance := range overrideAlloc {
		clean := stripHexPrefix(strings.Trim(addr, `"`))
		if clean == "" {
			continue
		}
		alloc[clean] = map[string]any{"balance": fmt.Sprintf("%v", balance)}
	}
	hasCreate2 := false
	c2Lower := strings.ToLower(create2ProxyAddress)
	for k := range alloc {
		if strings.ToLower(k) == c2Lower {
			hasCreate2 = true
			break
		}
	}
	if !hasCreate2 {
		alloc[create2ProxyAddress] = map[string]any{
			"code":    create2ProxyCode,
			"balance": "0x0",
		}
	}
	return alloc
}

// stripHexPrefix removes every "0x" / "0X" occurrence from addr (matching
// Python's str.replace behavior) and preserves case. Empty input -> "".
func stripHexPrefix(addr string) string {
	addr = strings.ReplaceAll(addr, "0x", "")
	addr = strings.ReplaceAll(addr, "0X", "")
	return addr
}

// pickExtraData implements metadata.extraData > overrides.extraData > "0x00".
func pickExtraData(meta, overrides map[string]any) string {
	if v, ok := meta["extraData"].(string); ok {
		return v
	}
	if v, ok := overrides["extraData"].(string); ok {
		return v
	}
	return "0x00"
}
