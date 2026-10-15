package testengine_test

import (
	"bytes"
	"compress/gzip"
	"context"
	"encoding/base64"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"net/http/httptest"
	"os"
	"path/filepath"
	"strconv"
	"strings"
	"sync"
	"testing"
	"time"

	"github.com/0xmhha/chainbench/internal/chainsetup"
	"github.com/0xmhha/chainbench/internal/chainsetup/verb"
	"github.com/0xmhha/chainbench/internal/core/rpc"
	"github.com/0xmhha/chainbench/internal/dsl"
	"github.com/0xmhha/chainbench/internal/testengine"
)

func TestNativeLegacyAuthorizationEstimates(t *testing.T) {
	families := []struct{ chain, variable string }{{"stablenet", "GSTABLE_BIN"}, {"wbft", "GWBFT_BIN"}}
	for _, f := range families {
		if os.Getenv(f.variable) == "" {
			t.Skip("native authorization estimate proof requires " + f.variable)
		}
	}
	for _, f := range families {
		if !t.Run(f.chain, func(t *testing.T) {
			root := failureRuntime(t)
			ctx, cancel := context.WithTimeout(t.Context(), 6*time.Minute)
			defer cancel()
			name := "17-estimategas-authorizationlist-cost.json"
			path := filepath.Join(repoRoot(t), "tests/tc/go-stablenet/post-v1.0.0-change/common-all", name)
			b, err := os.ReadFile(path)
			if err != nil {
				t.Fatal(err)
			}
			if err = os.WriteFile(filepath.Join(root, name), b, 0600); err != nil {
				t.Fatal(err)
			}
			raws, err := dsl.ReadFilesWithChainPreset([]string{path}, f.chain+"-bp4")
			if err != nil {
				t.Fatal(err)
			}
			input := testengine.RunSuiteIn{SpecContent: raws, DataDir: filepath.Join(root, "w"), ArtifactRoot: filepath.Join(root, "reports"), Binary: os.Getenv(f.variable), KeysSource: "generate", KeysDir: filepath.Join(root, "w/keys"), LaunchOpts: []string{"cache=64"}}
			up, plan, err := testengine.PlanSuiteLayout(ctx, input)
			if err != nil || plan.Keys.Source != "generate" || plan.Keys.Dir != input.KeysDir || up.KeysSource != "generate" {
				t.Fatal("refuse keys outside owned workspace", plan.Keys, err)
			}
			proof := map[string]any{"chain": f.chain, "binary": input.Binary, "plannedKeysDir": plan.Keys.Dir, "scope": "Unchanged public case run through attach interpreter after shared planner/ChainUp on owned BP4, with a byte-preserving HTTP recorder; original18 EOA transfer baseline plus existing 0/1/2 authorization cost bounds. Fresh code-free delegate, not original17 deployed Simple7702Delegate, original EN topology, or whole legacy/L1/87/Web."}
			defer func() {
				b, err := json.MarshalIndent(proof, "", "  ")
				if err != nil {
					t.Error(err)
					return
				}
				if err = os.WriteFile(filepath.Join(root, "reports/authorization-estimate-proof.json"), b, 0600); err != nil {
					t.Error(err)
				}
			}()
			if _, err = verb.ChainUp(ctx, chainsetup.Deps{}, up); err != nil {
				proof["setupError"] = err.Error()
				t.Fatal(err)
			}
			state, err := verb.NetworkStatus(ctx, chainsetup.Deps{}, verb.NetworkStatusIn{DataDir: input.DataDir})
			if err != nil || len(state.Nodes.Nodes) != 4 {
				t.Fatal("missing actual BP4", err)
			}
			for _, n := range state.Nodes.Nodes {
				client := rpc.Dial(n.RPCURL)
				for {
					head, e := client.BlockNumber(ctx)
					if e == nil && head >= 2 {
						break
					}
					select {
					case <-ctx.Done():
						t.Fatal("native readiness", ctx.Err(), e)
					case <-time.After(250 * time.Millisecond):
					}
				}
			}
			target := state.Nodes.Nodes[0].RPCURL
			var mu sync.Mutex
			observations := []map[string]any{}
			transport := &http.Client{Timeout: 30 * time.Second}
			proxy := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
				body, e := io.ReadAll(r.Body)
				if e != nil {
					http.Error(w, e.Error(), http.StatusBadRequest)
					return
				}
				request, e := http.NewRequestWithContext(r.Context(), http.MethodPost, target, bytes.NewReader(body))
				if e != nil {
					http.Error(w, e.Error(), http.StatusBadRequest)
					return
				}
				request.Header = r.Header.Clone()
				response, e := transport.Do(request)
				if e != nil {
					http.Error(w, e.Error(), http.StatusBadGateway)
					return
				}
				defer response.Body.Close()
				data, e := io.ReadAll(response.Body)
				if e != nil {
					http.Error(w, e.Error(), http.StatusBadGateway)
					return
				}
				if observation, ok := authorizationRPCObservation(body, data, response.Header.Get("Content-Encoding")); ok {
					observation["target"], observation["httpStatus"] = target, response.StatusCode
					mu.Lock()
					observations = append(observations, observation)
					mu.Unlock()
				}
				for key, values := range response.Header {
					w.Header()[key] = values
				}
				w.WriteHeader(response.StatusCode)
				_, _ = w.Write(data)
			}))
			defer func() {
				proxy.Close()
				mu.Lock()
				proof["observations"] = observations
				mu.Unlock()
			}()
			state.Nodes.Nodes[0].RPCURL = proxy.URL
			engine, err := testengine.NewAttachEngine(testengine.AttachConfig{Chain: f.chain, Nodes: state.Nodes, Caps: []string{"rpc", "family:wbft"}, ArtifactRoot: input.ArtifactRoot, KeysDir: input.KeysDir, ReadOnlyKeys: true})
			if err != nil {
				t.Fatal(err)
			}
			sessionRoot, err := engine.Run(ctx, raws)
			proof["session"] = sessionRoot
			if err != nil {
				t.Fatal(err)
			}
			b, err = os.ReadFile(filepath.Join(sessionRoot, "session.json"))
			if err != nil {
				t.Fatal(err)
			}
			var session struct {
				Summary struct{ Pass, Fail, Blocked, Skip int }
			}
			if err = json.Unmarshal(b, &session); err != nil {
				t.Fatal(err)
			}
			proof["sessionDocument"] = json.RawMessage(b)
			mu.Lock()
			defer mu.Unlock()
			proof["observations"] = observations
			if session.Summary.Pass != 1 || session.Summary.Fail != 0 || session.Summary.Blocked != 0 || session.Summary.Skip != 0 || len(observations) != 4 {
				t.Fatal("expected exactly one public pass and four unchanged RPC observations", session.Summary, len(observations))
			}
			code := observations[0]["response"].(map[string]any)
			if code["result"] != "0x" || code["error"] != nil {
				t.Fatal("receiver is not a code-free EOA")
			}
			var from, to string
			for i, o := range observations[1:] {
				q, a := o["request"].(map[string]any), o["response"].(map[string]any)
				tx := q["params"].([]any)[0].(map[string]any)
				if i == 0 {
					from, _ = tx["from"].(string)
					to, _ = tx["to"].(string)
				}
				auth, _ := tx["authorizationList"].([]any)
				gas, e := strconv.ParseUint(strings.TrimPrefix(fmt.Sprint(a["result"]), "0x"), 16, 64)
				upper := []uint64{21000, 46690, 72065}[i]
				if q["method"] != "eth_estimateGas" || o["httpStatus"] != http.StatusOK || a["error"] != nil || e != nil || gas < uint64(21000+i*25000) || gas > upper || from == "" || to == "" || from == to || tx["from"] != from || tx["to"] != to || tx["value"] != "0x1" || len(auth) != i {
					t.Fatal("wrong actual transfer baseline or authorization bound", i, gas, e)
				}
			}
		}) {
			t.Fatal("stop after preceding proof or cleanup failed")
		}
	}
}

// authorizationRPCObservation records a copy; the proxy forwards the original
// response bytes and headers without decoding or changing them.
func authorizationRPCObservation(body, data []byte, encoding string) (map[string]any, bool) {
	decoded := data
	if encoding == "gzip" {
		reader, err := gzip.NewReader(bytes.NewReader(data))
		if err != nil {
			return nil, false
		}
		defer reader.Close()
		decoded, err = io.ReadAll(reader)
		if err != nil {
			return nil, false
		}
	}
	var q, a map[string]any
	if json.Unmarshal(body, &q) != nil || json.Unmarshal(decoded, &a) != nil || (q["method"] != "eth_estimateGas" && q["method"] != "eth_getCode") {
		return nil, false
	}
	return map[string]any{"request": q, "response": a, "requestBody": string(body), "responseBody": string(decoded), "responseBytesBase64": base64.StdEncoding.EncodeToString(data), "contentEncoding": encoding}, true
}
