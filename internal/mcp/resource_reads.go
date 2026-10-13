package mcp

import (
	"bytes"
	"context"
	"encoding/json"
	"errors"
	"io"

	"github.com/0xmhha/chainbench/internal/app"
)

func resourceInspectTool() Tool {
	return Tool{Name: "chainbench_resource_inspect", ReadOnly: true, Description: "Observe a target's physical identity, files, bounded log bytes or recorded process without changing it.", InputSchema: map[string]any{
		"type": "object", "required": []string{"request"}, "properties": map[string]any{
			"request":      map[string]any{"type": "object", "description": "Typed resource observation: target {dataRoot,server,host,user,port}, inspect, exists/checksums/regular/executable path arrays, process {pid,binary,args,dataDir,configPath}, log {path,offset,limit}; serverSet and docker are optional."},
			"serverLogins": map[string]any{"type": "string"},
		}}, Handler: func(ctx context.Context, args map[string]any) (string, error) {
		raw, err := json.Marshal(args["request"])
		if err != nil {
			return "", err
		}
		if len(raw) > 1<<20 {
			return "", errors.New("resource inspection request exceeds one MiB")
		}
		var in app.ResourceReadIn
		decoder := json.NewDecoder(bytes.NewReader(raw))
		decoder.DisallowUnknownFields()
		if err = decoder.Decode(&in); err != nil {
			return "", err
		}
		if err = decoder.Decode(new(any)); !errors.Is(err, io.EOF) {
			return "", errors.New("one resource request is required")
		}
		out, err := invokeFeature[app.ResourceReadIn, app.ResourceReadOut](ctx, app.Deps{}.WithLogins(argString(args, "serverLogins", "")), "resource.inspect", in)
		if err != nil {
			return "", err
		}
		raw, err = json.Marshal(out)
		return string(raw), err
	}}
}

func resourceAccessTool() Tool {
	return Tool{Name: "chainbench_resource_access", ReadOnly: true, Description: "Check a server's SSH authentication and data-root permission without changing files or returning credentials.", InputSchema: map[string]any{
		"type": "object", "required": []string{"serverSet", "server", "dataRoot"}, "properties": map[string]any{
			"serverSet": map[string]any{"type": "string"}, "server": map[string]any{"type": "string"}, "dataRoot": map[string]any{"type": "string"}, "serverLogins": map[string]any{"type": "string"},
		}}, Handler: func(ctx context.Context, args map[string]any) (string, error) {
		out, err := invokeFeature[app.ResourceAccessIn, app.ResourceAccessOut](ctx, app.Deps{}.WithLogins(argString(args, "serverLogins", "")), "resource.access", app.ResourceAccessIn{ServerSet: argString(args, "serverSet", ""), Server: argString(args, "server", ""), DataRoot: argString(args, "dataRoot", "")})
		if err != nil {
			return "", err
		}
		raw, err := json.Marshal(out)
		return string(raw), err
	}}
}
