package preflight_test

import (
	"github.com/0xmhha/chainbench/internal/core/node"
	"github.com/0xmhha/chainbench/internal/core/preflight"
	"github.com/0xmhha/chainbench/internal/resource"
	"strings"
	"testing"
)

func goldenPorts(n int) []node.Endpoints {
	var ps []node.Endpoints
	for i := 1; i <= n; i++ {
		p, _ := resource.Plan(i, 30010, 10, 40010, 10, node.DefaultReservation)
		ps = append(ps, p)
	}
	return ps
}

func TestValidate_OK(t *testing.T) {
	p := preflight.NetworkPlan{
		NetworkIDs:     []int64{8285, 8285, 8285},
		Ports:          goldenPorts(3),
		Genesis:        []byte(`{"config":{"petersburgBlock":0,"croissantBlock":20,"croissant":{}}}`),
		WemixMembers:   []string{"0xProd"},
		WbftValidators: []string{"0xAAA", "0xBBB"},
	}
	if err := preflight.Validate(p); err != nil {
		t.Fatalf("valid plan rejected: %v", err)
	}
}

func TestValidate_Catches(t *testing.T) {
	base := func() preflight.NetworkPlan {
		return preflight.NetworkPlan{NetworkIDs: []int64{8285, 8285}, Ports: goldenPorts(2),
			Genesis: []byte(`{"config":{"petersburgBlock":0}}`)}
	}
	// network id mismatch
	p := base()
	p.NetworkIDs = []int64{8285, 1111}
	if err := preflight.Validate(p); err == nil || !strings.Contains(err.Error(), "mismatch") {
		t.Errorf("networkid mismatch not caught: %v", err)
	}
	// member/validator overlap
	p = base()
	p.WemixMembers = []string{"0xShared"}
	p.WbftValidators = []string{"0xSHARED"} // case-insensitive
	if err := preflight.Validate(p); err == nil || !strings.Contains(err.Error(), "both a wemix member") {
		t.Errorf("member/validator overlap not caught: %v", err)
	}
	// genesis missing petersburg
	p = base()
	p.Genesis = []byte(`{"config":{"croissantBlock":20}}`)
	if err := preflight.Validate(p); err == nil {
		t.Error("missing petersburg not caught")
	}
	// port collision (etcd overlaps http)
	p = base()
	p.Ports = []node.Endpoints{{P2P: 100, Etcd: 101, HTTP: 101, WS: 102, Auth: 103}}
	p.NetworkIDs = []int64{8285}
	if err := preflight.Validate(p); err == nil {
		t.Error("port collision not caught")
	}
}
