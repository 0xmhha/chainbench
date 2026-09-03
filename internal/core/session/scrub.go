package session

import "regexp"

// redacted replaces a secret value in an evidence artifact.
const redacted = "[redacted]"

var (
	// secretField matches a JSON field whose name marks its value as a secret
	// and captures the field so the value can be replaced. These are the DSL and
	// config secret carriers: a signing key, a saved private key, a password, a
	// mnemonic. Non-secret evidence (hashes under "hash", addresses under
	// "from"/"to"/"address") is not in the set, so it is left intact.
	secretField = regexp.MustCompile(`("(?:key|privateKey|saveKey|password|mnemonic|secret)"\s*:\s*)"[^"]*"`)
	// secretFlag matches a --password/--unlock command-line flag and its value in
	// a recorded command string (either "--password X" or "--password=X").
	secretFlag = regexp.MustCompile(`(--(?:password|unlock)(?:=|\s+))(\S+)`)
)

// Scrub redacts secret values from an evidence artifact before it is written,
// so no raw private key or password reaches the session tree (design §8: "no
// raw private key or password"). It replaces the values of sensitive JSON
// fields and the arguments of --password/--unlock flags in a recorded command;
// everything else — tx hashes, addresses, block numbers, provenance — is left
// exactly as-is, so a verdict stays traceable. It is applied at the session's
// write seam, not to the functional files a node reads.
func Scrub(b []byte) []byte {
	b = secretField.ReplaceAll(b, []byte(`${1}"`+redacted+`"`))
	b = secretFlag.ReplaceAll(b, []byte(`${1}`+redacted))
	return b
}
