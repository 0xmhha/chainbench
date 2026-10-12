package testhelper

import (
	"strings"
	"testing"

	"github.com/0xmhha/chainbench/internal/dsl"
)

func argumentCase(t *testing.T, steps string) dsl.Spec {
	t.Helper()
	s, err := dsl.Parse([]byte(`{"schemaVersion":"2","kind":"case","id":"c",` +
		`"chainPreset":{"schemaVersion":"2","kind":"chain-preset","id":"e","chain":"stablenet"},` +
		`"steps":[` + steps + `]}`))
	if err != nil {
		t.Fatalf("parse: %v", err)
	}
	return s
}

// An argument that is read only on another path, or that loses to the
// alternative written beside it, has no effect either, and is refused with
// the reason.
func TestIgnoredArgumentsOnTheOtherPath(t *testing.T) {
	for want, steps := range map[string]string{
		"chainId.onEach":           `{"expect":"chainId","on":"node1","onEach":["node2"],"is":"1"}`,
		"chainId.delta":            `{"expect":"chainId","is":"1","delta":"1"}`,
		"balanceAt.tol":            `{"expect":"balanceAt","address":"node1","compare":"InDelta","delta":"1","tol":"2","is":"1"}`,
		"waitFor.delta":            `{"do":"waitFor","source":"blockNumber","compare":"GreaterOrEqual","expected":"1","delta":"1"},{"expect":"chainId","is":"1"}`,
		"deployContract.data":      `{"do":"deployContract","from":"node1","bytecode":"0x00","data":"0x01"},{"expect":"chainId","is":"1"}`,
		"deployContract.gas":       `{"do":"newAccount","save":"a","saveKey":"k"},{"do":"deployContract","key":"$k","bytecode":"0x00","gas":"100000"},{"expect":"chainId","is":"1"}`,
		"deployContract.from":      `{"do":"newAccount","save":"a","saveKey":"k"},{"do":"deployContract","key":"$k","from":"node1","bytecode":"0x00"},{"expect":"chainId","is":"1"}`,
		"sendTx.from":              `{"do":"newAccount","save":"a","saveKey":"k"},{"do":"sendTx","key":"$k","from":"node1","to":"node2"},{"expect":"chainId","is":"1"}`,
		"contractChecksum.data":    `{"expect":"contractChecksum","bytecode":"0x00","data":"0x01","is":"x"}`,
		"read.address":             `{"do":"read","source":"contractChecksum","bytecode":"0x00","address":"node1","save":"c"},{"expect":"chainId","is":"1"}`,
		"createAddress.from":       `{"expect":"createAddress","deployer":"node1","from":"node2","is":"x"}`,
		"load.fillPercent":         `{"do":"load","from":"node1","gas":"21000","fillPercent":50},{"expect":"chainId","is":"1"}`,
		"sendTx.reason":            `{"do":"sendTx","from":"node1","to":"node2","reason":"x"},{"expect":"chainId","is":"1"}`,
		"sendTx.blocks":            `{"do":"sendTx","from":"node1","to":"node2","expect":"reject","blocks":3},{"expect":"chainId","is":"1"}`,
		"sendTx.expect":            `{"do":"sendTx","from":"node1","to":"node2","expectRevert":true,"expect":"receipt"},{"expect":"chainId","is":"1"}`,
		"startNode.reason":         `{"do":"startNode","on":"node1","reason":"x"},{"expect":"chainId","is":"1"}`,
		"startNode.expect":         `{"do":"startNode","on":"node1","expectFail":true,"expect":"fail"},{"expect":"chainId","is":"1"}`,
		"derive.selector":          `{"expect":"derive","op":"sum","of":["1"],"selector":"0x12345678","is":"1"}`,
		"derive.format":            `{"expect":"derive","op":"word","of":["0x00"],"format":"hex","is":"1"}`,
		"derive.index":             `{"expect":"derive","op":"sum","of":["1"],"index":1,"is":"1"}`,
		"logs.index":               `{"expect":"logs","address":"node1","index":1,"is":"0"}`,
		"receiptLog.topic":         `{"expect":"receiptLog","hash":"0x00","select":"data","topic":1,"is":"0x"}`,
		"receiptLog.select":        `{"expect":"receiptLog","hash":"0x00","select":"topics","is":"0x"}`,
		"partition.method":         `{"do":"partition","groups":[["node1"],["node2"]],"method":"bogus"},{"expect":"chainId","is":"1"}`,
		"swapNode.purpose":         `{"do":"swapNode","on":"node1","binary":"default","purpose":"x"},{"expect":"chainId","is":"1"}`,
		"blockInterval.maxSeconds": `{"expect":"blockInterval","maxSeconds":2,"maxMillis":1500}`,
		"blockInterval.minSeconds": `{"expect":"blockInterval","minSeconds":1,"minMillis":500}`,
		"validators.compare":       `{"expect":"validators","compare":"InDelta","is":4}`,
		"contractChecksum.compare": `{"expect":"contractChecksum","bytecode":"0x00","compare":"InDelta","is":"x"}`,
		"waitFor.compare":          `{"do":"waitFor","source":"validators","compare":"InDelta","expected":4},{"expect":"chainId","is":"1"}`,
	} {
		got := strings.Join(IgnoredArguments(argumentCase(t, steps)), ", ")
		if !strings.Contains(got, want) {
			t.Errorf("%s: ignored = %q, want %s", steps, got, want)
		}
	}
}

