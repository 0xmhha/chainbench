package suitecmd_test

import (
	"sync/atomic"
	"testing"
)

func nativeDurationControlCases() []nativeControlCase {
	return []nativeControlCase{
		{"action", "sendTx", "timeout", "0s"}, {"action", "sendTx", "pollInterval", "-1ms"},
		{"action", "waitBlock", "timeout", "invalid"}, {"action", "waitBlock", "pollInterval", "0s"},
		{"action", "waitFor", "timeout", true}, {"action", "waitFor", "pollInterval", "-1ms"},
		{"action", "load", "timeout", "0s"}, {"action", "load", "pollInterval", "invalid"},
		{"action", "faucet", "timeout", "0s"}, {"action", "faucet", "pollInterval", "-1ms"},
		{"action", "deployContract", "timeout", "0s"}, {"action", "deployContract", "pollInterval", "invalid"},
		{"action", "registerContract", "timeout", nil}, {"action", "registerContract", "pollInterval", "0s"},
		{"assertion", "blockHalt", "within", "0s"},
		{"assertion", "blockStalled", "timeout", "0s"}, {"assertion", "blockStalled", "pollInterval", "-1ms"},
		{"assertion", "blockAdvance", "timeout", "invalid"}, {"assertion", "blockAdvance", "pollInterval", "0s"},
		{"assertion", "wsSubscribe", "timeout", "9223372036854775808ns"}, {"assertion", "wsCollected", "timeout", "0s"},
	}
}

func TestNativeDurationControlFixturesUseOnlyConsumedArguments(t *testing.T) {
	preset := map[string]any{"schemaVersion": "2", "kind": "chain-preset", "id": "fixture", "chain": "stablenet", "attach": map[string]any{"rpc": []string{"http://127.0.0.1:1"}}}
	var count atomic.Int32
	requireNativeControlCases(t, t.TempDir(), preset, &count, "duration", nativeDurationControlCases(), true)
}
