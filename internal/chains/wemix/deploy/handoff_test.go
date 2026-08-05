package deploy

import "testing"

func TestHandoffConfirmed(t *testing.T) {
	producers := lowerSet([]string{"0xPRODUCER1", "0xproducer2"})

	// A validator (not a producer) sealed the post-fork block -> confirmed.
	if !handoffConfirmed("0xVALIDATOR", producers) {
		t.Error("validator sealer should confirm the handoff")
	}
	// A producer sealed it (case-insensitive) -> not yet handed off.
	if handoffConfirmed("0xproducer1", producers) {
		t.Error("producer sealer must not confirm")
	}
	if handoffConfirmed("0xPRODUCER2", producers) {
		t.Error("producer sealer (mixed case) must not confirm")
	}
	// Empty / zero miner -> not confirmed.
	if handoffConfirmed("", producers) || handoffConfirmed("0x0000000000000000000000000000000000000000", producers) {
		t.Error("empty/zero miner must not confirm")
	}
}

func TestProducerAddrs(t *testing.T) {
	a := &Accounts{Producers: []NodeAcct{
		{Server: 1, Addr: "0xa"},
		{Server: 2, Addr: ""}, // skipped
		{Server: 3, Addr: "0xc"},
	}}
	got := a.ProducerAddrs()
	if len(got) != 2 || got[0] != "0xa" || got[1] != "0xc" {
		t.Errorf("ProducerAddrs = %v", got)
	}
}
