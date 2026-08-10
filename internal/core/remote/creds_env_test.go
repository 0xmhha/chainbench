package remote_test

import (
	"testing"

	"github.com/0xmhha/chainbench/internal/core/remote"
)

func TestCredentialsFromEnv(t *testing.T) {
	// user from env overrides, password auth.
	env := map[string]string{"CHAINBENCH_REMOTE_USER": "envuser", "CHAINBENCH_REMOTE_PASS": "pw"}
	c, err := remote.CredentialsFromEnv("flaguser", "10.0.0.1", 2222, func(k string) string { return env[k] })
	if err != nil {
		t.Fatalf("creds: %v", err)
	}
	if c.User != "envuser" || c.Host != "10.0.0.1" || c.Port != 2222 || c.Password != "pw" {
		t.Fatalf("creds = %+v", c)
	}
	// no user -> error
	if _, err := remote.CredentialsFromEnv("", "h", 0, func(string) string { return "" }); err == nil {
		t.Fatal("expected error for no user")
	}
	// user but no auth -> error
	if _, err := remote.CredentialsFromEnv("u", "h", 0, func(string) string { return "" }); err == nil {
		t.Fatal("expected error for no auth")
	}
}
