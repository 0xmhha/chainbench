package adapters_test

import (
	"errors"
	"testing"

	"github.com/0xmhha/chainbench/network/internal/adapters"
)

func TestLoad_Stablenet(t *testing.T) {
	a, err := adapters.Load("stablenet")
	if err != nil {
		t.Fatalf("Load(stablenet): %v", err)
	}
	if a == nil {
		t.Fatal("Load returned nil adapter")
	}
	if ns := a.ConsensusRpcNamespace(); ns != "istanbul" {
		t.Errorf("stablenet namespace = %q, want istanbul", ns)
	}
}

func TestLoad_WBFT(t *testing.T) {
	a, err := adapters.Load("wbft")
	if err != nil {
		t.Fatalf("Load(wbft): %v", err)
	}
	if ns := a.ConsensusRpcNamespace(); ns != "istanbul" {
		t.Errorf("wbft namespace = %q, want istanbul", ns)
	}
}

func TestLoad_Wemix(t *testing.T) {
	a, err := adapters.Load("wemix")
	if err != nil {
		t.Fatalf("Load(wemix): %v", err)
	}
	if ns := a.ConsensusRpcNamespace(); ns != "wemix" {
		t.Errorf("wemix namespace = %q, want wemix", ns)
	}
}

func TestLoad_UnknownChainType(t *testing.T) {
	_, err := adapters.Load("ethereum")
	if !errors.Is(err, adapters.ErrUnknownChainType) {
		t.Errorf("err = %v, want ErrUnknownChainType", err)
	}
}
