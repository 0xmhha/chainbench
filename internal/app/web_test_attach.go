package app

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"os"
	"path/filepath"
	"strings"

	"github.com/0xmhha/chainbench/internal/core/collector"
	"github.com/0xmhha/chainbench/internal/core/lifecycle"
	"github.com/0xmhha/chainbench/internal/core/node"
	"github.com/0xmhha/chainbench/internal/dsl"
	"github.com/0xmhha/chainbench/internal/testengine"
)

// webTestAttachOperation runs saved attach cases against a network this
// service composed and recorded. It launches, stops and removes nothing.
const webTestAttachOperation = "test.attach"

// webTestAttach is the reviewed read-only run. A case's attach declaration
// names some network by address; on the Web the endpoints, key set and
// capabilities are the recorded network's, so nothing the case wrote is dialed.
type webTestAttach struct {
	Cases []webExecutableCase `json:"cases"`
	Nodes node.NodeSet        `json:"nodes"`
}

func (e *WebChainEngine) prepareTestAttach(ctx context.Context, p *webChainPayload, state State) error {
	cases, err := e.prepareTestCases(ctx, p.Binary.Chain, p.Arguments.CaseRefs)
	if err != nil {
		return err
	}
	for _, c := range cases {
		spec, err := dsl.Parse(c.Content)
		if err != nil {
			return err
		}
		if err = validateWebAttachInputs(spec); err != nil {
			return fmt.Errorf("case %s: %w", c.Document.ID, err)
		}
	}
	p.Attach = &webTestAttach{Cases: cases, Nodes: webRecordedNodeSet(state)}
	return nil
}

func validateWebAttachInputs(spec dsl.Spec) error {
	if spec.EnvAttach == nil {
		return errors.New("an attach job runs only cases that attach to a running network")
	}
	if spec.Chain.ManifestPath != "" || spec.Chain.TemplatePath != "" || spec.Chain.GenesisExisting != "" || spec.Chain.Config != "" || spec.EnvBlueprint != "" {
		return errors.New("test file references require registered immutable assets")
	}
	for label, account := range spec.EnvAccounts {
		if strings.TrimSpace(account.KeyFile) != "" {
			return fmt.Errorf("account %s reads a key file on this server; bind a private account credential instead", label)
		}
	}
	return nil
}

// webRecordedNodeSet is the recorded network as the test engine addresses it.
// A node answers on the address it recorded, else on the target's host; a
// local target records none and answers on loopback.
func webRecordedNodeSet(state State) node.NodeSet {
	host := state.Target.Host
	if host == "" {
		host = "127.0.0.1"
	}
	ns := node.NodeSet{Chain: state.Chain, Network: "web-recorded", Capabilities: append([]string(nil), state.Capabilities...), Nodes: make([]node.Node, 0, len(state.Nodes))}
	for _, r := range state.Nodes {
		h := r.Host
		if h == "" {
			h = host
		}
		n := node.Node{Index: r.Index, Role: node.Role(r.Role), Host: h, Ports: r.Endpoints, PID: r.PID}
		if r.HTTP > 0 {
			n.RPCURL = fmt.Sprintf("http://%s:%d", h, r.HTTP)
		}
		if r.WS > 0 {
			n.WSURL = fmt.Sprintf("ws://%s:%d", h, r.WS)
		}
		if r.Metrics > 0 {
			n.MetricsURL = collector.MetricsURLOn(fmt.Sprintf("http://%s:%d", h, r.Metrics))
		}
		ns.Nodes = append(ns.Nodes, n)
	}
	return ns
}

// webRecordedOperation reports a job that acts on the recorded network rather
// than composing one, so its claims are the recorded placement.
func webRecordedOperation(operation string) bool {
	return webNodeControlOperation(operation) || operation == webTestAttachOperation
}

// webCaseOperation reports a job whose saved cases bring their own assets.
func webCaseOperation(operation string) bool {
	return operation == "test.run" || operation == webTestAttachOperation
}

func (e *WebChainEngine) executeTestAttach(ctx context.Context, a DeploymentActor, p webChainPayload, report func(WebJobPhase) error) (WebJobResult, error) {
	result := WebJobResult{NodeDisposition: "retained"}
	if p.Attach == nil || len(p.Attach.Cases) == 0 {
		return result, errors.New("no reviewed test cases")
	}
	lookup, err := e.documents.jobCredentialLookup(ctx, a, p.Set.DeploymentDocumentInput, p.Input.CredentialBindings)
	if err != nil {
		return result, err
	}
	root := ""
	err = e.phase(ctx, a, webTestAttachOperation, report, func() error {
		raw, err := os.ReadFile(filepath.Join(p.ControlDir, "chain-record.json"))
		if err != nil || manifestHash(raw) != p.RecordDigest {
			return ErrDeploymentConflict
		}
		var state State
		if err = json.Unmarshal(raw, &state); err != nil {
			return ErrDeploymentConflict
		}
		if err = verifyWebNetworkProcesses(ctx, state, p, lookup, false); err != nil {
			return err
		}
		keys, err := e.bindWebConfigKeys(ctx, state)
		if err != nil || keys != p.Keys {
			return ErrDeploymentConflict
		}
		urls, specs := []string{}, make([][]byte, 0, len(p.Attach.Cases))
		for _, n := range p.Attach.Nodes.Nodes {
			if n.RPCURL != "" {
				urls = append(urls, n.RPCURL)
			}
		}
		for _, c := range p.Attach.Cases {
			specs = append(specs, c.Content)
		}
		declared := &AttachDecl{Chain: state.Chain, RPCURLs: urls, KeysDir: state.KeysDir, Provides: p.Attach.Nodes.Capabilities}
		var runErr error
		root, runErr = AttachRun(ctx, Deps{Command: "web test.attach by " + a.ID, ServerLookup: lookup}, AttachRunIn{
			At: lifecycle.AdoptChainByDeclaration, Declared: declared, ArtifactRoot: webOwnedSessions(e.root),
			Specs: specs, Nodes: p.Attach.Nodes, ReadOnlyKeys: true,
		})
		if runErr != nil {
			return runErr
		}
		if sum, err := testengine.ReadSessionSummary(root); err != nil || sum.Failed() {
			return errors.Join(errors.New("one or more selected test cases failed or were blocked"), err)
		}
		return nil
	})
	if root != "" {
		ref, refErr := webTestSessionRef(webOwnedSessions(e.root), root)
		if refErr != nil {
			err = errors.Join(err, refErr)
		} else {
			result.RunIDs = []string{ref}
		}
	}
	return result, err
}
