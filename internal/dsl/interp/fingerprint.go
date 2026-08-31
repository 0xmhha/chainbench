package interp

import (
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"fmt"

	"github.com/0xmhha/chainbench/internal/core/nodeconfig"
	"github.com/0xmhha/chainbench/internal/core/session"
	"github.com/0xmhha/chainbench/internal/dsl"
)

// fpInput is the canonical, order-independent hash input for a fingerprint.
// json.Marshal sorts map keys, so equal declared values hash equally regardless
// of insertion order.
type fpInput struct {
	Binary         string            `json:"binary"`
	Binaries       map[string]string `json:"binaries"`
	Config         string            `json:"config"`
	GenesisOverlay map[string]any    `json:"genesisOverlay"`
	Topology       map[string]any    `json:"topology"`
	Hardforks      map[string]int    `json:"hardforks"`
	Placement      string            `json:"placement"`
	Resolved       map[string]string `json:"resolved"`
}

// Fingerprint hashes the resolved declared values
// (binaries+genesis+config+topology+hardforks+placement) to a reuse key. config
// comes from resolved; the rest come from the spec. It never touches a chain.
//
// It lives with the interpreter rather than the grammar because a fingerprint
// keys an environment for reuse — a runtime concern — and reaching the session
// type keeps it out of the pure grammar package.
func Fingerprint(s dsl.Spec, resolved nodeconfig.Values) session.Fingerprint {
	in := fpInput{
		Binary:         s.Chain.Binary,
		Binaries:       s.Chain.Binaries,
		Config:         s.Chain.Config,
		GenesisOverlay: s.Chain.GenesisOverlay,
		Topology:       s.Topology,
		Hardforks:      s.Hardforks,
		Placement:      s.Placement,
		Resolved:       resolved,
	}
	b, err := json.Marshal(in)
	if err != nil {
		// Inputs originate from JSON, so this should not happen; fall back to a
		// stable representation rather than silently ignoring the error.
		b = []byte(fmt.Sprintf("fingerprint-error:%v", err))
	}
	sum := sha256.Sum256(b)
	return session.Fingerprint(hex.EncodeToString(sum[:]))
}
