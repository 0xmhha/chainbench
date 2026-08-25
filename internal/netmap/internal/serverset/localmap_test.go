package serverset_test

import (
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/0xmhha/chainbench/internal/netmap/internal/serverset"
)

func writeLocalMap(t *testing.T, dir string) string {
	t.Helper()
	path := filepath.Join(dir, serverset.LocalMapFile)
	body := `hosts:
  172.30.0.11:
    host: 127.0.0.1
    ports: { 22: 2201, 8600: 18601 }
  172.30.0.12:
    host: 127.0.0.1
    ports: { 22: 2202 }
`
	if err := os.WriteFile(path, []byte(body), 0o600); err != nil {
		t.Fatal(err)
	}
	return path
}

// TestLocalMap_TranslatesOnlyWhatItKnows pins the dial translation: a mapped
// host moves to its published port, an unmapped port keeps its number on the
// substitute host, and an unknown host passes through untouched.
func TestLocalMap_TranslatesOnlyWhatItKnows(t *testing.T) {
	m, err := serverset.LoadLocalMap(writeLocalMap(t, t.TempDir()))
	if err != nil {
		t.Fatal(err)
	}
	am := m.AddrMap(nil)

	cases := []struct {
		host     string
		port     int
		wantHost string
		wantPort int
	}{
		{"172.30.0.11", 22, "127.0.0.1", 2201},
		{"172.30.0.11", 8600, "127.0.0.1", 18601},
		{"172.30.0.12", 8600, "127.0.0.1", 8600}, // port not mapped: host swaps, port stays
		{"10.0.0.99", 22, "10.0.0.99", 22},       // unknown host: untouched
	}
	for _, c := range cases {
		h, p := am(c.host, c.port)
		if h != c.wantHost || p != c.wantPort {
			t.Errorf("map(%s:%d) = %s:%d, want %s:%d", c.host, c.port, h, p, c.wantHost, c.wantPort)
		}
	}
}

// TestLocalMap_MissingFileNamesTheFix pins that --docker without the map is a
// loud, actionable error — silently dialing unmapped addresses is the failure
// this design exists to prevent.
func TestLocalMap_MissingFileNamesTheFix(t *testing.T) {
	_, err := serverset.LoadLocalMap(filepath.Join(t.TempDir(), serverset.LocalMapFile))
	if err == nil {
		t.Fatal("missing localmap should be an error")
	}
	if !strings.Contains(err.Error(), "--docker") || !strings.Contains(err.Error(), "gen-env.sh") {
		t.Fatalf("error should say what is missing and how to make it: %v", err)
	}
}

// TestLocalMap_ReportsEachTranslationOnce pins the visibility contract: the
// harness never connects somewhere the operator cannot see, and a chatty
// repeat per dial would bury the report.
func TestLocalMap_ReportsEachTranslationOnce(t *testing.T) {
	m, err := serverset.LoadLocalMap(writeLocalMap(t, t.TempDir()))
	if err != nil {
		t.Fatal(err)
	}
	var got []string
	am := m.AddrMap(func(from, to string) { got = append(got, from+" -> "+to) })
	for i := 0; i < 3; i++ {
		am("172.30.0.11", 22)
	}
	am("10.0.0.99", 22) // unknown: no report
	if len(got) != 1 || got[0] != "172.30.0.11:22 -> 127.0.0.1:2201" {
		t.Fatalf("reports = %v, want exactly one translation", got)
	}
}

// TestLocalMapNear derives the map's path from the server set it translates.
func TestLocalMapNear(t *testing.T) {
	got := serverset.LocalMapNear("env/docker/build/server-set.yaml")
	if got != filepath.Join("env/docker/build", serverset.LocalMapFile) {
		t.Fatalf("LocalMapNear = %s", got)
	}
	if serverset.LocalMapNear("") != serverset.LocalMapFile {
		t.Fatalf("default location should be the working directory")
	}
}
