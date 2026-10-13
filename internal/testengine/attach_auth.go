package testengine

import (
	"fmt"
	"net/http"
	"os"
	"reflect"

	"github.com/0xmhha/chainbench/internal/core/node"
	"github.com/0xmhha/chainbench/internal/core/remote"
)

func attachAuthClients(cfg AttachConfig) (map[string]*http.Client, map[string]http.Header, error) {
	nodes := cfg.Nodes.Nodes
	if cfg.NodeSet != nil {
		nodes = cfg.NodeSet.Nodes
	}
	clients, headers, authByURL := map[string]*http.Client{}, map[string]http.Header{}, map[string]node.Auth{}
	for _, n := range nodes {
		client, err := remote.HTTPClientFromAuth(remote.Auth(n.Auth), os.Getenv)
		if err != nil {
			return nil, nil, fmt.Errorf("attach node%d: %w", n.Index, err)
		}
		header, err := remote.RequestHeaders(remote.Auth(n.Auth), os.Getenv)
		if err != nil {
			return nil, nil, fmt.Errorf("attach node%d: %w", n.Index, err)
		}
		for _, target := range []string{n.RPCURL, n.WSURL, n.MetricsURL} {
			if target == "" {
				continue
			}
			if previous, ok := authByURL[target]; ok && !reflect.DeepEqual(previous, n.Auth) {
				return nil, nil, fmt.Errorf("attach endpoint has conflicting authentication references")
			}
			authByURL[target] = n.Auth
			clients[target], headers[target] = client, header
		}
	}
	return clients, headers, nil
}
