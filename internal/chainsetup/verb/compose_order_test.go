package verb

import (
	"context"
	_ "github.com/0xmhha/chainbench/internal/chains/all"
	"github.com/0xmhha/chainbench/internal/chainsetup"
	"strings"
	"testing"
	"time"
)

// TestGenesis_RefusesBeforePlace is the same rule reached through the use case,
// so the guard is proven to be wired in and not merely present.
func TestGenesis_RefusesBeforePlace(t *testing.T) {
	dir := t.TempDir()
	d := chainsetup.Deps{Clock: func() time.Time { return time.Unix(0, 0).UTC() }}
	if _, err := NetNew(context.Background(), d, NetNewIn{DataDir: dir, Chain: "wbft"}); err != nil {
		t.Fatalf("new: %v", err)
	}
	_, err := NetGenesis(context.Background(), d, chainsetup.NetGenesisIn{DataDir: dir})
	if err == nil {
		t.Fatal("genesis composed with no placement")
	}
	if !strings.Contains(err.Error(), "place") {
		t.Errorf("error %q should tell the operator to run place", err)
	}
}
