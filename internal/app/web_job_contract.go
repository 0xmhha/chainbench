package app

import (
	"context"
	"encoding/json"
	"path"
	"strings"
	"time"
)

// WebPlanInput names immutable shared inputs; credential values never belong here.
type WebPlanInput struct {
	WorkspaceID        string                  `json:"workspaceId"`
	Operation          string                  `json:"operation"`
	DocumentRefs       []DeploymentDocumentRef `json:"documentRefs"`
	NodeIDs            []string                `json:"nodeIds,omitempty"`
	AssetRefs          []string                `json:"assetRefs,omitempty"`
	CredentialBindings map[string]string       `json:"credentialBindings,omitempty"`
	// AccountBindings maps a case account label to the caller's own account
	// key credential, for an attach job whose case names a key file.
	AccountBindings map[string]string `json:"accountBindings,omitempty"`
	Arguments       json.RawMessage   `json:"arguments,omitempty"`
	Retention       string            `json:"retention,omitempty"`
}

// WebResourceClaim comes only from an engine's resolved plan, never the browser.
// HostIdentity must bind aliases to the same physical host (including SSH key).
// Executable names a node binary the job launches on that host: the engine
// refuses to compose while the same binary runs there outside its workspace.
// Idle marks a server a spread plan inspects but places no node on: it stays
// in the plan so the job's input is pinned, and it holds nothing.
type WebResourceClaim struct {
	HostIdentity string `json:"hostIdentity"`
	DataPath     string `json:"dataPath,omitempty"`
	Ports        []int  `json:"ports,omitempty"`
	Executable   string `json:"executable,omitempty"`
	Idle         bool   `json:"idle,omitempty"`
}

func claimsOverlap(a, b WebResourceClaim) bool {
	if a.HostIdentity != b.HostIdentity || a.Idle || b.Idle {
		return false
	}
	if a.Executable != "" && a.Executable == b.Executable {
		return true
	}
	if a.DataPath != "" && b.DataPath != "" {
		aPath, bPath := path.Clean(a.DataPath), path.Clean(b.DataPath)
		if aPath == bPath || strings.HasPrefix(aPath, strings.TrimSuffix(bPath, "/")+"/") || strings.HasPrefix(bPath, strings.TrimSuffix(aPath, "/")+"/") {
			return true
		}
	}
	for _, p := range a.Ports {
		for _, q := range b.Ports {
			if p == q {
				return true
			}
		}
	}
	return false
}

// WebPreparedJob contains the engine's fully resolved immutable execution input.
// Fingerprint covers target identity, revisions, assets, bindings and arguments.
type WebPreparedJob struct {
	Fingerprint    string               `json:"fingerprint"`
	Claims         []WebResourceClaim   `json:"claims"`
	Phases         []string             `json:"phases"`
	Changes        []string             `json:"changes"`
	RequiredAccess []string             `json:"requiredAccess"`
	ChainSurfaces  []WebChainSurfaceRef `json:"chainSurfaces,omitempty"`
	Summary        *WebPlanSummary      `json:"summary,omitempty"`
	Payload        json.RawMessage      `json:"payload"`
}

// WebPlanSummary says what a plan will run and where, in the terms an operator
// reviews it by. It is display only: the fingerprint binds the plan.
type WebPlanSummary struct {
	Chain  string        `json:"chain"`
	Binary string        `json:"binary"`
	Cases  []string      `json:"cases,omitempty"`
	Nodes  []WebPlanNode `json:"nodes,omitempty"`
}

// WebPlanNode is one planned node and the server it lands on.
type WebPlanNode struct {
	Label  string `json:"label"`
	Role   string `json:"role"`
	Server string `json:"server"`
}

