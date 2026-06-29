package main

import (
	"context"
	"errors"
	"strings"
	"testing"

	"github.com/0xmhha/chainbench/network/internal/types"
)

// dialNode's ssh-remote branch error paths. The happy end-to-end (SSH tunnel →
// RPC) is covered at the driver level in the sshremote package test, which
// stands up an in-process SSH server; here we pin the boundary classification
// and the redaction contract (the password value never appears in an error).

func apiCode(t *testing.T, err error) string {
	t.Helper()
	var api *APIError
	if !errors.As(err, &api) {
		t.Fatalf("error is not *APIError: %v", err)
	}
	return string(api.Code)
}

func TestDialNode_SSHRemote_WrongAuthType(t *testing.T) {
	node := &types.Node{
		Id: "n", Provider: types.NodeProviderSshRemote, Http: "http://127.0.0.1:8545",
		Auth: types.Auth{"type": "api-key", "env": "X"},
	}
	_, err := dialNode(context.Background(), node)
	if got := apiCode(t, err); got != "INVALID_ARGS" {
		t.Errorf("code = %s, want INVALID_ARGS", got)
	}
}

func TestDialNode_SSHRemote_MissingFields(t *testing.T) {
	for _, missing := range []string{"user", "host", "env"} {
		auth := types.Auth{"type": "ssh-password", "user": "root", "host": "h", "env": "PW"}
		delete(auth, missing)
		node := &types.Node{
			Id: "n", Provider: types.NodeProviderSshRemote, Http: "http://127.0.0.1:8545", Auth: auth,
		}
		_, err := dialNode(context.Background(), node)
		if got := apiCode(t, err); got != "INVALID_ARGS" {
			t.Errorf("missing %q: code = %s, want INVALID_ARGS", missing, got)
		}
	}
}

func TestDialNode_SSHRemote_EnvUnset(t *testing.T) {
	node := &types.Node{
		Id: "n", Provider: types.NodeProviderSshRemote, Http: "http://127.0.0.1:8545",
		Auth: types.Auth{"type": "ssh-password", "user": "root", "host": "h", "env": "CHAINBENCH_TEST_SSH_PW_UNSET"},
	}
	_, err := dialNode(context.Background(), node)
	if got := apiCode(t, err); got != "UPSTREAM_ERROR" {
		t.Errorf("code = %s, want UPSTREAM_ERROR", got)
	}
	if !strings.Contains(err.Error(), "CHAINBENCH_TEST_SSH_PW_UNSET") {
		t.Errorf("error should name the env var: %v", err)
	}
}

// Redaction: when the SSH dial fails, the password value must not leak into the
// returned error. Use insecure host-key mode + an unroutable host so the dial
// fails fast without needing a server.
func TestDialNode_SSHRemote_PasswordNotLeakedOnDialFailure(t *testing.T) {
	const secret = "s3cr3t-ssh-password-sentinel"
	t.Setenv("CHAINBENCH_SSH_INSECURE_HOST_KEY", "1")
	t.Setenv("CHAINBENCH_TEST_SSH_PW", secret)
	node := &types.Node{
		Id: "n", Provider: types.NodeProviderSshRemote, Http: "http://127.0.0.1:8545",
		// 127.0.0.1:1 refuses quickly; no SSH server needed.
		Auth: types.Auth{"type": "ssh-password", "user": "root", "host": "127.0.0.1", "port": float64(1), "env": "CHAINBENCH_TEST_SSH_PW"},
	}
	_, err := dialNode(context.Background(), node)
	if err == nil {
		t.Fatal("expected dial failure")
	}
	if strings.Contains(err.Error(), secret) {
		t.Fatalf("password leaked into error: %v", err)
	}
	if got := apiCode(t, err); got != "UPSTREAM_ERROR" {
		t.Errorf("code = %s, want UPSTREAM_ERROR", got)
	}
}

func TestDialNode_UnsupportedProvider(t *testing.T) {
	node := &types.Node{Id: "n", Provider: types.NodeProvider("carrier-pigeon"), Http: "http://127.0.0.1:8545"}
	_, err := dialNode(context.Background(), node)
	if got := apiCode(t, err); got != "NOT_SUPPORTED" {
		t.Errorf("code = %s, want NOT_SUPPORTED", got)
	}
}
