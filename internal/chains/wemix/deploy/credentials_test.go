package deploy

import "testing"

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
	// env overrides win.
	env := map[string]string{"CHAINBENCH_REMOTE_USER": "envuser", "CHAINBENCH_REMOTE_PASS": "envpw"}
	rcEnv, _ := cr.For(c, c.Servers[0], func(k string) string { return env[k] })
	if rcEnv.User != "envuser" || rcEnv.Password != "envpw" {
		t.Errorf("env creds = %+v", rcEnv)
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
