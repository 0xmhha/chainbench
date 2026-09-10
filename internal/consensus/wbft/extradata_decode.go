package wbft

import (
	"encoding/hex"
	"encoding/json"
	"fmt"
	"strings"
)

// Reading the validator set back out of a genesis.
//
// ExtraData writes the set into the RLP extra-data; this reads it back, so a
// finished genesis can be checked against the keys a network will actually run
// with. A validator whose address is not one of the running node keys cannot
// sign, and the chain stalls with the cause far from the symptom — this turns
// that into a refusal at compose time.

// rlpNode is one decoded RLP item: a byte string, or a list of items.
type rlpNode struct {
	isList bool
	bytes  []byte
	list   []rlpNode
}

// GenesisValidators returns the validator addresses (0x-hex) a wbft-family
// genesis encodes in its extra-data, satisfying registry.GenesisValidatorReader
// so a composition can check an existing genesis against the keys it will run.
func (Family) GenesisValidators(genesisJSON []byte) ([]string, error) {
	return genesisValidatorsFromExtraData(genesisJSON)
}

// genesisValidatorsFromExtraData reads the validator addresses out of a genesis
// extra-data. At genesis every candidate is a validator (the Validators index
// list is 0..n-1), so the candidate addresses are the set. It errors if the
// genesis has no extra-data or the extra-data is not the shape ExtraData writes.
func genesisValidatorsFromExtraData(genesisJSON []byte) ([]string, error) {
	var g struct {
		ExtraData string `json:"extraData"`
	}
	if err := json.Unmarshal(genesisJSON, &g); err != nil {
		return nil, fmt.Errorf("wbft: genesis validators: parse genesis: %w", err)
	}
	if g.ExtraData == "" {
		return nil, fmt.Errorf("wbft: genesis validators: genesis has no extraData")
	}
	raw, err := hex.DecodeString(strings.TrimPrefix(g.ExtraData, "0x"))
	if err != nil {
		return nil, fmt.Errorf("wbft: genesis validators: extraData is not hex: %w", err)
	}
	root, rest, err := rlpDecode(raw)
	if err != nil {
		return nil, fmt.Errorf("wbft: genesis validators: decode extraData: %w", err)
	}
	if len(rest) != 0 {
		return nil, fmt.Errorf("wbft: genesis validators: %d trailing bytes after extraData", len(rest))
	}
	// WBFTExtra is a 10-item list; EpochInfo is the last, [candidates, indices,
	// blsKeys]; each candidate is [address, diligence].
	if !root.isList || len(root.list) != 10 {
		return nil, fmt.Errorf("wbft: genesis validators: extraData is not the expected WBFTExtra shape")
	}
	epoch := root.list[9]
	if !epoch.isList || len(epoch.list) < 1 || !epoch.list[0].isList {
		return nil, fmt.Errorf("wbft: genesis validators: extraData has no candidate list")
	}
	candidates := epoch.list[0].list
	out := make([]string, 0, len(candidates))
	for i, c := range candidates {
		if !c.isList || len(c.list) < 1 || c.list[0].isList || len(c.list[0].bytes) != addressLen {
			return nil, fmt.Errorf("wbft: genesis validators: candidate %d is not [address, diligence]", i)
		}
		out = append(out, "0x"+hex.EncodeToString(c.list[0].bytes))
	}
	return out, nil
}

// rlpDecode decodes one RLP item at the front of b, returning it and the
// remaining bytes. It handles the two string forms and the two list forms — the
// same grammar rlpEncode writes.
func rlpDecode(b []byte) (rlpNode, []byte, error) {
	if len(b) == 0 {
		return rlpNode{}, nil, fmt.Errorf("unexpected end of input")
	}
	p := b[0]
	switch {
	case p < 0x80: // a single byte, value < 0x80, is itself
		return rlpNode{bytes: []byte{p}}, b[1:], nil
	case p < 0xb8: // short string, length p-0x80
		n := int(p - 0x80)
		if len(b) < 1+n {
			return rlpNode{}, nil, fmt.Errorf("short string overruns input")
		}
		return rlpNode{bytes: b[1 : 1+n]}, b[1+n:], nil
	case p < 0xc0: // long string, length-of-length p-0xb7
		return rlpDecodeLong(b, p, 0xb7, false)
	case p < 0xf8: // short list, payload length p-0xc0
		return rlpDecodeList(b, 1, int(p-0xc0))
	default: // long list, length-of-length p-0xf7
		return rlpDecodeLong(b, p, 0xf7, true)
	}
}

// rlpDecodeLong decodes a long string or long list whose length is itself
// length-prefixed.
func rlpDecodeLong(b []byte, p, base byte, list bool) (rlpNode, []byte, error) {
	ll := int(p - base)
	if len(b) < 1+ll {
		return rlpNode{}, nil, fmt.Errorf("length header overruns input")
	}
	n := 0
	for _, c := range b[1 : 1+ll] {
		n = n<<8 | int(c)
	}
	start := 1 + ll
	if len(b) < start+n {
		return rlpNode{}, nil, fmt.Errorf("payload overruns input")
	}
	if list {
		return rlpDecodeList(b, start, n)
	}
	return rlpNode{bytes: b[start : start+n]}, b[start+n:], nil
}

// rlpDecodeList decodes n bytes of list payload starting at off into items.
func rlpDecodeList(b []byte, off, n int) (rlpNode, []byte, error) {
	if len(b) < off+n {
		return rlpNode{}, nil, fmt.Errorf("list payload overruns input")
	}
	payload := b[off : off+n]
	var items []rlpNode
	for len(payload) > 0 {
		item, rest, err := rlpDecode(payload)
		if err != nil {
			return rlpNode{}, nil, err
		}
		items = append(items, item)
		payload = rest
	}
	return rlpNode{isList: true, list: items}, b[off+n:], nil
}
