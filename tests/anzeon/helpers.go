// helpers.go gathers what the remaining Go-func cases in this package share:
// the system-contract addresses, the faucet key, the governance quorum walk,
// and the receipt/header readers. It was collected here when the ported cases
// (now DSL specs under tests/specs) were deleted, so the cases that stay do not
// each carry a copy. It goes away with them.
package anzeon

import (
	"context"
	"encoding/hex"
	"encoding/json"
	"math/big"
	"os"
	"path/filepath"
	"strings"
	"time"

	"github.com/0xmhha/chainbench/internal/accounts"
	"github.com/0xmhha/chainbench/internal/chains/stablenet/govbind"
	"github.com/0xmhha/chainbench/internal/core/keyring/store"
	"github.com/0xmhha/chainbench/internal/core/rpc"
	"github.com/0xmhha/chainbench/internal/testkit"
)

const accountManager = "0x0000000000000000000000000000000000B00003"

// addrTopic left-pads a 0x-hex address to a 32-byte (64-hex) indexed-topic word.
func addrTopic(addr string) string {
	a := strings.ToLower(strings.TrimPrefix(addr, "0x"))
	if len(a) >= 64 {
		return "0x" + a
	}
	return "0x" + strings.Repeat("0", 64-len(a)) + a
}

// burnRefundAmount is a modest payable burn value the proposer's coinbase covers.
var burnRefundAmount = big.NewInt(1_000_000_000_000_000) // 0.001 coin

// caller returns an accounts.EthCaller bound to the primary node.
func caller(t *testkit.T) accounts.EthCaller {
	p, ok := t.NodeSet().Primary()
	t.Truef(ok, "node set has no primary node")
	return rpc.Dial(p.RPCURL).EthCall
}

// councilProposalToQuorum proposes proposeSig(target) on GovCouncil from the
// first validator and approves it to quorum, returning the proposer for the
// post-execution reads.
func councilProposalToQuorum(t *testkit.T, proposeSig, target string) validator {
	ctx := t.Ctx()
	vals := discoverValidators(t)
	proposer := vals[0]

	data := accounts.EncodeCallArgs(proposeSig, accounts.Address(target))
	proposeHash, err := proposer.client.SendTransaction(ctx, rpc.SendTxArgs{
		From: proposer.addr, To: govCouncil, Data: data, Gas: govGas,
	})
	t.NoErr(err, proposeSig)

	proposalID := extractProposalID(t, proposer, proposeHash)
	approveToQuorum(t, govCouncil, proposalID, proposer, vals[1:])
	return proposer
}

// effectiveTip waits for the tx receipt and returns effectiveGasPrice minus the
// inclusion block's own baseFee — the priority fee actually charged. Using the
// inclusion block's baseFee (not a separately-read one) makes the charged tip
// exact, independent of any baseFee drift between blocks.
func effectiveTip(t *testkit.T, hash string) *big.Int {
	var rcpt struct {
		Status            string `json:"status"`
		EffectiveGasPrice string `json:"effectiveGasPrice"`
		BlockNumber       string `json:"blockNumber"`
	}
	t.WaitFor(func() bool {
		var raw json.RawMessage
		if err := t.Primary().Call(t.Ctx(), "eth_getTransactionReceipt", &raw, hash); err != nil {
			return false
		}
		if len(raw) == 0 || strings.TrimSpace(string(raw)) == "null" {
			return false
		}
		return json.Unmarshal(raw, &rcpt) == nil && rcpt.Status == "0x1"
	}, 90*time.Second, time.Second, "tx receipt with status 0x1")

	var block struct {
		BaseFeePerGas string `json:"baseFeePerGas"`
	}
	t.NoErr(t.Primary().Call(t.Ctx(), "eth_getBlockByNumber", &block, rcpt.BlockNumber, false),
		"eth_getBlockByNumber(inclusion)")
	egp := hexBig(t, rcpt.EffectiveGasPrice, "effectiveGasPrice")
	inclusionBase := hexBig(t, block.BaseFeePerGas, "baseFeePerGas")
	return new(big.Int).Sub(egp, inclusionBase)
}

// emittedEventForTarget reports whether a log with topic0 == keccak(eventSig) and
// the target address in its indexed topics was emitted by contract. It filters by
// topic0 and the left-padded target topic via eth_getLogs.
func emittedEventForTarget(t *testkit.T, c *rpc.Client, contract, eventSig, target string) bool {
	var logs []accounts.Log
	err := c.Call(t.Ctx(), "eth_getLogs", &logs, map[string]any{
		"address":   contract,
		"fromBlock": "earliest",
		"toBlock":   "latest",
		"topics":    []any{accounts.EventTopic(eventSig), addrTopic(target)},
	})
	return err == nil && len(logs) > 0
}

// gastipRecipient receives the probe transfer (regression c-01 TEST_ACC_B).
const gastipRecipient = "0x70997970C51812dc3A010C7d01b50e0d17dc79C8"

// govCouncil is the go-stablenet GovCouncil system contract (regression
// lib/common.sh GOV_COUNCIL).
const govCouncil = "0x0000000000000000000000000000000000001004"

// govMasterMinter is the go-stablenet GovMasterMinter system contract (regression
// lib/common.sh GOV_MASTER_MINTER).
const govMasterMinter = "0x0000000000000000000000000000000000001002"

