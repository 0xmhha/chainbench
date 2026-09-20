// Package surface holds what every command group needs to say about itself, so
// a second rendering of the command tree reads one declaration rather than a
// hand-maintained list beside it.
package surface

import "github.com/spf13/cobra"

// ReadOnlyKey is the annotation a command carries to declare that running it
// changes nothing and prints no secret.
//
// It is a DECLARATION, not an inference (surface-unification-design §4.4). No
// walk of the code can tell whether `node rpc` mutates — the method is an
// argument — and no naming rule can tell that `keyring show` is safe while
// `keyring export` is not, since both only print. The author of the command
// knows, says so here, and the projection reads it.
const ReadOnlyKey = "chainbench.readonly"

// ReadOnly marks a command as a query: it must not change a file, a process or
// chain state, and its output must carry no secret.
//
// The two halves are both load-bearing. `keyring export` prints a private key
// and changes nothing, and it is disqualified by the second half alone — which
// is why the property is declared per command rather than derived from whether
// a verb writes.
func ReadOnly(c *cobra.Command) *cobra.Command {
	if c.Annotations == nil {
		c.Annotations = map[string]string{}
	}
	c.Annotations[ReadOnlyKey] = "true"
	return c
}

// IsReadOnly reports whether c declared itself a query.
func IsReadOnly(c *cobra.Command) bool {
	return c.Annotations[ReadOnlyKey] == "true"
}
