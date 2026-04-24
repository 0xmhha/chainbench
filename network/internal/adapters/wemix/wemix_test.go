package wemix

import (
	"context"
	"errors"
	"testing"

	"github.com/0xmhha/chainbench/network/internal/adapters/spec"
)

func TestAdapter_ConsensusRpcNamespace(t *testing.T) {
	if ns := New().ConsensusRpcNamespace(); ns != "wemix" {
		t.Errorf("got %q, want wemix", ns)
	}
}

func TestAdapter_ExtraStartFlags(t *testing.T) {
	want := "--allow-insecure-unlock"
	if flags := New().ExtraStartFlags(spec.RoleValidator); flags != want {
		t.Errorf("got %q, want %q", flags, want)
	}
	if flags := New().ExtraStartFlags(spec.RoleEndpoint); flags != want {
		t.Errorf("got %q, want %q", flags, want)
	}
}

func TestAdapter_GenerateGenesis_NotImplemented(t *testing.T) {
	err := New().GenerateGenesis(context.Background(), spec.GenesisInput{})
	if !errors.Is(err, spec.ErrNotImplemented) {
		t.Errorf("err = %v, want ErrNotImplemented", err)
	}
}

func TestAdapter_GenerateToml_NotImplemented(t *testing.T) {
	err := New().GenerateToml(context.Background(), spec.TomlInput{})
	if !errors.Is(err, spec.ErrNotImplemented) {
		t.Errorf("err = %v, want ErrNotImplemented", err)
	}
}
