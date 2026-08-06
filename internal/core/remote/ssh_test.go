package remote

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
	"strconv"
	"strings"
	"testing"

	"golang.org/x/crypto/ssh"
	"golang.org/x/crypto/ssh/knownhosts"
)

// --- in-process SSH server harness (ported from the legacy sshremote tests) ---

type sshTestServer struct {
	addr    string
	hostKey ssh.PublicKey
}

// execFn fakes remote command execution. nil means the server rejects session
// channels (tunnel-only tests).
type execFn func(command string) (stdout string, exitCode int)

func startSSHServer(t *testing.T, wantUser, wantPassword, backendAddr string, exec execFn) *sshTestServer {
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
			return nil, errAuthFailed
		},
	}
	cfg.AddHostKey(signer)

	ln, err := net.Listen("tcp", "127.0.0.1:0")
	if err != nil {
		t.Fatalf("listen: %v", err)
	}
	go func() {
		for {
			conn, err := ln.Accept()
			if err != nil {
				return
			}
			go serveSSHConn(conn, cfg, backendAddr, exec)
		}
	}()
	t.Cleanup(func() { _ = ln.Close() })
	return &sshTestServer{addr: ln.Addr().String(), hostKey: signer.PublicKey()}
}

func serveSSHConn(nConn net.Conn, cfg *ssh.ServerConfig, backendAddr string, exec execFn) {
	sconn, chans, reqs, err := ssh.NewServerConn(nConn, cfg)
	if err != nil {
		_ = nConn.Close()
		return
	}
	defer sconn.Close()
	go ssh.DiscardRequests(reqs)
	for nc := range chans {
		switch nc.ChannelType() {
		case "direct-tcpip":
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
		case "session":
			if exec == nil {
				_ = nc.Reject(ssh.Prohibited, "session not allowed")
				continue
			}
			ch, chReqs, err := nc.Accept()
			if err != nil {
				continue
			}
			go handleSessionExec(ch, chReqs, exec)
		default:
			_ = nc.Reject(ssh.UnknownChannelType, "unsupported channel")
		}
	}
}

func handleSessionExec(ch ssh.Channel, reqs <-chan *ssh.Request, exec execFn) {
	defer ch.Close()
	for req := range reqs {
		if req.Type != "exec" {
			_ = req.Reply(false, nil)
			continue
		}
		var payload struct{ Command string }
		_ = ssh.Unmarshal(req.Payload, &payload)
		_ = req.Reply(true, nil)
		stdout, code := exec(payload.Command)
		_, _ = io.WriteString(ch, stdout)
		_, _ = ch.SendRequest("exit-status", false,
			ssh.Marshal(struct{ Status uint32 }{Status: uint32(code)}))
		return
	}
}

type authFailed struct{}

func (*authFailed) Error() string { return "auth failed" }

var errAuthFailed = &authFailed{}

func mockRPC(t *testing.T) *httptest.Server {
	t.Helper()
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		_, _ = w.Write([]byte(`{"jsonrpc":"2.0","id":1,"result":"0x539"}`))
	}))
	t.Cleanup(srv.Close)
	return srv
}

func credsFor(t *testing.T, srv *sshTestServer, password string) Credentials {
	t.Helper()
	host, portStr, _ := net.SplitHostPort(srv.addr)
	port, _ := strconv.Atoi(portStr)
	return Credentials{User: "alice", Host: host, Port: port, Password: password}
}

// --- tests ---

func TestDialTunnelClient_TunneledHTTP(t *testing.T) {
	rpc := mockRPC(t)
	srv := startSSHServer(t, "alice", "hunter2", strings.TrimPrefix(rpc.URL, "http://"), nil)

	client, closer, err := DialTunnelClient(credsFor(t, srv, "hunter2"), ssh.InsecureIgnoreHostKey())
	if err != nil {
		t.Fatalf("DialTunnelClient: %v", err)
	}
	defer closer.Close()

	resp, err := client.Post(rpc.URL, "application/json",
		strings.NewReader(`{"jsonrpc":"2.0","id":1,"method":"eth_chainId"}`))
	if err != nil {
		t.Fatalf("tunneled POST: %v", err)
	}
	defer resp.Body.Close()
	body, _ := io.ReadAll(resp.Body)
	if !strings.Contains(string(body), "0x539") {
		t.Errorf("tunneled response = %q, want 0x539", string(body))
	}
}