// headerGasTip reads istanbul_getWbftExtraInfo(blockHex).gasTip as a big.Int.
// The field may be hex- or decimal-encoded, so parseBig handles both.
func headerGasTip(t *testkit.T, blockHex string) *big.Int {
	var extra struct {
		GasTip string `json:"gasTip"`
	}
	t.NoErr(t.Primary().Call(t.Ctx(), "istanbul_getWbftExtraInfo", &extra, blockHex),
		"istanbul_getWbftExtraInfo")
	return parseBig(extra.GasTip)
}

// hexBig parses a 0x-hex quantity, failing the test on a malformed value.
func hexBig(t *testkit.T, hexQty, what string) *big.Int {
	v, ok := new(big.Int).SetString(strings.TrimPrefix(hexQty, "0x"), 16)
	t.Truef(ok, "%s %q is hex", what, hexQty)
	return v
}

// latestBaseFee reads and parses the latest block's baseFeePerGas.
func latestBaseFee(t *testkit.T) *big.Int {
	var block struct {
		BaseFeePerGas string `json:"baseFeePerGas"`
	}
	t.NoErr(t.Primary().Call(t.Ctx(), "eth_getBlockByNumber", &block, "latest", false), "eth_getBlockByNumber")
	bf, ok := new(big.Int).SetString(strings.TrimPrefix(block.BaseFeePerGas, "0x"), 16)
	t.Truef(ok, "baseFeePerGas %q is not hex", block.BaseFeePerGas)
	return bf
}

// latestBlockHex returns the latest block number as a 0x-hex string.
func latestBlockHex(t *testkit.T) string {
	var latest string
	t.NoErr(t.Primary().Call(t.Ctx(), "eth_blockNumber", &latest), "eth_blockNumber")
	return latest
}

const nativeCoinAdapter = "0x0000000000000000000000000000000000001000"

// parseBig parses a JSON-RPC numeric string that may be 0x-hex or decimal (the
// WBFTExtra.gasTip field is not consistently hex). Unparseable input yields 0.
func parseBig(s string) *big.Int {
	s = strings.TrimSpace(s)
	if s == "" {
		return big.NewInt(0)
	}
	if strings.HasPrefix(s, "0x") || strings.HasPrefix(s, "0X") {
		if v, ok := new(big.Int).SetString(s[2:], 16); ok {
			return v
		}
		return big.NewInt(0)
	}
	if v, ok := new(big.Int).SetString(s, 10); ok {
		return v
	}
	return big.NewInt(0)
}

// proposeBurnFrom proposes a burn (raw proof bytes; the Boho v2 GovMinter accepts
// arbitrary proofData) with the payable value from the proposer, returning the
// proposalId.
func proposeBurnFrom(t *testkit.T, proposer validator) *big.Int {
	proof, _ := hex.DecodeString("deadbeef")
	proposeHash, err := proposer.client.SendTransaction(t.Ctx(), rpc.SendTxArgs{
		From: proposer.addr, To: govMinter, Data: govbind.ProposeBurnCall(proof),
		Gas: govGas, Value: "0x" + burnRefundAmount.Text(16),
	})
	t.NoErr(err, "proposeBurn")
	return extractProposalID(t, proposer, proposeHash)
}

// receiptHasTopic reports whether a successful tx's logs include a log with topic0.
func receiptHasTopic(ctx context.Context, c *rpc.Client, hash, topic0 string) bool {
	logs, ok := receiptLogs(ctx, c, hash)
	if !ok {
		return false
	}
	_, found := accounts.FindLog(logs, topic0)
	return found
}

// receiptReverted reports whether a mined tx has status 0x0.
func receiptReverted(ctx context.Context, c *rpc.Client, hash string) bool {
	raw, err := c.TxReceipt(ctx, hash)
	if err != nil || len(raw) == 0 {
		return false
	}
	var r struct {
		Status string `json:"status"`
	}
	return json.Unmarshal(raw, &r) == nil && r.Status == "0x0"
}

// fundedKey is the private key of a genesis-funded account the cases spend
// from. An operator-supplied CHAINBENCH_FUNDED_KEY wins (that is how a case
// acts on a chain chainbench did not compose); otherwise the first preset
// node's key is used — every preset node account is in the genesis alloc, so
// there is no separate faucet key to keep anywhere.
func fundedKey(t *testkit.T) []byte {
	if k, ok := t.FundedKey(); ok {
		return k
	}
	dir, ok := presetDir()
	if !ok {
		t.Skip("no funded key: set CHAINBENCH_FUNDED_KEY or run from the repository (keys/preset)")
	}
	p, err := store.LoadPreset(dir)
	t.NoErr(err, "load preset")
	nk, ok := p.Node(1)
	t.Truef(ok, "preset has no node1")
	return nk.Nodekey.Bytes()
}

// presetDir finds the repository's preset by walking up from the working
// directory, so the cases run from any directory inside the checkout.
func presetDir() (string, bool) {
	dir, err := os.Getwd()
	if err != nil {
		return "", false
	}
	for {
		cand := filepath.Join(dir, "keys", "preset")
		if _, err := os.Stat(filepath.Join(cand, "metadata.json")); err == nil {
			return cand, true
		}
		parent := filepath.Dir(dir)
		if parent == dir {
			return "", false
		}
		dir = parent
	}
}
