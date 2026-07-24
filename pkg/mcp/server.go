package mcp

import (
	"context"
	"encoding/json"
)

// protocolVersion is the MCP revision this server implements.
const protocolVersion = "2024-11-05"

// Server holds registered tools and dispatches MCP JSON-RPC requests.
type Server struct {
	name    string
	version string
	tools   map[string]Tool
	order   []string
}

// NewServer returns an empty server.
func NewServer(name, version string) *Server {
	return &Server{name: name, version: version, tools: map[string]Tool{}}
}

// Register adds a tool (last registration of a name wins; order preserved on
// first registration).
func (s *Server) Register(t Tool) {
	if _, exists := s.tools[t.Name]; !exists {
		s.order = append(s.order, t.Name)
	}
	s.tools[t.Name] = t
}

type rpcRequest struct {
	JSONRPC string          `json:"jsonrpc"`
	ID      json.RawMessage `json:"id"`
	Method  string          `json:"method"`
	Params  json.RawMessage `json:"params"`
}

type rpcError struct {
	Code    int    `json:"code"`
	Message string `json:"message"`
}

// Handle processes one JSON-RPC request and returns the response bytes, or nil
// for a notification (a request without an id) that needs no reply.
func (s *Server) Handle(ctx context.Context, reqBytes []byte) []byte {
	var req rpcRequest
	if err := json.Unmarshal(reqBytes, &req); err != nil {
		return s.errorResponse(nil, -32700, "parse error")
	}
	isNotification := len(req.ID) == 0

	switch req.Method {
	case "initialize":
		return s.result(req.ID, map[string]any{
			"protocolVersion": protocolVersion,
			"capabilities":    map[string]any{"tools": map[string]any{}},
			"serverInfo":      map[string]any{"name": s.name, "version": s.version},
		})
	case "notifications/initialized", "notifications/cancelled":
		return nil // notifications: no response
	case "tools/list":
		return s.result(req.ID, map[string]any{"tools": s.toolList()})
	case "tools/call":
		return s.callTool(ctx, req)
	default:
		if isNotification {
			return nil
		}
		return s.errorResponse(req.ID, -32601, "method not found: "+req.Method)
	}
}

func (s *Server) toolList() []map[string]any {
	out := make([]map[string]any, 0, len(s.order))
	for _, name := range s.order {
		t := s.tools[name]
		schema := t.InputSchema
		if schema == nil {
			schema = map[string]any{"type": "object"}
		}
		out = append(out, map[string]any{
			"name":        t.Name,
			"description": t.Description,
			"inputSchema": schema,
		})
	}
	return out
}

func (s *Server) callTool(ctx context.Context, req rpcRequest) []byte {
	var params struct {
		Name      string         `json:"name"`
		Arguments map[string]any `json:"arguments"`
	}
	if err := json.Unmarshal(req.Params, &params); err != nil {
		return s.errorResponse(req.ID, -32602, "invalid params")
	}
	tool, ok := s.tools[params.Name]
	if !ok {
		return s.errorResponse(req.ID, -32602, "unknown tool: "+params.Name)
	}
	if params.Arguments == nil {
		params.Arguments = map[string]any{}
	}
	text, err := tool.Handler(ctx, params.Arguments)
	if err != nil {
		return s.result(req.ID, toolContent("error: "+err.Error(), true))
	}
	return s.result(req.ID, toolContent(text, false))
}

func toolContent(text string, isError bool) map[string]any {
	return map[string]any{
		"content": []map[string]any{{"type": "text", "text": text}},
		"isError": isError,
	}
}

func (s *Server) result(id json.RawMessage, result any) []byte {
	b, _ := json.Marshal(map[string]any{"jsonrpc": "2.0", "id": rawOrNull(id), "result": result})
	return b
}

func (s *Server) errorResponse(id json.RawMessage, code int, msg string) []byte {
	b, _ := json.Marshal(map[string]any{"jsonrpc": "2.0", "id": rawOrNull(id), "error": rpcError{Code: code, Message: msg}})
	return b
}

func rawOrNull(id json.RawMessage) any {
	if len(id) == 0 {
		return nil
	}
	return id
}
