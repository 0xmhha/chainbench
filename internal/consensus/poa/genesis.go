package poa

import (
	"encoding/json"
	"fmt"
	"strconv"
	"strings"
)

// zeroAddress is the default coinbase when none is supplied.
const zeroAddress = "0x0000000000000000000000000000000000000000"

// Placeholder tokens in the wemix base genesis template.
const (
	phChainID  = "__CHAIN_ID__"
	phCoinbase = "__COINBASE__"
)

// PrepareTemplate fills the base-genesis template's placeholders (chain id,
// coinbase; an empty coinbase defaults to the zero address).
//
// It is a preparation step, not a genesis generator, and the name says so now
// because the old one lied about what it produced. A wemix genesis is written
// by the binary — `wemix genesis` reads a governance config and this prepared
// template and emits the real thing, with the alloc, the extraData bootnode
// encoding and the wemix fork config in it. Using the substituted template as a
// genesis produced a file that parsed and initialized and left the node running
// ethash with no wemix RPC namespace: structurally valid, functionally dead,
// and worse than an error because nothing said so.
//
// The chain id is stamped here rather than left to the binary because
// `wemix genesis` passes the template's config through untouched, so the
// manifest's value only lands if it is already in the template.
func PrepareTemplate(template []byte, chainID int64, coinbase string) ([]byte, error) {
	if chainID <= 0 {
		return nil, fmt.Errorf("poa: prepare template: invalid chain id %d", chainID)
	}
	if coinbase == "" {
		coinbase = zeroAddress
	}
	out := string(template)
	out = strings.ReplaceAll(out, phChainID, strconv.FormatInt(chainID, 10))
	out = strings.ReplaceAll(out, `"`+phCoinbase+`"`, strconv.Quote(coinbase))
	out = strings.ReplaceAll(out, phCoinbase, coinbase)
	if !json.Valid([]byte(out)) {
		return nil, fmt.Errorf("poa: prepare template: substitution produced invalid JSON")
	}
	return []byte(out), nil
}
