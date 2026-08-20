package main

import "testing"

func TestRemoteDriver_EmptyHostIsLocal(t *testing.T) {
	d, err := remoteDriver("", "", 0)
	if err != nil || d != nil {
		t.Errorf("empty host should yield (nil, nil), got (%v, %v)", d, err)
	}
}

func TestRemoteDriver_PasswordRequiredFromEnv(t *testing.T) {
	t.Setenv("CHAINBENCH_REMOTE_PASS", "")
	if _, err := remoteDriver("10.0.0.5", "cb", 22); err == nil {
		t.Error("missing CHAINBENCH_REMOTE_PASS should error (password must not be a flag)")
	}
}

func TestRemoteDriver_BuildsFromEnv(t *testing.T) {
	t.Setenv("CHAINBENCH_REMOTE_PASS", "secret")
	t.Setenv("CHAINBENCH_SSH_INSECURE_HOST_KEY", "1") // avoid known_hosts in the test env
	d, err := remoteDriver("10.0.0.5", "cb", 2222)
	if err != nil || d == nil {
		t.Fatalf("remoteDriver should build from env: (%v, %v)", d, err)
	}
}
