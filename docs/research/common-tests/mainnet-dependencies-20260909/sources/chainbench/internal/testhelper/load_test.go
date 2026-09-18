package testhelper

import "testing"

func TestBurnGas(t *testing.T) {
	const blockLimit = 100_000_000
	cases := []struct {
		name        string
		blockLimit  uint64
		fillPercent uint64
		want        uint64
		wantErr     bool
	}{
		{name: "25 percent", blockLimit: blockLimit, fillPercent: 25, want: 25_000_000},
		{name: "10 percent", blockLimit: blockLimit, fillPercent: 10, want: 10_000_000},
		{name: "full block", blockLimit: blockLimit, fillPercent: 100, want: 100_000_000},
		{name: "zero percent rejected", blockLimit: blockLimit, fillPercent: 0, wantErr: true},
		{name: "over 100 rejected", blockLimit: blockLimit, fillPercent: 101, wantErr: true},
		{name: "zero block limit rejected", blockLimit: 0, fillPercent: 25, wantErr: true},
		{name: "below burner floor rejected", blockLimit: 1_000_000, fillPercent: 5, wantErr: true},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			got, err := burnGas(tc.blockLimit, tc.fillPercent)
			if tc.wantErr {
				if err == nil {
					t.Fatalf("burnGas(%d, %d) = %d, want error", tc.blockLimit, tc.fillPercent, got)
				}
				return
			}
			if err != nil {
				t.Fatalf("burnGas(%d, %d): unexpected error: %v", tc.blockLimit, tc.fillPercent, err)
			}
			if got != tc.want {
				t.Fatalf("burnGas(%d, %d) = %d, want %d", tc.blockLimit, tc.fillPercent, got, tc.want)
			}
		})
	}
}
