package sshremote

import (
	"context"
	"crypto/ed25519"
	"crypto/rand"
	"io"
	"net"
	"net/http"
	"net/http/httptest"
	"os"
	"path/filepath"
	"strings"
	"testing"
	"time"

	"golang.org/x/crypto/ssh"
	"golang.org/x/crypto/ssh/knownhosts"
)

// sshTestServer is an in-process SSH server that accepts password auth and
// forwards every direct-tcpip channel (the tunnel the driver opens) to a fixed
// backend address — here, a mock JSON-RPC server. It models the remote host
// whose RPC port the driver reaches over SSH.
type sshTestServer struct {
	addr     string
	hostKey  ssh.PublicKey
	listener net.Listener
}

func startSSHServer(t *testing.T, wantUser, wantPassword, backendAddr string) *sshTestServer {
	t.Helper()
	_, priv, err := ed25519.GenerateKey(rand.Reader)
	if err != nil {
		t.Fatalf("gen host key: %v", err)
	}
	signer, err := ssh.NewSignerFromKey(priv)
	if err != nil {
		t.Fatalf("signer: %v", err)
	}
	cfg := &ssh.ServerConfig{
		PasswordCallback: func(c ssh.ConnMetadata, pass []byte) (*ssh.Permissions, error) {
			if c.User() == wantUser && string(pass) == wantPassword {
				return &ssh.Permissions{}, nil
			}
			return nil, errAuth
		},
	}
	cfg.AddHostKey(signer)

	ln, err := net.Listen("tcp", "127.0.0.1:0")
	if err != nil {
		t.Fatalf("listen: %v", err)
	}
	srv := &sshTestServer{addr: ln.Addr().String(), hostKey: signer.PublicKey(), listener: ln}

	go func() {
		for {
			conn, err := ln.Accept()
			if err != nil {
				return // listener closed
			}
			go serveSSHConn(conn, cfg, backendAddr)
		}
	}()
	t.Cleanup(func() { _ = ln.Close() })
	return srv
}

func serveSSHConn(nConn net.Conn, cfg *ssh.ServerConfig, backendAddr string) {
	sconn, chans, reqs, err := ssh.NewServerConn(nConn, cfg)
	if err != nil {
		_ = nConn.Close()
		return
	}
	defer sconn.Close()
	go ssh.DiscardRequests(reqs)
	for nc := range chans {
		if nc.ChannelType() != "direct-tcpip" {
			_ = nc.Reject(ssh.UnknownChannelType, "only direct-tcpip")
			continue
		}
		ch, chReqs, err := nc.Accept()
		if err != nil {
			continue
		}
		go ssh.DiscardRequests(chReqs)
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
}

var errAuth = &authError{}

type authError struct{}

func (*authError) Error() string { return "auth failed" }

// mockRPC serves a minimal JSON-RPC endpoint: eth_blockNumber and eth_chainId.
func mockRPC(t *testing.T) *httptest.Server {
	t.Helper()
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		body, _ := io.ReadAll(r.Body)
		w.Header().Set("Content-Type", "application/json")
		switch {
		case strings.Contains(string(body), "eth_chainId"):
			_, _ = w.Write([]byte(`{"jsonrpc":"2.0","id":1,"result":"0x539"}`))
		default: // eth_blockNumber etc.
			_, _ = w.Write([]byte(`{"jsonrpc":"2.0","id":1,"result":"0x2a"}`))
		}
	}))
	t.Cleanup(srv.Close)
	return srv
}

// backendAddr strips the scheme from an httptest URL → host:port.
func backendAddr(t *testing.T, url string) string {
	t.Helper()
	return strings.TrimPrefix(url, "http://")
}

func TestDial_TunneledRPC_Happy(t *testing.T) {
	rpc := mockRPC(t)
	addr := backendAddr(t, rpc.URL)
	srv := startSSHServer(t, "alice", "hunter2", addr)

	host, portStr, _ := net.SplitHostPort(srv.addr)
	creds := Credentials{User: "alice", Host: host, Port: atoi(t, portStr), Password: "hunter2"}

	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	client, err := Dial(ctx, creds, rpc.URL, ssh.InsecureIgnoreHostKey())
	if err != nil {
		t.Fatalf("Dial: %v", err)
	}
	defer client.Close()

	bn, err := client.BlockNumber(ctx)
	if err != nil {
		t.Fatalf("BlockNumber over tunnel: %v", err)
	}
	if bn != 0x2a {
		t.Errorf("block_number = %d, want 42", bn)
	}
	cid, err := client.ChainID(ctx)
	if err != nil {
		t.Fatalf("ChainID over tunnel: %v", err)
	}
	if cid.Uint64() != 0x539 {
		t.Errorf("chain_id = %d, want 1337", cid.Uint64())
	}
}

func TestDial_BadPassword(t *testing.T) {
	rpc := mockRPC(t)
	srv := startSSHServer(t, "alice", "hunter2", backendAddr(t, rpc.URL))
	host, portStr, _ := net.SplitHostPort(srv.addr)
	creds := Credentials{User: "alice", Host: host, Port: atoi(t, portStr), Password: "WRONG"}

	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()
	_, err := Dial(ctx, creds, rpc.URL, ssh.InsecureIgnoreHostKey())
	if err == nil {
		t.Fatal("expected auth failure")
	}
	if strings.Contains(err.Error(), "WRONG") {
		t.Errorf("password leaked into error: %v", err)
	}
}

func TestDial_HostKeyMismatch_Rejected(t *testing.T) {
	rpc := mockRPC(t)
	srv := startSSHServer(t, "alice", "hunter2", backendAddr(t, rpc.URL))
	host, portStr, _ := net.SplitHostPort(srv.addr)

	// known_hosts entry for a DIFFERENT key → server's real key must be rejected.
	_, otherPriv, _ := ed25519.GenerateKey(rand.Reader)
	otherSigner, _ := ssh.NewSignerFromKey(otherPriv)
	khPath := writeKnownHosts(t, srv.addr, otherSigner.PublicKey())

	cb, err := ResolveHostKeyCallback(func(k string) string {
		if k == "CHAINBENCH_SSH_KNOWN_HOSTS" {
			return khPath
		}
		return ""
	})
	if err != nil {
		t.Fatalf("ResolveHostKeyCallback: %v", err)
	}
	creds := Credentials{User: "alice", Host: host, Port: atoi(t, portStr), Password: "hunter2"}
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()
	if _, err := Dial(ctx, creds, rpc.URL, cb); err == nil {
		t.Fatal("expected host key rejection")
	}
}

func TestResolveHostKeyCallback_InsecureOptIn(t *testing.T) {
	cb, err := ResolveHostKeyCallback(func(k string) string {
		if k == "CHAINBENCH_SSH_INSECURE_HOST_KEY" {
			return "1"
		}
		return ""
	})
	if err != nil {
		t.Fatalf("insecure opt-in should not error: %v", err)
	}
	if cb == nil {
		t.Fatal("nil callback")
	}
}

func writeKnownHosts(t *testing.T, addr string, key ssh.PublicKey) string {
	t.Helper()
	line := knownhosts.Line([]string{addr}, key)
	dir := t.TempDir()
	p := filepath.Join(dir, "known_hosts")
	if err := os.WriteFile(p, []byte(line+"\n"), 0o600); err != nil {
		t.Fatalf("write known_hosts: %v", err)
	}
	return p
}

func atoi(t *testing.T, s string) int {
	t.Helper()
	n := 0
	for _, c := range s {
		n = n*10 + int(c-'0')
	}
	return n
}
