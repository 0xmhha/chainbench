//go:build e2e

package main

import (
	"context"
	"encoding/hex"
	"encoding/json"
	"errors"
	"fmt"
	"math/big"
	"os"
	"path/filepath"
	"reflect"
	"strings"
	"sync"
	"testing"
	"time"

	"github.com/0xmhha/chainbench/internal/accounts"
	"github.com/0xmhha/chainbench/internal/core/rpc"
)

// observedRawRPC forwards every request unchanged to the real endpoint and
// retains only public signed raw bytes, never wallet keys.
type observedRawRPC struct {
	client *rpc.Client
	mu     sync.Mutex
	raws   []string
}

func (o *observedRawRPC) Call(ctx context.Context, method string, result any, params ...any) error {
	if method == "eth_sendRawTransaction" {
		if len(params) != 1 {
			return fmt.Errorf("unexpected raw submission arguments")
		}
		raw, ok := params[0].(string)
		if !ok {
			return fmt.Errorf("raw submission is not hex")
		}
		o.mu.Lock()
		o.raws = append(o.raws, raw)
		o.mu.Unlock()
	}
	return o.client.Call(ctx, method, result, params...)
}
func (o *observedRawRPC) submitted() []string {
	o.mu.Lock()
	defer o.mu.Unlock()
	return append([]string(nil), o.raws...)
}

func rawRefusalBefore(t *testing.T, c *rpc.Client, sender, payer string) rawRefusalProof {
	t.Helper()
	p := rawRefusalProof{}
	var err error
	p.BeforeHeight, err = c.BlockNumber(t.Context())
	if err != nil {
		t.Fatal(err)
	}
	p.NonceBefore, err = c.NonceAt(t.Context(), sender)
	if err != nil {
		t.Fatal(err)
	}
	p.SenderBefore = balanceOf(t, c, sender).String()
	p.PayerBefore = balanceOf(t, c, payer).String()
	return p
}

func completeRawRefusal(t *testing.T, c *rpc.Client, p rawRefusalProof, raw, returned string, submitErr error, sender, payer, reason string) rawRefusalProof {
	t.Helper()
	var rejection *rpc.ServerError
	if !errors.As(submitErr, &rejection) || rejection == nil {
		t.Fatal("not actual node submission refusal", submitErr)
	}
	p.Raw, p.ReturnedHash, p.Method, p.Code, p.Message = raw, returned, rejection.Method, rejection.Code, rejection.Message
	bytes, err := hex.DecodeString(strings.TrimPrefix(raw, "0x"))
	if err != nil {
		t.Fatal(err)
	}
	p.Hash = rawSubmissionHash(bytes)
	for _, phase := range []string{"initial", "later"} {
		if phase == "later" {
			waitForHead(t, c, p.BeforeHeight+3, 90*time.Second)
		}
		for _, method := range []string{"eth_getTransactionByHash", "eth_getTransactionReceipt"} {
			var result json.RawMessage
			if err := c.Call(t.Context(), method, &result, p.Hash); err != nil {
				t.Fatal(err)
			}
			p.Lookups = append(p.Lookups, rawNullLookup{Phase: phase, Method: method, Hash: p.Hash, Result: string(result)})
		}
	}
	p.AfterHeight, err = c.BlockNumber(t.Context())
	if err != nil {
		t.Fatal(err)
	}
	p.NonceAfter, err = c.NonceAt(t.Context(), sender)
	if err != nil {
		t.Fatal(err)
	}
	p.SenderAfter = balanceOf(t, c, sender).String()
	p.PayerAfter = balanceOf(t, c, payer).String()
	if err := verifyRawRefusal(p, reason); err != nil {
		t.Fatal(err, p)
	}
	return p
}

