package dashboard

import (
	"net"
	"net/http"
	"net/netip"
	"strings"
)

// localBrowserRequest also rejects DNS rebinding: a hostile domain resolving
// to loopback must not be able to provision its own administrator session.
func localBrowserRequest(r *http.Request) bool {
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
