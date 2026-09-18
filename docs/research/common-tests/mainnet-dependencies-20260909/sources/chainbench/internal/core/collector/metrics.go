package collector

import (
	"bufio"
	"context"
	"fmt"
	"io"
	"net/http"
	"strconv"
	"strings"
	"time"
)

// Metric verification source (background requirement #3): the
// third of the three verification sources (log, rpc, metric). A node started
// with --metrics serves Prometheus text at /debug/metrics/prometheus; Scrape
// reads it into name -> value samples an assertion can compare.

// metricsPath is the geth-family Prometheus endpoint path.
const metricsPath = "/debug/metrics/prometheus"

// scrapeTimeout bounds one scrape; the endpoint is local or LAN.
const scrapeTimeout = 5 * time.Second

// maxMetricsBody caps a scrape response (a metrics page is ~100KB; a body this
// large means we are not talking to a metrics endpoint).
const maxMetricsBody = 8 << 20

// MetricsURL derives the scrape URL for a node's metrics endpoint.
func MetricsURL(host string, port int) string {
	return fmt.Sprintf("http://%s:%d%s", host, port, metricsPath)
}

// ScrapeMetrics fetches and parses one node's metrics endpoint. The result
// maps metric name (labels stripped) to its last sample value — chainbench's
// assertions compare single gauges/counters, so label sets collapse to the
// final sample, which for geth's unlabeled metrics is the only one.
func ScrapeMetrics(ctx context.Context, url string) (map[string]float64, error) {
	ctx, cancel := context.WithTimeout(ctx, scrapeTimeout)
	defer cancel()
	req, err := http.NewRequestWithContext(ctx, http.MethodGet, url, nil)
	if err != nil {
		return nil, fmt.Errorf("collector: metrics: %w", err)
	}
	resp, err := http.DefaultClient.Do(req)
	if err != nil {
		return nil, fmt.Errorf("collector: metrics: %w", err)
	}
	defer func() { _ = resp.Body.Close() }()
	if resp.StatusCode != http.StatusOK {
		return nil, fmt.Errorf("collector: metrics: %s returned %s", url, resp.Status)
	}
	return ParseMetrics(io.LimitReader(resp.Body, maxMetricsBody))
}

// ParseMetrics parses Prometheus text-format samples. Comment and empty lines
// are skipped; a malformed value line is an error (a partial scrape must not
// silently pass an assertion).
func ParseMetrics(r io.Reader) (map[string]float64, error) {
	out := map[string]float64{}
	sc := bufio.NewScanner(r)
	sc.Buffer(make([]byte, 0, 64*1024), 1<<20)
	for sc.Scan() {
		line := strings.TrimSpace(sc.Text())
		if line == "" || strings.HasPrefix(line, "#") {
			continue
		}
		// "name{labels} value [timestamp]" | "name value [timestamp]"
		rest := line
		var name string
		if i := strings.IndexByte(rest, '{'); i >= 0 {
			name = rest[:i]
			j := strings.IndexByte(rest, '}')
			if j < i {
				return nil, fmt.Errorf("collector: metrics: malformed labels: %q", line)
			}
			rest = strings.TrimSpace(rest[j+1:])
		} else {
			fields := strings.Fields(rest)
			if len(fields) < 2 {
				return nil, fmt.Errorf("collector: metrics: malformed sample: %q", line)
			}
			name = fields[0]
			rest = strings.Join(fields[1:], " ")
		}
		fields := strings.Fields(rest)
		if len(fields) < 1 {
			return nil, fmt.Errorf("collector: metrics: no value: %q", line)
		}
		v, err := strconv.ParseFloat(fields[0], 64)
		if err != nil {
			return nil, fmt.Errorf("collector: metrics: bad value in %q: %w", line, err)
		}
		out[name] = v
	}
	if err := sc.Err(); err != nil {
		return nil, fmt.Errorf("collector: metrics: %w", err)
	}
	return out, nil
}
