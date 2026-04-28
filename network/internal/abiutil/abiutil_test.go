package abiutil

import (
	"encoding/hex"
	"math/big"
	"strings"
	"testing"

	"github.com/ethereum/go-ethereum/common"
	"github.com/ethereum/go-ethereum/core/types"
	"github.com/ethereum/go-ethereum/crypto"
)

// erc20MiniABI is a minimal ERC-20-like fixture covering the surface area
// the abiutil tests need: a transfer method, a Transfer event with one
// indexed + one non-indexed field, and a uint256 constructor.
const erc20MiniABI = `[
  {"type":"constructor","inputs":[{"name":"supply","type":"uint256"}],"stateMutability":"nonpayable"},
  {"type":"function","name":"transfer","stateMutability":"nonpayable",
   "inputs":[{"name":"to","type":"address"},{"name":"value","type":"uint256"}],
   "outputs":[{"name":"","type":"bool"}]},
  {"type":"event","name":"Transfer","anonymous":false,
   "inputs":[
     {"name":"from","type":"address","indexed":true},
     {"name":"to","type":"address","indexed":false},
     {"name":"value","type":"uint256","indexed":false}
   ]}
]`

// singleIndexedEventABI exercises the spec test case "1 indexed (from) +
// 1 non-indexed (value)" exactly. Separated from erc20MiniABI which has 1
// indexed + 2 non-indexed so the test assertion on field count is sharp.
const singleIndexedEventABI = `[
  {"type":"event","name":"Transfer","anonymous":false,
   "inputs":[
     {"name":"from","type":"address","indexed":true},
     {"name":"value","type":"uint256","indexed":false}
   ]}
]`

func TestParseABI_Happy(t *testing.T) {
	parsed, err := ParseABI(erc20MiniABI)
	if err != nil {
		t.Fatalf("ParseABI: %v", err)
	}
	if _, ok := parsed.Methods["transfer"]; !ok {
		t.Error("expected transfer method")
	}
	if _, ok := parsed.Events["Transfer"]; !ok {
		t.Error("expected Transfer event")
	}
	if len(parsed.Constructor.Inputs) != 1 {
		t.Errorf("constructor inputs = %d, want 1", len(parsed.Constructor.Inputs))
	}
}

func TestParseABI_BadJSON(t *testing.T) {
	_, err := ParseABI(`{not valid json`)
	if err == nil {
		t.Fatal("expected error")
	}
	if !strings.Contains(err.Error(), "abiutil.ParseABI:") {
		t.Errorf("err missing prefix: %v", err)
	}
}

func TestCoerceArgs_Uint256_Decimal(t *testing.T) {
	parsed, err := ParseABI(erc20MiniABI)
	if err != nil {
		t.Fatal(err)
	}
	method := parsed.Methods["transfer"]
	out, err := CoerceArgs(method.Inputs[1:2], []any{"42"})
	if err != nil {
		t.Fatalf("CoerceArgs: %v", err)
	}
	v, ok := out[0].(*big.Int)
	if !ok {
		t.Fatalf("type = %T, want *big.Int", out[0])
	}
	if v.Cmp(big.NewInt(42)) != 0 {
		t.Errorf("v = %v, want 42", v)
	}
}

func TestCoerceArgs_Uint256_Hex(t *testing.T) {
	parsed, _ := ParseABI(erc20MiniABI)
	method := parsed.Methods["transfer"]
	out, err := CoerceArgs(method.Inputs[1:2], []any{"0x2a"})
	if err != nil {
		t.Fatalf("CoerceArgs: %v", err)
	}
	v := out[0].(*big.Int)
	if v.Cmp(big.NewInt(42)) != 0 {
		t.Errorf("v = %v, want 42", v)
	}
}

