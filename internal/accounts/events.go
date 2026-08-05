package accounts

import (
	"encoding/hex"
	"math/big"
	"strings"

	sdkcrypto "github.com/0xmhha/accounts/crypto"
)

// EventTopic returns the 0x-hex event signature hash (topic0) for a Solidity
// event signature such as "Transfer(address,address,uint256)". It is the full
// keccak256 of the signature (unlike a function Selector, which is 4 bytes).
func EventTopic(eventSig string) string {
	return "0x" + hex.EncodeToString(sdkcrypto.Keccak256([]byte(eventSig)))
}

// TopicToAddress decodes a 32-byte indexed-address log topic to its 0x-hex
// address (the low 20 bytes). Returns "" for a malformed topic.
func TopicToAddress(topic string) string {
	h := strings.TrimPrefix(topic, "0x")
	if len(h) != 64 {
		return ""
	}
	return "0x" + h[24:]
}

// WordToBig decodes a 32-byte 0x-hex word (event data or a numeric topic) to a
// big.Int, reporting whether it parsed.
func WordToBig(word string) (*big.Int, bool) {
	return new(big.Int).SetString(strings.TrimPrefix(word, "0x"), 16)
}

// Log is one decoded receipt log entry (as returned by eth_getTransactionReceipt).
type Log struct {
	Address string   `json:"address"`
	Topics  []string `json:"topics"`
	Data    string   `json:"data"`
}

// FindLog returns the first log whose topic0 equals topic0Hex (an event
// signature hash, e.g. from EventTopic) and whether one was found.
func FindLog(logs []Log, topic0Hex string) (Log, bool) {
	for _, l := range logs {
		if len(l.Topics) > 0 && strings.EqualFold(l.Topics[0], topic0Hex) {
			return l, true
		}
	}
	return Log{}, false
}
