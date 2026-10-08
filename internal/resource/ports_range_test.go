package resource_test

import (
	"github.com/0xmhha/chainbench/internal/core/node"
	"github.com/0xmhha/chainbench/internal/resource"
	"math"
	"testing"
)

func TestPortBandsRefuseRangeAndArithmeticOverflow(t *testing.T) {
	for _, tc := range []struct {
		name                                      string
		index, p2pBase, p2pStep, rpcBase, rpcStep int
	}{
		{"derived HTTP overflow", 1, 31000, 10, 65535, 10},
		{"last slot overflow", 2, 65530, 10, 8600, 10},
		{"integer multiplication overflow", math.MaxInt, 31000, math.MaxInt, 8600, 10},
		{"negative base", 1, -1, 10, 8600, 10},
	} {
		t.Run(tc.name, func(t *testing.T) {
			if _, err := resource.Plan(tc.index, tc.p2pBase, tc.p2pStep, tc.rpcBase, tc.rpcStep, node.Reservation{}); err == nil {
				t.Fatal("unusable port plan accepted")
			}
		})
	}
	if _, err := resource.Plan(1, 31000, 10, 65532, 10, node.Reservation{}); err != nil {
		t.Fatal("valid upper boundary rejected:", err)
	}
	for _, p := range []node.Endpoints{{P2P: 65536}, {HTTP: -1}, {EtcdClient: 65536}} {
		if err := resource.ValidatePorts([]node.Endpoints{p}); err == nil {
			t.Fatal("invalid endpoint accepted")
		}
	}
}
