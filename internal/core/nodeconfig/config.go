// Config resolution (part of package nodeconfig): the three-source
// configuration resolution required by the chain setup surface (requirement
// #4): a value may come from a code default, a config file, or a runtime
// flag/env override, with later sources winning. Resolution produces one merged
// view that every pipeline phase and driver consumes.
//
// Config is represented as flat dot-path keys (e.g. "ports.base_http") mirroring
// the profile schema, so this stays free of any file format: callers Flatten a
// parsed YAML/JSON map into Values and Merge the layers. (Formerly the
// standalone config package, folded into nodeconfig in R1.)
package nodeconfig

import (
	"strconv"
	"time"
)

// Values is a flat map of dot-path key to raw string value.
type Values map[string]string

// Defaults returns the canonical code-default layer. These mirror
// network/schema/defaults.json and the profile defaults so a bare setup with no
// file or flags still resolves to a working local chain.
func Defaults() Values {
	return Values{
		"chain":              "stablenet",
		"nodes.validators":   "4",
		"nodes.endpoints":    "1",
		"nodes.verbosity":    "3",
		"nodes.gcmode":       "full",
		"nodes.cache":        "1024",
		"data.directory":     "data",
		"keys.mode":          "static",
		"keys.source":        "keys/preset",
		"ports.base_p2p":     "30301",
		"ports.base_http":    "8501",
		"ports.base_ws":      "9501",
		"ports.base_auth":    "8551",
		"ports.base_metrics": "6061",
		"logging.rotation":   "true",
		"logging.max_size":   "10M",
		"logging.max_files":  "5",
		"logging.directory":  "data/logs",
	}
}

// Merge overlays layers left-to-right: a key set in a later layer overrides the
// same key in an earlier one. Empty-string values still override (an explicit
// clear); callers wanting "unset" should omit the key. The result is a fresh
// map; inputs are not mutated.
func Merge(layers ...Values) Values {
	out := Values{}
	for _, layer := range layers {
		for k, v := range layer {
			out[k] = v
		}
	}
	return out
}

// Resolve is the standard three-source resolution: Defaults() first, then the
// file layer, then the override (flag/env) layer.
func Resolve(file, override Values) Values {
	return Merge(Defaults(), file, override)
}

// Flatten converts a nested map (as decoded from YAML/JSON) into dot-path
// Values. Nested maps recurse; scalars are stringified; slices/other complex
// values are skipped (they are handled by chain-specific code, not the core
// resolver). Keys with non-string map keys are ignored.
func Flatten(nested map[string]any) Values {
	out := Values{}
	flattenInto(out, "", nested)
	return out
}

func flattenInto(out Values, prefix string, m map[string]any) {
	for k, v := range m {
		key := k
		if prefix != "" {
			key = prefix + "." + k
		}
		switch t := v.(type) {
		case map[string]any:
			flattenInto(out, key, t)
		case nil:
			// skip: absence, not an explicit clear
		case string:
			out[key] = t
		case bool:
			out[key] = strconv.FormatBool(t)
		case int:
			out[key] = strconv.Itoa(t)
		case int64:
			out[key] = strconv.FormatInt(t, 10)
		case float64:
			// JSON numbers decode as float64; render integers without a
			// trailing ".0" so "8283" stays "8283".
			if t == float64(int64(t)) {
				out[key] = strconv.FormatInt(int64(t), 10)
			} else {
				out[key] = strconv.FormatFloat(t, 'f', -1, 64)
			}
		default:
			// slices, structs, etc. are not core-resolver concerns; skip.
		}
	}
}

// String returns the value for key, or def if absent.
func (v Values) String(key, def string) string {
	if s, ok := v[key]; ok {
		return s
	}
	return def
}

// Int returns the value for key parsed as an int, or def if absent/unparseable.
func (v Values) Int(key string, def int) int {
	if s, ok := v[key]; ok {
		if n, err := strconv.Atoi(s); err == nil {
			return n
		}
	}
	return def
}

// Bool returns the value for key parsed as a bool, or def if absent/unparseable.
// Accepts the usual truthy strings (true/false/1/0/yes/no, case-insensitive).
func (v Values) Bool(key string, def bool) bool {
	if s, ok := v[key]; ok {
		switch s {
		case "true", "True", "TRUE", "1", "yes", "Yes", "YES", "on":
			return true
		case "false", "False", "FALSE", "0", "no", "No", "NO", "off":
			return false
		}
	}
	return def
}

// Duration returns the value for key parsed as a time.Duration, or def if
// absent/unparseable.
func (v Values) Duration(key string, def time.Duration) time.Duration {
	if s, ok := v[key]; ok {
		if d, err := time.ParseDuration(s); err == nil {
			return d
		}
	}
	return def
}
