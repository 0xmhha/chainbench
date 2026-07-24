package consensus_test

import (
	"context"
	"errors"
	"testing"

	"github.com/0xmhha/chainbench/pkg/core/consensus"
)

type fakeCaller struct {
	method string
	vals   []string
	err    error
}

func (f *fakeCaller) Call(_ context.Context, method string, out any, _ ...any) error {
	f.method = method
	if f.err != nil {
		return f.err
	}
	*(out.(*[]string)) = f.vals
	return nil
}

func TestValidators(t *testing.T) {
	fc := &fakeCaller{vals: []string{"0xaaa", "0xbbb"}}
	got, err := consensus.Validators(context.Background(), fc, "istanbul_getValidators")
	if err != nil {
		t.Fatal(err)
	}
	if fc.method != "istanbul_getValidators" {
		t.Errorf("method used: %s", fc.method)
	}
	if len(got) != 2 || got[0] != "0xaaa" {
		t.Errorf("validators: %v", got)
	}
}

func TestValidators_Error(t *testing.T) {
	fc := &fakeCaller{err: errors.New("boom")}
	if _, err := consensus.Validators(context.Background(), fc, "wemix_getValidators"); err == nil {
		t.Error("expected error")
	}
}
