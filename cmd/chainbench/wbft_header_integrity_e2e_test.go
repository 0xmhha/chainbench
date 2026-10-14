//go:build e2e

package main

import (
	"context"
	"encoding/hex"
	"encoding/json"
	"fmt"
	"github.com/0xmhha/chainbench/internal/core/rpc"
	"math/big"
	"os"
	"strconv"
	"strings"
	"testing"
	"time"

	sdkcrypto "github.com/0xmhha/accounts/crypto"
)

type wbftHeaderSample struct {
	Number, Timestamp                        uint64
	Hash, ParentHash, Miner, MixHash, Reveal string
	Round                                    uint64
}

// verifyWbftHeaderWindow validates Croissant-v1 signatures, hashed-reveal XOR
// and every one-second interval in a healthy round-zero window.
func verifyWbftHeaderWindow(chainID uint64, samples []wbftHeaderSample) error {
	if chainID == 0 || len(samples) < 4 {
		return fmt.Errorf("need a chain ID, anchor and three consecutive headers")
	}
	for i := 1; i < len(samples); i++ {
		p, c := samples[i-1], samples[i]
		if p.Number == 0 || c.Number != p.Number+1 || p.Hash == "" || c.Hash == "" || c.ParentHash != p.Hash || c.Timestamp != p.Timestamp+1 || c.Round != 0 {
			return fmt.Errorf("block %d: broken height/parent, non-one-second interval or nonzero round", c.Number)
		}
		signature, err := hex.DecodeString(strings.TrimPrefix(c.Reveal, "0x"))
		if err != nil || len(signature) != 65 {
			return fmt.Errorf("block %d: invalid reveal", c.Number)
		}
		data := append(new(big.Int).SetUint64(chainID).Bytes(), byte(1))
		data = append(data, new(big.Int).SetUint64(c.Number).Bytes()...)
		signer, err := sdkcrypto.Recover(sdkcrypto.Keccak256(sdkcrypto.Keccak256(data)), signature)
		if err != nil || !strings.EqualFold(signer.Hex(), c.Miner) {
			return fmt.Errorf("block %d: reveal signer is not proposer: %v", c.Number, err)
		}
		prev, e1 := hex.DecodeString(strings.TrimPrefix(p.MixHash, "0x"))
		next, e2 := hex.DecodeString(strings.TrimPrefix(c.MixHash, "0x"))
		if e1 != nil || e2 != nil || len(prev) != 32 || len(next) != 32 {
			return fmt.Errorf("block %d: invalid mix length", c.Number)
		}
		hash := sdkcrypto.Keccak256(signature)
		for n := range prev {
			if next[n] != prev[n]^hash[n] {
				return fmt.Errorf("block %d: mix chain differs", c.Number)
			}
		}
	}
	return nil
}

func validWbftWindow(t *testing.T) []wbftHeaderSample {
	t.Helper()
	key, err := sdkcrypto.GenerateKey()
	if err != nil {
		t.Fatal(err)
	}
	result := []wbftHeaderSample{{Number: 20, Timestamp: 1020, Hash: fmt.Sprintf("0x%064x", 20), MixHash: "0x" + strings.Repeat("00", 32)}}
	for i := uint64(21); i <= 23; i++ {
		data := append(new(big.Int).SetUint64(8285).Bytes(), byte(1))
		data = append(data, new(big.Int).SetUint64(i).Bytes()...)
		signature, err := sdkcrypto.Sign(sdkcrypto.Keccak256(sdkcrypto.Keccak256(data)), key)
		if err != nil {
			t.Fatal(err)
		}
		prev, _ := hex.DecodeString(result[len(result)-1].MixHash[2:])
		revealHash := sdkcrypto.Keccak256(signature)
		for n := range prev {
			prev[n] ^= revealHash[n]
		}
		result = append(result, wbftHeaderSample{Number: i, Timestamp: 1000 + i, Hash: fmt.Sprintf("0x%064x", i), ParentHash: result[len(result)-1].Hash, Miner: sdkcrypto.PrivKeyToAddress(key).Hex(), MixHash: "0x" + hex.EncodeToString(prev), Reveal: "0x" + hex.EncodeToString(signature)})
	}
	return result
}

