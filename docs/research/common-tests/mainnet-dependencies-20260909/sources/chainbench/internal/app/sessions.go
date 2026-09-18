package app

import (
	"context"
	"errors"
	"fmt"
	"os"
	"time"

	"github.com/0xmhha/chainbench/internal/core/session"
)

// GCSessionsIn selects which session artifacts to delete. At least one policy
// must be given: with neither, the call would delete every completed session,
// which is never what a caller that forgot a flag meant (F16-O4).
type GCSessionsIn struct {
	// Root is the artifact root holding the session directories.
	Root string
	// OlderThan removes sessions whose verdict is older than this age. Zero
	// disables the age policy.
	OlderThan time.Duration
	// KeepLast protects the newest N sessions. Zero or less disables it.
	KeepLast int
}

// GCSessionsOut lists what was removed.
type GCSessionsOut struct {
	Removed []string
}

// GCSessions garbage-collects completed session directories. Only sessions that
// wrote a session.json are considered, so a run still in progress — which has
// not recorded its verdict yet — is preserved regardless of policy.
func GCSessions(_ context.Context, d Deps, in GCSessionsIn) (GCSessionsOut, error) {
	if in.Root == "" {
		return GCSessionsOut{}, errors.New("app: an artifact root is required")
	}
	if in.OlderThan <= 0 && in.KeepLast <= 0 {
		return GCSessionsOut{}, errors.New("app: session gc needs an older-than age and/or a keep-last count")
	}
	ids, err := session.List(in.Root)
	if err != nil {
		return GCSessionsOut{}, fmt.Errorf("app: session gc: %w", err)
	}

	// List is oldest-first, so the newest N are the tail.
	protected := map[string]bool{}
	if in.KeepLast > 0 {
		start := max(0, len(ids)-in.KeepLast)
		for _, id := range ids[start:] {
			protected[id] = true
		}
	}
	cutoff := d.now().Add(-in.OlderThan)

	var removed []string
	for _, id := range ids {
		if protected[id] {
			continue
		}
		if in.OlderThan > 0 {
			fi, statErr := os.Stat(session.SessionFilePath(in.Root, id))
			if statErr != nil || fi.ModTime().After(cutoff) {
				continue // unreadable or newer than the cutoff: keep
			}
		}
		if err := os.RemoveAll(session.SessionDir(in.Root, id)); err != nil {
			return GCSessionsOut{Removed: removed}, fmt.Errorf("app: session gc: remove %s: %w", id, err)
		}
		removed = append(removed, id)
	}
	return GCSessionsOut{Removed: removed}, nil
}
