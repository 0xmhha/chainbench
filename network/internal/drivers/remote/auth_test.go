package remote

import (
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"github.com/0xmhha/chainbench/network/internal/types"
)

func TestAPIKeyTransport_InjectsHeader(t *testing.T) {
	var gotHeader string
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		gotHeader = r.Header.Get("X-Api-Key")
		_, _ = w.Write([]byte("ok"))
	}))
	defer srv.Close()

	rt := APIKeyTransport(nil, "X-Api-Key", "secret123")
	req, _ := http.NewRequest(http.MethodGet, srv.URL, nil)
	resp, err := rt.RoundTrip(req)
	if err != nil {
		t.Fatalf("RoundTrip: %v", err)
	}
	resp.Body.Close()
	if gotHeader != "secret123" {
		t.Errorf("X-Api-Key = %q, want secret123", gotHeader)
	}
}

func TestAPIKeyTransport_DefaultHeader(t *testing.T) {
	var gotHeader string
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		gotHeader = r.Header.Get("Authorization")
		_, _ = w.Write([]byte("ok"))
	}))
	defer srv.Close()

	// Empty header name should default to "Authorization".
	rt := APIKeyTransport(nil, "", "raw-token")
	req, _ := http.NewRequest(http.MethodGet, srv.URL, nil)
	resp, err := rt.RoundTrip(req)
	if err != nil {
		t.Fatalf("RoundTrip: %v", err)
	}
	resp.Body.Close()
	if gotHeader != "raw-token" {
		t.Errorf("Authorization = %q, want raw-token", gotHeader)
	}
}

func TestBearerTokenTransport_InjectsAuthorization(t *testing.T) {
	var gotHeader string
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		gotHeader = r.Header.Get("Authorization")
		_, _ = w.Write([]byte("ok"))
	}))
	defer srv.Close()

	rt := BearerTokenTransport(nil, "eyJabc.def")
	req, _ := http.NewRequest(http.MethodGet, srv.URL, nil)
	resp, err := rt.RoundTrip(req)
	if err != nil {
		t.Fatalf("RoundTrip: %v", err)
	}
	resp.Body.Close()
	if gotHeader != "Bearer eyJabc.def" {
		t.Errorf("Authorization = %q, want Bearer eyJabc.def", gotHeader)
	}
}

func TestAuthFromNode_NilAuthReturnsNil(t *testing.T) {
	node := &types.Node{Id: "node1", Http: "http://x"}
	rt, err := AuthFromNode(node, func(string) string { return "" })
	if err != nil {
		t.Fatalf("err: %v", err)
	}
	if rt != nil {
		t.Errorf("expected nil RoundTripper for nil Auth, got %T", rt)
	}
}

func TestAuthFromNode_NilNodeReturnsNil(t *testing.T) {
	rt, err := AuthFromNode(nil, func(string) string { return "" })
	if err != nil {
		t.Fatalf("err: %v", err)
	}
	if rt != nil {
		t.Errorf("expected nil RoundTripper for nil node, got %T", rt)
	}
}

func TestAuthFromNode_APIKey(t *testing.T) {
	// types.Auth is map[string]interface{} (no typed union from generator).
	node := &types.Node{
		Id:   "node1",
		Http: "http://x",
		Auth: types.Auth{"type": "api-key", "env": "TEST_KEY"},
	}
	envs := map[string]string{"TEST_KEY": "abc"}
	rt, err := AuthFromNode(node, func(k string) string { return envs[k] })
	if err != nil {
		t.Fatalf("err: %v", err)
	}
	if rt == nil {
		t.Fatal("expected non-nil RoundTripper")
	}

	// Exercise the transport — default header is Authorization.
	var gotHeader string
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		gotHeader = r.Header.Get("Authorization")
		_, _ = w.Write([]byte("ok"))
	}))
	defer srv.Close()
	req, _ := http.NewRequest(http.MethodGet, srv.URL, nil)
	resp, err := rt.RoundTrip(req)
	if err != nil {
		t.Fatal(err)
	}
	resp.Body.Close()
	if gotHeader != "abc" {
		t.Errorf("got header %q, want abc", gotHeader)
	}
}

