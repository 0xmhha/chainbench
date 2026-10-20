package testengine_test

import (
	"fmt"
	"strings"
	"testing"
)

type statisticsBlock struct {
	Number                   uint64
	Hash, ParentHash, Author string
	Round                    uint64
	Seals                    map[string][]string
}
type statisticsStatus struct {
	Activity map[string]map[string]int                          `json:"sealerActivity"`
	Author   map[string]int                                     `json:"author"`
	Range    struct{ StartBlock, EndBlock, TotalBlocks uint64 } `json:"blockRange"`
	Rounds   struct {
		Distribution map[string]uint64 `json:"roundDistribution"`
	} `json:"roundStats"`
}

func verifyStatistics50(vals []string, blocks []statisticsBlock, status statisticsStatus) error {
	if len(vals) < 3 || len(blocks) != 50 || status.Range.StartBlock != 1 || status.Range.EndBlock != 50 || status.Range.TotalBlocks != 50 {
		return fmt.Errorf("not exact fifty-block window")
	}
	allowed := map[string]bool{}
	for _, v := range vals {
		v = strings.ToLower(v)
		if v == "" || allowed[v] {
			return fmt.Errorf("invalid validators")
		}
		allowed[v] = true
	}
	categories := []string{"prepared", "committed", "prevPrepared", "prevCommitted"}
	expected := map[string]map[string]int{"total": {}}
	authors := map[string]int{}
	rounds := map[string]uint64{}
	for _, v := range vals {
		authors[strings.ToLower(v)] = 0
		expected["total"][strings.ToLower(v)] = 0
	}
	for _, k := range categories {
		expected[k] = map[string]int{}
		for v := range allowed {
			expected[k][v] = 0
		}
	}
	quorum := len(vals) - (len(vals)-1)/3
	for i, b := range blocks {
		if b.Number != uint64(i+1) || b.Hash == "" || !allowed[strings.ToLower(b.Author)] || i > 0 && b.ParentHash != blocks[i-1].Hash {
			return fmt.Errorf("wrong linked header or author")
		}
		authors[strings.ToLower(b.Author)]++
		rounds[fmt.Sprint(b.Round)]++
		if len(b.Seals) != 4 || len(b.Seals["committed"]) < quorum {
			return fmt.Errorf("missing seals or quorum")
		}
		for _, k := range categories {
			signers, ok := b.Seals[k]
			if !ok {
				return fmt.Errorf("missing seal field")
			}
			seen := map[string]bool{}
			for _, v := range signers {
				v = strings.ToLower(v)
				if !allowed[v] || seen[v] {
					return fmt.Errorf("unknown/duplicate signer")
				}
				seen[v] = true
				expected[k][v]++
				expected["total"][v]++
			}
		}
	}
	equalCounts := func(actual, want map[string]int) bool {
		if len(actual) != len(want) {
			return false
		}
		seen := map[string]bool{}
		for v, n := range actual {
			v = strings.ToLower(v)
			expected, ok := want[v]
			if !ok || seen[v] || n < 0 || n != expected {
				return false
			}
			seen[v] = true
		}
		return true
	}
	if !equalCounts(status.Author, authors) || len(status.Activity) != 5 || len(status.Rounds.Distribution) != len(rounds) {
		return fmt.Errorf("wrong author/activity/round counts")
	}
	total, committed := 0, 0
	for k, want := range expected {
		if !equalCounts(status.Activity[k], want) {
			return fmt.Errorf("wrong %s seal counts", k)
		}
		for _, n := range status.Activity[k] {
			if k == "total" {
				total += n
			}
			if k == "committed" {
				committed += n
			}
		}
	}
	for k, n := range status.Rounds.Distribution {
		if rounds[k] != n || n == 0 {
			return fmt.Errorf("wrong round distribution")
		}
	}
	if total < 50*quorum || committed < 50*quorum {
		return fmt.Errorf("below signature quorum total")
	}
	return nil
}

func TestStatistics50RejectsPresenceOnlyCounts(t *testing.T) {
	for _, name := range []string{"valid", "short", "wrong start", "wrong total", "wrong author count", "unknown author", "missing author", "wrong round count", "wrong committed count", "wrong total seals", "missing seal category", "negative count", "no quorum", "duplicate signer", "unknown signer", "wrong parent", "wrong height"} {
		t.Run(name, func(t *testing.T) {
			vals := []string{"v0", "v1", "v2", "v3"}
			blocks := []statisticsBlock{}
			s := statisticsStatus{Activity: map[string]map[string]int{}, Author: map[string]int{"v0": 13, "v1": 13, "v2": 12, "v3": 12}}
			s.Range.StartBlock = 1
			s.Range.EndBlock = 50
			s.Range.TotalBlocks = 50
			s.Rounds.Distribution = map[string]uint64{"0": 50}
			for _, k := range []string{"total", "prepared", "committed", "prevPrepared", "prevCommitted"} {
				s.Activity[k] = map[string]int{}
				for _, v := range vals {
					s.Activity[k][v] = 0
				}
			}
			for n := uint64(1); n <= 50; n++ {
				blocks = append(blocks, statisticsBlock{Number: n, Hash: fmt.Sprint(n), ParentHash: fmt.Sprint(n - 1), Author: vals[(n-1)%4], Seals: map[string][]string{"prepared": nil, "committed": vals[:3], "prevPrepared": nil, "prevCommitted": nil}})
			}
			for _, v := range vals[:3] {
				s.Activity["total"][v] = 50
				s.Activity["committed"][v] = 50
			}
			switch name {
			case "short":
				blocks = blocks[:49]
			case "wrong start":
				s.Range.StartBlock = 2
			case "wrong total":
				s.Range.TotalBlocks = 49
			case "wrong author count":
				s.Author["v0"]--
			case "unknown author":
				s.Author["other"] = 1
			case "missing author":
				delete(s.Author, "v3")
			case "wrong round count":
				s.Rounds.Distribution["0"]--
			case "wrong committed count":
				s.Activity["committed"]["v0"]--
			case "wrong total seals":
				s.Activity["total"]["v0"]--
			case "missing seal category":
				delete(s.Activity, "prevPrepared")
			case "negative count":
				s.Activity["prepared"]["v3"] = -1
			case "no quorum":
				blocks[49].Seals["committed"] = vals[:2]
			case "duplicate signer":
				blocks[49].Seals["committed"] = []string{"v0", "v0", "v1"}
			case "unknown signer":
				blocks[49].Seals["committed"] = []string{"v0", "v1", "other"}
			case "wrong parent":
				blocks[49].ParentHash = "other"
			case "wrong height":
				blocks[49].Number = 51
			}
			err := verifyStatistics50(vals, blocks, s)
			if name == "valid" && err != nil {
				t.Fatal(err)
			}
			if name != "valid" && err == nil {
				t.Fatal("presence-only status accepted malformed evidence", strings.ReplaceAll(name, " ", "_"))
			}
		})
	}
}