func TestCoerceArgs_Uint256_Number(t *testing.T) {
	parsed, _ := ParseABI(erc20MiniABI)
	method := parsed.Methods["transfer"]
	out, err := CoerceArgs(method.Inputs[1:2], []any{float64(42)})
	if err != nil {
		t.Fatalf("CoerceArgs: %v", err)
	}
	v := out[0].(*big.Int)
	if v.Cmp(big.NewInt(42)) != 0 {
		t.Errorf("v = %v, want 42", v)
	}
	// Non-integral float must error.
	if _, err := CoerceArgs(method.Inputs[1:2], []any{float64(1.5)}); err == nil {
		t.Error("expected error for non-integral float")
	}
}

func TestCoerceArgs_Address(t *testing.T) {
	parsed, _ := ParseABI(erc20MiniABI)
	method := parsed.Methods["transfer"]
	addr := "0x000000000000000000000000000000000000dEaD"
	out, err := CoerceArgs(method.Inputs[0:1], []any{addr})
	if err != nil {
		t.Fatalf("CoerceArgs: %v", err)
	}
	got := out[0].(common.Address)
	if got != common.HexToAddress(addr) {
		t.Errorf("addr = %s, want %s", got.Hex(), addr)
	}
	// Bad address.
	if _, err := CoerceArgs(method.Inputs[0:1], []any{"not-an-address"}); err == nil {
		t.Error("expected error for invalid address")
	}
}

// bytesABI gives us a single-arg `bytes` method to exercise toBytes.
const bytesABI = `[
  {"type":"function","name":"f","stateMutability":"nonpayable",
   "inputs":[{"name":"data","type":"bytes"}],"outputs":[]}
]`

func TestCoerceArgs_Bytes(t *testing.T) {
	parsed, err := ParseABI(bytesABI)
	if err != nil {
		t.Fatal(err)
	}
	method := parsed.Methods["f"]
	out, err := CoerceArgs(method.Inputs, []any{"0xdeadbeef"})
	if err != nil {
		t.Fatalf("CoerceArgs: %v", err)
	}
	got := out[0].([]byte)
	want := []byte{0xde, 0xad, 0xbe, 0xef}
	if hex.EncodeToString(got) != hex.EncodeToString(want) {
		t.Errorf("bytes = %x, want %x", got, want)
	}
}

const bytes32ABI = `[
  {"type":"function","name":"f","stateMutability":"nonpayable",
   "inputs":[{"name":"data","type":"bytes32"}],"outputs":[]}
]`

func TestCoerceArgs_Bytes32_Happy(t *testing.T) {
	parsed, err := ParseABI(bytes32ABI)
	if err != nil {
		t.Fatal(err)
	}
	method := parsed.Methods["f"]
	hexStr := "0x" + strings.Repeat("ab", 32)
	out, err := CoerceArgs(method.Inputs, []any{hexStr})
	if err != nil {
		t.Fatalf("CoerceArgs: %v", err)
	}
	got, ok := out[0].([32]byte)
	if !ok {
		t.Fatalf("type = %T, want [32]byte", out[0])
	}
	if got[0] != 0xab || got[31] != 0xab {
		t.Errorf("bytes32 = %x", got)
	}
}

func TestCoerceArgs_Bytes32_WrongLength(t *testing.T) {
	parsed, _ := ParseABI(bytes32ABI)
	method := parsed.Methods["f"]
	_, err := CoerceArgs(method.Inputs, []any{"0xdeadbeef"})
	if err == nil {
		t.Fatal("expected error for wrong-length bytes32 input")
	}
}

const bytes16ABI = `[
  {"type":"function","name":"f","stateMutability":"nonpayable",
   "inputs":[{"name":"data","type":"bytes16"}],"outputs":[]}
]`

