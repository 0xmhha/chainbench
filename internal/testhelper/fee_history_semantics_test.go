package testhelper

import (
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"reflect"
	"testing"

	"github.com/0xmhha/chainbench/internal/core/session"
	"github.com/0xmhha/chainbench/internal/dsl"
	"github.com/0xmhha/chainbench/internal/dsl/interp"
)

func TestPublicFeeHistoryRejectsMalformedArrays(t *testing.T) {
	for _, name := range []string{"valid", "short base", "short ratio", "short reward", "zero last base", "negative last ratio", "ratio above one", "missing reward percentile"} {
		t.Run(name, func(t *testing.T) {
			bases := make([]any, 11)
			ratios := make([]any, 10)
			rewards := make([]any, 10)
			for i := range bases {
				bases[i] = "0x64"
			}
			for i := range ratios {
				ratios[i] = 0.5
				rewards[i] = []any{"0x1", "0x2"}
			}
			switch name {
			case "short base":
				bases = bases[:10]
			case "short ratio":
				ratios = ratios[:9]
			case "short reward":
				rewards = rewards[:9]
			case "zero last base":
				bases[10] = "0x0"
			case "negative last ratio":
				ratios[9] = -0.01
			case "ratio above one":
				ratios[9] = 1.01
			case "missing reward percentile":
				rewards[9] = []any{"0x1"}
			}
			srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
				var q struct {
					ID     any
					Method string
					Params []any
				}
				if err := json.NewDecoder(r.Body).Decode(&q); err != nil {
					t.Error(err)
					return
				}
				var result any = "0xa"
				if q.Method == "eth_feeHistory" {
					if !reflect.DeepEqual(q.Params, []any{"0xa", "latest", []any{float64(25), float64(75)}}) {
						t.Errorf("wrong original fee-history input: %v", q.Params)
					}
					result = map[string]any{"oldestBlock": "0x1", "baseFeePerGas": bases, "gasUsedRatio": ratios, "reward": rewards}
				}
				_ = json.NewEncoder(w).Encode(map[string]any{"jsonrpc": "2.0", "id": q.ID, "result": result})
			}))
			defer srv.Close()
			raws, err := dsl.ReadFilesWithChainPreset([]string{"../../tests/tc/common/fee/CT-FEE-012-fee-history.json"}, "../../presets/chain/stablenet-bp4.json")
			if err != nil {
				t.Fatal(err)
			}
			s, err := dsl.ParseV2(raws[0])
			if err != nil {
				t.Fatal(err)
			}
			// The native runner proves the block wait; unit controls exercise actual public assertions.
			selected := []dsl.Statement{}
			for _, st := range s.Sequence {
				if st.Expect != "" {
					selected = append(selected, st)
				}
			}
			s.Sequence = selected
			rec := &recordStub{}
			got, err := interp.NewInterpreter(deps()).Run(context.Background(), s, envWithNode(t, srv.URL), rec)
			want := session.StatusFail
			if name == "valid" {
				want = session.StatusPass
			}
			if err != nil || got != want {
				t.Fatalf("public history %s err=%v want=%s assertions=%+v", got, err, want, rec.asserts)
			}
		})
	}
}
