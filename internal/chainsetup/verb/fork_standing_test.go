package verb

import (
	"encoding/json"
	"os"
	"path/filepath"
	"testing"

	"github.com/0xmhha/chainbench/internal/chainsetup"
	"github.com/0xmhha/chainbench/internal/core/node"
)

func TestChainForkReportsPersistedHandoverWithoutRPC(t *testing.T) {
	for _, c := range []struct {
		name  string
		roles []string
		want  bool
	}{
		{"before", []string{"en", "en"}, false},
		{"partial", []string{"bp", "en"}, false},
		{"handed over", []string{"bp", "bp"}, true},
	} {
		t.Run(c.name, func(t *testing.T) {
			dir := t.TempDir()
			state := chainsetup.State{FormatVersion: chainsetup.StateFormatVersion, Chain: "wemix", Fork: &chainsetup.GenesisFork{Name: "croissant", At: 20, Binary: "next"}, Nodes: []node.Record{{Index: 1, Role: c.roles[0], Binary: "next"}, {Index: 2, Role: c.roles[1], Binary: "next"}, {Index: 5, Role: "bp"}}}
			data, err := json.Marshal(state)
			if err != nil {
				t.Fatal(err)
			}
			if err := os.WriteFile(filepath.Join(dir, "chain-record.json"), data, 0600); err != nil {
				t.Fatal(err)
			}
			out, err := ChainFork(t.Context(), chainsetup.Deps{}, ChainForkIn{DataDir: dir})
			if err != nil {
				t.Fatal(err)
			}
			data, err = json.Marshal(out)
			if err != nil {
				t.Fatal(err)
			}
			var got struct{ HandedOver bool }
			if err := json.Unmarshal(data, &got); err != nil {
				t.Fatal(err)
			}
			if got.HandedOver != c.want {
				t.Fatalf("recorded handover missing: %s", data)
			}
			if len(out.PreFork) != 1 || out.PreFork[0] != 5 {
				t.Fatal("producer identity changed", out.PreFork)
			}
		})
	}
}
