package testhelper

import "testing"

func TestAverageInterval(t *testing.T) {
	cases := []struct {
		name       string
		timestamps []uint64
		want       float64
		wantErr    bool
	}{
		{name: "even one-second cadence", timestamps: []uint64{100, 101, 102, 103}, want: 1},
		{name: "two samples", timestamps: []uint64{10, 13}, want: 3},
		{name: "uneven averages", timestamps: []uint64{0, 2, 4, 10}, want: 10.0 / 3.0},
		{name: "single sample rejected", timestamps: []uint64{5}, wantErr: true},
		{name: "empty rejected", timestamps: []uint64{}, wantErr: true},
		{name: "non-monotonic rejected", timestamps: []uint64{10, 9}, wantErr: true},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			got, err := averageInterval(tc.timestamps)
			if tc.wantErr {
				if err == nil {
					t.Fatalf("averageInterval(%v) = %v, want error", tc.timestamps, got)
				}
				return
			}
			if err != nil {
				t.Fatalf("averageInterval(%v): unexpected error: %v", tc.timestamps, err)
			}
			if diff := got - tc.want; diff > 1e-9 || diff < -1e-9 {
				t.Fatalf("averageInterval(%v) = %v, want %v", tc.timestamps, got, tc.want)
			}
		})
	}
}
