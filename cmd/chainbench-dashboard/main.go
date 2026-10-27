// Command chainbench-dashboard is the dashboard daemon (requirement #19): it hosts the
// obs event bus and run store behind an HTTP + SSE API and serves the dashboard
// page. Pipeline runs feed it live by POSTing obs events to /api/events; with
// -artifact-root it also serves completed-run session artifacts from disk.
package main

import (
	"context"
	"encoding/json"
	"flag"
	"fmt"
	"net"
	"net/http"
	"net/netip"
	"os"
	"strings"

	"github.com/0xmhha/chainbench/internal/app"
	_ "github.com/0xmhha/chainbench/internal/chains/all"
	"github.com/0xmhha/chainbench/internal/core/collector"
	"github.com/0xmhha/chainbench/internal/dashboard"
	"github.com/0xmhha/chainbench/internal/webcli"
)

// accountFreeRemoteRefusal is what an account-free start prints when its flags
// say remote browsers reach it through this machine.
const accountFreeRemoteRefusal = "account-free mode serves only a browser on this machine; team mode, -trusted-proxy and TLS need -accounts=true"

func main() {
	addr := flag.String("addr", "127.0.0.1:8787", "listen address")
	artifactRoot := flag.String("artifact-root", "", "directory of session artifacts to serve under /api/sessions (optional)")
	presetRoot := flag.String("chain-presets", "presets/chain", "chain preset directory")
	analysisRoot := flag.String("chain-analysis", "docs/chain-analysis", "captured chain binary documentation directory")
	deploymentRoot := flag.String("deployment-root", "chainbench-out/web-ui", "persistent deployment data directory")
	manifestAssets := flag.String("manifest-assets", "", "server-provisioned binary asset JSON enabling manifest management")
	manifestKeys := flag.String("manifest-keys", "presets/keys", "key preset for isolated manifest setup")
	webMode := flag.String("web-mode", "personal", "account mode: personal or team")
	accounts := flag.Bool("accounts", false, "require registered accounts (default: account-free local access)")
	legacy := flag.Bool("legacy-observation", false, "loopback-only legacy event observer without control APIs")
	tlsCert := flag.String("tls-cert", "", "TLS certificate for the web service")
	tlsKey := flag.String("tls-key", "", "TLS private key for the web service")
	trusted := flag.String("trusted-proxy", "", "comma-separated addresses or CIDRs of TLS terminators whose X-Forwarded-Proto is trusted")
	chainbenchBin := flag.String("chainbench", "", "chainbench CLI the Web runs (default: the one beside this binary, then PATH)")
	flag.Parse()
	if (*tlsCert == "") != (*tlsKey == "") {
		fmt.Fprintln(os.Stderr, "TLS certificate and key must be configured together")
		os.Exit(1)
	}
	var proxies []netip.Prefix
	for _, raw := range strings.Split(*trusted, ",") {
		if raw = strings.TrimSpace(raw); raw == "" {
			continue
		}
		prefix, err := netip.ParsePrefix(raw)
		if err != nil {
			ip, ipErr := netip.ParseAddr(raw)
			if ipErr != nil {
				fmt.Fprintln(os.Stderr, "trusted proxy:", err)
				os.Exit(1)
			}
			prefix = netip.PrefixFrom(ip, ip.BitLen())
		}
		proxies = append(proxies, prefix)
	}
	if !*legacy {
		if err := dashboard.ValidateServiceMode(*webMode, *addr, *tlsCert != "", proxies); err != nil {
			fmt.Fprintln(os.Stderr, err)
			os.Exit(1)
		}
		if !*accounts {
			if err := dashboard.ValidateServiceMode("personal", *addr, *tlsCert != "", proxies); err != nil {
				fmt.Fprintln(os.Stderr, "account-free mode requires loopback; use -accounts=true for remote access:", err)
				os.Exit(1)
			}
			// A same-host proxy forwards every remote browser from loopback, so
			// the listener alone cannot tell; these flags say remote use is meant.
			if *webMode == "team" || len(proxies) > 0 || *tlsCert != "" {
				fmt.Fprintln(os.Stderr, accountFreeRemoteRefusal)
				os.Exit(1)
			}
		}
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

	opts := []dashboard.Option{dashboard.WithTrustedProxies(proxies)}
	if *legacy {
		opts = append(opts, dashboard.WithLegacyObservation())
	}
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
		// Both modes share browser protection and execution authorization. Only
		// account-enabled access requires registration or a password.
		var auth *app.WebAuth
		var err error
		if *accounts {
			auth, err = app.OpenWebAuth(*deploymentRoot, *webMode)
		} else {
			auth, err = app.OpenLocalWebAuth(*deploymentRoot)
		}
		var authenticate dashboard.DeploymentAuthenticator
		if err == nil {
			authenticate = dashboard.WebAuthenticator(auth)
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
		authorize := auth.AuthorizeJobActor
		opts = append(opts, dashboard.WithWebSecurity(auth, deployments))
		if *accounts {
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
		// The Web runs the chainbench CLI for the networks it reads and changes,
		// so a dashboard without it refuses to start rather than fail a job.
		self, _ := os.Executable()
		cliPath, err := webcli.FindBinary(*chainbenchBin, self)
		if err != nil {
			fmt.Fprintln(os.Stderr, err)
			os.Exit(1)
		}
		engine.UseCLI(webcli.Runner{Binary: cliPath})
		deployments.UseCLI(webcli.Runner{Binary: cliPath})
		manifests.UseCLI(webcli.Runner{Binary: cliPath})
		// Plans pin the documented surface of every binary they run, so the
		// documentation has to be readable before the service starts.
		if err = app.CheckChainSurfaces(*analysisRoot); err != nil {
			fmt.Fprintln(os.Stderr, "chain analysis:", err)
			os.Exit(1)
		}
		engine.UseChainSurfaces(*analysisRoot)
		opts = append(opts, dashboard.WithChainSurfaces(engine, authenticate))
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
		opts = append(opts, dashboard.WithWebAudit(auth, deployments, jobs, history, manifests))
		monitor, err := app.OpenWebMonitor(*deploymentRoot, deployments.RedactWeb)
		if err != nil {
			fmt.Fprintln(os.Stderr, "observation store:", err)
			os.Exit(1)
		}
		monitor.UseCLI(webcli.Runner{Binary: cliPath})
		go monitor.Run(context.Background())
		// Local machines of jobs the restart interrupted are checked once,
		// read-only, so their records show what actually remains.
		go jobs.ReconcileInterrupted(context.Background())
		engine.UseMonitor(monitor)
		history.UseObservations(monitor)
		opts = append(opts, dashboard.WithWebMonitor(monitor, authenticate))
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
