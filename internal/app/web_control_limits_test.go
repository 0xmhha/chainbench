package app

import "testing"

// Every node control a node does not offer carries the reason it is refused,
// so a screen can show the button disabled with that reason.
func TestNodeControlLimitsExplainEveryRefusedControl(t *testing.T) {
	producer := webControlLimits(WebNode{Role: "bp", State: "running", SupportedControls: []string{"node.start", "node.stop", "node.restart", "node.swap"}}, "owned", false)
	if len(producer) != 1 || producer["node.reset"] == "" {
		t.Fatalf("producer limits = %v", producer)
	}
	unobserved := webControlLimits(WebNode{Role: "en", State: "unknown", SupportedControls: []string{}}, "owned", false)
	if len(unobserved) != 5 {
		t.Fatalf("unobserved limits = %v", unobserved)
	}
	attached := webControlLimits(WebNode{Role: "en", State: "running", SupportedControls: []string{}}, "attached", false)
	if len(attached) != 5 || attached["node.stop"] == unobserved["node.stop"] {
		t.Fatalf("attached limits = %v", attached)
	}
	configured := webControlLimits(WebNode{Role: "en", State: "running", SupportedControls: []string{"node.start", "node.stop", "node.restart", "node.reset"}}, "owned", true)
	if len(configured) != 1 || configured["node.swap"] == "" {
		t.Fatalf("config file limits = %v", configured)
	}
	if probe := webControlLimits(WebNode{Role: "en", State: "unknown", ObservationReason: "probe_unavailable", SupportedControls: []string{}}, "owned", false); probe["node.stop"] == unobserved["node.stop"] {
		t.Fatalf("an unreachable target reads like an unchecked one: %v", probe)
	}
	launch := webControlLimits(WebNode{Role: "en", State: "running", ObservationReason: "binary_launch_unavailable", SupportedControls: []string{"node.stop"}}, "owned", false)
	if len(launch) != 4 || launch["node.start"] == unobserved["node.start"] {
		t.Fatalf("launch limits = %v", launch)
	}
}
