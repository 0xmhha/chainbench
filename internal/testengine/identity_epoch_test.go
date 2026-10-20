package testengine_test

import (
	"encoding/hex"
	"fmt"
	"golang.org/x/crypto/sha3"
	"math/big"
	"strings"
	"testing"
)

type identityNode struct {
	Index                                                                       int
	Role, ExpectedAddress, Address, PublicKey, AdminPublicKey, AdminID, ChainID string
	Validator                                                                   *bool
}
type epochQuery struct {
	SignerNumber                     uint64
	SignerHash, Author, HeaderAuthor string
	At                               uint64
	Hash, ParentHash                 string
	Validators, Signers              []string
}
type identityEpochProof struct {
	GenesisPath, GenesisChainID string
	Fork, EpochLength           uint64
	Target                      int
	Nodes                       []identityNode
	Queries                     []epochQuery
}

func verifyIdentityEpoch(p identityEpochProof) error {
	if p.GenesisPath == "" || p.GenesisChainID == "" || p.Fork != 0 || p.EpochLength != 10 || p.Target != 4 || len(p.Nodes) != 5 || len(p.Queries) != 6 {
		return fmt.Errorf("wrong actual genesis/topology/epoch context")
	}
	chainID, ok := new(big.Int).SetString(p.GenesisChainID, 10)
	if !ok || chainID.Sign() <= 0 {
		return fmt.Errorf("invalid genesis chain ID")
	}
	indexes := map[int]bool{}
	identities := map[string]bool{}
	bp := map[string]bool{}
	for _, n := range p.Nodes {
		addr := strings.ToLower(n.Address)
		if n.Index < 1 || n.Index > 5 || indexes[n.Index] || identities[addr] || n.Validator == nil || addr != strings.ToLower(n.ExpectedAddress) || len(n.PublicKey) != 128 || n.AdminPublicKey != n.PublicKey {
			return fmt.Errorf("node identity differs")
		}
		raw, e := hex.DecodeString(n.PublicKey)
		if e != nil {
			return fmt.Errorf("invalid public key")
		}
		h := sha3.NewLegacyKeccak256()
		h.Write(raw)
		digest := h.Sum(nil)
		if addr != "0x"+hex.EncodeToString(digest[12:]) || n.AdminID != hex.EncodeToString(digest) {
			return fmt.Errorf("node RPC identity not derived from expected public key")
		}
		actual, ok := new(big.Int).SetString(n.ChainID, 0)
		if !ok || actual.Cmp(chainID) != 0 {
			return fmt.Errorf("RPC/genesis chain IDs differ")
		}
		producer := n.Index <= 4
		if producer && (n.Role != "bp" || !*n.Validator) || !producer && (n.Role != "en" || *n.Validator) {
			return fmt.Errorf("wrong BP/EN membership")
		}
		indexes[n.Index] = true
		identities[addr] = true
		if producer {
			bp[addr] = true
		}
	}
	for i, q := range p.Queries {
		expected := []uint64{9, 10, 11, 19, 20, 21}[i]
		if q.SignerNumber != q.At || q.SignerHash != q.Hash || q.Author != q.HeaderAuthor || !bp[strings.ToLower(q.Author)] {
			return fmt.Errorf("signer response belongs to different header/author")
		}
		if q.At != expected || q.Hash == "" || len(q.Validators) != p.Target || len(q.Signers) < 3 || i > 0 && q.At == p.Queries[i-1].At+1 && q.ParentHash != p.Queries[i-1].Hash {
			return fmt.Errorf("wrong tagged boundary/quorum")
		}
		seen := map[string]bool{}
		for _, v := range q.Validators {
			v = strings.ToLower(v)
			if !bp[v] || seen[v] {
				return fmt.Errorf("validator set differs from configured BP identities")
			}
			seen[v] = true
		}
		signers := map[string]bool{}
		for _, v := range q.Signers {
			v = strings.ToLower(v)
			if !seen[v] || signers[v] {
				return fmt.Errorf("unknown/duplicate committer")
			}
			signers[v] = true
		}
	}
	return nil
}

