package session_test

import (
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