func TestCoerceArgs_FixedBytes_NonStandardSize(t *testing.T) {
	parsed, err := ParseABI(bytes16ABI)
	if err != nil {
		t.Fatal(err)
	}
	method := parsed.Methods["f"]
	_, err = CoerceArgs(method.Inputs, []any{"0x" + strings.Repeat("ab", 16)})
	if err == nil {
		t.Fatal("expected error for bytes16 (only bytes32 supported)")
	}
	if !strings.Contains(err.Error(), "not supported") {
		t.Errorf("err should explain bytes16 unsupported: %v", err)
	}
}

const boolStringABI = `[
  {"type":"function","name":"setBool","stateMutability":"nonpayable",
   "inputs":[{"name":"b","type":"bool"}],"outputs":[]},
  {"type":"function","name":"setString","stateMutability":"nonpayable",
   "inputs":[{"name":"s","type":"string"}],"outputs":[]}
]`

func TestCoerceArgs_Bool(t *testing.T) {
	parsed, _ := ParseABI(boolStringABI)
	out, err := CoerceArgs(parsed.Methods["setBool"].Inputs, []any{true})
	if err != nil {
		t.Fatalf("CoerceArgs: %v", err)
	}
	if out[0].(bool) != true {
		t.Errorf("bool = %v, want true", out[0])
	}
}

func TestCoerceArgs_String(t *testing.T) {
	parsed, _ := ParseABI(boolStringABI)
	out, err := CoerceArgs(parsed.Methods["setString"].Inputs, []any{"hello"})
	if err != nil {
		t.Fatalf("CoerceArgs: %v", err)
	}
	if out[0].(string) != "hello" {
		t.Errorf("string = %v, want hello", out[0])
	}
}

const tupleABI = `[
  {"type":"function","name":"f","stateMutability":"nonpayable",
   "inputs":[{"name":"p","type":"tuple","components":[
     {"name":"a","type":"uint256"},
     {"name":"b","type":"address"}
   ]}],"outputs":[]}
]`

func TestCoerceArgs_TupleNotSupported(t *testing.T) {
	parsed, err := ParseABI(tupleABI)
	if err != nil {
		t.Fatal(err)
	}
	_, err = CoerceArgs(parsed.Methods["f"].Inputs, []any{map[string]any{"a": "1", "b": "0x01"}})
	if err == nil {
		t.Fatal("expected error for tuple")
	}
}

func TestCoerceArgs_LengthMismatch(t *testing.T) {
	parsed, _ := ParseABI(erc20MiniABI)
	method := parsed.Methods["transfer"]
	if _, err := CoerceArgs(method.Inputs, []any{"0x01"}); err == nil {
		t.Error("expected error for too few args")
	}
	if _, err := CoerceArgs(method.Inputs, []any{"0x01", "1", "extra"}); err == nil {
		t.Error("expected error for too many args")
	}
}

const noConstructorABI = `[
  {"type":"function","name":"f","stateMutability":"nonpayable","inputs":[],"outputs":[]}
]`

func TestPackConstructor_Empty(t *testing.T) {
	parsed, err := ParseABI(noConstructorABI)
	if err != nil {
		t.Fatal(err)
	}
	out, err := PackConstructor(parsed, nil)
	if err != nil {
		t.Fatalf("PackConstructor: %v", err)
	}
	if len(out) != 0 {
		t.Errorf("expected empty bytes, got %x", out)
	}
}

func TestPackConstructor_WithArgs(t *testing.T) {
	parsed, err := ParseABI(erc20MiniABI)
	if err != nil {
		t.Fatal(err)
	}
	out, err := PackConstructor(parsed, []any{"42"})
	if err != nil {
		t.Fatalf("PackConstructor: %v", err)
	}
	if len(out) != 32 {
		t.Fatalf("packed = %d bytes, want 32", len(out))
	}
	// 42 as 32-byte big-endian: 31 zero bytes + 0x2a.
	if out[31] != 0x2a {
		t.Errorf("last byte = %x, want 0x2a", out[31])
	}
	for i := 0; i < 31; i++ {
		if out[i] != 0 {
			t.Errorf("byte[%d] = %x, want 0", i, out[i])
		}
	}
}

