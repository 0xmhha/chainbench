package app

import (
	"encoding/json"
	"math"
	"strconv"
	"testing"

	"github.com/0xmhha/chainbench/internal/testhelper"
)

func TestUnsignedEditorRangesMatchRuntimeDomains(t *testing.T) {
	_, schemas, err := testhelper.EditorVocabulary()
	if err != nil {
		t.Fatal(err)
	}
	for _, c := range []struct {
		entry, field                          string
		positive, indexSized, exactFloatBound bool
	}{
		{"action-waitBlock", "target", false, false, false},
		{"action-sendTx", "blocks", true, false, false},
		{"action-sendTx", "gas", false, false, false},
		{"action-sendTx", "nonce", false, false, false},
		{"action-load", "blocks", true, true, false},
		{"action-readNodeLog", "maxBytes", true, true, false},
		{"assertion-blockHalt", "maxAdvance", false, false, false},
		{"assertion-blockInterval", "blocks", true, true, false},
		{"assertion-blockInterval", "maxSeconds", false, false, true},
		{"assertion-blockInterval", "minSeconds", false, false, true},
		{"assertion-blockInterval", "maxMillis", false, false, true},
		{"assertion-blockInterval", "minMillis", false, false, true},
		{"assertion-wsSubscribe", "count", true, true, false},
		{"assertion-wsCollected", "count", true, true, false},
		{"reader-derive", "index", false, true, false},
		{"reader-logs", "index", false, true, false},
		{"reader-receiptLog", "index", false, true, false},
		{"reader-receiptLog", "topic", false, true, false},
		{"reader-createAddress", "nonce", false, false, false},
	} {
		schema := schemas[c.entry]["properties"].(map[string]any)[c.field].(map[string]any)
		maximum := uint64(math.MaxUint64)
		if c.indexSized {
			maximum = uint64(math.MaxInt)
		}
		if c.exactFloatBound {
			maximum = 9007199254740991
		}
		cases := []struct {
			value any
			valid bool
		}{
			{1.5, false}, {-1, false}, {nil, false}, {true, false}, {float64(9007199254740992), false},
			{0, !c.positive}, {"0", !c.positive}, {"0X000", !c.positive},
			{1, true}, {"0001", true}, {"0X01", true}, {"$quantity", true},
			{strconv.FormatUint(maximum, 10), true}, {"0X" + strconv.FormatUint(maximum, 16), true},
			{"18446744073709551616", false},
		}
		if maximum < math.MaxUint64 {
			cases = append(cases, struct {
				value any
				valid bool
			}{strconv.FormatUint(maximum+1, 10), false}, struct {
				value any
				valid bool
			}{"0x" + strconv.FormatUint(maximum+1, 16), false})
		}
		for _, v := range cases {
			raw, _ := json.Marshal(v.value)
			err := validateEditorSchema(raw, schema, schema)
			if (err == nil) != v.valid {
				t.Errorf("%s.%s %v: valid=%v error=%v", c.entry, c.field, v.value, v.valid, err)
			}
		}
	}
}
