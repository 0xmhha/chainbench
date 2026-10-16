package testengine

import (
	"context"
	"encoding/json"
	"net"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
	"time"

	"github.com/0xmhha/chainbench/internal/core/node"
	"github.com/0xmhha/chainbench/internal/core/registry"
	"github.com/0xmhha/chainbench/internal/core/rpc"
)

func TestConsensusProbeRecordsFailureKindWithoutUpstreamText(t *testing.T) {
	for _, tc := range []struct {
		name, chain, reply, kind, reason, status string
		code                                     *int
	}{
		{name: "wrong result type", chain: "wemix", reply: `{"result":""}`, kind: "response_type", reason: "rpc_probe_unavailable", status: "unknown"},
		{name: "invalid JSON", chain: "wemix", reply: `{"result":`, kind: "response_syntax", reason: "rpc_probe_unavailable", status: "unknown"},
		{name: "RPC failure", chain: "stablenet", reply: `{"error":{"code":-32000,"message":"private-upstream-secret"}}`, kind: "rpc_error", reason: "rpc_probe_unavailable", status: "unknown", code: intPointer(-32000)},
		{name: "zero RPC code", chain: "stablenet", reply: `{"error":{"code":0,"message":"private-upstream-secret"}}`, kind: "rpc_error", reason: "rpc_probe_unavailable", status: "unknown", code: intPointer(0)},
		{name: "missing method", chain: "stablenet", reply: `{"error":{"code":-32601,"message":"private-upstream-secret"}}`, kind: "rpc_error", reason: "rpc_method_unavailable", status: "unsupported", code: intPointer(-32601)},
		{name: "HTTP failure", chain: "wemix", reply: "http-failure", kind: "probe_error", reason: "rpc_probe_unavailable", status: "unknown"},
		{name: "unready governance", chain: "wemix", reply: `{"result":{"governance":""}}`, kind: "probe_error", reason: "rpc_probe_unavailable", status: "unknown"},
	} {
		t.Run(tc.name, func(t *testing.T) {
			server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
				if tc.reply == "http-failure" {
					http.Error(w, "private-upstream-secret", http.StatusServiceUnavailable)
					return
				}
				_, _ = w.Write([]byte(tc.reply))
			}))
			defer server.Close()
			plugin, err := registry.Get(tc.chain)
			if err != nil {
				t.Fatal(err)
			}
			probe := probeConsensusNode(context.Background(), plugin, node.Node{Index: 5, Role: node.RoleEN, RPCURL: server.URL}, rpc.Dial)
			data, err := json.Marshal(probe)
			if err != nil {
				t.Fatal(err)
			}
			var fields map[string]any
			if err := json.Unmarshal(data, &fields); err != nil {
				t.Fatal(err)
			}
			if fields["failureKind"] != tc.kind || fields["status"] != tc.status || fields["reason"] != tc.reason {
				t.Errorf("failure lost its kind or changed its verdict: %s", data)
			}
			if tc.code != nil && fields["rpcErrorCode"] != float64(*tc.code) {
				t.Errorf("RPC error code missing: %s", data)
			}
			if strings.Contains(string(data), "private-upstream-secret") {
				t.Fatalf("observation copied private upstream text: %s", data)
			}
		})
	}
}

func TestConsensusProbeRecordsCancellationAndTransportKinds(t *testing.T) {
	plugin, err := registry.Get("wemix")
	if err != nil {
		t.Fatal(err)
	}
	for _, tc := range []struct {
		name, kind string
		ctx        func() (context.Context, context.CancelFunc)
	}{
		{"deadline", "deadline", func() (context.Context, context.CancelFunc) {
			return context.WithDeadline(context.Background(), time.Unix(1, 0))
		}},
		{"cancelled", "cancelled", func() (context.Context, context.CancelFunc) {
			ctx, cancel := context.WithCancel(context.Background())
			cancel()
			return ctx, cancel
		}},
		{"transport", "transport_error", func() (context.Context, context.CancelFunc) { return context.WithCancel(context.Background()) }},
	} {
		t.Run(tc.name, func(t *testing.T) {
			server := httptest.NewServer(http.NotFoundHandler())
			url := server.URL
			server.Close()
			ctx, cancel := tc.ctx()
			defer cancel()
			probe := probeConsensusNode(ctx, plugin, node.Node{Index: 5, Role: node.RoleEN, RPCURL: url}, rpc.Dial)
			data, _ := json.Marshal(probe)
			var fields map[string]any
			_ = json.Unmarshal(data, &fields)
			if fields["failureKind"] != tc.kind || fields["status"] != "unknown" || fields["reason"] != "rpc_probe_unavailable" {
				t.Fatalf("failure kind/verdict: %s", data)
			}
		})
	}
}

func intPointer(value int) *int { return &value }

func TestConsensusProbeSupportedObservationOmitsFailureFields(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		_, _ = w.Write([]byte(`{"result":["0xpublic-validator"]}`))
	}))
	defer server.Close()
	plugin, err := registry.Get("stablenet")
	if err != nil {
		t.Fatal(err)
	}
	probe := probeConsensusNode(context.Background(), plugin, node.Node{Index: 1, Role: node.RoleBP, RPCURL: server.URL}, rpc.Dial)
	data, _ := json.Marshal(probe)
	var fields map[string]any
	_ = json.Unmarshal(data, &fields)
	if fields["status"] != "supported" || fields["validatorCount"] != float64(1) || fields["failureKind"] != nil || fields["rpcErrorCode"] != nil {
		t.Fatalf("successful observation acquired failure metadata: %s", data)
	}
}

type probeTimeoutTransport struct{}

func (probeTimeoutTransport) RoundTrip(*http.Request) (*http.Response, error) {
	return nil, &net.DNSError{IsTimeout: true, Name: "private-upstream-secret"}
}

func TestConsensusProbeRecordsNetworkTimeout(t *testing.T) {
	plugin, err := registry.Get("wemix")
	if err != nil {
		t.Fatal(err)
	}
	client := &http.Client{Transport: probeTimeoutTransport{}}
	probe := probeConsensusNode(context.Background(), plugin, node.Node{Index: 5, Role: node.RoleEN, RPCURL: "http://unused.invalid"}, func(url string) *rpc.Client { return rpc.DialWithClient(url, client) })
	data, _ := json.Marshal(probe)
	var fields map[string]any
	_ = json.Unmarshal(data, &fields)
	if fields["failureKind"] != "transport_timeout" || fields["reason"] != "rpc_probe_unavailable" || fields["status"] != "unknown" {
		t.Fatalf("network timeout lost its category: %s", data)
	}
	if strings.Contains(string(data), "private-upstream-secret") || strings.Contains(string(data), "unused.invalid") {
		t.Fatalf("network details leaked into observation: %s", data)
	}
}