func TestPackMethodCall_Happy(t *testing.T) {
	parsed, err := ParseABI(erc20MiniABI)
	if err != nil {
		t.Fatal(err)
	}
	out, err := PackMethodCall(parsed, "transfer",
		[]any{"0x000000000000000000000000000000000000dEaD", "42"})
	if err != nil {
		t.Fatalf("PackMethodCall: %v", err)
	}
	// Selector = first 4 bytes of keccak256("transfer(address,uint256)").
	want := crypto.Keccak256([]byte("transfer(address,uint256)"))[:4]
	if hex.EncodeToString(out[:4]) != hex.EncodeToString(want) {
		t.Errorf("selector = %x, want %x", out[:4], want)
	}
	// Total = 4 selector + 32 (address pad) + 32 (uint256) = 68.
	if len(out) != 68 {
		t.Errorf("len = %d, want 68", len(out))
	}
	if out[67] != 0x2a {
		t.Errorf("uint256 last byte = %x, want 0x2a", out[67])
	}
}

func TestPackMethodCall_UnknownMethod(t *testing.T) {
	parsed, _ := ParseABI(erc20MiniABI)
	_, err := PackMethodCall(parsed, "nonexistent", nil)
	if err == nil {
		t.Fatal("expected error")
	}
	if !strings.Contains(err.Error(), "nonexistent") {
		t.Errorf("err should mention method name: %v", err)
	}
}

const uintReturnABI = `[
  {"type":"function","name":"balanceOf","stateMutability":"view",
   "inputs":[{"name":"who","type":"address"}],
   "outputs":[{"name":"","type":"uint256"}]}
]`

func TestUnpackMethodResult_Uint256(t *testing.T) {
	parsed, err := ParseABI(uintReturnABI)
	if err != nil {
		t.Fatal(err)
	}
	// 32-byte big-endian encoding of 12345.
	encoded := make([]byte, 32)
	big.NewInt(12345).FillBytes(encoded)
	values, err := UnpackMethodResult(parsed, "balanceOf", encoded)
	if err != nil {
		t.Fatalf("Unpack: %v", err)
	}
	if len(values) != 1 {
		t.Fatalf("values = %d, want 1", len(values))
	}
	v, ok := values[0].(*big.Int)
	if !ok {
		t.Fatalf("type = %T, want *big.Int", values[0])
	}
	if v.Cmp(big.NewInt(12345)) != 0 {
		t.Errorf("v = %v, want 12345", v)
	}
}

func TestDecodeLog_Happy(t *testing.T) {
	parsed, err := ParseABI(singleIndexedEventABI)
	if err != nil {
		t.Fatal(err)
	}
	event := parsed.Events["Transfer"]
	// topic[0] = event signature; topic[1] = indexed `from` left-padded.
	from := common.HexToAddress("0x000000000000000000000000000000000000dEaD")
	topicFrom := common.BytesToHash(from.Bytes())
	// data = ABI-encoded `value` (32-byte uint256).
	data := make([]byte, 32)
	big.NewInt(12345).FillBytes(data)

	log := types.Log{
		Topics: []common.Hash{event.ID, topicFrom},
		Data:   data,
	}
	out, err := DecodeLog(parsed, "Transfer", log)
	if err != nil {
		t.Fatalf("DecodeLog: %v", err)
	}
	gotFrom, ok := out["from"].(common.Address)
	if !ok {
		t.Fatalf("from type = %T, want common.Address", out["from"])
	}
	if gotFrom != from {
		t.Errorf("from = %s, want %s", gotFrom.Hex(), from.Hex())
	}
	gotVal, ok := out["value"].(*big.Int)
	if !ok {
		t.Fatalf("value type = %T, want *big.Int", out["value"])
	}
	if gotVal.Cmp(big.NewInt(12345)) != 0 {
		t.Errorf("value = %v, want 12345", gotVal)
	}
}
