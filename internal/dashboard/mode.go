package dashboard

import (
	"errors"
	"fmt"
	"net"
	"net/http"
	"net/netip"
	"strings"
)

// ValidateServiceMode refuses a listener the account mode does not allow. A
// personal service answers only on loopback. A team service reachable from
// other machines must terminate TLS itself or sit behind a TLS terminator
// it trusts; on loopback, traffic never leaves the machine.
func ValidateServiceMode(mode, addr string, tls bool, proxies []netip.Prefix) error {
	host, _, err := net.SplitHostPort(addr)
	if err != nil {
		return fmt.Errorf("listen address %q: %w", addr, err)
	}
	ip, err := netip.ParseAddr(strings.Trim(host, "[]"))
	loopback := err == nil && ip.IsLoopback()
	switch mode {
	case "personal":
		if !loopback {
			return errors.New("personal mode listens only on a loopback address; use team mode to serve other machines")
		}
	case "team":
		if !loopback && !tls && len(proxies) == 0 {
			return errors.New("team mode on a reachable address needs -tls-cert/-tls-key or -trusted-proxy naming the TLS terminator")
		}
	default:
		return fmt.Errorf("unknown account mode %q (personal or team)", mode)
	}
	return nil
}

// WithTrustedProxies names the TLS terminators whose X-Forwarded-Proto the
// server believes. A header from any other peer is ignored.
func WithTrustedProxies(proxies []netip.Prefix) Option {
	return func(s *Server) { s.trustedProxies = proxies }
}

// requestScheme is the scheme the browser used: https when this server
// terminates TLS, or when a trusted terminator says it did.
func (s *Server) requestScheme(r *http.Request) string {
	if r.TLS != nil {
		return "https"
	}
	host, _, err := net.SplitHostPort(r.RemoteAddr)
	if err != nil {
		return "http"
	}
	peer, err := netip.ParseAddr(host)
	if err != nil {
		return "http"
	}
	for _, p := range s.trustedProxies {
		if p.Contains(peer.Unmap()) {
			if strings.EqualFold(strings.TrimSpace(strings.Split(r.Header.Get("X-Forwarded-Proto"), ",")[0]), "https") {
				return "https"
			}
			return "http"
		}
	}
	return "http"
}
