package deploy

import (
	"github.com/0xmhha/chainbench/internal/core/remote"
	"os"
	"path/filepath"
	"testing"
)

func TestCredentials_For(t *testing.T) {
	cr := &Credentials{
		User:     "ubuntu",
		Password: "globalpw",
		Overrides: map[int]CredsOverride{
			3: {User: "adminuser"},
		},
	}
	c := &Cluster{SSHPort: 10022, Servers: []Server{
		{Index: 1, Host: "10.0.0.1", Role: RoleWbftBP},
		{Index: 3, Host: "10.0.0.3", Role: RoleWbftBP},
	}}
	noEnv := func(string) string { return "" }

	// global user/password + cluster SSH port.
	rc, err := cr.For(c, c.Servers[0], noEnv)
	if err != nil {
		t.Fatal(err)
	}
	if rc.User != "ubuntu" || rc.Password != "globalpw" || rc.Host != "10.0.0.1" || rc.Port != 10022 {
		t.Errorf("server1 creds = %+v", rc)
	}
	// per-server user override, global password.
	rc3, _ := cr.For(c, c.Servers[1], noEnv)
	if rc3.User != "adminuser" || rc3.Password != "globalpw" {
		t.Errorf("server3 creds = %+v", rc3)
	}
	// With no credentials file, the environment is the source.
	env := map[string]string{remote.EnvUser: "envuser", remote.EnvPass: "envpw"}
	rcEnv, _ := cr.For(c, c.Servers[0], func(k string) string { return env[k] })
	if rcEnv.User != "envuser" || rcEnv.Password != "envpw" {
		t.Errorf("env creds = %+v", rcEnv)
	}
}

// TestCredentials_FileIsTheSingleSource pins the same rule the server set
// enforces: once a credentials file is loaded, a leftover CHAINBENCH_REMOTE_*
// export must not silently redirect a login.
func TestCredentials_FileIsTheSingleSource(t *testing.T) {
	p := filepath.Join(t.TempDir(), "credentials")
	if err := os.WriteFile(p, []byte("user: ubuntu\npassword: filepw\n"), 0o600); err != nil {
		t.Fatal(err)
	}
	cr, err := LoadCredentials(p)
	if err != nil {
		t.Fatal(err)
	}
	c := &Cluster{SSHPort: 22, Servers: []Server{{Index: 1, Host: "10.0.0.1", Role: RoleWbftBP}}}
	env := map[string]string{remote.EnvUser: "envuser", remote.EnvPass: "envpw"}
	rc, err := cr.For(c, c.Servers[0], func(k string) string { return env[k] })
	if err != nil {
		t.Fatal(err)
	}
	if rc.User != "ubuntu" || rc.Password != "filepw" {
		t.Errorf("the environment redirected a login the file defines: %+v", rc)
	}
}

func TestCredentials_MissingAuth(t *testing.T) {
	c := &Cluster{Servers: []Server{{Index: 1, Host: "h", Role: RoleEndpoint}}}
	noEnv := func(string) string { return "" }
	if _, err := (&Credentials{Password: "p"}).For(c, c.Servers[0], noEnv); err == nil {
		t.Error("expected error for missing user")
	}
	if _, err := (&Credentials{User: "u"}).For(c, c.Servers[0], noEnv); err == nil {
		t.Error("expected error for missing password")
	}
}
