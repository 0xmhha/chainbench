package app

import (
	"strings"

	"github.com/0xmhha/chainbench/internal/resource"
)

// WebAccessError refuses a plan for SSH servers the user has bound no personal
// credential to. Missing names each as "ssh:<server>", the same form a bundle
// export lists as a required binding. The plan input lacks those bindings, so
// it is an input the user corrects, not a permission they lack.
type WebAccessError struct {
	Missing []string
}

func (e *WebAccessError) Error() string {
	return "bind your SSH credential to these servers first: " + strings.Join(e.Missing, ", ")
}

// webMissingAccess lists the target servers reached over SSH that have no
// binding in the plan input. Local targets need none.
func webMissingAccess(servers []resource.Server, bindings map[string]string) []string {
	missing := []string{}
	for _, s := range servers {
		if target := resource.TargetOf(s, ""); target.Server != "" && bindings[s.Name] == "" {
			missing = append(missing, "ssh:"+s.Name)
		}
	}
	return missing
}
