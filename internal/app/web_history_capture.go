package app

import (
	"encoding/json"
	"sort"
	"strings"

	"github.com/0xmhha/chainbench/internal/core/session"
)

func (s *WebHistory) jobHistory() ([]WebRun, map[string]WebRun) {
	out, refs := []WebRun{}, map[string]WebRun{}
	if s.jobs == nil {
		return out, refs
	}
	s.jobs.mu.Lock()
	b, _ := json.Marshal(s.jobs.state)
	s.jobs.mu.Unlock()
	var state webJobState
	_ = json.Unmarshal(b, &state)
	for _, job := range state.Jobs {
		plan := state.Plans[job.PlanID]
		var payload webChainPayload
		_ = json.Unmarshal(plan.Prepared.Payload, &payload)
		run := WebRun{ID: "job-" + job.ID, JobID: job.ID, WorkspaceID: job.WorkspaceID, Chain: payload.Binary.Chain, ActorID: job.ActorID, State: job.State, StartedAt: job.CreatedAt.Format("2006-01-02T15:04:05.999999999Z07:00"), SessionRefs: append([]string{}, job.RunIDs...), Fingerprints: map[string]string{"target": plan.Prepared.Fingerprint}, ArtifactRefs: []string{"job.json"}, Summary: map[string]any{"kind": "job", "operation": job.Operation, "phases": job.Phases, "partialEffects": job.PartialEffects, "unresolvedResources": job.UnresolvedResources, "nodeDisposition": job.NodeDisposition, "error": job.Error, "missingDimensions": []string{"test-results", "metrics", "logs"}}}
		if payload.Binary.SHA256 != "" {
			run.Fingerprints["binary"] = payload.Binary.SHA256
		}
		data, _ := json.Marshal(run)
		_ = json.Unmarshal([]byte(s.redact(string(data))), &run)
		out = append(out, run)
		for _, ref := range job.RunIDs {
			refs[ref] = run
		}
	}
	return out, refs
}

func (s *WebHistory) sessionHistory(id, ref string, captured session.CapturedSession, linked WebRun) webHistoryCapture {
	files := map[string]string{}
	for name, text := range captured.Files {
		files[name] = s.redactHistoryText(text)
	}
	var result session.Result
	_ = json.Unmarshal([]byte(files["session.json"]), &result)
	state := "succeeded"
	missing := []string{"metrics", "logs", "finishedAt"}
	if len(result.Tests) == 0 {
		state = "unknown"
	}
	for _, test := range result.Tests {
		switch test.Status {
		case "fail", "blocked":
			if state != "unknown" {
				state = "failed"
			}
		case "pass", "skip":
		default:
			state = "unknown"
		}
	}
	run := WebRun{ID: id, JobID: linked.JobID, WorkspaceID: linked.WorkspaceID, Chain: linked.Chain, ActorID: linked.ActorID, State: state, StartedAt: result.StartedAt, SessionRefs: []string{ref}, Fingerprints: map[string]string{}, ArtifactRefs: []string{}, Summary: map[string]any{"kind": "engine-session", "command": result.Command, "tests": result.Tests, "counts": result.Summary, "captureGaps": captured.Gaps}}
	if linked.JobID == "" {
		missing = append(missing, "actorId", "workspaceId", "chain", "binary")
	} else if linked.Fingerprints["binary"] != "" {
		run.Fingerprints["binary"] = linked.Fingerprints["binary"]
	}
	details := map[string]any{}
	for name, text := range files {
		run.ArtifactRefs = append(run.ArtifactRefs, name)
		var value map[string]any
		if strings.HasPrefix(name, "environments/") && strings.HasSuffix(name, "/env.json") && json.Unmarshal([]byte(text), &value) == nil {
			if fingerprint, ok := value["fingerprint"].(string); ok && fingerprint != "" {
				run.Fingerprints[name] = fingerprint
			}
		}
		if strings.HasPrefix(name, "tests/") && !strings.HasSuffix(name, "/spec.json") && json.Valid([]byte(text)) {
			details[name] = json.RawMessage(text)
		}
	}
	sort.Strings(run.ArtifactRefs)
	run.Summary["missingDimensions"], run.Summary["results"] = missing, details
	return webHistoryCapture{run, files}
}

func (s *WebHistory) redactHistoryText(text string) string {
	var value any
	if json.Valid([]byte(text)) && decodeHistoryJSON([]byte(text), &value) == nil {
		value = historySecrets(value)
		b, _ := json.Marshal(value)
		return s.redact(string(b))
	}
	// Chainstate uses JSONL; each line is still a structured evidence object.
	lines := strings.Split(text, "\n")
	for i, line := range lines {
		if json.Valid([]byte(line)) && decodeHistoryJSON([]byte(line), &value) == nil {
			b, _ := json.Marshal(historySecrets(value))
			lines[i] = s.redact(string(b))
		} else {
			lines[i] = s.redact(string(session.Scrub([]byte(line))))
		}
	}
	return strings.Join(lines, "\n")
}

func historySecrets(value any) any {
	switch v := value.(type) {
	case map[string]any:
		for key, item := range v {
			switch strings.ToLower(key) {
			case "key", "privatekey", "private_key", "savekey", "password", "mnemonic", "secret", "passphrase", "passwordhash", "setuptoken":
				v[key] = "[REDACTED]"
			default:
				v[key] = historySecrets(item)
			}
		}
	case []any:
		for i := range v {
			v[i] = historySecrets(v[i])
		}
	}
	return value
}
