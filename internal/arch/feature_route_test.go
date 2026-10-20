package arch_test

import (
	"testing"

	"github.com/0xmhha/chainbench/internal/arch"
)

func TestFeatureBindingRouteKeepsCoreBypassesVisible(t *testing.T) {
	for _, sample := range []struct {
		packages []string
		via      string
		bypass   bool
	}{
		{[]string{"feature"}, "app", false},
		{[]string{"app", "feature"}, "app", false},
		{[]string{"feature", "resource"}, "app+core", true},
		{[]string{"core/rpc"}, "core", true},
	} {
		entry := arch.Entry{Pkgs: sample.packages}
		if entry.Via() != sample.via || entry.ReachesPastApp() != sample.bypass {
			t.Errorf("route %v: via=%s bypass=%t", sample.packages, entry.Via(), entry.ReachesPastApp())
		}
	}
}
