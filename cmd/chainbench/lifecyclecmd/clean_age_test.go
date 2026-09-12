package lifecyclecmd

import (
	"strings"
	"testing"
	"time"
)

// `clean --older-than` decides which session directories get REMOVED, and its
// parser was uncovered. The stakes are asymmetric: a value read too small keeps
// files, a value read wrongly deletes work.
//
// The Nd/Nw suffixes exist because time.ParseDuration has no day or week, so
// this is a hand-written parser in front of a destructive operation -- the kind
// of place where being wrong does not fail.

func TestParseAge_Table(t *testing.T) {
	cases := []struct {
		in    string
		want  time.Duration
		errIs string
	}{
		{in: "1d", want: 24 * time.Hour},
		{in: "30d", want: 30 * 24 * time.Hour},
		{in: "1w", want: 7 * 24 * time.Hour},
		{in: "2w", want: 14 * 24 * time.Hour},
		{in: " 3d ", want: 3 * 24 * time.Hour}, // surrounding space is the operator's, not a value
		{in: "90m", want: 90 * time.Minute},    // the durations ParseDuration already knows
		{in: "36h", want: 36 * time.Hour},

		// A non-positive age is not a small one: GCSessions reads it as "no age
		// policy", so the operator's condition would be dropped rather than
		// applied, on a command that removes directories.
		{in: "-7d", errIs: "must be positive"},
		{in: "-1w", errIs: "must be positive"},
		{in: "0", errIs: "must be positive"},
		{in: "0d", errIs: "must be positive"},
		{in: "-30m", errIs: "must be positive"},

		{in: "d", errIs: "invalid syntax"},
		{in: "w", errIs: "invalid syntax"},
		{in: "soon", errIs: "invalid duration"},
		{in: "", errIs: "invalid duration"},
		{in: "1 d", errIs: "invalid syntax"}, // a space inside the value is a typo
	}
	for _, c := range cases {
		t.Run(c.in, func(t *testing.T) {
			got, err := parseAge(c.in)
			if c.errIs != "" {
				if err == nil {
					t.Fatalf("parseAge(%q) = %s, wanted a refusal", c.in, got)
				}
				if !strings.Contains(err.Error(), c.errIs) {
					t.Errorf("refusal %q does not say %q", err, c.errIs)
				}
				return
			}
			if err != nil {
				t.Fatalf("parseAge(%q): %v", c.in, err)
			}
			if got != c.want {
				t.Errorf("parseAge(%q) = %s, want %s", c.in, got, c.want)
			}
		})
	}
}

// TestParseAge_DaysAndWeeksAreNotDurationSuffixes is why this parser exists at
// all: if time.ParseDuration understood them, the hand-written branch would be
// dead code and its bugs would be free ones.
func TestParseAge_DaysAndWeeksAreNotDurationSuffixes(t *testing.T) {
	for _, s := range []string{"1d", "1w"} {
		if _, err := time.ParseDuration(s); err == nil {
			t.Errorf("time.ParseDuration now understands %q; the hand-written branch is redundant and should go", s)
		}
	}
}
