package remote

import (
	"errors"
	"net/http"
)

// The lease is checked for each request, including an existing keepalive
// connection. Checking TCP dials alone would leave reused RPC channels open.
type guardedTunnelTransport struct {
	transport *http.Transport
	authorize func() error
}

func (t guardedTunnelTransport) RoundTrip(r *http.Request) (*http.Response, error) {
	if err := r.Context().Err(); err != nil {
		return nil, err
	}
	if t.authorize() != nil {
		return nil, errors.New("remote: access authorization denied")
	}
	return t.transport.RoundTrip(r)
}

func (t guardedTunnelTransport) CloseIdleConnections() { t.transport.CloseIdleConnections() }
