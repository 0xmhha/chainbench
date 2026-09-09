package process

import (
	"reflect"
	"testing"
)

func TestSplitCmdline(t *testing.T) {
	cases := []struct {
		name string
		raw  string
		want []string
	}{
		{"trailing nul dropped", "gwbft\x00--datadir\x00/data/n1\x00", []string{"gwbft", "--datadir", "/data/n1"}},
		{"no trailing nul", "gwbft\x00--config\x00/c/n1.toml", []string{"gwbft", "--config", "/c/n1.toml"}},
		{"empty", "", nil},
		{"path with space kept intact", "gwbft\x00--config\x00/c/my node.toml", []string{"gwbft", "--config", "/c/my node.toml"}},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			got := splitCmdline(tc.raw)
			if len(got) == 0 && len(tc.want) == 0 {
				return
			}
			if !reflect.DeepEqual(got, tc.want) {
				t.Fatalf("splitCmdline(%q) = %v, want %v", tc.raw, got, tc.want)
			}
		})
	}
}
