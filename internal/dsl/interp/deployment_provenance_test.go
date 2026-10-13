package interp_test

import (
	"context"
	"testing"

	"github.com/0xmhha/chainbench/internal/core/session"
	"github.com/0xmhha/chainbench/internal/dsl"
	"github.com/0xmhha/chainbench/internal/dsl/interp"
)

type deploymentEvidenceAction struct{ provenance *session.ContractDeployment }

func (a deploymentEvidenceAction) Do(_ context.Context, ctx *interp.ActionCtx) error {
	ctx.Hash = "0xtx"
	ctx.Deployment = a.provenance
	return nil
}

func TestInterpreterPreservesPublicDeploymentProvenance(t *testing.T) {
	nonce := uint64(0)
	wanted := &session.ContractDeployment{Deployer: "0xpublic", Nonce: &nonce, ExpectedAddress: "0xexpected", ActualAddress: "0xactual", BytecodeSHA256: "public-checksum", Artifact: "sha256:public-checksum"}
	registry := interp.NewRegistry()
	registry.RegisterAction("deploy", deploymentEvidenceAction{provenance: wanted})
	interpreter := interp.NewInterpreter(interp.Deps{Actions: registry})
	record := &fakeRecord{}
	if _, err := interpreter.Run(context.Background(), dsl.Spec{Steps: []map[string]any{{"deploy": map[string]any{}}}}, testEnv(t), record); err != nil {
		t.Fatal(err)
	}
	if len(record.stepResults) != 1 || record.stepResults[0].Deployment != wanted || record.stepResults[0].Deployment.Nonce == nil || *record.stepResults[0].Deployment.Nonce != 0 {
		t.Fatal("deployment output was lost at the interpreter/recorder boundary")
	}
}
