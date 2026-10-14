package collector_test

import (
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"github.com/0xmhha/chainbench/internal/core/collector"
)

func TestOversizedMetricsCannotPassUsingTheValidPrefix(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		_, _ = w.Write([]byte("needed_counter 1\n"))
		for i := 0; i < 9000; i++ {
			_, _ = w.Write([]byte("# " + strings.Repeat("x", 1024) + "\n"))
		}
	}))
	defer server.Close()
	if samples, err := collector.ScrapeMetrics(t.Context(), server.URL); err == nil {
		t.Fatal("truncated telemetry was accepted", samples)
	}
}
