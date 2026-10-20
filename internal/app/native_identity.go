package app

import (
	"debug/buildinfo"
	"errors"
	"regexp"
	"strings"
)

// manifestSourceCommit preserves the legacy self-reported stamp. A binary
// without that stamp must carry an exact, unmodified Git revision in its Go
// build metadata; a malformed or conflicting stamp is never silently replaced.
func manifestSourceCommit(path string, version []byte) (string, error) {
	stamp := regexp.MustCompile(`(?m)^Git Commit: ([0-9a-f]{40})\s*$`).FindAllSubmatch(version, -1)
	if len(stamp) == 1 {
		return string(stamp[0][1]), nil
	}
	if len(stamp) > 1 || strings.Contains(string(version), "Git Commit:") {
		return "", errors.New("binary has an invalid source commit stamp")
	}
	info, err := buildinfo.ReadFile(path)
	if err != nil {
		return "", errors.New("binary lacks a verified source commit identity")
	}
	settings := map[string]string{}
	for _, setting := range info.Settings {
		settings[setting.Key] = setting.Value
	}
	revision := settings["vcs.revision"]
	if settings["vcs"] != "git" || settings["vcs.modified"] != "false" || !regexp.MustCompile(`^[0-9a-f]{40}$`).MatchString(revision) {
		return "", errors.New("binary lacks an unmodified Git build identity")
	}
	return revision, nil
}
