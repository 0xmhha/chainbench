package app

import (
	"regexp"
	"strings"
)

var testCaseErrorPointer = regexp.MustCompile(`^(/content(?:/[A-Za-z0-9_.$-]+)*):`)

// TestCaseErrorPath is the JSON pointer an engine error about a case starts
// with, so an editor can select the statement or field it is about. An error
// that names no place in the case belongs to the whole case, /content.
func TestCaseErrorPath(err error) string {
	if m := testCaseErrorPointer.FindStringSubmatch(strings.TrimSpace(err.Error())); m != nil {
		return m[1]
	}
	return "/content"
}