func TestWbftHeaderWindowRejectsPresenceOnlyEvidence(t *testing.T) {
	for _, tc := range []struct {
		name   string
		change func([]wbftHeaderSample)
	}{
		{"changed signature", func(p []wbftHeaderSample) { p[1].Reveal = "0x" + strings.Repeat("01", 65) }},
		{"wrong proposer", func(p []wbftHeaderSample) { p[1].Miner = "0x1111111111111111111111111111111111111111" }},
		{"wrong mix", func(p []wbftHeaderSample) { p[1].MixHash = "0x" + strings.Repeat("ff", 32) }},
		{"missing height", func(p []wbftHeaderSample) { p[1].Number++ }},
		{"broken parent", func(p []wbftHeaderSample) { p[1].ParentHash = p[2].Hash }},
		{"two-second interval", func(p []wbftHeaderSample) { p[1].Timestamp++ }},
		{"same timestamp", func(p []wbftHeaderSample) { p[1].Timestamp-- }},
		{"round change in healthy window", func(p []wbftHeaderSample) { p[1].Round = 1 }},
	} {
		t.Run(tc.name, func(t *testing.T) {
			p := validWbftWindow(t)
			tc.change(p)
			if verifyWbftHeaderWindow(8285, p) == nil {
				t.Fatal("invalid WBFT header window accepted")
			}
		})
	}
}

func TestWbftHeaderWindowAcceptsValidCryptographicChain(t *testing.T) {
	p := validWbftWindow(t)
	if err := verifyWbftHeaderWindow(8285, p); err != nil {
		t.Fatal(err)
	}
	if verifyWbftHeaderWindow(8286, p) == nil {
		t.Fatal("wrong chain ID accepted")
	}
	if verifyWbftHeaderWindow(8285, p[:3]) == nil {
		t.Fatal("too short a chain accepted")
	}
}

// TestWemixWbftHeaderIntegrityE2E strengthens WBFT-002/010 after the actual fork.
// Presence-only assertions and average intervals cannot satisfy this window.
func TestWemixWbftHeaderIntegrityE2E(t *testing.T) {
	from, to := os.Getenv("CHAINBENCH_E2E_FROM_BIN"), os.Getenv("CHAINBENCH_E2E_TO_BIN")
	if from == "" || to == "" {
		t.Skip("set FROM/TO binaries for real WBFT header proof")
	}
	overlay := map[string]any{"config": map[string]any{"croissant": map[string]any{"wBFT": map[string]any{"blockPeriodSeconds": 1, "requestTimeoutSeconds": 1}}}}
	url := runGovHandoffOverlay(t, from, to, overlay)
	c := rpc.Dial(url)
	ctx, cancel := context.WithTimeout(t.Context(), 2*time.Minute)
	defer cancel()
	chainID, err := c.ChainID(ctx)
	if err != nil {
		t.Fatal(err)
	}
	start, err := c.BlockNumber(ctx)
	if err != nil || start <= 20 {
		t.Fatal("not past real fork", start, err)
	}
	waitForHead(t, c, start+10, 90*time.Second)
	samples := []wbftHeaderSample{}
	for n := start; n <= start+10; n++ {
		tag := fmt.Sprintf("0x%x", n)
		var h struct{ Number, Hash, ParentHash, Miner, MixHash, Timestamp string }
		if err := c.Call(ctx, "eth_getBlockByNumber", &h, tag, false); err != nil {
			t.Fatal(err)
		}
		var extra struct{ RandaoReveal, Round string }
		if err := c.Call(ctx, "istanbul_getWbftExtraInfo", &extra, tag); err != nil {
			t.Fatal(err)
		}
		number, e1 := strconv.ParseUint(strings.TrimPrefix(h.Number, "0x"), 16, 64)
		timestamp, e2 := strconv.ParseUint(strings.TrimPrefix(h.Timestamp, "0x"), 16, 64)
		round, e3 := strconv.ParseUint(strings.TrimPrefix(extra.Round, "0x"), 16, 64)
		if e1 != nil || e2 != nil || e3 != nil || number != n {
			t.Fatal("invalid exact header query", tag, h, extra)
		}
		samples = append(samples, wbftHeaderSample{Number: number, Timestamp: timestamp, Hash: h.Hash, ParentHash: h.ParentHash, Miner: h.Miner, MixHash: h.MixHash, Reveal: extra.RandaoReveal, Round: round})
	}
	proof := map[string]any{"chainID": chainID, "anchor": start, "followingConsecutiveHeaders": 10, "blockPeriodSeconds": 1, "requestTimeoutSeconds": 1, "randaoVersion": 1, "samples": samples, "signing": "keccak256(keccak256(chainID minimal bytes || version minimal bytes || number minimal bytes))", "mix": "previous mix XOR keccak256(reveal)"}
	raw, err := json.Marshal(proof)
	if err != nil {
		t.Fatal(err)
	}
	t.Logf("wbft-header-proof: %s", raw)
	if err := verifyWbftHeaderWindow(chainID, samples); err != nil {
		t.Fatal(err)
	}
}
