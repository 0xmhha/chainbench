package collector_test

import (
	"context"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"github.com/0xmhha/chainbench/internal/core/collector"
)

const sampleMetrics = `# HELP chain_head_block head block number
# TYPE chain_head_block gauge
chain_head_block 42
txpool_pending 7
p2p_peers 3
system_memory_used{quantile="0.5"} 1.048576e+06
rpc_duration_all{quantile="0.99"} 0.001
`

func TestParseMetrics(t *testing.T) {
	got, err := collector.ParseMetrics(strings.NewReader(sampleMetrics))
	if err != nil {
		t.Fatal(err)
	}
	for name, want := range map[string]float64{
		"chain_head_block":   42,
		"txpool_pending":     7,
		"p2p_peers":          3,
		"system_memory_used": 1048576,
	} {
		if got[name] != want {
			t.Errorf("%s = %v, want %v", name, got[name], want)
		}
	}
}

func TestParseMetricsRejectsMalformed(t *testing.T) {
	for _, bad := range []string{
		"chain_head_block", // no value
		"chain_head_block not-a-number",
	} {
		if _, err := collector.ParseMetrics(strings.NewReader(bad)); err == nil {
			t.Errorf("%q must fail", bad)
		}
	}
}

func TestScrapeMetrics(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path != "/debug/metrics/prometheus" {
			http.NotFound(w, r)
			return
		}
		_, _ = w.Write([]byte(sampleMetrics))
	}))
	defer srv.Close()

	got, err := collector.ScrapeMetrics(context.Background(), srv.URL+"/debug/metrics/prometheus")
	if err != nil {
		t.Fatal(err)
	}
	if got["chain_head_block"] != 42 {
		t.Fatalf("chain_head_block = %v", got["chain_head_block"])
	}

	if _, err := collector.ScrapeMetrics(context.Background(), srv.URL+"/nope"); err == nil {
		t.Fatal("non-200 must fail")
	}
}
