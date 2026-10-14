//go:build e2e

package main

import (
	"context"
	"encoding/hex"
	"encoding/json"
	"fmt"
	"os"
	"strings"
	"testing"
	"time"

	"github.com/0xmhha/chainbench/internal/consensus/wbft"
	"github.com/0xmhha/chainbench/internal/core/rpc"
)

func TestWemixWbftAggregateSealE2E(t *testing.T) {
	from, to := os.Getenv("CHAINBENCH_E2E_FROM_BIN"), os.Getenv("CHAINBENCH_E2E_TO_BIN")
	if from == "" || to == "" {
		t.Skip("set FROM/TO binaries for actual aggregate proof")
	}
	url := runGovHandoffOverlay(t, from, to, map[string]any{"config": map[string]any{"croissant": map[string]any{"wBFT": map[string]any{"blockPeriodSeconds": 1, "requestTimeoutSeconds": 1, "epochLength": 10}}}})
	c := rpc.Dial(url)
	ctx, cancel := context.WithTimeout(t.Context(), 2*time.Minute)
	defer cancel()
	start, err := c.BlockNumber(ctx)
	if err != nil || start <= 20 {
		t.Fatal("not beyond fork", start, err)
	}
	waitForHead(t, c, start+10, 90*time.Second)
	type epoch struct{ Validators []struct{ Addr, BLS string } }
	cache := map[uint64]*epoch{}
	proofs := []map[string]any{}
	ordinary, boundary := 0, 0
	for n := start; n <= start+10; n++ {
		tag := fmt.Sprintf("0x%x", n)
		var h map[string]any
		if err = c.Call(ctx, "eth_getBlockByNumber", &h, tag, false); err != nil {
			t.Fatal(err)
		}
		var addresses []string
		if err = c.Call(ctx, "istanbul_getValidators", &addresses, tag); err != nil || len(addresses) < 4 {
			t.Fatal("missing current validators", err)
		}
		var contextEpoch *epoch
		var epochBlock uint64
		for p := n - 1; p >= 20; p-- {
			e, ok := cache[p]
			if !ok {
				var x struct{ EpochInfo *epoch }
				if err = c.Call(ctx, "istanbul_getWbftExtraInfo", &x, fmt.Sprintf("0x%x", p)); err != nil {
					t.Fatal(err)
				}
				e = x.EpochInfo
				cache[p] = e
			}
			if e != nil && len(e.Validators) > 0 {
				contextEpoch = e
				epochBlock = p
				break
			}
			if p == 20 {
				break
			}
		}
		if contextEpoch == nil {
			t.Fatal("no preceding actual epoch keys", n)
		}
		byAddress := map[string]string{}
		for _, v := range contextEpoch.Validators {
			byAddress[strings.ToLower(v.Addr)] = v.BLS
		}
		keys := [][]byte{}
		keyHex := []string{}
		for _, addr := range addresses {
			value, ok := byAddress[strings.ToLower(addr)]
			if !ok {
				t.Fatal("current validator missing from preceding epoch", addr)
			}
			b, err := hex.DecodeString(strings.TrimPrefix(value, "0x"))
			if err != nil {
				t.Fatal(err)
			}
			keys = append(keys, b)
			keyHex = append(keyHex, value)
		}
		var encoded string
		if err = c.Call(ctx, "debug_getRawHeader", &encoded, map[string]any{"blockHash": h["hash"]}); err != nil {
			t.Fatal(err)
		}
		raw, err := hex.DecodeString(strings.TrimPrefix(encoded, "0x"))
		if err != nil {
			t.Fatal(err)
		}
		if err = wbft.VerifyCommittedHeader(raw, keys); err != nil {
			t.Fatalf("actual header %d aggregate: %v", n, err)
		}
		wrong := append([][]byte(nil), keys...)
		wrong[0] = keys[1]
		if wbft.VerifyCommittedHeader(raw, wrong) == nil {
			t.Fatal("actual duplicate epoch key accepted")
		}
		var extra map[string]any
		if err = c.Call(ctx, "istanbul_getWbftExtraInfo", &extra, tag); err != nil {
			t.Fatal(err)
		}
		if err = verifyExtraEpochShape(n, 10, extra); err != nil {
			t.Fatalf("header %d extra shape: %v", n, err)
		}
		if extra["epochInfo"] == nil {
			ordinary++
		} else {
			boundary++
		}
		proofs = append(proofs, map[string]any{"number": n, "hash": h["hash"], "headerRLP": encoded, "epochBlock": epochBlock, "validators": addresses, "blsKeys": keyHex, "signatureVerified": true, "rpcExtra": extra})
	}
	if ordinary < 9 || boundary < 1 {
		t.Fatal("missing actual ordinary/boundary controls", ordinary, boundary)
	}
	b, err := json.Marshal(map[string]any{"scope": "Actual managed WEMIX-to-WBFT fork, current preceding epoch key ordering, eleven committed headers. Not complete87/mainnet/Linux/final Web acceptance.", "headers": proofs})
	if err != nil {
		t.Fatal(err)
	}
	t.Logf("wbft-aggregate-proof: %s", b)
}