func TestWemixRawSubmissionRefusalsE2E(t *testing.T) {
	from, to := os.Getenv("CHAINBENCH_E2E_FROM_BIN"), os.Getenv("CHAINBENCH_E2E_TO_BIN")
	if from == "" || to == "" {
		t.Skip("set native handoff binaries")
	}
	senderKey, sender, err := accounts.GenerateKey()
	if err != nil {
		t.Fatal(err)
	}
	payerKey, payer, err := accounts.GenerateKey()
	if err != nil {
		t.Fatal(err)
	}
	overlay := map[string]any{"alloc": map[string]any{sender: map[string]any{"balance": handoffBalance}, payer: map[string]any{"balance": handoffBalance}}}
	workspace, url := runGovHandoffWorkspace(t, from, to, writeHandoffCase(t, overlay))
	root := filepath.Dir(workspace)
	c := rpc.Dial(url)
	ctx, cancel := context.WithTimeout(t.Context(), 3*time.Minute)
	defer cancel()
	observed := &observedRawRPC{client: c}
	provider, err := accounts.ForChainWithRPC("wbft", func(string) accounts.RPCRequester { return observed })
	if err != nil {
		t.Fatal(err)
	}
	wallet, err := provider.OpenWallet(ctx, senderKey, url)
	if err != nil {
		t.Fatal(err)
	}
	proof := map[string]any{"sender": sender, "payer": payer, "genesis": filepath.Join(workspace, "genesis.json"), "scope": "TX012 raw legacy block gasLimit+1; TX014 only inner Sender.S corrupted while fee-payer signature remains over original signed sender. Actual typed node errors and raw hash transaction/receipt null immediately and after >=3 blocks; nonce and funded non-validator balances unchanged. Valid same-nonce dual-signature control mines. Current invalid-feePayer binding refusal is recorded, not described as ErrInvalidSig. Not public DSL re-execution/full87/Linux/mainnet/legacy/final Web."}
	defer func() {
		b, e := json.MarshalIndent(proof, "", "  ")
		if e != nil {
			t.Error(e)
			return
		}
		if e = os.WriteFile(filepath.Join(root, "reports/raw-refusal-proof.json"), b, 0600); e != nil {
			t.Error(e)
		}
	}()
	chainID, err := c.ChainID(ctx)
	if err != nil {
		t.Fatal(err)
	}
	proof["chainId"] = chainID
	before := rawRefusalBefore(t, c, sender, payer)
	var header map[string]any
	if err := c.Call(ctx, "eth_getBlockByNumber", &header, hexUint(before.BeforeHeight), false); err != nil {
		t.Fatal(err)
	}
	gasLimit, ok := new(big.Int).SetString(strings.TrimPrefix(fmt.Sprint(header["gasLimit"]), "0x"), 16)
	if !ok || !gasLimit.IsUint64() || gasLimit.Uint64() == ^uint64(0) {
		t.Fatal("invalid actual block gas limit")
	}
	nonce := before.NonceBefore
	recipient := "0x00000000000000000000000000000000c0ffee14"
	returned, sendErr := wallet.SendLegacyTx(ctx, accounts.LegacyTxArgs{ToHex: recipient, Value: big.NewInt(1), Gas: gasLimit.Uint64() + 1, GasPrice: big.NewInt(1000000000), Nonce: &nonce})
	raws := observed.submitted()
	if len(raws) != 1 {
		t.Fatal("legacy transaction not actually submitted once", len(raws), sendErr)
	}
	gasProof := completeRawRefusal(t, c, before, raws[0], returned, sendErr, sender, payer, "exceeds block gas limit")
	proof["gasHeader"], proof["gasRefusal"] = header, gasProof
	raw, err := hex.DecodeString(raws[0][2:])
	if err != nil {
		t.Fatal(err)
	}
	decoded, n, err := decodeForkRLP(raw, 0)
	if err != nil || n != len(raw) || !decoded.List || len(decoded.Items) != 9 {
		t.Fatal("not actual signed legacy RLP", err)
	}
	field := func(i int) *big.Int { return new(big.Int).SetBytes(decoded.Items[i].Bytes) }
	signingV := new(big.Int).Add(new(big.Int).Mul(new(big.Int).SetUint64(chainID), big.NewInt(2)), big.NewInt(35))
	if field(0).Uint64() != nonce || field(1).Cmp(big.NewInt(1000000000)) != 0 || field(2).Cmp(new(big.Int).Add(gasLimit, big.NewInt(1))) != 0 || "0x"+hex.EncodeToString(decoded.Items[3].Bytes) != recipient || field(4).Cmp(big.NewInt(1)) != 0 || len(decoded.Items[5].Bytes) != 0 || new(big.Int).Sub(field(6), signingV).Sign() < 0 || new(big.Int).Sub(field(6), signingV).Cmp(big.NewInt(1)) > 0 || field(7).Sign() == 0 || field(8).Sign() == 0 {
		t.Fatal("signed legacy input differs from original gas-limit boundary")
	}
	var priceHex string
	if err := c.Call(ctx, "eth_gasPrice", &priceHex); err != nil {
		t.Fatal(err)
	}
	price, ok := new(big.Int).SetString(strings.TrimPrefix(priceHex, "0x"), 16)
	if !ok || price.Sign() <= 0 {
		t.Fatal("invalid node gas price")
	}
	feeCap := new(big.Int).Mul(price, big.NewInt(2))
	tip := new(big.Int).Set(price)
	before = rawRefusalBefore(t, c, sender, payer)
	nonce = before.NonceBefore
	tampered, err := accounts.EncodeFeeDelegatedTampered(senderKey, payerKey, recipient, big.NewInt(1), int64(chainID), nonce, feeCap, tip, "sender")
	if err != nil {
		t.Fatal(err)
	}
	returned = ""
	sendErr = c.Call(ctx, "eth_sendRawTransaction", &returned, "0x"+hex.EncodeToString(tampered))
	tamperProof := completeRawRefusal(t, c, before, "0x"+hex.EncodeToString(tampered), returned, sendErr, sender, payer, "fee delegation: invalid feePayer")
	proof["tamperedRefusal"] = tamperProof
	validHash, err := wallet.SendFeeDelegatedTx(ctx, payerKey, accounts.DynamicTxArgs{ToHex: recipient, Value: big.NewInt(1), Gas: 21000, GasFeeCap: feeCap, GasTipCap: tip, Nonce: &nonce})
	if err != nil || validHash == "" {
		t.Fatal("unchanged signed control not admitted", err)
	}
	raws = observed.submitted()
	if len(raws) != 2 {
		t.Fatal("control not actually submitted once", len(raws))
	}
	validRaw, err := hex.DecodeString(raws[1][2:])
	if err != nil || rawSubmissionHash(validRaw) != validHash {
		t.Fatal("control raw/hash differs", err)
	}
	bad, bn, err := decodeForkRLP(tampered[1:], 0)
	if err != nil || bn != len(tampered)-1 || tampered[0] != 0x16 {
		t.Fatal(err)
	}
	good, gn, err := decodeForkRLP(validRaw[1:], 0)
	if err != nil || gn != len(validRaw)-1 || validRaw[0] != 0x16 || !good.List || !bad.List || len(good.Items) != 5 || len(bad.Items) != 5 || len(good.Items[0].Items) != 12 || len(bad.Items[0].Items) != 12 {
		t.Fatal("not exact fee-delegated envelopes", err)
	}
	if !reflect.DeepEqual(good.Items[1:], bad.Items[1:]) || !reflect.DeepEqual(good.Items[0].Items[:11], bad.Items[0].Items[:11]) || new(big.Int).SetBytes(bad.Items[0].Items[11].Bytes).Cmp(new(big.Int).Xor(new(big.Int).SetBytes(good.Items[0].Items[11].Bytes), big.NewInt(0xff))) != 0 {
		t.Fatal("more than sender S changed between refused and valid raw transactions")
	}
	receipt := policyReceipt(t, c, validHash)
	var tx map[string]any
	if err := c.Call(ctx, "eth_getTransactionByHash", &tx, validHash); err != nil {
		t.Fatal(err)
	}
	if tx["hash"] != validHash || tx["type"] != "0x16" || !strings.EqualFold(fmt.Sprint(tx["from"]), sender) || tx["blockHash"] != receipt["blockHash"] || tx["blockNumber"] != receipt["blockNumber"] {
		t.Fatal("actual signed control identity differs", tx, receipt)
	}
	proof["validRaw"], proof["validHash"], proof["validTransaction"], proof["validReceipt"] = raws[1], validHash, tx, receipt
}
