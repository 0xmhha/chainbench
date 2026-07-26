package netid

import (
	"strings"
	"testing"
)

func TestResolve(t *testing.T) {
	if id, err := Resolve(8285); err != nil || id != 8285 {
		t.Errorf("Resolve(8285) = %d, %v", id, err)
	}
	if _, err := Resolve(0); err == nil {
		t.Error("Resolve(0) should error (no default)")
	}
}

func TestFlag(t *testing.T) {
	got := Flag(8285)
	if len(got) != 2 || got[0] != "--networkid" || got[1] != "8285" {
		t.Errorf("Flag(8285) = %v", got)
	}
}

func TestValidateUniform(t *testing.T) {
	if err := ValidateUniform([]int64{8285, 8285, 8285}); err != nil {
		t.Errorf("uniform ids should pass: %v", err)
	}
	err := ValidateUniform([]int64{8285, 1111})
	if err == nil || !strings.Contains(err.Error(), "mismatch") {
		t.Errorf("mismatch should error, got %v", err)
	}
	if err := ValidateUniform([]int64{8285, 0}); err == nil {
		t.Error("invalid id should error")
	}
	if err := ValidateUniform(nil); err == nil {
		t.Error("empty should error")
	}
}
