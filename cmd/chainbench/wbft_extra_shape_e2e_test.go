//go:build e2e

package main

import (
	"encoding/hex"
	"fmt"
	"strconv"
	"strings"
	"testing"
)

func verifyExtraEpochShape(number, epochLength uint64, extra map[string]any) error {
	if epochLength == 0 {
		return fmt.Errorf("unknown actual epoch length")
	}
	epoch, exists := extra["epochInfo"]
	if !exists {
		return fmt.Errorf("epochInfo omitted rather than null")
	}
	if number%epochLength != 0 && epoch != nil {
		return fmt.Errorf("ordinary header carries epoch")
	}
	if number%epochLength == 0 {
		object, ok := epoch.(map[string]any)
		if !ok {
			return fmt.Errorf("boundary epoch absent")
		}
		validators, ok := object["validators"].([]any)
		if !ok || len(validators) == 0 {
			return fmt.Errorf("boundary validator keys absent")
		}
	}
	round, ok := extra["prevRound"].(string)
	if !ok || !strings.HasPrefix(round, "0x") {
		return fmt.Errorf("previous round absent")
	}
	if _, err := strconv.ParseUint(round[2:], 16, 32); err != nil {
		return fmt.Errorf("previous round invalid")
	}
	decode := func(value any, size int) bool {
		v, ok := value.(string)
		if !ok || !strings.HasPrefix(v, "0x") {
			return false
		}
		raw, err := hex.DecodeString(v[2:])
		return err == nil && len(raw) == size
	}
	if !decode(extra["randaoReveal"], 65) {
		return fmt.Errorf("invalid actual randao reveal")
	}
	seal, ok := extra["committedSeal"].(map[string]any)
	if !ok || !decode(seal["signature"], 96) {
		return fmt.Errorf("invalid committed signature")
	}
	return nil
}

func TestExtraEpochShapeRejectsImplicitNull(t *testing.T) {
	fixture := func() map[string]any {
		return map[string]any{"epochInfo": nil, "prevRound": "0x0", "randaoReveal": "0x" + strings.Repeat("ab", 65), "committedSeal": map[string]any{"signature": "0x" + strings.Repeat("cd", 96)}}
	}
	if err := verifyExtraEpochShape(25, 10, fixture()); err != nil {
		t.Fatal(err)
	}
	for _, tc := range []struct {
		name   string
		change func(map[string]any)
	}{
		{"missing epoch field", func(p map[string]any) { delete(p, "epochInfo") }},
		{"nonboundary epoch object", func(p map[string]any) { p["epochInfo"] = map[string]any{} }},
		{"missing previous round", func(p map[string]any) { delete(p, "prevRound") }},
		{"invalid previous round", func(p map[string]any) { p["prevRound"] = "oops" }},
		{"short randao", func(p map[string]any) { p["randaoReveal"] = "0xab" }},
		{"short signature", func(p map[string]any) { p["committedSeal"] = map[string]any{"signature": "0xab"} }},
	} {
		t.Run(tc.name, func(t *testing.T) {
			p := fixture()
			tc.change(p)
			if err := verifyExtraEpochShape(25, 10, p); err == nil {
				t.Fatal("implicit or malformed null accepted")
			}
		})
	}
	if err := verifyExtraEpochShape(30, 10, fixture()); err == nil {
		t.Fatal("boundary accepted with null epoch")
	}
}

func TestExtraEpochShapeRequiresBoundaryControl(t *testing.T) {
	p := map[string]any{"epochInfo": map[string]any{"validators": []any{map[string]any{"addr": "validator", "bls": "key"}}}, "prevRound": "0x0", "randaoReveal": "0x" + strings.Repeat("ab", 65), "committedSeal": map[string]any{"signature": "0x" + strings.Repeat("cd", 96)}}
	if err := verifyExtraEpochShape(30, 10, p); err != nil {
		t.Fatal(err)
	}
	p["epochInfo"] = map[string]any{"validators": []any{}}
	if err := verifyExtraEpochShape(30, 10, p); err == nil {
		t.Fatal("empty boundary control accepted")
	}
	if err := verifyExtraEpochShape(25, 0, p); err == nil {
		t.Fatal("unknown epoch divisor accepted")
	}
}
