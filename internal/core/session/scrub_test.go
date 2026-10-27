package session_test

import (
	"encoding/json"
	"strings"
	"testing"

	"github.com/0xmhha/chainbench/internal/core/session"
)

func TestScrub(t *testing.T) {
	// A 64-hex value with the shape of a private key, built at runtime so no real
	// key is committed. Its point is the shape, not the value.
	key := "0x" + strings.Repeat("ab", 32)
	cases := []struct {
		name   string
		in     string
		redact bool // whether the secret substring must be gone
		secret string
		keep   string // a substring that must survive
	}{
		{
			name: "signing key field", redact: true, secret: key,
			in:   `{"do":"sendTx","from":"0xabc","key":"` + key + `","to":"0xdef"}`,
			keep: `"from":"0xabc"`,
		},
		{
			name: "saveKey field", redact: true, secret: key,
			in:   `{"do":"newAccount","saveKey":"` + key + `"}`,
			keep: `"do":"newAccount"`,
		},
		{
			name: "password field", redact: true, secret: "hunter2",
			in:   `{"unlock":"0xabc","password":"hunter2"}`,
			keep: `"unlock":"0xabc"`,
		},
		{
			name: "password flag in command", redact: true, secret: "s3cr3t",
			in:   `gstable --datadir /d --unlock 0xabc --password s3cr3t --http`,
			keep: `--datadir /d`,
		},
		{
			name: "password flag equals form", redact: true, secret: "s3cr3t",
			in:   `gstable --password=s3cr3t --http`,
			keep: `--http`,
		},
		{
			// A tx hash is 64 hex like a key, but under "hash" it is evidence, not
			// a secret, and must survive.
			name: "tx hash survives", redact: false,
			in:   `{"hash":"` + key + `","from":"0xabc"}`,
			keep: key,
		},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			got := string(session.Scrub([]byte(tc.in)))
			if tc.redact && strings.Contains(got, tc.secret) {
				t.Errorf("secret %q not redacted: %s", tc.secret, got)
			}
			if tc.redact && !strings.Contains(got, "[redacted]") {
				t.Errorf("no [redacted] marker: %s", got)
			}
			if !strings.Contains(got, tc.keep) {
				t.Errorf("non-secret %q was lost: %s", tc.keep, got)
			}
		})
	}
}

// TestScrubKeepsARecordedCommandLineInsideItsJSONString: a --password value
// inside a JSON string is replaced up to the string's end, not through its
// closing quote, so a scrubbed process ledger still parses.
func TestScrubKeepsARecordedCommandLineInsideItsJSONString(t *testing.T) {
	in := []byte(`[{"PID":4242,"Command":"/opt/gstable --password launch-secret-word","Host":"10.0.0.5"},{"PID":1,"Command":"gstable --unlock=0xabc --http"}]`)
	out := session.Scrub(in)
	var procs []map[string]any
	if err := json.Unmarshal(out, &procs); err != nil {
		t.Fatalf("scrubbed ledger no longer parses: %v\n%s", err, out)
	}
	if procs[0]["Command"] != "/opt/gstable --password [redacted]" || procs[0]["Host"] != "10.0.0.5" || procs[1]["Command"] != "gstable --unlock=[redacted] --http" {
		t.Fatalf("scrubbed ledger = %s", out)
	}
	if got := string(session.Scrub([]byte("gstable --password secret --http"))); got != "gstable --password [redacted] --http" {
		t.Fatalf("plain command = %q", got)
	}
}

// TestScrubHidesQuotedAndEscapedFlagValues: a quoted value, a value quoted
// inside a JSON string, and a value with escaped characters are hidden whole,
// as the unquoted forms are.
func TestScrubHidesQuotedAndEscapedFlagValues(t *testing.T) {
	for in, want := range map[string]string{
		`gstable --password "hunter2"`:                      `gstable --password [redacted]`,
		`gstable --password "hunter2 x" --http`:             `gstable --password [redacted] --http`,
		`gstable --unlock '0xabc def'`:                      `gstable --unlock [redacted]`,
		`{"cmd":"gstable --password=\"hunter2\" --http"}`:   `{"cmd":"gstable --password=[redacted] --http"}`,
		`{"cmd":"g --password=pa\\ss"}`:                     `{"cmd":"g --password=[redacted]"}`,
		`{"cmd":"g --password /keys/pw","host":"10.0.0.5"}`: `{"cmd":"g --password [redacted]","host":"10.0.0.5"}`,
	} {
		if got := string(session.Scrub([]byte(in))); got != want {
			t.Errorf("Scrub(%s) = %s, want %s", in, got, want)
		}
	}
}

// TestScrubHidesASecretFieldWithAnEscapedQuote: a secret value holding an
// escaped quote is replaced whole, not up to the escape.
func TestScrubHidesASecretFieldWithAnEscapedQuote(t *testing.T) {
	in := `{"key":"abc\"defsecret","next":"kept"}`
	if got := string(session.Scrub([]byte(in))); got != `{"key":"[redacted]","next":"kept"}` {
		t.Fatalf("Scrub = %s", got)
	}
}

// TestScrubNeverHidesLessThanTheGreedyFlagRule: every input the original
// greedy rule (--password \S+) hid stays hidden, and JSON stays valid JSON.
func TestScrubNeverHidesLessThanTheGreedyFlagRule(t *testing.T) {
	for _, in := range []string{
		`g --password "hunter2`,
		`g --password \"hunter2`,
		`g --password=ab"cd`,
		`g --password=''x`,
		`g --password="a"b`,
		`{"c":"g --password=ab\"cd x"}`,
		`{"c":"g --password=ab\\\"cd x"}`,
		`{"c":"g --password \"ab\\\"cd\" --http"}`,
	} {
		out := string(session.Scrub([]byte(in)))
		for _, leaked := range []string{"hunter2", "cd", "x\"", "b"} {
			if leaked == "b" && !strings.HasSuffix(in, `"a"b`) || leaked == "x\"" && !strings.HasSuffix(in, `''x`) {
				continue
			}
			if strings.Contains(out, leaked) && !strings.Contains(in, "--http") || leaked == "cd" && strings.Contains(out, "cd") {
				t.Errorf("Scrub(%s) = %s shows %q", in, out, leaked)
			}
		}
		if strings.HasPrefix(in, "{") && !json.Valid([]byte(out)) {
			t.Errorf("Scrub(%s) = %s is no longer JSON", in, out)
		}
	}
	// A flag with nothing after it inside a JSON string leaves the JSON whole.
	if out := string(session.Scrub([]byte(`{"c":"g --password ","k":"v"}`))); out != `{"c":"g --password ","k":"v"}` {
		t.Errorf("empty flag value = %s", out)
	}
	// Malformed secret fields the old rule caught stay caught.
	for in, want := range map[string]string{
		`"key":"abc\"`: `"key":"[redacted]"`,
	} {
		if got := string(session.Scrub([]byte(in))); got != want {
			t.Errorf("Scrub(%s) = %s, want %s", in, got, want)
		}
	}
	// Formatting outside the scrubbed values is kept byte for byte.
	in := "{\n  \"spec\": \"id\",\n  \"cmd\": \"g --password p1 <x>\"\n}"
	if out := string(session.Scrub([]byte(in))); out != "{\n  \"spec\": \"id\",\n  \"cmd\": \"g --password [redacted] <x>\"\n}" {
		t.Errorf("formatting changed: %q", out)
	}
}
