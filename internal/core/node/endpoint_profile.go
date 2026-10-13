package node

import (
	"fmt"
	"net/url"
	"regexp"
	"strings"
)

// Validate checks profile addresses and secret references without dialing.
func (ep RPCEndpoint) Validate() error {
	if ep.HTTPPort < 0 || ep.HTTPPort > 65535 {
		return fmt.Errorf("endpoint httpPort is outside 0..65535")
	}
	if strings.TrimSpace(ep.RPCURL) == "" {
		return fmt.Errorf("endpoint rpc URL is empty")
	}
	for _, endpoint := range []struct {
		value     string
		websocket bool
	}{{ep.RPCURL, false}, {ep.MetricsURL, false}, {ep.WSURL, true}} {
		if endpoint.value == "" || strings.Contains(endpoint.value, "${") {
			continue
		}
		parsed, err := url.Parse(endpoint.value)
		if err != nil || parsed.Host == "" {
			return fmt.Errorf("endpoint URL needs a host")
		}
		valid := parsed.Scheme == "http" || parsed.Scheme == "https"
		if endpoint.websocket {
			valid = parsed.Scheme == "ws" || parsed.Scheme == "wss"
		}
		if !valid || parsed.User != nil {
			return fmt.Errorf("endpoint URL has unsupported scheme or embedded credentials")
		}
	}
	if ep.Auth != nil {
		if ep.Auth.Type != "api-key" && ep.Auth.Type != "jwt" {
			return fmt.Errorf("endpoint auth must be HTTP api-key or jwt")
		}
		if !regexp.MustCompile(`^[A-Za-z_][A-Za-z0-9_]*$`).MatchString(ep.Auth.Env) {
			return fmt.Errorf("endpoint auth env must name an environment variable")
		}
		if ep.Auth.Header != "" && (!regexp.MustCompile("^[!#$%&'*+.^_`|~0-9A-Za-z-]+$").MatchString(ep.Auth.Header) || ep.Auth.Type == "jwt") {
			return fmt.Errorf("endpoint auth header is invalid or ignored by jwt")
		}
	}
	for _, api := range ep.APIs {
		if strings.TrimSpace(api) == "" {
			return fmt.Errorf("endpoint api name is empty")
		}
	}
	return nil
}
