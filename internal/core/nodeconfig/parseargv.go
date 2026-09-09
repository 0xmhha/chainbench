package nodeconfig

import "strings"

// RunView is what a node's launch argv reveals about how it was configured: the
// binary it runs and the paths it was given. It is the inverse of Argv — enough
// to recover a running node's environment from its command line alone, when no
// record of the composition survives.
type RunView struct {
	// Binary is argv[0], the executable the node runs.
	Binary string
	// DataDir is the node's --datadir, empty when the flag was absent.
	DataDir string
	// ConfigPath is the node's --config file, empty when the flag was absent.
	ConfigPath string
}

// ParseArgv recovers a RunView from a process's argv (argv[0] is the binary).
// It reads the --datadir and --config flags in either spelling — "--flag value"
// or "--flag=value", one dash or two — and ignores every flag it does not know.
// The flag names are the KeyDataDir/KeyConfig constants, so Argv (which writes
// them) and ParseArgv (which reads them) name the flags in one place.
func ParseArgv(argv []string) RunView {
	var v RunView
	if len(argv) == 0 {
		return v
	}
	v.Binary = argv[0]
	for i := 1; i < len(argv); i++ {
		name, inline, hasInline := splitFlag(argv[i])
		switch name {
		case string(KeyDataDir):
			v.DataDir = flagValue(argv, &i, inline, hasInline)
		case string(KeyConfig):
			v.ConfigPath = flagValue(argv, &i, inline, hasInline)
		}
	}
	return v
}

// splitFlag strips leading dashes and splits an =inline value; it returns the
// bare flag name, the inline value, and whether one was present. A non-flag
// argument yields an empty name.
func splitFlag(arg string) (name, inline string, hasInline bool) {
	if !strings.HasPrefix(arg, "-") {
		return "", "", false
	}
	a := strings.TrimLeft(arg, "-")
	if i := strings.IndexByte(a, '='); i >= 0 {
		return a[:i], a[i+1:], true
	}
	return a, "", false
}

// flagValue returns a flag's value: the inline part when the flag was --k=v,
// otherwise the following argv element, which it consumes by advancing i.
func flagValue(argv []string, i *int, inline string, hasInline bool) string {
	if hasInline {
		return inline
	}
	if *i+1 < len(argv) {
		*i++
		return argv[*i]
	}
	return ""
}
