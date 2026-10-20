package testhelper

import (
	"bytes"
	"context"
	"encoding/hex"
	"encoding/json"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/0xmhha/chainbench/internal/core/keyring"
	"github.com/0xmhha/chainbench/internal/core/keyring/derive"
	"github.com/0xmhha/chainbench/internal/core/session"
	"github.com/0xmhha/chainbench/internal/dsl"
	"github.com/0xmhha/chainbench/internal/dsl/interp"
)

type captureBurnProposal struct{ data string }

func (c *captureBurnProposal) Do(_ context.Context, ac *interp.ActionCtx) error {
	c.data, _ = ac.Args["data"].(string)
	return nil
}

// The public input must work with private generated identities, while every
// other word of the original proof fixture stays byte-for-byte unchanged.
func TestPublicBurnProofUsesActualProposer(t *testing.T) {
	body, err := os.ReadFile("testdata/burn-proof-payloads.json")
	if err != nil {
		t.Fatal(err)
	}
	var originals map[string]string
	if err = json.Unmarshal(body, &originals); err != nil {
		t.Fatal(err)
	}
	for file, baseline := range originals {
		for i := 0; i < 2; i++ {
			t.Run(file+string(rune('A'+i)), func(t *testing.T) {
				d, ring := depsWithRing(t)
				for _, label := range []keyring.Label{"node3", "node4"} {
					if _, e := ring.Add(context.Background(), label, keyring.RandomSource{}, derive.AccountOnly); e != nil {
						t.Fatal(e)
					}
				}
				raw, e := dsl.ReadFiles([]string{filepath.Join("../../tests/tc/go-stablenet/post-v1.0.0-change/common-all", file)})
				if e != nil {
					t.Fatal(e)
				}
				spec, e := dsl.ParseV2(raw[0])
				if e != nil {
					t.Fatal(e)
				}
				sequence := []dsl.Statement{}
				proposer := ""
				for _, s := range spec.Sequence {
					if s.Do == "read" && s.Args["source"] == "derive" {
						sequence = append(sequence, s)
					}
					if s.Do == "sendTx" && s.Args["save"] == "burnHash" {
						sequence = append(sequence, s)
						proposer, _ = s.Args["from"].(string)
						break
					}
				}
				if proposer == "" {
					t.Fatal("public burn input missing")
				}
				spec.Sequence = sequence
				capture := &captureBurnProposal{}
				d.RPC = deps().RPC
				d.Actions = testhelperRegistry()
				d.Actions.RegisterAction("sendTx", capture)
				status, e := interp.NewInterpreter(*d).Run(context.Background(), spec, envOf(t, "http://unused1", "http://unused2", "http://unused3", "http://unused4"), &recordStub{})
				if e != nil || status != session.StatusPass {
					t.Fatalf("public input preparation failed: %s %v", status, e)
				}
				want, e := hex.DecodeString(strings.TrimPrefix(baseline, "0x"))
				if e != nil || len(want) != 452 {
					t.Fatal("original proof bytes invalid", e)
				}
				entry, ok := ring.Get(keyring.Label(proposer))
				if !ok {
					t.Fatal("proposer identity missing")
				}
				address, e := hex.DecodeString(strings.TrimPrefix(entry.Address, "0x"))
				if e != nil || len(address) != 20 {
					t.Fatal("bad generated address", e)
				}
				copy(want[4+2*32+12:4+3*32], address)
				got, e := hex.DecodeString(strings.TrimPrefix(capture.data, "0x"))
				if e != nil || !bytes.Equal(got, want) {
					t.Fatalf("public proof not bound to generated %s, or other proof bytes changed", proposer)
				}
			})
		}
	}
}
