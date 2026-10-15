package testengine_test

import (
	"encoding/hex"
	"encoding/json"
	"fmt"
	"math"
	"strconv"
	"strings"
	"testing"
)

type w4Sample struct {
	Height             uint64
	PeerCount          string
	Source, Reader     map[string]any
	Prometheus, Expvar string
}
type w4Witness struct {
	Revision, BinarySHA, InitTrace string
	Index, PID                     int
	Role                           string
	IssueResolved                  bool
	Samples                        []w4Sample
}

func verifyW4Collision(p w4Witness) error {
	sha, err := hex.DecodeString(p.BinarySHA)
	les := strings.Index(p.InitTrace, "init github.com/ethereum/go-ethereum/les/fetcher @")
	eth := strings.Index(p.InitTrace, "init github.com/ethereum/go-ethereum/eth/fetcher @")
	if p.Revision != "4538a9f5f2e4bcd2cb9515f8c466fabc24f73987" || err != nil || len(sha) != 32 || p.Index != 5 || p.Role != "en" || p.PID <= 0 || p.IssueResolved || les < 0 || eth <= les || len(p.Samples) != 2 {
		return fmt.Errorf("not a provenance-bound unresolved live EN collision")
	}
	if p.Samples[0].Height == 0 || p.Samples[1].Height < p.Samples[0].Height || p.Samples[1].Height-p.Samples[0].Height < 20 {
		return fmt.Errorf("no twenty newly received blocks")
	}
	for _, s := range p.Samples {
		peers, e := strconv.ParseUint(strings.TrimPrefix(s.PeerCount, "0x"), 16, 64)
		if e != nil || !strings.HasPrefix(s.PeerCount, "0x") || peers == 0 {
			return fmt.Errorf("no live peer")
		}
		if s.Source["number"] != fmt.Sprintf("0x%x", s.Height) || s.Reader["number"] != s.Source["number"] {
			return fmt.Errorf("unpinned block")
		}
		for _, key := range []string{"hash", "stateRoot"} {
			value, ok := s.Source[key].(string)
			b, e := hex.DecodeString(strings.TrimPrefix(value, "0x"))
			if !ok || !strings.HasPrefix(value, "0x") || e != nil || len(b) != 32 || s.Reader[key] != value {
				return fmt.Errorf("wrong received header/root")
			}
		}
		var expvar map[string]json.RawMessage
		if e := json.Unmarshal([]byte(s.Expvar), &expvar); e != nil {
			return fmt.Errorf("missing raw expvar: %w", e)
		}
		for _, kind := range []string{"broadcasts", "announces"} {
			name := "eth_fetcher_block_" + kind + "_in"
			count := 0
			for _, line := range strings.Split(s.Prometheus, "\n") {
				fields := strings.Fields(line)
				if len(fields) == 0 || fields[0] != name {
					continue
				}
				if len(fields) != 2 {
					return fmt.Errorf("malformed selected exporter sample")
				}
				v, e := strconv.ParseFloat(fields[1], 64)
				if e != nil || math.IsNaN(v) || math.IsInf(v, 0) || v != 0 {
					return fmt.Errorf("not the exact zero meter collision")
				}
				count++
			}
			var value *float64
			if count != 1 || json.Unmarshal(expvar["eth/fetcher/block/"+kind+"/in.count"], &value) != nil || value == nil || math.IsNaN(*value) || math.IsInf(*value, 0) || *value != 0 {
				return fmt.Errorf("exporter absent, duplicate or differs")
			}
		}
	}
	return nil
}

func w4Fixture() w4Witness {
	p := w4Witness{Revision: "4538a9f5f2e4bcd2cb9515f8c466fabc24f73987", BinarySHA: strings.Repeat("a", 64), Index: 5, PID: 123, Role: "en", InitTrace: "init github.com/ethereum/go-ethereum/les/fetcher @1 ms\ninit github.com/ethereum/go-ethereum/eth/fetcher @2 ms\n"}
	for _, h := range []uint64{5, 25} {
		b := map[string]any{"number": fmt.Sprintf("0x%x", h), "hash": "0x" + strings.Repeat("11", 32), "stateRoot": "0x" + strings.Repeat("22", 32)}
		r := map[string]any{}
		for k, v := range b {
			r[k] = v
		}
		p.Samples = append(p.Samples, w4Sample{Height: h, PeerCount: "0x1", Source: b, Reader: r, Prometheus: "eth_fetcher_block_broadcasts_in 0\neth_fetcher_block_announces_in 0\n", Expvar: `{"eth/fetcher/block/broadcasts/in.count":0,"eth/fetcher/block/announces/in.count":0}`})
	}
	return p
}

func TestW4CollisionRejectsZeroOnlyDiagnosis(t *testing.T) {
	if err := verifyW4Collision(w4Fixture()); err != nil {
		t.Fatal(err)
	}
	for _, tc := range []struct {
		name   string
		change func(*w4Witness)
	}{
		{"missing expvar", func(p *w4Witness) { p.Samples[0].Expvar = "" }},
		{"missing expvar counter", func(p *w4Witness) { p.Samples[1].Expvar = `{}` }},
		{"expvar positive", func(p *w4Witness) {
			p.Samples[1].Expvar = `{"eth/fetcher/block/broadcasts/in.count":1,"eth/fetcher/block/announces/in.count":0}`
		}},
		{"prometheus positive announces", func(p *w4Witness) {
			p.Samples[1].Prometheus = "eth_fetcher_block_broadcasts_in 0\neth_fetcher_block_announces_in 1\n"
		}},
		{"prometheus NaN", func(p *w4Witness) {
			p.Samples[0].Prometheus = "eth_fetcher_block_broadcasts_in NaN\neth_fetcher_block_announces_in 0\n"
		}},
		{"prometheus absent", func(p *w4Witness) { p.Samples[0].Prometheus = "" }},
		{"negative expvar", func(p *w4Witness) {
			p.Samples[0].Expvar = `{"eth/fetcher/block/broadcasts/in.count":-1,"eth/fetcher/block/announces/in.count":0}`
		}},
		{"no twenty block progress", func(p *w4Witness) { p.Samples[1].Height = 24 }},
		{"wrong block root", func(p *w4Witness) { p.Samples[1].Reader["stateRoot"] = "different" }},
		{"wrong block hash", func(p *w4Witness) { p.Samples[1].Reader["hash"] = "different" }},
		{"unpinned tag", func(p *w4Witness) { p.Samples[0].Source["number"] = "latest" }},
		{"no peers", func(p *w4Witness) { p.Samples[1].PeerCount = "0x0" }},
		{"reverse initialization", func(p *w4Witness) {
			p.InitTrace = "init github.com/ethereum/go-ethereum/eth/fetcher @1 ms\ninit github.com/ethereum/go-ethereum/les/fetcher @2 ms\n"
		}},
		{"no source revision", func(p *w4Witness) { p.Revision = "" }},
		{"no process identity", func(p *w4Witness) { p.PID = 0 }},
		{"producer instead of EN", func(p *w4Witness) { p.Role = "bp" }},
		{"diagnosis claimed resolution", func(p *w4Witness) { p.IssueResolved = true }},
	} {
		t.Run(tc.name, func(t *testing.T) {
			data, err := json.Marshal(w4Fixture())
			if err != nil {
				t.Fatal(err)
			}
			var p w4Witness
			if err = json.Unmarshal(data, &p); err != nil {
				t.Fatal(err)
			}
			tc.change(&p)
			if verifyW4Collision(p) == nil {
				t.Fatal("zero-only observation hid missing collision evidence")
			}
		})
	}
}
