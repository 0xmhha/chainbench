package deploy

import (
	"testing"
)

// TestSample_Loads guards the committed cluster.yaml.sample: it must parse and
// validate, so a copy is a working starting point.
func TestSample_Loads(t *testing.T) {
	c, err := LoadCluster("cluster.yaml.sample")
	if err != nil {
		t.Fatalf("load sample: %v", err)
	}
	if c.CroissantBlock != 100 || c.RPCPort != 8601 {
		t.Errorf("sample params: croissant=%d rpc=%d", c.CroissantBlock, c.RPCPort)
	}
	if len(c.Producers()) != 2 || len(c.Validators()) != 5 || len(c.Endpoints()) != 1 || len(c.Bootnodes()) != 1 {
		t.Errorf("sample roles: prod=%d val=%d en=%d pn=%d",
			len(c.Producers()), len(c.Validators()), len(c.Endpoints()), len(c.Bootnodes()))
	}
}

const twoServer = `
rpc_port: 9000
ws_port: 9001
ssh_port: 2222
wemix_binary: /r/gwemix3
wbft_binary: /r/gwemix4
servers:
  - index: 1
    host: 10.0.0.1
    role: wemix_bp
  - index: 2
    host: 10.0.0.2
    role: en
    ssh_port: 2200
    binary: /r/custom
    sync_mode: snap
`

func TestParse_Resolution(t *testing.T) {
	c, err := ParseCluster([]byte(twoServer))
	if err != nil {
		t.Fatalf("parse: %v", err)
	}
	p, _ := c.ServerByIndex(1)
	e, _ := c.ServerByIndex(2)

	// SSH port: cluster default vs server override.
	if got := c.SSHPortFor(p); got != 2222 {
		t.Errorf("producer ssh port = %d, want 2222 (cluster default)", got)
	}
	if got := c.SSHPortFor(e); got != 2200 {
		t.Errorf("endpoint ssh port = %d, want 2200 (override)", got)
	}
	// Binary: role default vs override.
	if got := c.BinaryFor(p); got != "/r/gwemix3" {
		t.Errorf("producer binary = %q, want wemix_binary", got)
	}
	if got := c.BinaryFor(e); got != "/r/custom" {
		t.Errorf("endpoint binary = %q, want override", got)
	}
	// Sync mode default vs override.
	if got := c.SyncModeFor(p); got != "full" {
		t.Errorf("producer sync = %q, want full", got)
	}
	if got := c.SyncModeFor(e); got != "snap" {
		t.Errorf("endpoint sync = %q, want snap", got)
	}
	// URLs.
	if got := c.RPCURL(p); got != "http://10.0.0.1:9000" {
		t.Errorf("rpc url = %q", got)
	}
	if got := c.WSURL(e); got != "ws://10.0.0.2:9001" {
		t.Errorf("ws url = %q", got)
	}
}

func TestLaunchOrder_EndpointsFirst(t *testing.T) {
	c, err := ParseCluster([]byte(twoServer))
	if err != nil {
		t.Fatal(err)
	}
	order := c.LaunchOrder()
	if len(order) != 2 || order[0].Role != RoleEndpoint || order[1].Role != RoleWemixBP {
		t.Errorf("launch order = %+v, want endpoint before producer", order)
	}
}

func TestSingleServer(t *testing.T) {
	c, err := ParseCluster([]byte(`
rpc_port: 8601
servers:
  - index: 1
    host: 127.0.0.1
    role: wbft_bp
`))
	if err != nil {
		t.Fatalf("single-server parse: %v", err)
	}
	if len(c.Servers) != 1 || len(c.Validators()) != 1 {
		t.Errorf("single-server: %d servers", len(c.Servers))
	}
	if p := c.SSHPortFor(c.Servers[0]); p != 22 {
		t.Errorf("default ssh port = %d, want 22", p)
	}
}

func TestValidate_Rejects(t *testing.T) {
	cases := map[string]string{
		"no servers":      "rpc_port: 8601\nservers: []\n",
		"no rpc_port":     "servers:\n  - index: 1\n    host: h\n    role: en\n",
		"missing host":    "rpc_port: 8601\nservers:\n  - index: 1\n    role: en\n",
		"invalid role":    "rpc_port: 8601\nservers:\n  - index: 1\n    host: h\n    role: bogus\n",
		"duplicate index": "rpc_port: 8601\nservers:\n  - index: 1\n    host: a\n    role: en\n  - index: 1\n    host: b\n    role: pn\n",
		"bad index":       "rpc_port: 8601\nservers:\n  - index: 0\n    host: h\n    role: en\n",
	}
	for name, yml := range cases {
		if _, err := ParseCluster([]byte(yml)); err == nil {
			t.Errorf("%s: expected error, got nil", name)
		}
	}
}

func TestVariableCount(t *testing.T) {
	// Build an N-server config and confirm it loads and counts correctly.
	yml := "rpc_port: 8601\nservers:\n"
	for i := 1; i <= 12; i++ {
		role := "wbft_bp"
		if i == 1 {
			role = "pn"
		}
		yml += "  - index: " + itoa(i) + "\n    host: 10.0.0." + itoa(i) + "\n    role: " + role + "\n"
	}
	c, err := ParseCluster([]byte(yml))
	if err != nil {
		t.Fatalf("N-server parse: %v", err)
	}
	if len(c.Servers) != 12 || len(c.Validators()) != 11 || len(c.Bootnodes()) != 1 {
		t.Errorf("N-server counts: servers=%d val=%d pn=%d", len(c.Servers), len(c.Validators()), len(c.Bootnodes()))
	}
}

func itoa(n int) string {
	if n == 0 {
		return "0"
	}
	var b []byte
	for n > 0 {
		b = append([]byte{byte('0' + n%10)}, b...)
		n /= 10
	}
	return string(b)
}
