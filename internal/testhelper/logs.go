package testhelper

import (
	"context"
	"fmt"
	"strconv"
	"strings"

	"github.com/0xmhha/chainbench/internal/core/rpc"
)

// assertLogs is the event-log assertion name.
const assertLogs = "logs"

// Selectors for what a logs assertion compares. "count" is the default because
// most event checks are "did this fire, and how often".
const (
	logSelectCount = "count"
	logSelectData  = "data"
	logSelectAddr  = "address"
	logSelectBlock = "blockNumber"
	logSelectTx    = "txHash"
)

// readLogs queries eth_getLogs and returns the value the spec selected: the
// number of matching logs, or a field of one of them. It is registered as a
// normal RPC-reading assertion, so it also works as a "read" source and can be
// saved and compared like any other value.
//
// Spec: address, topics ([]string; "" is a wildcard position), fromBlock,
// toBlock, select (count | data | address | blockNumber | txHash | topic0..3),
// index (which matching log, default 0).
func readLogs(ctx context.Context, c *rpc.Client, spec map[string]any) (any, error) {
	filter := rpc.LogFilter{}
	filter.Address, _ = spec["address"].(string)
	filter.FromBlock, _ = spec["fromBlock"].(string)
	filter.ToBlock, _ = spec["toBlock"].(string)
	if raw, ok := spec["topics"].([]any); ok {
		for _, t := range raw {
			s, _ := t.(string) // a non-string (or null) is a wildcard position
			filter.Topics = append(filter.Topics, s)
		}
	}

	logs, err := c.Logs(ctx, filter)
	if err != nil {
		return nil, err
	}

	sel, _ := spec["select"].(string)
	if sel == "" {
		sel = logSelectCount
	}
	if sel == logSelectCount {
		return strconv.Itoa(len(logs)), nil
	}

	idx := 0
	if n, ok := uintArg(spec["index"]); ok {
		idx = int(n)
	}
	if idx >= len(logs) {
		return nil, fmt.Errorf("testspec: logs: selected %s of log %d but only %d log(s) matched", sel, idx, len(logs))
	}
	return logField(logs[idx], sel)
}

// logField extracts one field of a log. Block numbers come back as decimal so
// they compare against a plain number in the spec; hashes and data stay 0x-hex.
func logField(log map[string]any, sel string) (any, error) {
	if strings.HasPrefix(sel, "topic") {
		n, err := strconv.Atoi(strings.TrimPrefix(sel, "topic"))
		if err != nil {
			return nil, fmt.Errorf("testspec: logs: bad topic selector %q", sel)
		}
		topics, _ := log["topics"].([]any)
		if n >= len(topics) {
			return nil, fmt.Errorf("testspec: logs: topic%d requested but the log has %d topic(s)", n, len(topics))
		}
		return topics[n], nil
	}

	switch sel {
	case logSelectData, logSelectAddr, logSelectTx:
		v, ok := log[sel].(string)
		if !ok {
			return nil, fmt.Errorf("testspec: logs: log has no %q", sel)
		}
		return v, nil
	case logSelectBlock:
		v, ok := log[sel].(string)
		if !ok {
			return nil, fmt.Errorf("testspec: logs: log has no blockNumber")
		}
		n, err := strconv.ParseUint(strings.TrimPrefix(v, "0x"), 16, 64)
		if err != nil {
			return nil, fmt.Errorf("testspec: logs: blockNumber %q: %w", v, err)
		}
		return strconv.FormatUint(n, 10), nil
	default:
		return nil, fmt.Errorf("testspec: logs: unknown select %q (count|data|address|blockNumber|txHash|topicN)", sel)
	}
}
