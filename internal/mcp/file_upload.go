package mcp

import (
	"context"
	"encoding/json"
	"fmt"
	"os"
	"strconv"

	"github.com/0xmhha/chainbench/internal/app"
)

func fileUploadTool() Tool {
	return Tool{Name: "chainbench_file_upload", Description: "Upload accepted local files to a workspace target, checking SHA256 before and after transfer and setting an octal permission mode.", InputSchema: map[string]any{
		"type": "object", "required": []string{"localPaths", "workspaceConfig"}, "properties": map[string]any{
			"localPaths": map[string]any{"type": "array", "items": map[string]any{"type": "string"}, "minItems": 1},
			"serverSet":  map[string]any{"type": "string"}, "server": map[string]any{"type": "string"}, "serverLogins": map[string]any{"type": "string"},
			"workspaceConfig": map[string]any{"type": "string"}, "docker": map[string]any{"type": "boolean"},
			"purpose": map[string]any{"type": "string", "enum": []string{"bin", "binaries", "genesis", "configs", "keystore", "keyrings"}}, "remote": map[string]any{"type": "string"},
			"sha256": map[string]any{"type": "string"}, "mode": map[string]any{"type": "string", "description": "octal permission such as 0755; empty preserves source mode"}, "forceUpload": map[string]any{"type": "boolean"},
		}}, Handler: func(ctx context.Context, args map[string]any) (string, error) {
		var mode os.FileMode
		if s := argString(args, "mode", ""); s != "" {
			n, err := strconv.ParseUint(s, 8, 32)
			if err != nil || n == 0 || n > 0777 {
				return "", fmt.Errorf("mode %q: want an octal permission such as 0755", s)
			}
			mode = os.FileMode(n)
		}
		out, err := invokeFeature[app.UploadIn, app.UploadOut](ctx, app.Deps{}.WithLogins(argString(args, "serverLogins", "")), "file.upload", app.UploadIn{
			Target:     app.TransferServer{ServerSet: argString(args, "serverSet", ""), Server: argString(args, "server", ""), WorkspaceConfigPath: argString(args, "workspaceConfig", ""), Docker: argBool(args, "docker", false)},
			LocalPaths: argStrings(args, "localPaths"), Purpose: argString(args, "purpose", ""), RemotePath: argString(args, "remote", ""), SHA256: argString(args, "sha256", ""), Mode: mode, Force: argBool(args, "forceUpload", false),
		})
		raw, marshalErr := json.Marshal(out)
		if err != nil {
			return string(raw), err
		}
		return string(raw), marshalErr
	}}
}
