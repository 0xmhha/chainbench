package remote

import (
	"context"
	"errors"
	"io"
	"net"
	"net/http"
	"net/http/httptest"
	"strconv"
	"strings"
	"sync/atomic"
	"testing"
	"time"

	"golang.org/x/crypto/ssh"
)

func TestCachedSSHCredentialsRecheckAuthorizationBeforeEveryConnection(t *testing.T) {
	var revoked atomic.Bool
	var commands, handshakes atomic.Int32
	srv := startSSHServer(t, "alice", "guard-password", "", func(string) (string, int) {
		commands.Add(1)
		return "ok", 0
	})
	creds := credsFor(t, srv, "guard-password")
	creds.BeforeDial = func() error {
		if revoked.Load() {
			return errors.New("private-revocation-reason-must-not-leak")
		}
		return nil
	}
	callback := func(hostname string, addr net.Addr, key ssh.PublicKey) error {
		handshakes.Add(1)
		return ssh.InsecureIgnoreHostKey()(hostname, addr, key)
	}
	if _, err := Exec(context.Background(), creds, callback, "allowed"); err != nil {
		t.Fatal(err)
	}
	revoked.Store(true)
	for _, call := range []func() error{
		func() error { _, err := Exec(context.Background(), creds, callback, "denied"); return err },
		func() error { _, _, err := DialTunnelClient(creds, callback); return err },
	} {
		err := call()
		if err == nil || strings.Contains(err.Error(), "private-revocation-reason") || !strings.Contains(err.Error(), "authorization denied") {
			t.Fatal("revoked access or private reason leaked", err)
		}
	}
	if commands.Load() != 1 || handshakes.Load() != 1 {
		t.Fatal("a new connection/command began after revocation", commands.Load(), handshakes.Load())
	}
	ctx, cancel := context.WithCancel(context.Background())
	cancel()
	if _, err := Exec(ctx, creds, callback, "cancelled"); !errors.Is(err, context.Canceled) {
		t.Fatal("cancelled access reached transport", err)
	}
}

func TestSSHTunnelRechecksRevocationOnReusedHTTPConnection(t *testing.T) {
	var requests atomic.Int32
	backend := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		requests.Add(1)
		_, _ = w.Write([]byte("ok"))
	}))
	defer backend.Close()
	srv := startSSHServer(t, "alice", "guard-password", strings.TrimPrefix(backend.URL, "http://"), nil)
	creds := credsFor(t, srv, "guard-password")
	var revoked atomic.Bool
	creds.BeforeDial = func() error {
		if revoked.Load() {
			return errors.New("revoked")
		}
		return nil
	}
	client, closer, err := DialTunnelClient(creds, ssh.InsecureIgnoreHostKey())
	if err != nil {
		t.Fatal(err)
	}
	defer func() { _ = closer.Close() }()
	defer client.CloseIdleConnections()
	response, err := client.Get(backend.URL)
	if err != nil {
		t.Fatal(err)
	}
	_, _ = io.Copy(io.Discard, response.Body)
	_ = response.Body.Close()
	revoked.Store(true)
	if _, err = client.Get(backend.URL); err == nil {
		t.Fatal("cached tunnel request accepted after revocation")
	}
	if requests.Load() != 1 {
		t.Fatal("revoked request reached backend", requests.Load())
	}
}

func TestSSHCommandCancellationReturnsBeforeRemoteCommandCompletes(t *testing.T) {
	started, release := make(chan struct{}), make(chan struct{})
	srv := startSSHServer(t, "alice", "guard-password", "", func(string) (string, int) {
		close(started)
		<-release
		return "late output", 0
	})
	defer close(release)
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()
	done := make(chan error, 1)
	go func() {
		_, err := Exec(ctx, credsFor(t, srv, "guard-password"), ssh.InsecureIgnoreHostKey(), "blocked")
		done <- err
	}()
	select {
	case <-started:
	case <-time.After(5 * time.Second):
		t.Fatal("command did not start")
	}
	cancel()
	select {
	case err := <-done:
		if !errors.Is(err, context.Canceled) {
			t.Fatal("cancelled command did not return context error", err)
		}
	case <-time.After(time.Second):
		t.Fatal("SSH execution remained blocked after cancellation")
	}
}

func TestSSHAuthenticationCancellationClosesStalledConnection(t *testing.T) {
	listener, err := net.Listen("tcp", "127.0.0.1:0")
	if err != nil {
		t.Fatal(err)
	}
	defer listener.Close()
	accepted := make(chan net.Conn, 1)
	go func() {
		conn, err := listener.Accept()
		if err == nil {
			accepted <- conn
		}
	}()
	host, port, err := net.SplitHostPort(listener.Addr().String())
	if err != nil {
		t.Fatal(err)
	}
	p, err := strconv.Atoi(port)
	if err != nil {
		t.Fatal(err)
	}
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()
	done := make(chan error, 1)
	go func() {
		_, err := Exec(ctx, Credentials{Host: host, Port: p, User: "alice", Password: "guard-password"}, ssh.InsecureIgnoreHostKey(), "unreachable")
		done <- err
	}()
	var conn net.Conn
	select {
	case conn = <-accepted:
	case <-time.After(5 * time.Second):
		t.Fatal("connection did not start")
	}
	defer conn.Close()
	cancel()
	select {
	case err := <-done:
		if !errors.Is(err, context.Canceled) {
			t.Fatal("authentication cancellation not propagated", err)
		}
	case <-time.After(time.Second):
		t.Fatal("SSH handshake remained blocked after cancellation")
	}
}