type WebPlan struct {
	ID                string                  `json:"id"`
	ActorID           string                  `json:"actorId"`
	WorkspaceID       string                  `json:"workspaceId"`
	Operation         string                  `json:"operation"`
	DocumentRefs      []DeploymentDocumentRef `json:"documentRefs"`
	TargetFingerprint string                  `json:"targetFingerprint"`
	ExpiresAt         time.Time               `json:"expiresAt"`
	Phases            []string                `json:"phases"`
	Changes           []string                `json:"changes"`
	Resources         []string                `json:"resources"`
	RequiredAccess    []string                `json:"requiredAccess"`
	Validation        map[string]any          `json:"validation"`
	ChainSurfaces     []WebChainSurfaceRef    `json:"chainSurfaces"`
	Summary           *WebPlanSummary         `json:"summary,omitempty"`
}

type WebJobPhase struct {
	Name       string    `json:"name"`
	State      string    `json:"state"`
	StartedAt  time.Time `json:"startedAt"`
	FinishedAt time.Time `json:"finishedAt,omitzero"`
	Message    string    `json:"message,omitempty"`
	// Failure is the kind of failure that stopped a failed phase, as the
	// engine classified it, when it did.
	Failure *WebPhaseFailure `json:"failure,omitempty"`
}

// WebPhaseFailure is an engine-classified failure: State names the designed
// failure state (for example ChainLaunchNodesFailPortBusy) and Detail what the
// engine recorded about it.
type WebPhaseFailure struct {
	State  string `json:"state"`
	Detail string `json:"detail,omitempty"`
}

type WebJob struct {
	ID          string `json:"id"`
	ActorID     string `json:"actorId"`
	WorkspaceID string `json:"workspaceId"`
	PlanID      string `json:"planId"`
	Operation   string `json:"operation"`
	State       string `json:"state"`
	// Terminal and NeedsCheck are derived on every read so screens do not keep
	// their own list of final states or their own reconcile rule.
	Terminal        bool          `json:"terminal"`
	NeedsCheck      bool          `json:"needsCheck"`
	Phases          []WebJobPhase `json:"phases"`
	CreatedAt       time.Time     `json:"createdAt"`
	Retention       string        `json:"retention"`
	NodeDisposition string        `json:"nodeDisposition"`
	CancelReason    string        `json:"cancelReason,omitempty"`
	CancelMessage   string        `json:"cancelMessage,omitempty"`
	// CredentialRefs are the private credential ids the job uses, never their
	// material.
	CredentialRefs []string `json:"credentialRefs"`
	RequestID      string   `json:"requestId,omitempty"`
	// Reconciliation is the latest read-only check of an interrupted job's
	// machines.
	Reconciliation *WebReconciliation `json:"reconciliation,omitempty"`
	RunIDs         []string           `json:"runIds"`
	// Cases names the test cases a job runs, so a list of jobs can be read.
	Cases               []string          `json:"cases,omitempty"`
	PartialEffects      []string          `json:"partialEffects"`
	UnresolvedResources []string          `json:"unresolvedResources"`
	Error               map[string]string `json:"error,omitempty"`
}

type WebJobResult struct {
	RunIDs              []string
	PartialEffects      []string
	UnresolvedResources []string
	NodeDisposition     string
}

// WebJobEngine uses existing engine verbs, with permission checks at every remote
// access. A nil engine disables creation rather than accepting simulated work.
type WebJobEngine interface {
	Prepare(context.Context, DeploymentActor, WebPlanInput) (WebPreparedJob, error)
	Execute(context.Context, DeploymentActor, WebPreparedJob, func(WebJobPhase) error) (WebJobResult, error)
	Cleanup(context.Context, DeploymentActor, WebPreparedJob) (WebJobResult, error)
}

func terminalWebJob(state string) bool {
	return state == "succeeded" || state == "failed" || state == "cancelled" || state == "interrupted"
}

// webJobNeedsCheck is whether the job ended without knowing what it left on
// its machines, which is when Reconcile may run.
func webJobNeedsCheck(job WebJob) bool {
	return terminalWebJob(job.State) && (job.State == "interrupted" || job.NodeDisposition == "unknown")
}
