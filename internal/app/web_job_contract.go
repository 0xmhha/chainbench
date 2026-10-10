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
type WebResourceClaim struct {
	HostIdentity string `json:"hostIdentity"`
	DataPath     string `json:"dataPath,omitempty"`
	Ports        []int  `json:"ports,omitempty"`
	Executable   string `json:"executable,omitempty"`
}

func claimsOverlap(a, b WebResourceClaim) bool {
	if a.HostIdentity != b.HostIdentity {
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
	Fingerprint    string             `json:"fingerprint"`
	Claims         []WebResourceClaim `json:"claims"`
	Phases         []string           `json:"phases"`
	Changes        []string           `json:"changes"`
	RequiredAccess []string           `json:"requiredAccess"`
	Payload        json.RawMessage    `json:"payload"`
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
	ChainSurfaces     []any                   `json:"chainSurfaces"`
}

type WebJobPhase struct {
	Name       string    `json:"name"`
	State      string    `json:"state"`
	StartedAt  time.Time `json:"startedAt,omitempty"`
	FinishedAt time.Time `json:"finishedAt,omitempty"`
	Message    string    `json:"message,omitempty"`
}

type WebJob struct {
	ID                  string            `json:"id"`
	ActorID             string            `json:"actorId"`
	WorkspaceID         string            `json:"workspaceId"`
	PlanID              string            `json:"planId"`
	Operation           string            `json:"operation"`
	State               string            `json:"state"`
	Phases              []WebJobPhase     `json:"phases"`
	CreatedAt           time.Time         `json:"createdAt"`
	Retention           string            `json:"retention"`
	NodeDisposition     string            `json:"nodeDisposition"`
	CancelReason        string            `json:"cancelReason,omitempty"`
	RunIDs              []string          `json:"runIds"`
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
