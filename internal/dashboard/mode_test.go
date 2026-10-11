package dashboard

import (
	"encoding/json"
	"net/http/httptest"
	"net/netip"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/0xmhha/chainbench/internal/app"
	"github.com/0xmhha/chainbench/internal/core/collector"
)

func TestServiceModesRefuseUnsafeListeners(t *testing.T) {
	proxy := []netip.Prefix{netip.MustParsePrefix("10.0.0.5/32")}
	for _, c := range []struct {
		mode, addr string
		tls        bool
		proxies    []netip.Prefix
		ok         bool
	}{
		{"personal", "127.0.0.1:8787", false, nil, true},
		{"personal", "[::1]:8787", false, nil, true},
		{"personal", "0.0.0.0:8787", false, nil, false},
		{"personal", "192.168.1.10:8787", true, nil, false},
		{"personal", ":8787", false, nil, false},
		{"team", "127.0.0.1:8787", false, nil, true},
		{"team", "0.0.0.0:8443", true, nil, true},
		{"team", "0.0.0.0:8787", false, proxy, true},
		{"team", "0.0.0.0:8787", false, nil, false},
		{"team", ":8787", false, nil, false},
		{"other", "127.0.0.1:8787", false, nil, false},
	} {
		err := ValidateServiceMode(c.mode, c.addr, c.tls, c.proxies)
		if (err == nil) != c.ok {
			t.Errorf("%s %s tls=%v proxies=%v: %v", c.mode, c.addr, c.tls, c.proxies, err)
		}
	}
}

func TestTrustedProxyLetsHTTPSOriginsWriteAndMarksCookiesSecure(t *testing.T) {
	root := t.TempDir()
	auth, err := app.OpenWebAuth(root, "team")
	if err != nil {
		t.Fatal(err)
	}
	store, err := app.OpenDeploymentStore(root)
	if err != nil {
		t.Fatal(err)
	}
	bus := collector.NewBus()
	defer bus.Close()
	s := NewServer(bus, nil, WithDeployments(store, WebAuthenticator(auth)), WithWebSecurity(auth, store), WithTrustedProxies([]netip.Prefix{netip.MustParsePrefix("10.0.0.5/32")}))
	setup, _ := os.ReadFile(filepath.Join(root, "setup.token"))
	body, _ := json.Marshal(map[string]string{"username": "admin", "password": "long-admin-password", "setupToken": string(setup)})
	call := func(peer, proto string) *httptest.ResponseRecorder {
		r := httptest.NewRequest("POST", "http://chainbench.example/api/v1/bootstrap", strings.NewReader(string(body)))
		r.RemoteAddr = peer
		r.Header.Set("Origin", "https://chainbench.example")
		if proto != "" {
			r.Header.Set("X-Forwarded-Proto", proto)
		}
		w := httptest.NewRecorder()
		s.ServeHTTP(w, r)
		return w
	}
	if w := call("192.0.2.9:4000", "https"); w.Code != 403 {
		t.Fatalf("an untrusted peer's forwarded scheme was believed: %d", w.Code)
	}
	w := call("10.0.0.5:4000", "https")
	if w.Code != 201 {
		t.Fatalf("a trusted TLS terminator's write was refused: %d %s", w.Code, w.Body)
	}
	if c := w.Result().Cookies(); len(c) == 0 || !c[0].Secure {
		t.Fatalf("session cookie behind TLS is not Secure: %+v", c)
	}
}
