package testkit

import "testing"

func TestReportCoverage(t *testing.T) {
	mk := func(statuses ...Status) Report {
		r := Report{}
		for _, s := range statuses {
			r.Results = append(r.Results, Result{Status: s})
		}
		return r
	}

	cases := []struct {
		name       string
		rep        Report
		applicable int
		want       int
	}{
		{"unset applicable -> unknown -> 100", mk(StatusPass, StatusSkip), 0, 100},
		{"all applicable ran", mk(StatusPass, StatusFail), 2, 100},
		{"2 of 3 applicable ran", mk(StatusPass, StatusFail, StatusSkip), 3, 66},
		{"none ran", mk(StatusSkip, StatusSkip), 2, 0},
	}
	for _, c := range cases {
		t.Run(c.name, func(t *testing.T) {
			c.rep.Applicable = c.applicable
			if got := c.rep.Coverage(); got != c.want {
				t.Errorf("Coverage() = %d, want %d", got, c.want)
			}
		})
	}
}