func TestDialTunnelClient_BadPasswordNoLeak(t *testing.T) {
	rpc := mockRPC(t)
	srv := startSSHServer(t, "alice", "hunter2", strings.TrimPrefix(rpc.URL, "http://"), nil)
	_, _, err := DialTunnelClient(credsFor(t, srv, "WRONGPASS"), ssh.InsecureIgnoreHostKey())
	if err == nil {
		t.Fatal("expected auth failure")
	}
	if strings.Contains(err.Error(), "WRONGPASS") {
		t.Errorf("password leaked into error: %v", err)
	}
}

func TestDialTunnelClient_HostKeyMismatch(t *testing.T) {
	rpc := mockRPC(t)
	srv := startSSHServer(t, "alice", "hunter2", strings.TrimPrefix(rpc.URL, "http://"), nil)

	// known_hosts entry for a DIFFERENT key -> the server's real key is rejected.
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
	if _, _, err := DialTunnelClient(credsFor(t, srv, "hunter2"), cb); err == nil {
		t.Fatal("expected host key rejection")
	}
}

func TestExec(t *testing.T) {
	var gotCmd string
	srv := startSSHServer(t, "alice", "hunter2", "", func(cmd string) (string, int) {
		gotCmd = cmd
		return "ok-output\n", 0
	})
	res, err := Exec(context.Background(), credsFor(t, srv, "hunter2"), ssh.InsecureIgnoreHostKey(), "systemctl restart gstable")
	if err != nil {
		t.Fatalf("Exec: %v", err)
	}
	if res.ExitCode != 0 || res.Stdout != "ok-output\n" || gotCmd != "systemctl restart gstable" {
		t.Errorf("unexpected result %+v (cmd %q)", res, gotCmd)
	}
}

func TestExec_NonZeroExitIsNotError(t *testing.T) {
	srv := startSSHServer(t, "alice", "hunter2", "", func(string) (string, int) { return "boom\n", 7 })
	res, err := Exec(context.Background(), credsFor(t, srv, "hunter2"), ssh.InsecureIgnoreHostKey(), "false")
	if err != nil {
		t.Fatalf("non-zero exit should not error: %v", err)
	}
	if res.ExitCode != 7 {
		t.Errorf("ExitCode = %d, want 7", res.ExitCode)
	}
}

func TestExec_BadPasswordNoLeak(t *testing.T) {
	srv := startSSHServer(t, "alice", "hunter2", "", func(string) (string, int) { return "", 0 })
	_, err := Exec(context.Background(), credsFor(t, srv, "WRONGPASS"), ssh.InsecureIgnoreHostKey(), "echo hi")
	if err == nil || strings.Contains(err.Error(), "WRONGPASS") {
		t.Errorf("expected auth failure without leaking password: %v", err)
	}
}

func TestDialSSH_Validation(t *testing.T) {
	cb := ssh.InsecureIgnoreHostKey()
	if _, _, err := DialTunnelClient(Credentials{Host: "h", Password: "p"}, cb); err == nil {
		t.Error("missing user should error")
	}
	if _, _, err := DialTunnelClient(Credentials{User: "u", Host: "h"}, cb); err == nil {
		t.Error("empty password should error")
	}
	if _, _, err := DialTunnelClient(Credentials{User: "u", Host: "h", Password: "p"}, nil); err == nil {
		t.Error("nil host key callback should error")
	}
}

func TestResolveHostKeyCallback_InsecureOptIn(t *testing.T) {
	cb, err := ResolveHostKeyCallback(func(k string) string {
		if k == "CHAINBENCH_SSH_INSECURE_HOST_KEY" {
			return "1"
		}
		return ""
	})
	if err != nil || cb == nil {
		t.Fatalf("insecure opt-in should yield a callback: %v", err)
	}
}

func writeKnownHosts(t *testing.T, addr string, key ssh.PublicKey) string {
	t.Helper()
	line := knownhosts.Line([]string{addr}, key)
	p := filepath.Join(t.TempDir(), "known_hosts")
	if err := os.WriteFile(p, []byte(line+"\n"), 0o600); err != nil {
		t.Fatalf("write known_hosts: %v", err)
	}
	return p
}