// The same arguments on the path that reads them are accepted.
func TestReadArgumentsOnTheirPath(t *testing.T) {
	for _, steps := range []string{
		`{"expect":"balanceAt","address":"node1","compare":"InDelta","delta":"1","is":"1"}`,
		`{"expect":"balanceAt","address":"node1","compare":"InDelta","tol":"1","is":"1"}`,
		`{"do":"deployContract","from":"node1","data":"0x01","gas":"100000"},{"expect":"chainId","is":"1"}`,
		`{"do":"sendTx","from":"node1","to":"node2","expect":"reject","reason":"x"},{"expect":"chainId","is":"1"}`,
		`{"do":"sendTx","from":"node1","to":"node2","expect":"keptOut","reason":"x","blocks":3},{"expect":"chainId","is":"1"}`,
		`{"do":"sendTx","from":"node1","to":"node2","expectReject":true,"reason":"x"},{"expect":"chainId","is":"1"}`,
		`{"do":"startNode","on":"node1","expect":"fail","reason":"x"},{"expect":"chainId","is":"1"}`,
		`{"expect":"derive","op":"abiCall","selector":"0x12345678","of":["1"],"is":"1"}`,
		`{"expect":"derive","op":"word","of":["0x00"],"index":1,"is":"1"}`,
		`{"expect":"derive","op":"sum","of":["1"],"format":"hex","is":"1"}`,
		`{"expect":"logs","address":"node1","select":"data","index":1,"is":"0x"}`,
		`{"expect":"receiptLog","hash":"0x00","select":"data","is":"0x"}`,
		`{"expect":"receiptLog","hash":"0x00","topic":2,"is":"0x"}`,
		`{"expect":"chainId","onEach":["node1","node2"],"is":"1"}`,
		`{"do":"partition","groups":[["node1"],["node2"]],"method":"peers"},{"do":"healPartition","groups":[["node1"],["node2"]],"method":"peers"},{"expect":"chainId","is":"1"}`,
		`{"do":"swapNode","on":"node1","config":{"cache":512},"purpose":"x"},{"expect":"chainId","is":"1"}`,
		`{"expect":"blockInterval","blocks":5,"maxMillis":1500,"minSeconds":1}`,
		`{"expect":"validators","compare":"Len","is":4}`,
		// blocks is read on the chain whose outcome is keptOut.
		`{"do":"sendTx","from":"node1","to":"node2","expect":"reject","expectPerChain":{"wbft":"keptOut"},"blocks":5,"reason":"x"},{"expect":"chainId","is":"1"}`,
	} {
		if got := IgnoredArguments(argumentCase(t, steps)); len(got) > 0 {
			t.Errorf("%s: ignored = %v, want none", steps, got)
		}
	}
}
