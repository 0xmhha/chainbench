package remote

import (
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
)

func TestAPIKeyTransport_InjectsHeader(t *testing.T) {
	var got string
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		got = r.Header.Get("X-API-Key")
	}))
	defer srv.Close()
	c := &http.Client{Transport: APIKeyTransport(nil, "X-API-Key", "secret")}
	resp, err := c.Get(srv.URL)
	if err != nil {
		t.Fatal(err)
	}
	resp.Body.Close()
	if got != "secret" {
		t.Errorf("header = %q, want secret", got)
	}
}

func TestBearerTokenTransport_InjectsAuthorization(t *testing.T) {
	var got string
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		got = r.Header.Get("Authorization")
	}))
	defer srv.Close()
	c := &http.Client{Transport: BearerTokenTransport(nil, "tok")}
	resp, err := c.Get(srv.URL)
	if err != nil {
		t.Fatal(err)
	}
	resp.Body.Close()
	if got != "Bearer tok" {
		t.Errorf("authorization = %q, want 'Bearer tok'", got)
	}
}

func TestTransportFromAuth(t *testing.T) {
	env := func(k string) string { return map[string]string{"KEY": "v", "TOK": "jwt"}[k] }

	// empty auth -> no transport
	if rt, err := TransportFromAuth(nil, env); rt != nil || err != nil {
		t.Errorf("empty auth: %v %v", rt, err)
	}
	// api-key with custom header
	rt, err := TransportFromAuth(Auth{"type": "api-key", "env": "KEY", "header": "X-K"}, env)
	if err != nil || rt == nil {
		t.Fatalf("api-key: %v", err)
	}
	// jwt
	if _, err := TransportFromAuth(Auth{"type": "jwt", "env": "TOK"}, env); err != nil {
		t.Errorf("jwt: %v", err)
	}
}

func TestTransportFromAuth_Errors(t *testing.T) {
	empty := func(string) string { return "" }
	cases := map[string]Auth{
		"missing type":        {"env": "X"},
		"unknown type":        {"type": "weird"},
		"api-key missing env": {"type": "api-key"},
		"ssh not for http":    {"type": "ssh-password", "user": "u", "host": "h", "env": "P"},
	}
	for name, a := range cases {
		if _, err := TransportFromAuth(a, empty); err == nil {
			t.Errorf("%s: expected error", name)
		}
	}
	// empty env value -> error naming the env var, never the value
	_, err := TransportFromAuth(Auth{"type": "api-key", "env": "MISSING"}, empty)
	if err == nil || !strings.Contains(err.Error(), "MISSING") {
		t.Errorf("empty env should error referencing env name: %v", err)
	}
}

func TestValidateAuth(t *testing.T) {
	ok := []Auth{
		nil,
		{"type": "api-key", "env": "K"},
		{"type": "jwt", "env": "T"},
		{"type": "ssh-password", "user": "u", "host": "h", "env": "P"},
	}
	for _, a := range ok {
		if err := ValidateAuth(a); err != nil {
			t.Errorf("valid auth %v rejected: %v", a, err)
		}
	}
	bad := []Auth{
		{"env": "K"},                          // missing type
		{"type": "weird"},                     // unknown
		{"type": "api-key"},                   // missing env
		{"type": "ssh-password", "user": "u"}, // missing host/env
	}
	for _, a := range bad {
		if err := ValidateAuth(a); err == nil {
			t.Errorf("invalid auth %v accepted", a)
		}
	}
}

func TestHTTPClientFromAuth(t *testing.T) {
	// empty -> default client
	c, err := HTTPClientFromAuth(nil, func(string) string { return "" })
	if err != nil || c != http.DefaultClient {
		t.Errorf("empty auth should yield default client: %v %v", c, err)
	}
	// api-key -> custom client
	c, err = HTTPClientFromAuth(Auth{"type": "api-key", "env": "K"}, func(string) string { return "v" })
	if err != nil || c == http.DefaultClient || c.Transport == nil {
		t.Errorf("api-key should yield a custom client: %v %v", c, err)
	}
}
