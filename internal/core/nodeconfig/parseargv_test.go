package nodeconfig

import (
	"reflect"
	"testing"
)

func TestParseArgv(t *testing.T) {
	cases := []struct {
		name string
		argv []string
		want RunView
	}{
		{
			"space form",
			[]string{"/data/bin/gwbft", "--datadir", "/data/n1", "--config", "/c/n1.toml", "--http"},
			RunView{Binary: "/data/bin/gwbft", DataDir: "/data/n1", ConfigPath: "/c/n1.toml"},
		},
		{
			"equals form",
			[]string{"gwbft", "--datadir=/data/n2", "--config=/c/n2.toml"},
			RunView{Binary: "gwbft", DataDir: "/data/n2", ConfigPath: "/c/n2.toml"},
		},
		{
			"single dash",
			[]string{"gstable", "-datadir", "/d", "-config", "/c.toml"},
			RunView{Binary: "gstable", DataDir: "/d", ConfigPath: "/c.toml"},
		},
		{
			"unknown flags ignored, no config",
			[]string{"gwbft", "--verbosity", "3", "--datadir", "/d", "--nodiscover"},
			RunView{Binary: "gwbft", DataDir: "/d"},
		},
		{
			"empty argv",
			nil,
			RunView{},
		},
		{
			"trailing flag with no value",
			[]string{"gwbft", "--datadir"},
			RunView{Binary: "gwbft"},
		},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			if got := ParseArgv(tc.argv); !reflect.DeepEqual(got, tc.want) {
				t.Fatalf("ParseArgv(%v) = %+v, want %+v", tc.argv, got, tc.want)
			}
		})
	}
}

// TestParseArgv_RoundTripsArgv proves the reader inverts the writer for the two
// flags it recovers: what Argv spells, ParseArgv reads back.
func TestParseArgv_RoundTripsArgv(t *testing.T) {
	a := NewArgs(Geth114())
	a.Set(KeyDataDir, "/data/n5", LayerRole)
	a.Set(KeyConfig, "/c/n5.toml", LayerRole)
	argv := append([]string{"gwbft"}, a.Argv()...)

	v := ParseArgv(argv)
	if v.DataDir != "/data/n5" || v.ConfigPath != "/c/n5.toml" {
		t.Fatalf("round trip lost paths: %+v (argv %v)", v, argv)
	}
}
