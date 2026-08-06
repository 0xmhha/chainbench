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

// BuildGenesis substitutes the poa (wemix) base-genesis template. Unlike the
// wbft family, the validator set is NOT in genesis — wemix resolves membership
// at bootstrap via governance contracts + etcd (see BootstrapPlan). So this only
// fills the base fields (chain id, coinbase); an empty coinbase defaults to the
// zero address.
func BuildGenesis(template []byte, chainID int64, coinbase string) ([]byte, error) {
	if chainID <= 0 {
		return nil, fmt.Errorf("poa genesis: invalid chain id %d", chainID)
	}
	if coinbase == "" {
		coinbase = zeroAddress
	}
	out := string(template)
	out = strings.ReplaceAll(out, phChainID, strconv.FormatInt(chainID, 10))
	out = strings.ReplaceAll(out, `"`+phCoinbase+`"`, strconv.Quote(coinbase))
	out = strings.ReplaceAll(out, phCoinbase, coinbase)
	if !json.Valid([]byte(out)) {
		return nil, fmt.Errorf("poa genesis: substitution produced invalid JSON")
	}
	return []byte(out), nil
}
