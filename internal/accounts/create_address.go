package accounts

import (
	"fmt"

	sdktx "github.com/0xmhha/accounts/tx"
	sdktypes "github.com/0xmhha/accounts/types"
)

// CreateAddress returns the contract address a CREATE deployment from deployer
// at nonce lands at — keccak(rlp(deployer, nonce)), the address a spec can
// assert a deploy reached before it happens. It reuses the accounts SDK's own
// computation rather than re-deriving the hash, so it matches what the node
// assigns. deployer is a 0x-hex address; an unparseable one is an error.
func CreateAddress(deployer string, nonce uint64) (string, error) {
	addr, err := sdktypes.HexToAddress(deployer)
	if err != nil {
		return "", fmt.Errorf("accounts: create address: deployer %q: %w", deployer, err)
	}
	return sdktx.CreateAddress(addr, nonce).Hex(), nil
}
