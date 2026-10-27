package app

import (
	"bytes"
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"slices"

	"go.yaml.in/yaml/v3"

	"github.com/0xmhha/chainbench/internal/core/filestore"
	"github.com/0xmhha/chainbench/internal/resource"
)

// webLocalMapField holds, in a server-set document, how this machine reaches
// servers that are local docker containers. The engine reads the same map as
// localmap.yaml beside the server set and applies it only with --docker, so a
// document that carries one runs every job on its pool in docker mode.
const webLocalMapField = "localmap"

// splitWebServerSet separates a server-set document into the server set the
// engine reads and its localmap, raw, when it has one.
func splitWebServerSet(content json.RawMessage) (map[string]any, json.RawMessage, error) {
	var fields map[string]json.RawMessage
	if err := json.Unmarshal(content, &fields); err != nil {
		return nil, nil, err
	}
	localmap := fields[webLocalMapField]
	delete(fields, webLocalMapField)
	rest, err := json.Marshal(fields)
	if err != nil {
		return nil, nil, err
	}
	var set map[string]any
	if err = json.Unmarshal(rest, &set); err != nil {
		return nil, nil, err
	}
	return set, localmap, nil
}

// webDocker reports whether the pool's servers are docker containers, which a
// localmap in its server-set document says.
func webDocker(doc DeploymentDocumentInput) bool {
	if doc.Kind != "server-set" {
		return false
	}
	_, localmap, err := splitWebServerSet(doc.Content)
	return err == nil && len(localmap) != 0
}

// parseWebLocalMap reads a document's localmap and refuses one the engine
// would misread: no hosts, an address the pool does not have, an empty
// substitute host, a port no machine listens on, or a field it does not know.
func parseWebLocalMap(raw json.RawMessage, set *resource.Set) (*resource.LocalMap, error) {
	var top struct {
		Hosts map[string]json.RawMessage `json:"hosts"`
	}
	dec := json.NewDecoder(bytes.NewReader(raw))
	dec.DisallowUnknownFields()
	if err := dec.Decode(&top); err != nil {
		return nil, contentFieldError(errors.New("localmap holds only hosts"), webLocalMapField)
	}
	if len(top.Hosts) == 0 {
		return nil, contentFieldError(errors.New("localmap maps no hosts"), webLocalMapField, "hosts")
	}
	pool := map[string]bool{}
	for _, h := range set.PoolSpec.Hosts {
		pool[h.Addr] = true
	}
	addrs := make([]string, 0, len(top.Hosts))
	for addr := range top.Hosts {
		addrs = append(addrs, addr)
	}
	slices.Sort(addrs)
	out := &resource.LocalMap{Hosts: map[string]resource.HostPorts{}}
	for _, addr := range addrs {
		at := []string{webLocalMapField, "hosts", addr}
		if !pool[addr] {
			return nil, contentFieldError(fmt.Errorf("localmap names %s, which is not an address of the pool", addr), at...)
		}
		var entry resource.HostPorts
		dec := json.NewDecoder(bytes.NewReader(top.Hosts[addr]))
		dec.DisallowUnknownFields()
		if err := dec.Decode(&entry); err != nil {
			return nil, contentFieldError(fmt.Errorf("localmap entry for %s holds a host and its ports only", addr), at...)
		}
		if entry.Host == "" {
			return nil, contentFieldError(fmt.Errorf("localmap entry for %s needs the host this machine dials", addr), append(at, "host")...)
		}
		for from, to := range entry.Ports {
			if from < 1 || from > resource.MaxListenPort || to < 1 || to > resource.MaxListenPort {
				return nil, contentFieldError(fmt.Errorf("localmap ports for %s must map 1..%d onto 1..%d", addr, resource.MaxListenPort, resource.MaxListenPort), append(at, "ports")...)
			}
		}
		out.Hosts[addr] = entry
	}
	return out, nil
}

// webServerSetFiles is a server-set document as the engine reads it: the
// server set, and for a docker pool the localmap that sits beside it (nil
// otherwise).
func webServerSetFiles(doc DeploymentDocumentInput) (set, localmap []byte, err error) {
	if err = ValidateDeploymentDocument(doc); err != nil {
		return nil, nil, err
	}
	fields, raw, err := splitWebServerSet(doc.Content)
	if err != nil {
		return nil, nil, err
	}
	if set, err = yaml.Marshal(fields); err != nil || len(raw) == 0 {
		return set, nil, err
	}
	parsed, err := deploymentSet(doc)
	if err != nil {
		return nil, nil, err
	}
	m, err := parseWebLocalMap(raw, parsed)
	if err != nil {
		return nil, nil, err
	}
	localmap, err = yaml.Marshal(m)
	return set, localmap, err
}

// writeWebServerSet writes a server-set document as the engine reads it: the
// server set at setPath and, for a docker pool, localmap.yaml beside it.
func writeWebServerSet(ctx context.Context, doc DeploymentDocumentInput, setPath string) error {
	set, localmap, err := webServerSetFiles(doc)
	if err != nil {
		return err
	}
	files := filestore.Local{}
	if err = files.Write(ctx, setPath, set, 0o600); err != nil || localmap == nil {
		return err
	}
	return files.Write(ctx, resource.LocalMapNear(setPath), localmap, 0o600)
}
