// Command chainbench-dashboard is the dashboard daemon (requirement #19): it hosts the
// obs event bus and run store behind an HTTP + SSE API and serves the dashboard
// page. Pipeline runs feed it live by POSTing obs events to /api/events; with
// -artifact-root it also serves completed-run session artifacts from disk.
package main

import (
	"encoding/json"
	"flag"
	"fmt"
	"net"
	"net/http"
	"os"

	"github.com/0xmhha/chainbench/internal/app"
	_ "github.com/0xmhha/chainbench/internal/chains/all"
	"github.com/0xmhha/chainbench/internal/core/collector"
	"github.com/0xmhha/chainbench/internal/dashboard"
)

func main() {
	addr := flag.String("addr", "127.0.0.1:8787", "listen address")
	artifactRoot := flag.String("artifact-root", "", "directory of session artifacts to serve under /api/sessions (optional)")
	presetRoot := flag.String("chain-presets", "presets/chain", "chain preset directory")
	deploymentAccounts := flag.String("deployment-accounts", "", "private provisioned account file enabling shared deployment editor")
	deploymentRoot := flag.String("deployment-root", "chainbench-out/web-ui", "persistent deployment data directory")
	manifestAssets := flag.String("manifest-assets", "", "server-provisioned binary asset JSON enabling manifest management")
	manifestKeys := flag.String("manifest-keys", "presets/keys", "key preset for isolated manifest setup")
	webMode := flag.String("web-mode", "personal", "account mode: personal or team")
	legacy := flag.Bool("legacy-observation", false, "loopback-only legacy event observer without control APIs")
	tlsCert := flag.String("tls-cert", "", "TLS certificate for the web service")
	tlsKey := flag.String("tls-key", "", "TLS private key for the web service")
	flag.Parse()
	if (*tlsCert == "") != (*tlsKey == "") {
		fmt.Fprintln(os.Stderr, "TLS certificate and key must be configured together")
		os.Exit(1)
	}
	if *legacy {
		host, _, err := net.SplitHostPort(*addr)
		if err != nil || net.ParseIP(host) == nil || !net.ParseIP(host).IsLoopback() {
			fmt.Fprintln(os.Stderr, "legacy observation requires an explicit loopback listen address")
			os.Exit(1)
		}
	}

	bus := collector.NewBus()
	defer bus.Close()
	store := collector.NewMemStore()

	var opts []dashboard.Option
	if !*legacy && *presetRoot != "" {
		opts = append(opts, dashboard.WithChainPresets(*presetRoot))
	}
	if *artifactRoot != "" {
		opts = append(opts, dashboard.WithArtifactRoot(*artifactRoot))
	}
	if !*legacy {
		if token := os.Getenv("CHAINBENCH_PUBLISHER_TOKEN"); token != "" {
			if len(token) < 32 {
				fmt.Fprintln(os.Stderr, "publisher token must contain at least 32 bytes")
				os.Exit(1)
			}
			opts = append(opts, dashboard.WithPublisherToken(token))
		}
		var authenticate dashboard.DeploymentAuthenticator
		var auth *app.WebAuth
		var err error
		if *deploymentAccounts != "" {
			authenticate, err = dashboard.DeploymentAccounts(*deploymentAccounts)
		} else {
			auth, err = app.OpenWebAuth(*deploymentRoot, *webMode)
			if err == nil {
				authenticate = dashboard.WebAuthenticator(auth)
			}
		}
		if err != nil {
			fmt.Fprintln(os.Stderr, "deployment accounts:", err)
			os.Exit(1)
		}
		deployments, err := app.OpenDeploymentStore(*deploymentRoot)
		if err != nil {
			fmt.Fprintln(os.Stderr, "deployment store:", err)
			os.Exit(1)
		}
		opts = append(opts, dashboard.WithDeployments(deployments, authenticate), dashboard.WithTestCases(authenticate))
		assets := []app.ManifestBinary{}
		var authorize func(app.DeploymentActor) error
		if auth != nil {
			authorize = auth.AuthorizeJobActor
		}
		if auth != nil {
			opts = append(opts, dashboard.WithWebSecurity(auth, deployments))
			fmt.Fprintln(os.Stderr, "First administrator setup uses the private setup.token file in the web data directory.")
		}
		if *manifestAssets != "" {
			raw, err := os.ReadFile(*manifestAssets)
			if err != nil {
				fmt.Fprintln(os.Stderr, err)
				os.Exit(1)
			}
			if err = json.Unmarshal(raw, &assets); err != nil {
				fmt.Fprintln(os.Stderr, err)
				os.Exit(1)
			}
		}
		manifests, err := app.OpenManifestStore(*deploymentRoot)
		if err != nil {
			fmt.Fprintln(os.Stderr, err)
			os.Exit(1)
		}
		opts = append(opts, dashboard.WithManifests(manifests, authenticate, assets, *manifestKeys))
		engine := app.NewWebChainEngine(*deploymentRoot, *manifestKeys, deployments, manifests, assets, authorize)
		jobs, err := app.OpenWebJobs(*deploymentRoot, engine, deployments.RedactWeb, authorize)
		if err != nil {
			fmt.Fprintln(os.Stderr, "job store:", err)
			os.Exit(1)
		}
		opts = append(opts, dashboard.WithWebJobs(jobs, authenticate))
		history, err := app.OpenWebHistory(*deploymentRoot, *artifactRoot, jobs, deployments.RedactWeb)
		if err != nil {
			fmt.Fprintln(os.Stderr, "history store:", err)
			os.Exit(1)
		}
		opts = append(opts, dashboard.WithWebHistory(history, authenticate))
	}
	srv := dashboard.NewServer(bus, store, opts...)

	scheme := "http"
	if *tlsCert != "" {
		scheme = "https"
	}
	fmt.Fprintf(os.Stderr, "chainbench-dashboard listening on %s://%s\n", scheme, *addr)
	var serveErr error
	if *tlsCert != "" {
		serveErr = http.ListenAndServeTLS(*addr, *tlsCert, *tlsKey, srv)
	} else {
		serveErr = http.ListenAndServe(*addr, srv)
	}
	if serveErr != nil {
		fmt.Fprintln(os.Stderr, "chainbench-dashboard:", serveErr)
		os.Exit(1)
	}
}
