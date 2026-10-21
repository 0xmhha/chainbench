package main

import (
	"context"
	"encoding/json"
	"flag"
	"net"
	"net/http"
	"os"
	"os/exec"
	"path/filepath"
	"slices"
	"strings"
	"testing"
	"time"
)

const dashboardTestProcess = "CHAINBENCH_TEST_DASHBOARD_PROCESS"

// TestDashboardProcess runs the actual entry point in an owned subprocess, so
// default flags are verified together with all production service wiring.
func TestDashboardProcess(t *testing.T) {
	if os.Getenv(dashboardTestProcess) != "1" {
		return
	}
	separator := slices.Index(os.Args, "--")
	if separator < 0 {
		t.Fatal("missing daemon arguments")
	}
	os.Args = append([]string{os.Args[0]}, os.Args[separator+1:]...)
	flag.CommandLine = flag.NewFlagSet("chainbench-dashboard", flag.ExitOnError)
	main()
}

func dashboardCommand(t *testing.T, ctx context.Context, args ...string) *exec.Cmd {
	t.Helper()
	binary, err := os.Executable()
	if err != nil {
		t.Fatal(err)
	}
	command := exec.CommandContext(ctx, binary, append([]string{"-test.run=^TestDashboardProcess$", "--"}, args...)...)
	command.Env = append(os.Environ(), dashboardTestProcess+"=1", "CHAINBENCH_PUBLISHER_TOKEN=")
	return command
}

func TestAccountsDefaultAndExplicitOverrides(t *testing.T) {
	for _, tc := range []struct {
		name     string
		flags    []string
		accounts bool
	}{
		{name: "default"},
		{name: "explicit false", flags: []string{"-accounts=false"}},
		{name: "override true", flags: []string{"-accounts=true"}, accounts: true},
		{name: "boolean shorthand", flags: []string{"-accounts"}, accounts: true},
	} {
		t.Run(tc.name, func(t *testing.T) {
			listener, err := net.Listen("tcp", "127.0.0.1:0")
			if err != nil {
				t.Fatal(err)
			}
			address := listener.Addr().String()
			if err := listener.Close(); err != nil {
				t.Fatal(err)
			}
			root := t.TempDir()
			cli := filepath.Join(root, "chainbench")
			if err := os.WriteFile(cli, []byte("#!/bin/sh\nexit 1\n"), 0700); err != nil {
				t.Fatal(err)
			}
			analysis, err := filepath.Abs("../../docs/chain-analysis")
			if err != nil {
				t.Fatal(err)
			}
			ctx, cancel := context.WithTimeout(context.Background(), 20*time.Second)
			t.Cleanup(cancel)
			args := append([]string{"-addr", address, "-deployment-root", root, "-chainbench", cli, "-chain-analysis", analysis, "-chain-presets=", "-manifest-keys="}, tc.flags...)
			command := dashboardCommand(t, ctx, args...)
			if err := command.Start(); err != nil {
				t.Fatal(err)
			}
			t.Cleanup(func() { cancel(); _ = command.Wait() })
			client := &http.Client{Timeout: time.Second}
			ticker := time.NewTicker(20 * time.Millisecond)
			defer ticker.Stop()
			for {
				select {
				case <-ctx.Done():
					t.Fatal("dashboard did not start")
				case <-ticker.C:
					response, err := client.Get("http://" + address + "/api/v1/auth/config")
					if err != nil {
						continue
					}
					var config struct{ AccountsEnabled, BootstrapRequired *bool }
					err = json.NewDecoder(response.Body).Decode(&config)
					_ = response.Body.Close()
					if err != nil || response.StatusCode != 200 || config.AccountsEnabled == nil || config.BootstrapRequired == nil {
						t.Fatal("invalid access config", err)
					}
					if *config.AccountsEnabled != tc.accounts || *config.BootstrapRequired != tc.accounts {
						t.Fatalf("accounts=%v bootstrap=%v, expected %v", *config.AccountsEnabled, *config.BootstrapRequired, tc.accounts)
					}
					_, setupErr := os.Stat(filepath.Join(root, "setup.token"))
					if (setupErr == nil) != tc.accounts {
						t.Fatal("setup token does not match the account mode")
					}
					return
				}
			}
		})
	}
}

func TestAccountFreeModeRefusesReachableTeamListener(t *testing.T) {
	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()
	command := dashboardCommand(t, ctx, "-web-mode=team", "-addr=0.0.0.0:0", "-trusted-proxy=127.0.0.1")
	output, err := command.CombinedOutput()
	if err == nil || !strings.Contains(string(output), "account-free mode requires loopback") {
		t.Fatal("default account-free mode allowed a reachable listener")
	}
}
