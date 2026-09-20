package testhelper

import "testing"

// A comparison value that is a key-set name is resolved to an address, which is
// right when the chain answers with an address and wrong when it answers with
// the name. The rule reads the answer rather than the position, so both shapes
// are decided by the same line.
//
// The wrong half was measured on 2026-09-19: on a 15-node wemix network
// default-on-routes-every-step compared admin_wemixInfo.self.name — which is
// "node1" — against 0x23222a98…, and failed while the network was correct.
func TestComparedAsWritten(t *testing.T) {
	for _, tc := range []struct {
		name   string
		actual any
		want   bool
	}{
		{"a node name keeps the name", "node1", true},
		{"a block's miner keeps the address", "0x23222a986facbd40933e9ca842d744d73d568ee2", false},
		{"upper-case hex is still an address", "0X23222A98", false},
		{"an empty answer decides nothing", "", false},
		{"a non-string answer decides nothing", 42, false},
		{"a hex quantity is not a name", "0x1", false},
	} {
		t.Run(tc.name, func(t *testing.T) {
			if got := ComparedAsWritten(tc.actual); got != tc.want {
				t.Errorf("ComparedAsWritten(%#v) = %v, want %v", tc.actual, got, tc.want)
			}
		})
	}
}
