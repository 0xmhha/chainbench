package resource

import "testing"

func TestParseSize(t *testing.T) {
	for in, want := range map[string]uint64{
		"0": 0, "1024": 1024, "2GiB": 2 << 30, "500MiB": 500 << 20, "1 GB": 1 << 30, "3KiB": 3 << 10, "7B": 7,
	} {
		got, err := ParseSize(in)
		if err != nil || got != want {
			t.Errorf("ParseSize(%q) = %d, %v; want %d", in, got, err, want)
		}
	}
	for _, bad := range []string{"", "two", "2XB", "-1GiB"} {
		if _, err := ParseSize(bad); err == nil {
			t.Errorf("ParseSize(%q) accepted a value that is not a size", bad)
		}
	}
}
