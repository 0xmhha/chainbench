//go:build e2e

package main

import (
	"encoding/hex"
	"fmt"
	"strings"
	"testing"
)

type forkHeaderTransitionProof struct {
	Before, At                            map[string]any
	BeforeErrorCode                       int
	BeforeErrorMethod, BeforeErrorMessage string
	EpochLength                           uint64
	Extra                                 map[string]any
}

type forkRLPNode struct {
	Bytes []byte
	Items []forkRLPNode
	List  bool
}

// Decode just the RLP wire grammar for the boundary proof, independently of
// the node API's already-decoded fields. Bounds and trailing bytes are checked.
func decodeForkRLP(b []byte, depth int) (forkRLPNode, int, error) {
	if len(b) == 0 || depth > 32 {
		return forkRLPNode{}, 0, fmt.Errorf("invalid RLP input/depth")
	}
	tag := b[0]
	if tag < 0x80 {
		return forkRLPNode{Bytes: b[:1]}, 1, nil
	}
	list := tag >= 0xc0
	base := byte(0x80)
	if list {
		base = 0xc0
	}
	off, n := 1, int(tag-base)
	if n > 55 {
		ll := n - 55
		if ll > 8 || len(b) < 1+ll || b[1] == 0 {
			return forkRLPNode{}, 0, fmt.Errorf("invalid long RLP header")
		}
		size := uint64(0)
		for _, v := range b[1 : 1+ll] {
			size = size<<8 | uint64(v)
		}
		off = 1 + ll
		if size < 56 || size > uint64(len(b)-off) {
			return forkRLPNode{}, 0, fmt.Errorf("invalid long RLP size")
		}
		n = int(size)
	}
	if n > len(b)-off {
		return forkRLPNode{}, 0, fmt.Errorf("RLP overruns input")
	}
	payload := b[off : off+n]
	node := forkRLPNode{List: list}
	if !list {
		if n == 1 && payload[0] < 0x80 {
			return forkRLPNode{}, 0, fmt.Errorf("noncanonical RLP string")
		}
		node.Bytes = payload
		return node, off + n, nil
	}
	for len(payload) > 0 {
		child, size, err := decodeForkRLP(payload, depth+1)
		if err != nil {
			return forkRLPNode{}, 0, err
		}
		node.Items = append(node.Items, child)
		payload = payload[size:]
	}
	return node, off + n, nil
}

func verifyForkHeaderTransition(p forkHeaderTransitionProof) error {
	if p.Before["number"] != "0x63" || p.At["number"] != "0x64" || p.Before["hash"] == nil || p.At["parentHash"] != p.Before["hash"] || p.At["hash"] == nil || p.At["hash"] == p.Before["hash"] {
		return fmt.Errorf("wrong actual fork header linkage")
	}
	if p.BeforeErrorCode != -32000 || p.BeforeErrorMethod != "istanbul_getWbftExtraInfo" || p.BeforeErrorMessage != "block is not a wbft block" {
		return fmt.Errorf("not the exact pre-fork node refusal")
	}
	before, e1 := hex.DecodeString(strings.TrimPrefix(fmt.Sprint(p.Before["extraData"]), "0x"))
	after, e2 := hex.DecodeString(strings.TrimPrefix(fmt.Sprint(p.At["extraData"]), "0x"))
	if e1 != nil || e2 != nil || len(before) > 32 || !strings.HasPrefix(string(before), "gwemix/v") || len(after) <= 32 {
		return fmt.Errorf("missing WPOA client vanity or WBFT encoding")
	}
	if err := verifyExtraEpochShape(100, p.EpochLength, p.Extra); err != nil {
		return err
	}
	root, n, err := decodeForkRLP(after, 0)
	if err != nil || n != len(after) || !root.List || len(root.Items) != 9 {
		return fmt.Errorf("not Croissant v1 nine-field RLP")
	}
	reveal, ok := p.Extra["randaoReveal"].(string)
	if !ok || "0x"+hex.EncodeToString(root.Items[1].Bytes) != reveal {
		return fmt.Errorf("wire randao differs from API")
	}
	seal := root.Items[7]
	api, ok := p.Extra["committedSeal"].(map[string]any)
	if !ok || !seal.List || len(seal.Items) != 2 || "0x"+hex.EncodeToString(seal.Items[1].Bytes) != api["signature"] {
		return fmt.Errorf("wire committed seal differs from API")
	}
	epoch := root.Items[8]
	if !epoch.List || len(epoch.Items) != 4 {
		return fmt.Errorf("wire epoch absent")
	}
	return nil
}

