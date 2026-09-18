package accounts_test

import (
	"strings"
	"testing"

	"github.com/0xmhha/chainbench/internal/accounts"
)

func TestEventTopic(t *testing.T) {
	// Canonical ERC-20 Transfer event signature hash.
	const transferSig = "0xddf252ad1be2c89b69c2b068fc378daa952ba7f163c4a11628f55a4df523b3ef"
	if got := accounts.EventTopic("Transfer(address,address,uint256)"); !strings.EqualFold(got, transferSig) {
		t.Errorf("EventTopic(Transfer) = %s, want %s", got, transferSig)
	}
}

func TestTopicToAddressAndWord(t *testing.T) {
	// an indexed address topic: 12 zero bytes + the 20-byte address.
	topic := "0x" + strings.Repeat("0", 24) + "00000000000000000000000000000000000000ab"
	if got := accounts.TopicToAddress(topic); !strings.EqualFold(got, "0x00000000000000000000000000000000000000ab") {
		t.Errorf("TopicToAddress = %s", got)
	}
	if accounts.TopicToAddress("0xshort") != "" {
		t.Error("malformed topic should yield empty")
	}
	// a data word: uint256 = 255.
	v, ok := accounts.WordToBig("0x" + strings.Repeat("0", 62) + "ff")
	if !ok || v.Int64() != 255 {
		t.Errorf("WordToBig = %v (%v)", v, ok)
	}
}

func TestFindLog(t *testing.T) {
	topic0 := accounts.EventTopic("Transfer(address,address,uint256)")
	logs := []accounts.Log{
		{Address: "0xa", Topics: []string{"0xdeadbeef"}},
		{Address: "0xb", Topics: []string{topic0, "0xfrom", "0xto"}, Data: "0x01"},
	}
	got, ok := accounts.FindLog(logs, topic0)
	if !ok || got.Address != "0xb" {
		t.Errorf("FindLog = %+v (%v)", got, ok)
	}
	if _, ok := accounts.FindLog(logs, "0xnotpresent"); ok {
		t.Error("absent topic should not be found")
	}
}