func identityEpochFixture() identityEpochProof {
	p := identityEpochProof{GenesisPath: "genesis.json", GenesisChainID: "8284", EpochLength: 10, Target: 4}
	vals := []string{}
	for i := 1; i <= 5; i++ {
		pub := fmt.Sprintf("%0128x", i)
		raw, _ := hex.DecodeString(pub)
		h := sha3.NewLegacyKeccak256()
		h.Write(raw)
		digest := h.Sum(nil)
		addr := "0x" + hex.EncodeToString(digest[12:])
		id := hex.EncodeToString(digest)
		role := "bp"
		flag := true
		if i == 5 {
			role = "en"
			flag = false
		} else {
			vals = append(vals, addr)
		}
		p.Nodes = append(p.Nodes, identityNode{Index: i, Role: role, ExpectedAddress: addr, Address: addr, PublicKey: pub, AdminPublicKey: pub, AdminID: id, ChainID: "0x205c", Validator: &flag})
	}
	for _, n := range []uint64{9, 10, 11, 19, 20, 21} {
		p.Queries = append(p.Queries, epochQuery{At: n, SignerNumber: n, SignerHash: fmt.Sprint(n), Author: vals[0], HeaderAuthor: vals[0], Hash: fmt.Sprint(n), ParentHash: fmt.Sprint(n - 1), Validators: append([]string{}, vals...), Signers: append([]string{}, vals[:3]...)})
	}
	return p
}

func TestIdentityEpochRejectsPresenceOnlyQueries(t *testing.T) {
	for _, name := range []string{"valid", "missing en", "duplicate node index", "duplicate address", "wrong node address", "wrong admin identity", "wrong genesis id", "wrong RPC id", "en validator flag", "missing validator flag", "wrong BP flag", "wrong epoch length", "wrong boundary tag", "short boundaries", "wrong validator count", "en in validators", "short quorum", "duplicate signer", "unknown signer", "wrong linked parent", "wrong signer number", "wrong signer hash", "unknown author", "wrong returned author"} {
		t.Run(name, func(t *testing.T) {
			p := identityEpochFixture()
			switch name {
			case "missing en":
				p.Nodes = p.Nodes[:4]
			case "duplicate node index":
				p.Nodes[4].Index = 1
			case "duplicate address":
				p.Nodes[4].Address = p.Nodes[0].Address
			case "wrong node address":
				p.Nodes[0].Address = p.Nodes[1].Address
			case "wrong admin identity":
				p.Nodes[4].AdminID = p.Nodes[0].AdminID
			case "wrong genesis id":
				p.GenesisChainID = "8285"
			case "wrong RPC id":
				p.Nodes[4].ChainID = "0x205d"
			case "en validator flag":
				b := true
				p.Nodes[4].Validator = &b
			case "missing validator flag":
				p.Nodes[4].Validator = nil
			case "wrong BP flag":
				b := false
				p.Nodes[0].Validator = &b
			case "wrong epoch length":
				p.EpochLength = 9
			case "wrong boundary tag":
				p.Queries[1].At = 12
			case "short boundaries":
				p.Queries = p.Queries[:3]
			case "wrong validator count":
				p.Queries[0].Validators = p.Queries[0].Validators[:3]
			case "en in validators":
				p.Queries[0].Validators[3] = p.Nodes[4].Address
			case "short quorum":
				p.Queries[0].Signers = p.Queries[0].Signers[:2]
			case "duplicate signer":
				p.Queries[0].Signers[1] = p.Queries[0].Signers[0]
			case "unknown signer":
				p.Queries[0].Signers[0] = p.Nodes[4].Address
			case "wrong returned author":
				p.Queries[0].Author = p.Queries[0].Validators[1]
			case "wrong signer number":
				p.Queries[0].SignerNumber++
			case "wrong signer hash":
				p.Queries[0].SignerHash = "other"
			case "unknown author":
				p.Queries[0].Author = p.Nodes[4].Address
			case "wrong linked parent":
				p.Queries[1].ParentHash = "other"
			}
			err := verifyIdentityEpoch(p)
			if name == "valid" && err != nil {
				t.Fatal(err)
			}
			if name != "valid" && err == nil {
				t.Fatal("presence-only query accepted invalid identity/epoch/genesis", name)
			}
		})
	}
}
