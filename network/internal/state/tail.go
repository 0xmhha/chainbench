package state

import (
	"bufio"
	"fmt"
	"os"
)

// TailFile returns the last n lines of the file at path (n must be >= 1).
// Lines are returned without their trailing newline. If the file has fewer
// than n lines, all lines are returned.
//
// Uses a ring buffer — O(file-size) read time, O(n * avg_line_len) memory.
// Acceptable for log files up to hundreds of MB with modest n.
func TailFile(path string, n int) ([]string, error) {
	if n < 1 {
		return nil, fmt.Errorf("state: TailFile: n must be >= 1, got %d", n)
	}
	f, err := os.Open(path)
	if err != nil {
		return nil, fmt.Errorf("state: TailFile open: %w", err)
	}
	defer f.Close()

	scanner := bufio.NewScanner(f)
	// Allow large lines — some node logs may exceed the default 64 KiB.
	scanner.Buffer(make([]byte, 0, 64*1024), 4*1024*1024)

	ring := make([]string, 0, n)
	for scanner.Scan() {
		if len(ring) == n {
			// Drop oldest (first) — shift in place.
			copy(ring, ring[1:])
			ring = ring[:n-1]
		}
		ring = append(ring, scanner.Text())
	}
	if err := scanner.Err(); err != nil {
		return nil, fmt.Errorf("state: TailFile read: %w", err)
	}
	return ring, nil
}
