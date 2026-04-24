// Package remote — auth helpers for injecting API-key / JWT credentials into
// outbound RPC traffic via http.RoundTripper.
//
// The design:
//   - headerTransport is a private RoundTripper that clones requests and sets
//     a single header. APIKeyTransport / BearerTokenTransport are the typed
//     public constructors callers should use.
//   - AuthFromNode bridges types.Node.Auth (go-jsonschema-generated
//     map[string]interface{}) to a RoundTripper. Auth values themselves live
//     in env vars; the Node.Auth map only carries the env-var name plus
//     optional header override. Errors reference env-var names only — never
//     values — so log/stderr output cannot leak credentials.
package remote

import (
	"fmt"
	"net/http"

	"github.com/0xmhha/chainbench/network/internal/types"
)

// headerTransport clones the request and sets a single header before
// delegating to the base transport. Kept unexported so callers go through
// APIKeyTransport / BearerTokenTransport which encode the header convention.
type headerTransport struct {
	base   http.RoundTripper
	header string
	value  string
}

func (t *headerTransport) RoundTrip(req *http.Request) (*http.Response, error) {
	// Per net/http contract a RoundTripper must not mutate the request it
	// receives; clone before setting the header so retries/caching callers
	// see the original request untouched.
	clone := req.Clone(req.Context())
	clone.Header.Set(t.header, t.value)
	base := t.base
	if base == nil {
		base = http.DefaultTransport
	}
	return base.RoundTrip(clone)
}

// APIKeyTransport wraps base to inject "<header>: <value>" on every request.
// An empty header name defaults to "Authorization". Passing nil for base uses
// http.DefaultTransport.
func APIKeyTransport(base http.RoundTripper, header, value string) http.RoundTripper {
	if header == "" {
		header = "Authorization"
	}
	return &headerTransport{base: base, header: header, value: value}
}

// BearerTokenTransport wraps base to inject "Authorization: Bearer <token>".
// Passing nil for base uses http.DefaultTransport.
func BearerTokenTransport(base http.RoundTripper, token string) http.RoundTripper {
	return &headerTransport{base: base, header: "Authorization", value: "Bearer " + token}
}

// AuthFromNode reads node.Auth (a loose map[string]interface{} emitted by
// go-jsonschema for the network.json oneOf) and returns a RoundTripper
// matching the configured type. Returns (nil, nil) when node is nil or
// node.Auth is empty (unauthenticated — existing bare Dial path).
//
// envLookup is injected for testability; production callers pass os.Getenv.
// Auth material never appears in returned errors — only env-var names do.
func AuthFromNode(node *types.Node, envLookup func(string) string) (http.RoundTripper, error) {
	if node == nil || len(node.Auth) == 0 {
		return nil, nil
	}
	rawType, ok := node.Auth["type"].(string)
	if !ok || rawType == "" {
		return nil, fmt.Errorf("remote.AuthFromNode: missing or non-string 'type' field")
	}
	switch rawType {
	case "api-key":
		envName, _ := node.Auth["env"].(string)
		if envName == "" {
			return nil, fmt.Errorf("remote.AuthFromNode(api-key): 'env' field is required")
		}
		value := envLookup(envName)
		if value == "" {
			return nil, fmt.Errorf("remote.AuthFromNode(api-key): env var %q is empty", envName)
		}
		header, _ := node.Auth["header"].(string) // optional; APIKeyTransport defaults to Authorization.
		return APIKeyTransport(nil, header, value), nil
	case "jwt":
		envName, _ := node.Auth["env"].(string)
		if envName == "" {
			return nil, fmt.Errorf("remote.AuthFromNode(jwt): 'env' field is required")
		}
		token := envLookup(envName)
		if token == "" {
			return nil, fmt.Errorf("remote.AuthFromNode(jwt): env var %q is empty", envName)
		}
		return BearerTokenTransport(nil, token), nil
	case "ssh-password":
		// SSH credentials belong to the (future) SSHRemoteDriver, not the RPC client.
		return nil, fmt.Errorf("remote.AuthFromNode: 'ssh-password' auth not applicable to RPC client")
	default:
		return nil, fmt.Errorf("remote.AuthFromNode: unknown auth type %q", rawType)
	}
}
