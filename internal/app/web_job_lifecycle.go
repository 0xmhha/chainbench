package app

import (
	"context"
	"fmt"
	"slices"
	"strings"
)

// WebBusyError refuses a job whose physical resources another job holds. It
// names each holder the caller may read, so the refusal says who to ask.
type WebBusyError struct {
	Holders []WebResourceConflict
}

func (e *WebBusyError) Error() string {
	ids := make([]string, 0, len(e.Holders))
	for _, h := range e.Holders {
		ids = append(ids, fmt.Sprintf("job %s by %s", h.JobID, h.ActorID))
	}
	return "physical resources are in use by " + strings.Join(ids, ", ")
}

func (e *WebBusyError) Is(target error) bool { return target == ErrDeploymentConflict }

type webRequestIDKey struct{}

// ContextWithRequestID carries the HTTP request id into a job it starts, so
// the job and its error name the request that created it.
func ContextWithRequestID(ctx context.Context, id string) context.Context {
	return context.WithValue(ctx, webRequestIDKey{}, id)
}

func requestIDFrom(ctx context.Context) string {
	id, _ := ctx.Value(webRequestIDKey{}).(string)
	return id
}

// webJobRevalidator is an engine that can recheck a plan's targets, assets
// and surfaces without executing it, before the job enters running.
type webJobRevalidator interface {
	Revalidate(context.Context, DeploymentActor, WebPreparedJob) error
}

// webJobCredentialRefs lists the private credential ids a job uses: its SSH
// bindings and its account-key bindings, sorted and without duplicates.
func webJobCredentialRefs(in WebPlanInput) []string {
	seen := map[string]bool{}
	out := []string{}
	for _, bindings := range []map[string]string{in.CredentialBindings, in.AccountBindings} {
		for _, id := range bindings {
			if id != "" && !seen[id] {
				seen[id] = true
				out = append(out, id)
			}
		}
	}
	slices.Sort(out)
	return out
}
