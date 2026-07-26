// Package remote gives the Go core access to remote chain RPC endpoints: HTTP
// auth transports (API key / JWT) and an SSH tunnel, both producing an
// *http.Client the core rpc client dials through (rpc.DialWithClient). It is the
// absorption of the legacy network/ module's remote + sshremote drivers into the
// core, so remote/attach flows no longer route through the separate wire module.
//
// Auth material never appears in returned errors — only env-var names and hosts
// do — so log/stderr output cannot leak credentials.
package remote

import (
	"fmt"
	"net/http"
)

// Auth is a node's authentication descriptor: a loose map carrying the auth
// "type" plus the env-var name that holds the secret (never the secret itself).
// e.g. {"type":"api-key","env":"MY_KEY","header":"X-API-Key"}.
type Auth map[string]any

// headerTransport clones the request and sets a single header before delegating
// to the base transport. Unexported so callers go through the typed
// constructors which encode the header convention.
type headerTransport struct {
	base   http.RoundTripper
	header string
	value  string
}

func (t *headerTransport) RoundTrip(req *http.Request) (*http.Response, error) {
	// net/http requires a RoundTripper not mutate the request it receives; clone
	// before setting the header so retry/cache callers see the original.
	clone := req.Clone(req.Context())
	clone.Header.Set(t.header, t.value)
	base := t.base
	if base == nil {
		base = http.DefaultTransport
	}
	return base.RoundTrip(clone)
}

// APIKeyTransport wraps base to inject "<header>: <value>" on every request. An
// empty header name defaults to "Authorization"; nil base uses DefaultTransport.
func APIKeyTransport(base http.RoundTripper, header, value string) http.RoundTripper {
	if header == "" {
		header = "Authorization"
	}
	return &headerTransport{base: base, header: header, value: value}
}

// BearerTokenTransport wraps base to inject "Authorization: Bearer <token>". Nil
// base uses DefaultTransport.
func BearerTokenTransport(base http.RoundTripper, token string) http.RoundTripper {
	return &headerTransport{base: base, header: "Authorization", value: "Bearer " + token}
}

// ValidateAuth reports whether auth is structurally valid. Empty auth is valid
// (unauthenticated). Callers at input boundaries use this to fail fast on
// malformed configuration before persistence.
func ValidateAuth(auth Auth) error {
	if len(auth) == 0 {
		return nil
	}
	rawType, ok := auth["type"].(string)
	if !ok || rawType == "" {
		return fmt.Errorf("auth: missing or non-string 'type' field")
	}
	switch rawType {
	case "api-key", "jwt":
		if envName, _ := auth["env"].(string); envName == "" {
			return fmt.Errorf("auth(%s): 'env' field is required", rawType)
		}
	case "ssh-password":
		// Consumed by the SSH tunnel, not the HTTP RPC client. Validate the
		// structural fields here so a malformed ssh node is rejected early.
		for _, f := range []string{"user", "host", "env"} {
			if s, _ := auth[f].(string); s == "" {
				return fmt.Errorf("auth(ssh-password): %q field is required", f)
			}
		}
	default:
		return fmt.Errorf("auth: unknown type %q", rawType)
	}
	return nil
}

// TransportFromAuth returns the HTTP RoundTripper for an auth descriptor, or
// (nil, nil) when auth is empty (unauthenticated). envLookup is injected for
// testability (production passes os.Getenv). Auth material never appears in
// errors — only env-var names do.
func TransportFromAuth(auth Auth, envLookup func(string) string) (http.RoundTripper, error) {
	if len(auth) == 0 {
		return nil, nil
	}
	rawType, ok := auth["type"].(string)
	if !ok || rawType == "" {
		return nil, fmt.Errorf("remote: missing or non-string auth 'type' field")
	}
	switch rawType {
	case "api-key":
		envName, _ := auth["env"].(string)
		if envName == "" {
			return nil, fmt.Errorf("remote(api-key): 'env' field is required")
		}
		value := envLookup(envName)
		if value == "" {
			return nil, fmt.Errorf("remote(api-key): env var %q is empty", envName)
		}
		header, _ := auth["header"].(string) // optional; defaults to Authorization.
		return APIKeyTransport(nil, header, value), nil
	case "jwt":
		envName, _ := auth["env"].(string)
		if envName == "" {
			return nil, fmt.Errorf("remote(jwt): 'env' field is required")
		}
		token := envLookup(envName)
		if token == "" {
			return nil, fmt.Errorf("remote(jwt): env var %q is empty", envName)
		}
		return BearerTokenTransport(nil, token), nil
	case "ssh-password":
		return nil, fmt.Errorf("remote: 'ssh-password' auth is for the SSH tunnel, not the HTTP client")
	default:
		return nil, fmt.Errorf("remote: unknown auth type %q", rawType)
	}
}

// HTTPClientFromAuth builds an *http.Client whose transport applies the auth
// descriptor, ready for rpc.DialWithClient. Empty auth yields the default
// client. It is the convenience the core attach/remote flows use.
func HTTPClientFromAuth(auth Auth, envLookup func(string) string) (*http.Client, error) {
	rt, err := TransportFromAuth(auth, envLookup)
	if err != nil {
		return nil, err
	}
	if rt == nil {
		return http.DefaultClient, nil
	}
	return &http.Client{Transport: rt}, nil
}
