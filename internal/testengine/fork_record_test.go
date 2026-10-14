package testengine

import (
	"encoding/json"
	"testing"

	"github.com/0xmhha/chainbench/internal/chainsetup/verb"
	"github.com/0xmhha/chainbench/internal/core/health"
)

func TestRecordedHandoverCannotParkOnReattachment(t *testing.T) {
	var f verb.ChainForkOut
	if err := json.Unmarshal([]byte(`{"fork":{"name":"croissant","at":20,"binary":"next"},"preFork":[5],"handedOver":true}`), &f); err != nil {
		t.Fatal(err)
	}
	g := forkGateFromRecord(f)
	rep := health.Report{Nodes: []health.NodeInfo{{Index: 1, OK: true, BlockNumber: 19}}}
	if g.parked(rep) {
		t.Fatal("reattachment forgot handover and accepted fork-1 as producing")
	}
	if !g.retired(5, 20) {
		t.Fatal("producer retirement lost")
	}
	if got := forkGateFromRecord(verb.ChainForkOut{HaltsAt: 20}); !got.parked(rep) {
		t.Fatal("non-handover genesis halt behavior changed")
	}
}
