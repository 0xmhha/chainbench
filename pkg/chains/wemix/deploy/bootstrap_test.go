package deploy

import (
	"strings"
	"testing"
)

func TestPoaCommand(t *testing.T) {
	got := poaCommand("/r/gwemix", []string{"wemix", "deploy-governance", "--url", "/r/gwemix.ipc"})
	want := "'/r/gwemix' 'wemix' 'deploy-governance' '--url' '/r/gwemix.ipc'"
	if got != want {
		t.Errorf("poaCommand = %q, want %q", got, want)
	}
}

func TestBootProducer(t *testing.T) {
	c := &Cluster{Servers: []Server{
		{Index: 1, Host: "a", Role: RoleWbftBP},
		{Index: 2, Host: "b", Role: RoleWemixBP},
		{Index: 3, Host: "c", Role: RoleWemixBP},
	}}
	boot, ok := BootProducer(c)
	if !ok || boot.Index != 2 {
		t.Errorf("boot producer = %+v ok=%v, want index 2", boot, ok)
	}
	if _, ok := BootProducer(&Cluster{Servers: []Server{{Index: 1, Host: "a", Role: RoleWbftBP}}}); ok {
		t.Error("expected no boot producer when there are no wemix_bp servers")
	}
}

func TestBuildWemixConfig(t *testing.T) {
	c := &Cluster{
		RPCPort: 8601, P2PPort: 30303,
		Servers: []Server{
			{Index: 1, Host: "10.0.0.1", Role: RoleWemixBP},
			{Index: 2, Host: "10.0.0.2", Role: RoleWemixBP},
			{Index: 3, Host: "10.0.0.3", Role: RoleWbftBP},
		},
	}
	id1 := strings.Repeat("ab", 64)        // 128-hex idv5 (no 0x)
	id2 := "0x" + strings.Repeat("cd", 64) // already 0x-prefixed
	a := &Accounts{
		Producers: []NodeAcct{
			{Server: 1, Addr: "0xproducer1", NodeID: id1, Stake: "3000000000000000000000000000"},
			{Server: 2, Addr: "0xproducer2", NodeID: id2},
		},
		Validators: []NodeAcct{{Server: 3, Addr: "0xval3"}},
	}
	cfg, err := BuildWemixConfig(c, a)
	if err != nil {
		t.Fatalf("build: %v", err)
	}
	if len(cfg.Members) != 2 {
		t.Fatalf("members = %d, want 2", len(cfg.Members))
	}
	if cfg.Staker != "0xproducer1" {
		t.Errorf("staker = %q, want producer1", cfg.Staker)
	}
	m0 := cfg.Members[0]
	if !m0.Bootnode || m0.ID != "0x"+id1 || m0.IP != "10.0.0.1" || m0.Port != 30303 {
		t.Errorf("member0 = %+v", m0)
	}
	if cfg.Members[1].Bootnode {
		t.Error("only the first producer is the bootnode")
	}
	if cfg.Members[1].ID != id2 { // already-0x id passes through unchanged
		t.Errorf("member1 id = %q", cfg.Members[1].ID)
	}
	// producers (2) + validator (1) all funded.
	if len(cfg.Accounts) != 3 {
		t.Errorf("funded accounts = %d, want 3", len(cfg.Accounts))
	}
}

func TestBuildWemixConfig_Errors(t *testing.T) {
	c := &Cluster{RPCPort: 8601, Servers: []Server{{Index: 1, Host: "h", Role: RoleWemixBP}}}
	// no producer account
	if _, err := BuildWemixConfig(c, &Accounts{}); err == nil {
		t.Error("expected error for missing producer account")
	}
	// producer without node_id
	if _, err := BuildWemixConfig(c, &Accounts{Producers: []NodeAcct{{Server: 1, Addr: "0xa"}}}); err == nil || !strings.Contains(err.Error(), "node_id") {
		t.Errorf("expected node_id error, got %v", err)
	}
}
