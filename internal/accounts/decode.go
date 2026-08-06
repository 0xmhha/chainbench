package accounts

import (
	"context"
	"encoding/hex"
	"math/big"
	"strings"
)

// DecodeString decodes an ABI-encoded dynamic string/bytes return value — a
// single dynamic value: a 32-byte offset, and at that offset a 32-byte length
// followed by the data. Returns "" on malformed input. It is the inverse of the
// dynamic encoding StringArg/Bytes produce, used to read e.g. an ERC-20 name().
func DecodeString(hexRet string) string {
	b, err := hex.DecodeString(strings.TrimPrefix(hexRet, "0x"))
	if err != nil || len(b) < 64 {
		return ""
	}
	offset := new(big.Int).SetBytes(b[:32])
	if !offset.IsInt64() {
		return ""
	}
	off := int(offset.Int64())
	if off < 0 || off+32 > len(b) {
		return ""
	}
	length := new(big.Int).SetBytes(b[off : off+32])
	if !length.IsInt64() {
		return ""
	}
	n := int(length.Int64())
	start := off + 32
	if n < 0 || start+n > len(b) {
		return ""
	}
	return string(b[start : start+n])
}

// ReadString calls to.methodSig(args...) via eth_call and decodes the returned
// dynamic string. args are fixed-size words (use EncodeCall's arg form). It
// composes EncodeCall + DecodeString for string reads (e.g. name(), symbol()).
func ReadString(ctx context.Context, call EthCaller, to, methodSig string, args ...[]byte) (string, error) {
	ret, err := call(ctx, to, EncodeCall(methodSig, args...))
	if err != nil {
		return "", err
	}
	return DecodeString(ret), nil
}
