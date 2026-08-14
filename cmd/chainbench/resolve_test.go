package main

import "testing"

func TestRemoteDriver_EmptyHostIsLocal(t *testing.T) {
	d, sink, err := remoteDriver("", "", 0)
	if err != nil || d != nil || sink != nil {
		t.Errorf("empty host should yield (nil, nil, nil), got (%v, %v, %v)", d, sink, err)
	}
}

func TestRemoteDriver_PasswordRequiredFromEnv(t *testing.T) {
	t.Setenv("CHAINBENCH_REMOTE_PASS", "")
	if _, _, err := remoteDriver("10.0.0.5", "cb", 22); err == nil {
		t.Error("missing CHAINBENCH_REMOTE_PASS should error (password must not be a flag)")
	}
}

func TestRemoteDriver_BuildsFromEnv(t *testing.T) {
	t.Setenv("CHAINBENCH_REMOTE_PASS", "secret")
	t.Setenv("CHAINBENCH_SSH_INSECURE_HOST_KEY", "1") // avoid known_hosts in the test env
	d, sink, err := remoteDriver("10.0.0.5", "cb", 2222)
	if err != nil || d == nil || sink == nil {
		t.Fatalf("remoteDriver should build a driver + sink from env: (%v, %v, %v)", d, sink, err)
	}
}
