package app

import (
	"reflect"
	"testing"

	"github.com/0xmhha/chainbench/internal/core/node"
	"github.com/0xmhha/chainbench/internal/resource"
)

func TestPlanSummaryNamesWhatRunsAndWhere(t *testing.T) {
	layout, err := node.NewMap([]node.Placement{
		{Index: 1, Label: "node1", Role: "bp", Host: "192.0.2.6"},
		{Index: 2, Label: "node2", Role: "en", Host: "192.0.2.7"},
		{Index: 3, Label: "node3", Role: "pn", Host: "192.0.2.9"},
	})
	if err != nil {
		t.Fatal(err)
	}
	servers := []resource.Server{{Name: "server6", Host: "192.0.2.6"}, {Name: "server7", Host: "192.0.2.7"}}
	got := webPlanSummary("wbft", "/data/bin/abc/gwbft", []string{"round trip"}, servers, layout)
	want := &WebPlanSummary{Chain: "wbft", Binary: "gwbft", Cases: []string{"round trip"}, Nodes: []WebPlanNode{
		{Label: "node1", Role: "bp", Server: "server6"},
		{Label: "node2", Role: "en", Server: "server7"},
		{Label: "node3", Role: "pn", Server: "192.0.2.9"},
	}}
	if !reflect.DeepEqual(got, want) {
		t.Fatalf("got %+v, want %+v", got, want)
	}
}
