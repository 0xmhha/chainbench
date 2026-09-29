package testhelper

import (
	"context"
	"errors"
	"strings"
	"testing"

	"github.com/0xmhha/chainbench/internal/core/rpc"
	"github.com/0xmhha/chainbench/internal/dsl/interp"
)

// TestReadValidators_AsksTheRunHowThisChainAnswers.
//
// The point of the reader is that it holds no method name. Whatever the run
// bound as Deps.Validators is what answers, so the same case reads a wbft
// chain through istanbul_getValidators and a wemix chain through its
// governance contract without saying either.
func TestReadValidators_AsksTheRunHowThisChainAnswers(t *testing.T) {
	want := []string{"0xaaa", "0xbbb", "0xccc"}
	var got *rpc.Client
	d := &interp.Deps{Validators: func(_ context.Context, c *rpc.Client) ([]string, error) {
		got = c
		return want, nil
	}}

	client := rpc.Dial("http://127.0.0.1:1")
	out, err := readValidators(t.Context(), d, client, nil)
	if err != nil {
		t.Fatal(err)
	}
	vals, ok := out.([]string)
	if !ok {
		t.Fatalf("validators read back as %T, want a slice so Len and Contains both work", out)
	}
	if len(vals) != len(want) {
		t.Errorf("got %v, want %v", vals, want)
	}
	if got != client {
		t.Error("the reader asked some other node than the one it was given")
	}
}

// TestReadValidators_ARunWithNoChainSaysSo: a run that resolved no chain has
// no way to know the route, and saying that beats calling a guessed method and
// reporting the node's "method not found" as if the chain were at fault.
func TestReadValidators_ARunWithNoChainSaysSo(t *testing.T) {
	_, err := readValidators(t.Context(), &interp.Deps{}, nil, nil)
	if err == nil || !strings.Contains(err.Error(), "no chain") {
		t.Errorf("want a refusal naming the missing chain, got %v", err)
	}
}

// TestReadValidators_TheChainsOwnFailureIsCarried: whatever went wrong asking
// this chain is what the case is told, wrapped rather than replaced.
func TestReadValidators_TheChainsOwnFailureIsCarried(t *testing.T) {
	boom := errors.New("governance contract answered nothing")
	d := &interp.Deps{Validators: func(context.Context, *rpc.Client) ([]string, error) { return nil, boom }}
	_, err := readValidators(t.Context(), d, nil, nil)
	if !errors.Is(err, boom) {
		t.Errorf("the chain's own error was lost: %v", err)
	}
}
