package testhelper

import (
	"context"
	"errors"
	"strings"
	"testing"

	"github.com/0xmhha/chainbench/internal/core/node"
	"github.com/0xmhha/chainbench/internal/dsl/interp"
)

type signatureLogControl struct {
	fakeNodeControl
	text   string
	logErr error
}

func (c *signatureLogControl) Log(_ context.Context, _ node.Node, limit int) (string, error) {
	if len(c.text) > limit {
		return c.text[len(c.text)-limit:], c.logErr
	}
	return c.text, c.logErr
}

func TestCompleteNodeLogCannotSilentlyUseATruncatedTail(t *testing.T) {
	for _, tt := range []struct {
		name, text string
		complete   any
		logErr     error
		wantErr    bool
	}{
		{"complete short log", "started\n", true, nil, false},
		{"empty complete log refused", "", true, nil, true},
		{"empty legacy tail allowed", "", false, nil, false},
		{"ambiguous exact limit", strings.Repeat("x", 16), true, nil, true},
		{"error hidden before tail", "invalid signature\n" + strings.Repeat("x", 16), true, nil, true},
		{"legacy tail still allowed", strings.Repeat("x", 32), false, nil, false},
		{"invalid complete control", "started\n", "true", nil, true},
		{"unreadable log", "", true, errors.New("log unavailable"), true},
	} {
		t.Run(tt.name, func(t *testing.T) {
			control := &signatureLogControl{text: tt.text, logErr: tt.logErr}
			d := faultDeps(control)
			action, _ := d.Actions.Action("readNodeLog")
			ac := &interp.ActionCtx{Env: envWithNodes(t, 1, "http://unused"), Deps: &d, Args: map[string]any{"on": "node1", "maxBytes": 16, "requireComplete": tt.complete}}
			err := action.Do(context.Background(), ac)
			if (err != nil) != tt.wantErr {
				t.Fatalf("error=%v, want refusal=%v; value=%#v", err, tt.wantErr, ac.Value)
			}
		})
	}
}

func TestDerivedRegexpCountRequiresAnIntactLogInterval(t *testing.T) {
	pattern := `(?i)ecrecover failed|invalid[^[:cntrl:]]*sig|verify[^[:cntrl:]]*error|signature[^[:cntrl:]]*(error|fail|invalid)|secp256k1[^[:cntrl:]]*(error|fail|invalid)`
	for _, tt := range []struct {
		name          string
		of            []any
		since         any
		pattern, want string
		wantErr       bool
	}{
		{"healthy interval", []any{"boot\nimported block\n"}, "boot\n", pattern, "0", false},
		{"historical error excluded", []any{"invalid signature\nboot\nimported block\n"}, "invalid signature\nboot\n", pattern, "0", false},
		{"mixed case signature error", []any{"boot\nSignature verification FAILED\n"}, "boot\n", pattern, "1", false},
		{"multiple error forms", []any{"ecrecover failed\ninvalid block sig\nsecp256k1 error\n"}, nil, pattern, "3", false},
		{"unchanged readable log", []any{"boot\n"}, "boot\n", pattern, "0", false},
		{"normalized newline boundary", []any{"boot\nimported block\n"}, "boot", pattern, "0", false},
		{"partial line hides signature prefix", []any{"boot\nSignature verification FAILED\n"}, "boot\nSignature", pattern, "", true},
		{"rotation loses prefix", []any{"new file\n"}, "boot\n", pattern, "", true},
		{"invalid regexp", []any{"boot\n"}, nil, "[", "", true},
		{"missing pattern", []any{"boot\n"}, nil, "", "", true},
		{"nonstring log", []any{12}, nil, pattern, "", true},
		{"nonstring baseline", []any{"boot\n"}, true, pattern, "", true},
		{"multiple streams ambiguous", []any{"first", "second"}, nil, pattern, "", true},
	} {
		t.Run(tt.name, func(t *testing.T) {
			d := deps()
			reader, _ := d.Actions.Reader("derive")
			args := map[string]any{"op": "regexpCount", "of": tt.of, "pattern": tt.pattern}
			if tt.since != nil {
				args["since"] = tt.since
			}
			got, err := reader(context.Background(), &d, nil, args)
			if (err != nil) != tt.wantErr || !tt.wantErr && got != tt.want {
				t.Fatalf("value=%#v err=%v, want %q, refusal=%v", got, err, tt.want, tt.wantErr)
			}
		})
	}
}
