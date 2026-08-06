package accounts

import (
	"encoding/hex"
	"math/big"
	"strings"
)

// Arg is one ABI argument for EncodeABI / EncodeCallArgs: a static 32-byte word
// (address, uintN, bool) or a dynamic value (bytes, string). Build one with
// Uint, Address, Word, Bytes, or StringArg. This is a minimal encoder for a flat
// argument list (no arrays or tuples) — enough for the ERC-20 and go-stablenet
// governance signatures, including the dynamic bytes/string args (e.g.
// proposeMint(bytes)) that the fixed-word EncodeCall cannot express.
type Arg struct {
	word    []byte // static: the 32-byte head word
	tail    []byte // dynamic: [length word || padded data]
	dynamic bool
}

// Uint encodes a uint256 argument.
func Uint(n *big.Int) Arg {
	if n == nil {
		n = big.NewInt(0)
	}
	return Arg{word: padWord32(n.Bytes())}
}

// Address encodes a 0x-hex address argument.
func Address(hexAddr string) Arg { return Arg{word: padWord32(AddressArg(hexAddr))} }

// Word encodes a raw <=32-byte value as a left-padded static word (e.g. a bool
// as {0} / {1}, or a bytes32).
func Word(b []byte) Arg { return Arg{word: padWord32(b)} }

// Bytes encodes a dynamic bytes argument (offset in the head, length+padded data
// in the tail).
func Bytes(b []byte) Arg { return Arg{tail: encodeDynamicData(b), dynamic: true} }

// StringArg encodes a dynamic string argument (UTF-8 bytes).
func StringArg(s string) Arg { return Bytes([]byte(s)) }

// encodeDynamicData is the ABI tail for a dynamic value: the 32-byte length
// followed by the data right-padded to a 32-byte multiple.
func encodeDynamicData(b []byte) []byte {
	out := padWord32(big.NewInt(int64(len(b))).Bytes())
	out = append(out, b...)
	if r := len(b) % 32; r != 0 {
		out = append(out, make([]byte, 32-r)...)
	}
	return out
}

// EncodeABI encodes a flat argument list per the Solidity ABI: static args are
// inline 32-byte head words; dynamic args (bytes/string) are a 32-byte offset in
// the head plus [length, padded data] in the tail. Returns the raw bytes. This
// is the abi.encode(...) used both for call arguments and for a nested payload
// (e.g. a governance proof passed as bytes).
func EncodeABI(args ...Arg) []byte {
	headLen := 32 * len(args)
	head := make([]byte, 0, headLen)
	var tail []byte
	for _, a := range args {
		if a.dynamic {
			offset := headLen + len(tail)
			head = append(head, padWord32(big.NewInt(int64(offset)).Bytes())...)
			tail = append(tail, a.tail...)
		} else {
			head = append(head, a.word...)
		}
	}
	return append(head, tail...)
}

// padWord32 left-pads b to a 32-byte word (b must be <=32 bytes).
func padWord32(b []byte) []byte {
	if len(b) >= 32 {
		return b[:32]
	}
	w := make([]byte, 32)
	copy(w[32-len(b):], b)
	return w
}

// EncodeCallArgs builds eth_call/tx calldata: the 4-byte selector of methodSig
// followed by EncodeABI(args...). Returns 0x-hex. Unlike EncodeCall (fixed words
// only), this supports dynamic bytes/string arguments.
func EncodeCallArgs(methodSig string, args ...Arg) string {
	sel, _ := hex.DecodeString(strings.TrimPrefix(Selector(methodSig), "0x"))
	return "0x" + hex.EncodeToString(append(sel, EncodeABI(args...)...))
}
