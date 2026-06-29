package main

import (
	"context"
	"crypto/ed25519"
	"crypto/rand"
	"encoding/json"
	"errors"
	"io"
	"net"
	"strconv"
	"strings"
	"testing"

	"golang.org/x/crypto/ssh"

	"github.com/0xmhha/chainbench/network/internal/state"
	"github.com/0xmhha/chainbench/network/internal/types"
)

// startExecSSHServer is a lean in-process SSH server for handler-level
// process/fs tests: it accepts password auth and answers "exec" requests via
// the supplied responder. (The richer tunnel+exec server lives in the
// sshremote package test; handler tests only need exec.)
func startExecSSHServer(t *testing.T, user, password string, exec func(cmd string) (string, int)) string {
	t.Helper()
	_, priv, err := ed25519.GenerateKey(rand.Reader)
	if err != nil {
		t.Fatalf("gen host key: %v", err)
	}
	signer, _ := ssh.NewSignerFromKey(priv)
	cfg := &ssh.ServerConfig{
		PasswordCallback: func(c ssh.ConnMetadata, pass []byte) (*ssh.Permissions, error) {
			if c.User() == user && string(pass) == password {
				return &ssh.Permissions{}, nil
			}
			return nil, errBadAuth
		},
	}
	cfg.AddHostKey(signer)
	ln, err := net.Listen("tcp", "127.0.0.1:0")
	if err != nil {
		t.Fatalf("listen: %v", err)
	}
	t.Cleanup(func() { _ = ln.Close() })
	go func() {
		for {
			conn, err := ln.Accept()
			if err != nil {
				return
			}
			go func(c net.Conn) {
				sc, chans, reqs, err := ssh.NewServerConn(c, cfg)
				if err != nil {
					_ = c.Close()
					return
				}
				defer sc.Close()
				go ssh.DiscardRequests(reqs)
				for nc := range chans {
					if nc.ChannelType() != "session" {
						_ = nc.Reject(ssh.UnknownChannelType, "session only")
						continue
					}
					ch, creqs, err := nc.Accept()
					if err != nil {
						continue
					}
					go func() {
						defer ch.Close()
						for req := range creqs {
							if req.Type != "exec" {
								_ = req.Reply(false, nil)
								continue
							}
							var p struct{ Command string }
							_ = ssh.Unmarshal(req.Payload, &p)
							_ = req.Reply(true, nil)
							out, code := exec(p.Command)
							_, _ = io.WriteString(ch, out)
							_, _ = ch.SendRequest("exit-status", false,
								ssh.Marshal(struct{ Status uint32 }{Status: uint32(code)}))
							return
						}
					}()
				}
			}(conn)
		}
	}()
	return ln.Addr().String()
}

type badAuthErr struct{}

func (*badAuthErr) Error() string { return "auth failed" }

var errBadAuth = &badAuthErr{}

// saveSSHNode persists a single-node ssh-remote network whose auth points at the
// given SSH server address, with the supplied provider_meta. Sets the password
// env var and insecure host-key opt-in for the duration of the test.
func saveSSHNode(t *testing.T, stateDir, network, sshAddr string, meta map[string]any) {
	t.Helper()
	host, portStr, _ := net.SplitHostPort(sshAddr)
	port, _ := strconv.Atoi(portStr)
	t.Setenv("CHAINBENCH_SSH_INSECURE_HOST_KEY", "1")
	t.Setenv("CHAINBENCH_TEST_SSH_PW", "hunter2")
	netw := &types.Network{
		Name: network, ChainType: "ethereum", ChainId: 1337,
		Nodes: []types.Node{{
			Id: "node1", Provider: types.NodeProviderSshRemote, Http: "http://127.0.0.1:8545",
			Auth: types.Auth{
				"type": "ssh-password", "user": "alice", "host": host,
				"port": float64(port), "env": "CHAINBENCH_TEST_SSH_PW",
			},
			ProviderMeta: meta,
		}},
	}
	if err := state.SaveRemote(stateDir, netw); err != nil {
		t.Fatalf("SaveRemote: %v", err)
	}
}

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

func TestHandleNodeStop_SSHRemote_Happy(t *testing.T) {
	var gotCmd string
	addr := startExecSSHServer(t, "alice", "hunter2", func(cmd string) (string, int) {
		gotCmd = cmd
		return "", 0
	})
	stateDir := t.TempDir()
	saveSSHNode(t, stateDir, "ssh", addr, map[string]any{"stop_cmd": "systemctl stop gstable"})

	h := newHandleNodeStop(stateDir, t.TempDir())
	bus, _ := newTestBus(t)
	args, _ := json.Marshal(map[string]any{"network": "ssh", "node_id": "node1"})
	data, err := h(args, bus)
	if err != nil {
		t.Fatalf("handler: %v", err)
	}
	if data["stopped"] != true {
		t.Errorf("stopped = %v, want true", data["stopped"])
	}
	if gotCmd != "systemctl stop gstable" {
		t.Errorf("remote saw command %q", gotCmd)
	}
}

