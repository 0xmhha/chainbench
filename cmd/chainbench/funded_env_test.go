package main

import "testing"

func TestFundedKeyFromEnv(t *testing.T) {
	t.Setenv("CHAINBENCH_FUNDED_KEY", "")
	if k, err := fundedKeyFromEnv(); err != nil || k != nil {
		t.Errorf("empty env: key=%v err=%v, want nil,nil", k, err)
	}
	t.Setenv("CHAINBENCH_FUNDED_KEY", "0xdeadbeef")
	if k, err := fundedKeyFromEnv(); err != nil || len(k) != 4 {
		t.Errorf("0x hex: key=%x err=%v, want 4 bytes", k, err)
	}
	t.Setenv("CHAINBENCH_FUNDED_KEY", "deadbeef")
	if k, err := fundedKeyFromEnv(); err != nil || len(k) != 4 {
		t.Errorf("bare hex: key=%x err=%v, want 4 bytes", k, err)
	}
	t.Setenv("CHAINBENCH_FUNDED_KEY", "not-hex")
	if _, err := fundedKeyFromEnv(); err == nil {
		t.Error("invalid hex should error")
	}
}
