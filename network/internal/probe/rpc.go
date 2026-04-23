package probe

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
)

type jsonRPCError struct {
	Code    int    `json:"code"`
	Message string `json:"message"`
}

type jsonRPCResponse struct {
	JSONRPC string          `json:"jsonrpc"`
	ID      int             `json:"id"`
	Result  json.RawMessage `json:"result,omitempty"`
	Error   *jsonRPCError   `json:"error,omitempty"`
}

// jsonRPCCall fires a single JSON-RPC POST and returns the Response.
// Network errors, non-200 HTTP, and malformed JSON bubble up as err.
// RPC-level errors (response.Error != nil) are returned in the response — caller decides.
func jsonRPCCall(ctx context.Context, client *http.Client, url, method string, params []any) (*jsonRPCResponse, error) {
	body, err := json.Marshal(map[string]any{
		"jsonrpc": "2.0",
		"id":      1,
		"method":  method,
		"params":  params,
	})
	if err != nil {
		return nil, fmt.Errorf("marshal rpc body: %w", err)
	}
	req, err := http.NewRequestWithContext(ctx, http.MethodPost, url, bytes.NewReader(body))
	if err != nil {
		return nil, fmt.Errorf("build rpc request: %w", err)
	}
	req.Header.Set("Content-Type", "application/json")
	resp, err := client.Do(req)
	if err != nil {
		return nil, fmt.Errorf("rpc http: %w", err)
	}
	defer resp.Body.Close()
	raw, err := io.ReadAll(resp.Body)
	if err != nil {
		return nil, fmt.Errorf("read rpc body: %w", err)
	}
	if resp.StatusCode != http.StatusOK {
		return nil, fmt.Errorf("rpc http status %d: %s", resp.StatusCode, string(raw))
	}
	var out jsonRPCResponse
	if err := json.Unmarshal(raw, &out); err != nil {
		return nil, fmt.Errorf("decode rpc response: %w", err)
	}
	return &out, nil
}
