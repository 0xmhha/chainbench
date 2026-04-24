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

// JWT parallel of TestAuthFromNode_EmptyEnvIsError. Closes the coverage
// asymmetry between api-key and jwt branches so a future refactor can't
// silently break one while the other's test stays green.
func TestAuthFromNode_JWT_EmptyEnvIsError(t *testing.T) {
	node := &types.Node{
		Id:   "node1",
		Http: "http://x",
		Auth: types.Auth{"type": "jwt", "env": "MISSING_JWT"},
	}
	_, err := AuthFromNode(node, func(string) string { return "" })
	if err == nil {
		t.Fatal("expected error for empty jwt env value")
	}
	if !strings.Contains(err.Error(), "MISSING_JWT") {
		t.Errorf("err should reference env name: %v", err)
	}
}

// JWT parallel of TestAuthFromNode_MissingEnvField.
func TestAuthFromNode_JWT_MissingEnvField(t *testing.T) {
	node := &types.Node{
		Id:   "node1",
		Http: "http://x",
		Auth: types.Auth{"type": "jwt"}, // no env
	}
	_, err := AuthFromNode(node, func(string) string { return "tok" })
	if err == nil {
		t.Fatal("expected error when 'env' field is missing for jwt")
	}
}

func TestValidateAuth_NilIsOK(t *testing.T) {
	if err := ValidateAuth(nil); err != nil {
		t.Errorf("nil auth should be valid: %v", err)
	}
	if err := ValidateAuth(types.Auth{}); err != nil {
		t.Errorf("empty auth should be valid: %v", err)
	}
}

func TestValidateAuth_ValidAPIKey(t *testing.T) {
	if err := ValidateAuth(types.Auth{"type": "api-key", "env": "KEY"}); err != nil {
		t.Errorf("valid api-key should pass: %v", err)
	}
}

func TestValidateAuth_ValidJWT(t *testing.T) {
	if err := ValidateAuth(types.Auth{"type": "jwt", "env": "TOK"}); err != nil {
		t.Errorf("valid jwt should pass: %v", err)
	}
}

func TestValidateAuth_SSHPasswordPasses(t *testing.T) {
	// ssh-password is persisted but ignored by RPC client; attach accepts it.
	if err := ValidateAuth(types.Auth{"type": "ssh-password", "user": "root", "host": "h"}); err != nil {
		t.Errorf("ssh-password should pass attach validation: %v", err)
	}
}

func TestValidateAuth_UnknownType(t *testing.T) {
	if err := ValidateAuth(types.Auth{"type": "totally-made-up"}); err == nil {
		t.Error("unknown type should fail")
	}
}

func TestValidateAuth_MissingType(t *testing.T) {
	if err := ValidateAuth(types.Auth{"env": "KEY"}); err == nil {
		t.Error("missing type should fail")
	}
}

func TestValidateAuth_APIKey_MissingEnv(t *testing.T) {
	if err := ValidateAuth(types.Auth{"type": "api-key"}); err == nil {
		t.Error("api-key without env should fail")
	}
}

func TestValidateAuth_JWT_MissingEnv(t *testing.T) {
	if err := ValidateAuth(types.Auth{"type": "jwt"}); err == nil {
		t.Error("jwt without env should fail")
	}
}
