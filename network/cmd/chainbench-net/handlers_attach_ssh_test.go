package main

import (
	"crypto/ed25519"
	"crypto/rand"
	"encoding/json"
	"io"
	"net"
	"net/http"
	"net/http/httptest"
	"strconv"
	"strings"
	"testing"

	"golang.org/x/crypto/ssh"

	"github.com/0xmhha/chainbench/network/internal/state"
	"github.com/0xmhha/chainbench/network/internal/types"
)

// startTunnelSSHServer is an in-process SSH server that forwards every
// direct-tcpip channel to backendAddr. It models the remote host whose RPC port
// network.attach reaches over the tunnel during probe. (handlers_ssh_test.go's
// server is exec/session-only; attach needs the tunnel.)
func startTunnelSSHServer(t *testing.T, user, password, backendAddr string) string {
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
					if nc.ChannelType() != "direct-tcpip" {
						_ = nc.Reject(ssh.UnknownChannelType, "tunnel only")
						continue
					}
					ch, creqs, err := nc.Accept()
					if err != nil {
						continue
					}
					go ssh.DiscardRequests(creqs)
					go func() {
						defer ch.Close()
						backend, err := net.Dial("tcp", backendAddr)
						if err != nil {
							return
						}
						defer backend.Close()
						go func() { _, _ = io.Copy(backend, ch) }()
						_, _ = io.Copy(ch, backend)
					}()
				}
			}(conn)
		}
	}()
	return ln.Addr().String()
}

// mockStablenetRPC answers the probe's eth_chainId (0x205b = 8283) and
// istanbul_getValidators ([]), which Detect classifies as chain_type "stablenet".
func mockStablenetRPC(t *testing.T) *httptest.Server {
	t.Helper()
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		var req struct {
			Method string          `json:"method"`
			ID     json.RawMessage `json:"id"`
		}
		body, _ := io.ReadAll(r.Body)
		_ = json.Unmarshal(body, &req)
		w.Header().Set("Content-Type", "application/json")
		switch req.Method {
		case "eth_chainId":
			_, _ = w.Write([]byte(`{"jsonrpc":"2.0","id":1,"result":"0x205b"}`))
		case "istanbul_getValidators":
			_, _ = w.Write([]byte(`{"jsonrpc":"2.0","id":1,"result":[]}`))
		default:
			_, _ = w.Write([]byte(`{"jsonrpc":"2.0","id":1,"error":{"code":-32601,"message":"nf"}}`))
		}
	}))
	t.Cleanup(srv.Close)
	return srv
}

func TestHandleNetworkAttach_SSHRemote_Happy(t *testing.T) {
	rpc := mockStablenetRPC(t)
	backend := strings.TrimPrefix(rpc.URL, "http://")
	sshAddr := startTunnelSSHServer(t, "deploy", "hunter2", backend)
	host, portStr, _ := net.SplitHostPort(sshAddr)

	t.Setenv("CHAINBENCH_SSH_INSECURE_HOST_KEY", "1")
	t.Setenv("CHAINBENCH_ATTACH_PW", "hunter2")
	stateDir := t.TempDir()
	h := newHandleNetworkAttach(stateDir)
	bus, _ := newTestBus(t)

	args, _ := json.Marshal(map[string]any{
		"name":     "sshnet",
		"rpc_url":  rpc.URL, // RPC URL as reachable from the remote host (here, the mock)
		"provider": "ssh-remote",
		"auth": map[string]any{
			"type": "ssh-password", "user": "deploy", "host": host,
			"port": strconvAtoiFloat(portStr), "env": "CHAINBENCH_ATTACH_PW",
		},
		"provider_meta": map[string]any{
			"log_file": "/var/log/node.log", "stop_cmd": "systemctl stop x",
		},
	})
	data, err := h(args, bus)
	if err != nil {
		t.Fatalf("attach: %v", err)
	}
	if data["chain_type"] != "stablenet" {
		t.Errorf("chain_type = %v, want stablenet (probed over tunnel)", data["chain_type"])
	}
	if fmtInt(data["chain_id"]) != "8283" {
		t.Errorf("chain_id = %v, want 8283", data["chain_id"])
	}

	// Persisted node must be ssh-remote with auth + provider_meta intact.
	net, lerr := state.LoadActive(state.LoadActiveOptions{StateDir: stateDir, Name: "sshnet"})
	if lerr != nil {
		t.Fatalf("load saved network: %v", lerr)
	}
	n := net.Nodes[0]
	if n.Provider != types.NodeProviderSshRemote {
		t.Errorf("provider = %q, want ssh-remote", n.Provider)
	}
	if n.Auth["type"] != "ssh-password" {
		t.Errorf("auth.type = %v", n.Auth["type"])
	}
	if n.ProviderMeta["stop_cmd"] != "systemctl stop x" {
		t.Errorf("provider_meta.stop_cmd not persisted: %v", n.ProviderMeta)
	}
}

func TestHandleNetworkAttach_SSHRemote_MissingAuth(t *testing.T) {
	h := newHandleNetworkAttach(t.TempDir())
	bus, _ := newTestBus(t)
	args, _ := json.Marshal(map[string]any{
		"name": "x", "rpc_url": "http://127.0.0.1:8545", "provider": "ssh-remote",
	})
	_, err := h(args, bus)
	if got := apiCode(t, err); got != "INVALID_ARGS" {
		t.Errorf("code = %s, want INVALID_ARGS", got)
	}
}

func TestHandleNetworkAttach_UnknownProvider(t *testing.T) {
	h := newHandleNetworkAttach(t.TempDir())
	bus, _ := newTestBus(t)
	args, _ := json.Marshal(map[string]any{
		"name": "x", "rpc_url": "http://127.0.0.1:8545", "provider": "carrier-pigeon",
	})
	_, err := h(args, bus)
	if got := apiCode(t, err); got != "INVALID_ARGS" {
		t.Errorf("code = %s, want INVALID_ARGS", got)
	}
}

func strconvAtoiFloat(s string) float64 {
	n, _ := strconv.Atoi(s)
	return float64(n)
}

func fmtInt(v any) string {
	switch n := v.(type) {
	case int:
		return strconv.Itoa(n)
	case int64:
		return strconv.FormatInt(n, 10)
	case float64:
		return strconv.Itoa(int(n))
	default:
		return ""
	}
}
