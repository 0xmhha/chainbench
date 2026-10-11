package app

import (
	"regexp"
	"strings"
)

var (
	contentErrorPointer = regexp.MustCompile(`^(/content(?:/[A-Za-z0-9_.$-]+)*):`)
	contentErrorFields  = regexp.MustCompile(`^([A-Za-z][A-Za-z0-9_-]*(?:\.[A-Za-z0-9_-]+)*):`)
)

// ContentErrorPath is the JSON pointer an engine error about a document starts
// with (or its dotted field names as a pointer), so an editor can select the statement or field it is about. An error
// that names no place in the document belongs to all of it, /content.
func ContentErrorPath(err error) string {
	text := strings.TrimSpace(err.Error())
	if m := contentErrorPointer.FindStringSubmatch(text); m != nil {
		return m[1]
	}
	// Resource validators name the field as dotted keys, launch.all.flag.
	if m := contentErrorFields.FindStringSubmatch(text); m != nil {
		return "/content/" + strings.ReplaceAll(m[1], ".", "/")
	}
	return "/content"
}
