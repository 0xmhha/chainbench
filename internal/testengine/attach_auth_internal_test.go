package testengine

import (
	"net/http"
	"net/http/httptest"
	"os"
	"sync/atomic"
	"testing"

	"github.com/0xmhha/chainbench/internal/core/node"
)

func TestEndpointAuthenticationDoesNotCrossNodesOrRedirects(t *testing.T) {
	t.Setenv("CHAINBENCH_TEST_ENDPOINT_TOKEN", "test-token-only")
	var externalCalls atomic.Int32
	external := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		externalCalls.Add(1)
		if r.Header.Get("Authorization") != "" {
			t.Error("authentication crossed to another node")
		}
		w.WriteHeader(http.StatusOK)
	}))
	defer external.Close()
	source := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) { http.Redirect(w, r, external.URL, http.StatusFound) }))
	defer source.Close()
	clients, _, err := attachAuthClients(AttachConfig{Nodes: node.NodeSet{Nodes: []node.Node{
		{Index: 1, RPCURL: source.URL, Auth: node.Auth{"type": "jwt", "env": "CHAINBENCH_TEST_ENDPOINT_TOKEN"}},
		{Index: 2, RPCURL: external.URL},
	}}})
	if err != nil {
		t.Fatal(err)
	}
	response, err := clients[source.URL].Get(source.URL)
	if err != nil {
		t.Fatal(err)
	}
	_ = response.Body.Close()
	if response.StatusCode != http.StatusFound || externalCalls.Load() != 0 {
		t.Fatal("authenticated cross-origin redirect was followed")
	}
	response, err = clients[external.URL].Get(external.URL)
	if err != nil {
		t.Fatal(err)
	}
	_ = response.Body.Close()
	if externalCalls.Load() != 1 {
		t.Fatal("second node was not read")
	}
}

func TestEndpointProfileMissingSecretFailsBeforeRecording(t *testing.T) {
	t.Setenv("CHAINBENCH_TEST_ENDPOINT_TOKEN", "")
	root := t.TempDir()
	_, err := NewAttachEngine(AttachConfig{Chain: "stablenet", ArtifactRoot: root, Nodes: node.NodeSet{Nodes: []node.Node{{Index: 1, RPCURL: "http://unused", Auth: node.Auth{"type": "jwt", "env": "CHAINBENCH_TEST_ENDPOINT_TOKEN"}}}}})
	if err == nil {
		t.Fatal("missing authentication was ignored")
	}
	if entries, err := os.ReadDir(root); err != nil || len(entries) != 0 {
		t.Fatal("authentication refusal wrote session data", err)
	}
}
