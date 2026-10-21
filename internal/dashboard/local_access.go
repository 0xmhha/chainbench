package dashboard

import (
	"net"
	"net/http"
	"net/netip"
	"strings"
)

// proxyForwardHeaders are the headers a reverse proxy adds. A browser on this
// machine talking to the service directly sends none of them.
var proxyForwardHeaders = []string{"Forwarded", "X-Forwarded-For", "X-Forwarded-Host", "X-Forwarded-Proto", "X-Real-IP"}

// localBrowserRequest also rejects DNS rebinding: a hostile domain resolving
// to loopback must not be able to provision its own administrator session.
// It rejects proxied requests too, because a same-host proxy makes a remote
// browser's peer and Host both look like loopback.
func localBrowserRequest(r *http.Request) bool {
	for _, header := range proxyForwardHeaders {
		if r.Header.Get(header) != "" {
			return false
		}
	}
	peer, _, err := net.SplitHostPort(r.RemoteAddr)
	if err != nil {
		return false
	}
	ip, err := netip.ParseAddr(peer)
	if err != nil || !ip.Unmap().IsLoopback() {
		return false
	}
	host := r.Host
	if parsed, _, err := net.SplitHostPort(host); err == nil {
		host = parsed
	}
	if strings.EqualFold(host, "localhost") {
		return true
	}
	ip, err = netip.ParseAddr(strings.Trim(host, "[]"))
	return err == nil && ip.Unmap().IsLoopback()
}
