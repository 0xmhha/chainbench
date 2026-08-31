package resource_test

import (
	"github.com/0xmhha/chainbench/internal/resource"

	"strings"
	"testing"
)

func TestNetworkID(t *testing.T) {
	if id, err := resource.NetworkID(8285); err != nil || id != 8285 {
		t.Errorf("resource.NetworkID(8285) = %d, %v", id, err)
	}
	if _, err := resource.NetworkID(0); err == nil {
		t.Error("resource.NetworkID(0) should error (no default)")
	}
}

func TestNetworkIDFlag(t *testing.T) {
	got := resource.NetworkIDFlag(8285)
	if len(got) != 2 || got[0] != "--networkid" || got[1] != "8285" {
		t.Errorf("resource.NetworkIDFlag(8285) = %v", got)
	}
}

func TestValidateNetworkIDs(t *testing.T) {
	if err := resource.ValidateNetworkIDs([]int64{8285, 8285, 8285}); err != nil {
		t.Errorf("uniform ids should pass: %v", err)
	}
	err := resource.ValidateNetworkIDs([]int64{8285, 1111})
	if err == nil || !strings.Contains(err.Error(), "mismatch") {
		t.Errorf("mismatch should error, got %v", err)
	}
	if err := resource.ValidateNetworkIDs([]int64{8285, 0}); err == nil {
		t.Error("invalid id should error")
	}
	if err := resource.ValidateNetworkIDs(nil); err == nil {
		t.Error("empty should error")
	}
}
