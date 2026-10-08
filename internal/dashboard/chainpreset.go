package dashboard

import (
	"encoding/json"
	"io"
	"net/http"

	"github.com/0xmhha/chainbench/internal/app"
	"github.com/0xmhha/chainbench/internal/dsl"
)

// WithChainPresets enables a read/validate-only composition editor. No document
// storage, filesystem writes or execution authority is granted by this option.
func WithChainPresets(root string) Option {
	return func(s *Server) {
		s.mux.HandleFunc("GET /api/v1/chain-presets", func(w http.ResponseWriter, r *http.Request) {
			presets, err := app.ChainPresets(root)
			if err != nil {
				http.Error(w, "cannot load chain presets", http.StatusInternalServerError)
				return
			}
			w.Header().Set("Content-Type", "application/json")
			_ = json.NewEncoder(w).Encode(presets)
		})
		s.mux.HandleFunc("GET /api/v1/contracts/chain-preset", func(w http.ResponseWriter, r *http.Request) {
			w.Header().Set("Content-Type", "application/json")
			if r.URL.Query().Get("chain") == "" {
				_, _ = w.Write(dsl.SchemaV2)
				return
			}
			schema, err := app.ChainPresetContract(r.URL.Query().Get("chain"))
			if err != nil {
				http.Error(w, "unsupported chain", http.StatusBadRequest)
				return
			}
			_ = json.NewEncoder(w).Encode(schema)
		})
		s.validateDocument = func(w http.ResponseWriter, r *http.Request) {
			var in struct {
				Kind            string          `json:"kind"`
				Name            string          `json:"name"`
				ContractVersion string          `json:"contractVersion"`
				Content         json.RawMessage `json:"content"`
				AssetRefs       []string        `json:"assetRefs"`
			}
			dec := json.NewDecoder(http.MaxBytesReader(w, r.Body, 1<<20))
			dec.DisallowUnknownFields()
			if err := dec.Decode(&in); err != nil {
				http.Error(w, "invalid document input", http.StatusBadRequest)
				return
			}
			if err := dec.Decode(new(any)); err != io.EOF {
				http.Error(w, "expected one document", http.StatusBadRequest)
				return
			}
			errors := []map[string]string{}
			switch {
			case in.Kind != "chain-preset" || in.ContractVersion != "2" || in.Name == "":
				errors = append(errors, map[string]string{"path": "/", "code": "unsupported", "message": "expected named chain-preset with contractVersion 2"})
			default:
				if err := app.ValidateDeploymentDocument(app.DeploymentDocumentInput{Kind: in.Kind, Name: in.Name, ContractVersion: in.ContractVersion, Content: in.Content, AssetRefs: in.AssetRefs}); err != nil {
					errors = append(errors, map[string]string{"path": "/content", "code": "invalid", "message": err.Error()})
				}
			}
			w.Header().Set("Content-Type", "application/json")
			_ = json.NewEncoder(w).Encode(map[string]any{"valid": len(errors) == 0, "contractVersion": "2", "errors": errors, "warnings": []string{"Target access and binary compatibility require execution preflight."}})
		}
	}
}
