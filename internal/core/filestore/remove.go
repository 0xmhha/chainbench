package filestore

import (
	"fmt"
	"path"
	"strings"
)

// minRemovableSegments is how many path segments a removable path must have
// below the filesystem root.
//
// Two. "/data" is one and is refused; "/data/chainbench" is two and is allowed.
// The number is not a claim about which directories matter — it is a floor that
// makes the catastrophic mistakes unrepresentable. A path assembled from an
// empty variable collapses toward "/" or "/data", and those are exactly the
// values that must never reach an rm -rf on somebody's server.
const minRemovableSegments = 2

// CheckRemovable is the coarse backstop every Store applies before it deletes.
//
// It cannot know whether a path is the RIGHT one — only the caller holding the
// target's data root knows that, and [CheckWithin] is where that check lives.
// What this rules out is the class of path that is wrong no matter who asked:
// empty, relative, the filesystem root, or one segment below it.
//
// Both halves are load-bearing, and they are here rather than in each Store
// because a guard that exists in one implementation and not the other protects
// exactly the target that is easiest to damage.
func CheckRemovable(p string) error {
	if strings.TrimSpace(p) == "" {
		return fmt.Errorf("filestore: refusing to remove an empty path")
	}
	if !path.IsAbs(p) {
		return fmt.Errorf("filestore: refusing to remove a relative path %q — a delete must name where it lands", p)
	}
	if strings.ContainsAny(p, "$~*?") {
		return fmt.Errorf("filestore: refusing to remove %q — a delete path must be literal, not expanded", p)
	}
	clean := path.Clean(p)
	if clean == "/" || clean == "." {
		return fmt.Errorf("filestore: refusing to remove the filesystem root")
	}
	if n := len(strings.Split(strings.Trim(clean, "/"), "/")); n < minRemovableSegments {
		return fmt.Errorf("filestore: refusing to remove %q — too close to the root (needs at least %d path segments)", clean, minRemovableSegments)
	}
	return nil
}

// CheckWithin reports whether p is inside root, which is the precise check
// [CheckRemovable] cannot make.
//
// A caller that knows the target's data root uses this so a delete cannot walk
// out of it: a path built from a bad join, or one carrying "..", is refused
// before it becomes an argument to rm. root itself is refused too — clearing a
// composition removes what the composition put there, not the directory the
// operator pointed the tool at.
func CheckWithin(root, p string) error {
	if err := CheckRemovable(p); err != nil {
		return err
	}
	if strings.TrimSpace(root) == "" {
		return fmt.Errorf("filestore: refusing to remove %q with no target root to confine it to", p)
	}
	cleanRoot, clean := path.Clean(root), path.Clean(p)
	if clean == cleanRoot {
		return fmt.Errorf("filestore: refusing to remove the target root %q itself", cleanRoot)
	}
	if !strings.HasPrefix(clean, strings.TrimSuffix(cleanRoot, "/")+"/") {
		return fmt.Errorf("filestore: refusing to remove %q — outside the target root %q", clean, cleanRoot)
	}
	return nil
}
