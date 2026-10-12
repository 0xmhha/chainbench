package app

import (
	"errors"
	"testing"
)

// The editor links an engine error to its statement or field by the path the
// error message starts with, as a pointer or as dotted field names; an error
// without one belongs to the whole document.
func TestContentErrorPathTakesTheLeadingPointer(t *testing.T) {
	for msg, want := range map[string]string{
		`/content/steps/1/do: "nope" is not a registered builtin`:                    "/content/steps/1/do",
		`/content/hooks/pre/0: no unique supported oneOf variant (/content/hooks/x)`: "/content/hooks/pre/0",
		`spec x has unresolved references (nothing composed): ref:missing`:           "/content",
		`/etc/passwd: not a case path`:                                               "/content",
		`launch.all.nosuchflag: boolean overrides must enable a boolean knob`:        "/content/launch/all/nosuchflag",
		`topology: at least one node is required`:                                    "/content/topology",
		`a sentence: with a colon`:                                                   "/content",
	} {
		if got := ContentErrorPath(errors.New(msg)); got != want {
			t.Errorf("%q: %q, want %q", msg, got, want)
		}
	}
}
