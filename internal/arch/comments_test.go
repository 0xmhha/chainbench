package arch

import (
	"sort"
	"strings"
	"testing"
)

// enforced are the claim kinds a comment may not make falsely. They are the
// ones a parser decides outright: a path either exists or it does not, and a
// package either has that name or it does not.
//
// doc-opens-with-prose is deliberately absent. A test documented by the
// scenario it proves ("A governance flow: send a transaction, then ...") reads
// better than one that repeats its own name, and that is a style choice rather
// than a false statement.
var enforced = map[string]bool{
	KindDeadFileCite:     true,
	KindDeadFileName:     true,
	KindDeadPkgPath:      true,
	KindDeadDocPath:      true,
	KindWrongPackageName: true,
	KindNamesOtherSymbol: true,
}

// TestCommentsDoNotContradictTheCode keeps comments from outliving what they
// describe.
//
// It exists because they did. A comment claiming the legacy role spellings
// survived "because they are written into persisted state" was read as fact in
// a 2026-09-14 review and produced the wrong decision; so did one saying the
// poa family has no proxy tier, which its own SupportsRole contradicts. Neither
// is machine-decidable, but the same drift left 85 claims that are: package
// comments naming packages that no longer exist, paths from before the pkg/ to
// internal/ move, and docs still opening with a symbol's previous name.
//
// Those are the ones this test holds at zero.
func TestCommentsDoNotContradictTheCode(t *testing.T) {
	claims, err := CommentClaims(moduleRoot)
	if err != nil {
		t.Fatalf("collect comment claims: %v", err)
	}

	byKind := map[string][]CommentClaim{}
	for _, c := range claims {
		if enforced[c.Kind] {
			byKind[c.Kind] = append(byKind[c.Kind], c)
		}
	}
	if len(byKind) == 0 {
		return
	}

	kinds := make([]string, 0, len(byKind))
	for k := range byKind {
		kinds = append(kinds, k)
	}
	sort.Strings(kinds)

	var b strings.Builder
	for _, k := range kinds {
		b.WriteString("\n" + k + ":")
		for _, c := range byKind[k] {
			b.WriteString("\n\t" + c.Pos + "  " + c.Detail)
		}
	}
	t.Errorf("%d comment(s) contradict the code:%s", len(claims), b.String())
}