func forkHeaderTransitionFixture() forkHeaderTransitionProof {
	reveal := "0x" + strings.Repeat("ab", 65)
	seal := "0x" + strings.Repeat("cd", 96)
	extra := map[string]any{"randaoReveal": reveal, "prevRound": "0x0", "epochInfo": map[string]any{"validators": []any{map[string]any{"addr": "validator", "bls": "key"}}}, "committedSeal": map[string]any{"signature": seal}}
	return forkHeaderTransitionProof{Before: map[string]any{"number": "0x63", "hash": "0x" + strings.Repeat("11", 32), "extraData": "0x" + hex.EncodeToString([]byte("gwemix/v1.0.0/darwin/go1.25.2"))}, At: map[string]any{"number": "0x64", "hash": "0x" + strings.Repeat("22", 32), "parentHash": "0x" + strings.Repeat("11", 32), "extraData": func() string {
		payload := "80b841" + reveal[2:] + "80c0c080c0f86301b860" + seal[2:] + "c4c0c0c080"
		return fmt.Sprintf("0xf8%02x%s", len(payload)/2, payload)
	}()}, BeforeErrorCode: -32000, BeforeErrorMethod: "istanbul_getWbftExtraInfo", BeforeErrorMessage: "block is not a wbft block", EpochLength: 10, Extra: extra}
}

func TestForkHeaderTransitionRejectsPresenceOnly(t *testing.T) {
	if err := verifyForkHeaderTransition(forkHeaderTransitionFixture()); err != nil {
		t.Fatal(err)
	}
	for _, tc := range []struct {
		name   string
		change func(*forkHeaderTransitionProof)
	}{
		{"wrong predecessor", func(p *forkHeaderTransitionProof) { p.Before["number"] = "0x13" }},
		{"wrong fork", func(p *forkHeaderTransitionProof) { p.At["number"] = "0x14" }},
		{"different parent", func(p *forkHeaderTransitionProof) { p.At["parentHash"] = p.At["hash"] }},
		{"no before extra", func(p *forkHeaderTransitionProof) { p.Before["extraData"] = "0x" }},
		{"no after extra", func(p *forkHeaderTransitionProof) { p.At["extraData"] = "0x" }},
		{"transport failure", func(p *forkHeaderTransitionProof) { p.BeforeErrorCode = 0 }},
		{"unrelated node refusal", func(p *forkHeaderTransitionProof) { p.BeforeErrorMessage = "block not found" }},
		{"wrong rejected method", func(p *forkHeaderTransitionProof) { p.BeforeErrorMethod = "eth_getBlockByNumber" }},
		{"missing epoch", func(p *forkHeaderTransitionProof) { p.Extra["epochInfo"] = nil }},
		{"wrong epoch length", func(p *forkHeaderTransitionProof) { p.EpochLength = 0 }},
		{"reveal differs from wire", func(p *forkHeaderTransitionProof) { p.Extra["randaoReveal"] = "0x" + strings.Repeat("ee", 65) }},
		{"committed seal differs from wire", func(p *forkHeaderTransitionProof) {
			p.Extra["committedSeal"] = map[string]any{"signature": "0x" + strings.Repeat("ef", 96)}
		}},
	} {
		t.Run(tc.name, func(t *testing.T) {
			p := forkHeaderTransitionFixture()
			tc.change(&p)
			if verifyForkHeaderTransition(p) == nil {
				t.Fatal("presence-only proof hid incorrect fork header transition")
			}
		})
	}
}
