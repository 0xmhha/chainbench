//go:build e2e

package main

import (
	"fmt"
	"github.com/0xmhha/chainbench/internal/core/rpc"
	"strings"
	"testing"
	"time"
)

type validatorObservation struct {
	URL, Address      string
	Boundary, Height  uint64
	Default, AtHeight bool
	Validators        []string
}

func verifyValidatorTransition(rows []validatorObservation, subject string, epochLength uint64) error {
	if len(rows) != 3 || epochLength == 0 || subject == "" || !rows[0].Default || rows[1].Default || !rows[2].Default {
		return fmt.Errorf("wrong actual transition context")
	}
	for i, row := range rows {
		if row.URL == "" || row.URL != rows[0].URL || !strings.EqualFold(row.Address, subject) || row.Boundary%epochLength != 0 || row.Height <= row.Boundary || row.Default != row.AtHeight || len(row.Validators) == 0 || i > 0 && (row.Boundary <= rows[i-1].Boundary || row.Height <= rows[i-1].Height) {
			return fmt.Errorf("different node or unapplied epoch")
		}
		seen := map[string]bool{}
		for _, v := range row.Validators {
			v = strings.ToLower(v)
			if v == "" || seen[v] {
				return fmt.Errorf("invalid current validator set")
			}
			seen[v] = true
		}
		if seen[strings.ToLower(subject)] != row.AtHeight {
			return fmt.Errorf("validator RPC disagrees with current membership")
		}
	}
	return nil
}
func observeValidatorTransition(t *testing.T, c *rpc.Client, url, subject string, boundary uint64) validatorObservation {
	t.Helper()
	waitForHead(t, c, boundary+1, 90*time.Second)
	p := validatorObservation{URL: url, Boundary: boundary}
	var err error
	if p.Height, err = c.BlockNumber(t.Context()); err != nil {
		t.Fatal(err)
	}
	if err = c.Call(t.Context(), "istanbul_nodeAddress", &p.Address); err != nil || !strings.EqualFold(p.Address, subject) {
		t.Fatal("not subject node identity", p.Address, err)
	}
	var flag *bool
	if err = c.Call(t.Context(), "istanbul_isValidator", &flag); err != nil || flag == nil {
		t.Fatal(err)
	}
	p.Default = *flag
	flag = nil
	tag := fmt.Sprintf("0x%x", p.Height)
	if err = c.Call(t.Context(), "istanbul_isValidator", &flag, tag); err != nil || flag == nil {
		t.Fatal(err)
	}
	p.AtHeight = *flag
	if err = c.Call(t.Context(), "istanbul_getValidators", &p.Validators, tag); err != nil {
		t.Fatal(err)
	}
	return p
}

func TestValidatorTransitionRejectsStaticRoleEvidence(t *testing.T) {
	valid := func() []validatorObservation {
		return []validatorObservation{
			{URL: "same", Address: "subject", Boundary: 30, Height: 31, Default: true, AtHeight: true, Validators: []string{"subject", "support"}},
			{URL: "same", Address: "subject", Boundary: 40, Height: 41, Default: false, AtHeight: false, Validators: []string{"support"}},
			{URL: "same", Address: "subject", Boundary: 50, Height: 51, Default: true, AtHeight: true, Validators: []string{"subject", "support"}},
		}
	}
	if err := verifyValidatorTransition(valid(), "subject", 10); err != nil {
		t.Fatal(err)
	}
	for _, tc := range []struct {
		name   string
		change func([]validatorObservation)
	}{
		{"different endpoint", func(p []validatorObservation) { p[1].URL = "other" }},
		{"different identity", func(p []validatorObservation) { p[1].Address = "other" }},
		{"boundary not applied", func(p []validatorObservation) { p[1].Height = p[1].Boundary }},
		{"historical epoch", func(p []validatorObservation) { p[1].Boundary = 20 }},
		{"not boundary", func(p []validatorObservation) { p[1].Boundary = 39 }},
		{"explicit RPC mismatch", func(p []validatorObservation) { p[1].AtHeight = true }},
		{"excluded still in set", func(p []validatorObservation) { p[1].Validators = append(p[1].Validators, "subject") }},
		{"empty validators", func(p []validatorObservation) { p[1].Validators = nil }},
	} {
		t.Run(tc.name, func(t *testing.T) {
			p := valid()
			tc.change(p)
			if err := verifyValidatorTransition(p, "subject", 10); err == nil {
				t.Fatal("static role or wrong epoch accepted")
			}
		})
	}
}