func TestHandleNodeStop_SSHRemote_NonZeroExit_Upstream(t *testing.T) {
	addr := startExecSSHServer(t, "alice", "hunter2", func(cmd string) (string, int) {
		return "service not found\n", 5
	})
	stateDir := t.TempDir()
	saveSSHNode(t, stateDir, "ssh", addr, map[string]any{"stop_cmd": "systemctl stop gstable"})

	h := newHandleNodeStop(stateDir, t.TempDir())
	bus, _ := newTestBus(t)
	args, _ := json.Marshal(map[string]any{"network": "ssh", "node_id": "node1"})
	_, err := h(args, bus)
	if got := apiCode(t, err); got != "UPSTREAM_ERROR" {
		t.Errorf("code = %s, want UPSTREAM_ERROR", got)
	}
}

func TestHandleNodeStop_SSHRemote_MissingStopCmd_NotSupported(t *testing.T) {
	addr := startExecSSHServer(t, "alice", "hunter2", func(cmd string) (string, int) { return "", 0 })
	stateDir := t.TempDir()
	// provider_meta has no stop_cmd → node does not provide that operation.
	saveSSHNode(t, stateDir, "ssh", addr, map[string]any{"log_file": "/var/log/x"})

	h := newHandleNodeStop(stateDir, t.TempDir())
	bus, _ := newTestBus(t)
	args, _ := json.Marshal(map[string]any{"network": "ssh", "node_id": "node1"})
	_, err := h(args, bus)
	if got := apiCode(t, err); got != "NOT_SUPPORTED" {
		t.Errorf("code = %s, want NOT_SUPPORTED", got)
	}
}

func TestHandleNodeRestart_SSHRemote_ComposesStopStart(t *testing.T) {
	var cmds []string
	addr := startExecSSHServer(t, "alice", "hunter2", func(cmd string) (string, int) {
		cmds = append(cmds, cmd)
		return "", 0
	})
	stateDir := t.TempDir()
	// No restart_cmd → composed from stop_cmd + start_cmd.
	saveSSHNode(t, stateDir, "ssh", addr, map[string]any{
		"stop_cmd": "stop-it", "start_cmd": "start-it",
	})

	h := newHandleNodeRestart(stateDir, t.TempDir())
	bus, _ := newTestBus(t)
	args, _ := json.Marshal(map[string]any{"network": "ssh", "node_id": "node1"})
	data, err := h(args, bus)
	if err != nil {
		t.Fatalf("handler: %v", err)
	}
	if data["restarted"] != true {
		t.Errorf("restarted = %v, want true", data["restarted"])
	}
	if len(cmds) != 2 || cmds[0] != "stop-it" || cmds[1] != "start-it" {
		t.Errorf("commands = %v, want [stop-it start-it]", cmds)
	}
}

func TestHandleNodeTailLog_SSHRemote_Happy(t *testing.T) {
	var gotCmd string
	addr := startExecSSHServer(t, "alice", "hunter2", func(cmd string) (string, int) {
		gotCmd = cmd
		return "line-a\nline-b\nline-c\n", 0
	})
	stateDir := t.TempDir()
	saveSSHNode(t, stateDir, "ssh", addr, map[string]any{"log_file": "/var/lib/gstable/node.log"})

	h := newHandleNodeTailLog(stateDir)
	bus, _ := newTestBus(t)
	args, _ := json.Marshal(map[string]any{"network": "ssh", "node_id": "node1", "lines": 3})
	data, err := h(args, bus)
	if err != nil {
		t.Fatalf("handler: %v", err)
	}
	lines, ok := data["lines"].([]string)
	if !ok || len(lines) != 3 || lines[0] != "line-a" {
		t.Errorf("lines = %v, want [line-a line-b line-c]", data["lines"])
	}
	if !strings.Contains(gotCmd, "tail -n 3 --") || !strings.Contains(gotCmd, "/var/lib/gstable/node.log") {
		t.Errorf("tail command = %q", gotCmd)
	}
}

func TestHandleNodeTailLog_SSHRemote_MissingLogFile_NotSupported(t *testing.T) {
	addr := startExecSSHServer(t, "alice", "hunter2", func(cmd string) (string, int) { return "", 0 })
	stateDir := t.TempDir()
	saveSSHNode(t, stateDir, "ssh", addr, map[string]any{"stop_cmd": "x"}) // no log_file

	h := newHandleNodeTailLog(stateDir)
	bus, _ := newTestBus(t)
	args, _ := json.Marshal(map[string]any{"network": "ssh", "node_id": "node1"})
	_, err := h(args, bus)
	if got := apiCode(t, err); got != "NOT_SUPPORTED" {
		t.Errorf("code = %s, want NOT_SUPPORTED", got)
	}
}
