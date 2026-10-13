package app

import (
	"context"
	"encoding/json"
	"errors"
	"testing"
)

func TestCredentialCheckUsesCLIWithPrivateInputs(t *testing.T) {
	s, a, w, c := leaseFixture(t)
	var d DeploymentDocument
	var err error
	for _, ref := range w.Documents {
		candidate, e := s.DocumentRevision(ref.ID, ref.Revision)
		if e != nil {
			t.Fatal(e)
		}
		if candidate.Kind == "server-set" {
			d = candidate
			break
		}
	}
	var content map[string]any
	if err = json.Unmarshal(d.Content, &content); err != nil {
		t.Fatal(err)
	}
	ssh := map[string]any{}
	content["ssh"] = ssh
	ssh["insecure_host_key"] = false
	ssh["known_hosts_file"] = "/nonexistent/chainbench-unit-known-hosts"
	d.Content, err = json.Marshal(content)
	if err != nil {
		t.Fatal(err)
	}
	d, err = s.SaveDocument(a, d.ID, d.Revision, d.DeploymentDocumentInput)
	if err != nil {
		t.Fatal(err)
	}
	for i, ref := range w.Documents {
		if ref.ID == d.ID {
			w.Documents[i] = DeploymentDocumentRef{ID: d.ID, Revision: d.Revision}
		}
	}
	w, err = s.SaveWorkspace(a, w.ID, w.Revision, w.DeploymentWorkspaceInput)
	if err != nil {
		t.Fatal(err)
	}
	stop := errors.New("CLI access check refused")
	cli := &runCLI{err: stop}
	s.UseCLI(cli)
	_, err = s.CheckCredential(context.Background(), a, w.ID, "ssh", c.ID)
	if !errors.Is(err, stop) || len(cli.args) < 2 || cli.args[0] != "resource" || cli.args[1] != "access" {
		t.Fatalf("private check bypassed CLI: %v %v", err, cli.args)
	}
}