func TestAuthFromNode_APIKeyCustomHeader(t *testing.T) {
	node := &types.Node{
		Id:   "node1",
		Http: "http://x",
		Auth: types.Auth{"type": "api-key", "env": "TEST_KEY", "header": "X-Api-Key"},
	}
	envs := map[string]string{"TEST_KEY": "abc"}
	rt, err := AuthFromNode(node, func(k string) string { return envs[k] })
	if err != nil {
		t.Fatalf("err: %v", err)
	}
	if rt == nil {
		t.Fatal("expected non-nil RoundTripper")
	}

	var gotHeader string
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		gotHeader = r.Header.Get("X-Api-Key")
		_, _ = w.Write([]byte("ok"))
	}))
	defer srv.Close()
	req, _ := http.NewRequest(http.MethodGet, srv.URL, nil)
	resp, err := rt.RoundTrip(req)
	if err != nil {
		t.Fatal(err)
	}
	resp.Body.Close()
	if gotHeader != "abc" {
		t.Errorf("got X-Api-Key %q, want abc", gotHeader)
	}
}

func TestAuthFromNode_JWT(t *testing.T) {
	node := &types.Node{
		Id:   "node1",
		Http: "http://x",
		Auth: types.Auth{"type": "jwt", "env": "JWT_ENV"},
	}
	envs := map[string]string{"JWT_ENV": "jwt-token"}
	rt, err := AuthFromNode(node, func(k string) string { return envs[k] })
	if err != nil {
		t.Fatalf("err: %v", err)
	}
	if rt == nil {
		t.Fatal("expected non-nil RoundTripper")
	}

	var gotHeader string
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		gotHeader = r.Header.Get("Authorization")
		_, _ = w.Write([]byte("ok"))
	}))
	defer srv.Close()
	req, _ := http.NewRequest(http.MethodGet, srv.URL, nil)
	resp, err := rt.RoundTrip(req)
	if err != nil {
		t.Fatal(err)
	}
	resp.Body.Close()
	if gotHeader != "Bearer jwt-token" {
		t.Errorf("got Authorization %q, want Bearer jwt-token", gotHeader)
	}
}

func TestAuthFromNode_EmptyEnvIsError(t *testing.T) {
	node := &types.Node{
		Id:   "node1",
		Http: "http://x",
		Auth: types.Auth{"type": "api-key", "env": "MISSING_KEY"},
	}
	_, err := AuthFromNode(node, func(string) string { return "" })
	if err == nil {
		t.Fatal("expected error for empty env value")
	}
	// Error must reference the env name (so operators can debug) but must not
	// accidentally echo any value we could mistake for leaked material.
	if !strings.Contains(err.Error(), "MISSING_KEY") {
		t.Errorf("err should reference env name: %v", err)
	}
}

func TestAuthFromNode_MissingEnvField(t *testing.T) {
	node := &types.Node{
		Id:   "node1",
		Http: "http://x",
		Auth: types.Auth{"type": "api-key"},
	}
	_, err := AuthFromNode(node, func(string) string { return "" })
	if err == nil {
		t.Fatal("expected error when 'env' field is missing for api-key")
	}
}

func TestAuthFromNode_MissingTypeField(t *testing.T) {
	node := &types.Node{
		Id:   "node1",
		Http: "http://x",
		Auth: types.Auth{"env": "FOO"},
	}
	_, err := AuthFromNode(node, func(string) string { return "bar" })
	if err == nil {
		t.Fatal("expected error when 'type' field is missing")
	}
}

func TestAuthFromNode_SSHPasswordRejected(t *testing.T) {
	node := &types.Node{
		Id:   "node1",
		Http: "http://x",
		Auth: types.Auth{"type": "ssh-password", "user": "root", "env": "PWD"},
	}
	_, err := AuthFromNode(node, func(string) string { return "x" })
	if err == nil {
		t.Fatal("expected error rejecting ssh-password for RPC client")
	}
}

func TestAuthFromNode_UnknownType(t *testing.T) {
	node := &types.Node{
		Id:   "node1",
		Http: "http://x",
		Auth: types.Auth{"type": "totally-made-up"},
	}
	_, err := AuthFromNode(node, func(string) string { return "x" })
	if err == nil {
		t.Fatal("expected error for unknown auth type")
	}
}
