package session

import (
	"bytes"
	"encoding/json"
	"regexp"
)

// redacted replaces a secret value in an evidence artifact.
const redacted = "[redacted]"

var (
	// secretField matches a JSON field whose name marks its value as a secret
	// and captures the field so the value can be replaced. These are the DSL and
	// config secret carriers: a signing key, a saved private key, a password, a
	// mnemonic. Non-secret evidence (hashes under "hash", addresses under
	// "from"/"to"/"address") is not in the set, so it is left intact.
	// The value may hold escaped characters, a quote among them.
	secretField = regexp.MustCompile(`("(?:key|privateKey|saveKey|password|mnemonic|secret)"\s*:\s*)"(?:\\.|[^"\\])*"`)
	// secretFieldLoose is the same field without escape rules, applied after
	// secretField so a malformed value (a line cut after a backslash) is still
	// hidden up to the next quote.
	secretFieldLoose = regexp.MustCompile(`("(?:key|privateKey|saveKey|password|mnemonic|secret)"\s*:\s*)"[^"]*"`)
	// secretFlag matches a --password/--unlock command-line flag and its value in
	// a command line (either "--password X" or "--password=X"). A quoted value
	// is taken whole with whatever is joined to it; anything else is taken up
	// to the next space, quotes included. In plain text a broken line is
	// acceptable and a leaked value is not, so the rule errs towards taking more.
	secretFlag = regexp.MustCompile(`(--(?:password|unlock)(?:=|\s+))(\\"(?:[^"\\]|\\[^"])*\\"\S*|"(?:[^"\\]|\\.)*"\S*|'[^']*'\S*|\S+)`)
	// jsonString is one string literal in valid JSON.
	jsonString = regexp.MustCompile(`"(?:\\.|[^"\\])*"`)
)

// Scrub redacts secret values from an evidence artifact before it is written,
// so no raw private key or password reaches the session tree (design §8: "no
// raw private key or password"). It replaces the values of sensitive JSON
// fields and the arguments of --password/--unlock flags in a recorded command;
// everything else — tx hashes, addresses, block numbers, provenance — is left
// exactly as-is, so a verdict stays traceable. It is applied at the session's
// write seam, not to the functional files a node reads.
//
// In valid JSON a command line is a string value: each string is decoded,
// scrubbed and encoded again on its own, so the flag rule sees the command as
// written and the document stays valid with every other byte unchanged.
func Scrub(b []byte) []byte {
	b = secretField.ReplaceAll(b, []byte(`${1}"`+redacted+`"`))
	b = secretFieldLoose.ReplaceAll(b, []byte(`${1}"`+redacted+`"`))
	if !json.Valid(b) {
		return secretFlag.ReplaceAll(b, []byte(`${1}`+redacted))
	}
	return jsonString.ReplaceAllFunc(b, func(literal []byte) []byte {
		var value string
		if json.Unmarshal(literal, &value) != nil {
			return literal
		}
		scrubbed := secretFlag.ReplaceAllString(value, `${1}`+redacted)
		if scrubbed == value {
			return literal
		}
		var out bytes.Buffer
		encoder := json.NewEncoder(&out)
		encoder.SetEscapeHTML(false)
		if encoder.Encode(scrubbed) != nil {
			return []byte(`"` + redacted + `"`)
		}
		return bytes.TrimSuffix(out.Bytes(), []byte("\n"))
	})
}
