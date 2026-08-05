package accounts

import (
	"encoding/hex"
	"strings"

	sdkcrypto "github.com/0xmhha/accounts/crypto"
)

// Selector returns the 4-byte function selector (0x-hex) for a Solidity method
// signature such as "isAuthorized(address)" or "totalSupply()". It is the first
// four bytes of keccak256(sig).
func Selector(methodSig string) string {
	sum := sdkcrypto.Keccak256([]byte(methodSig))
	return "0x" + hex.EncodeToString(sum[:4])
}

// EncodeCall builds eth_call calldata for methodSig with the given arguments:
// the 4-byte selector followed by each argument left-padded to 32 bytes (the
// ABI head encoding for fixed-size words — addresses and uint256, which are the
// argument kinds the system-contract read cases use). args are raw big-endian
// bytes (a 20-byte address, or a uint256's minimal bytes). Returns 0x-hex.
func EncodeCall(methodSig string, args ...[]byte) string {
	var b strings.Builder
	b.WriteString(Selector(methodSig))
	for _, a := range args {
		b.WriteString(leftPad32(a))
	}
	return b.String()
}

// leftPad32 hex-encodes b left-padded to 32 bytes (64 hex chars). Longer inputs
// are hex-encoded as-is (callers pass <=32-byte words).
func leftPad32(b []byte) string {
	h := hex.EncodeToString(b)
	if len(h) < 64 {
		h = strings.Repeat("0", 64-len(h)) + h
	}
	return h
}

// AddressArg converts a 0x-hex address to the 20 raw bytes EncodeCall pads to a
// 32-byte word. Invalid input yields nil (encoded as a zero word).
func AddressArg(hexAddr string) []byte {
	raw, err := hex.DecodeString(strings.TrimPrefix(hexAddr, "0x"))
	if err != nil {
		return nil
	}
	return raw
}
