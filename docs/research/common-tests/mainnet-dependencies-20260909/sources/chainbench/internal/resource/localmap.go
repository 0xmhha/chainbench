package resource

import (
	"fmt"
	"net"
	"os"
	"path/filepath"
	"strconv"
	"sync"

	"go.yaml.in/yaml/v3"

	"github.com/0xmhha/chainbench/internal/core/remote"
)

// LocalMapFile is the mapping file's conventional name, looked for next to
// the server set it translates.
const LocalMapFile = "localmap.yaml"

// LocalMap translates servers' real addresses onto addresses this machine can
// dial — loopback-published docker ports. The server set keeps the real
// addresses (the file has the same shape a production server set has, and the
// nodes use those addresses to reach each other); this map applies only to
// the harness's own dials, and only when the operator passes --docker.
//
// The file's presence alone activates nothing: a leftover map must not be
// able to redirect a real-remote run.
type LocalMap struct {
	// Hosts maps a server's real address onto how it is reached from here.
	Hosts map[string]HostPorts `yaml:"hosts"`
}

// HostPorts is one server's local substitute address.
type HostPorts struct {
	// Host replaces the address itself (typically 127.0.0.1).
	Host string `yaml:"host"`
	// Ports maps the set-uniform ports onto the per-server published ones.
	// A port with no entry passes through unchanged.
	Ports map[int]int `yaml:"ports"`
}

// LocalMapNear returns the mapping file's path next to the server set at
// setPath (empty means the default server-set location).
func LocalMapNear(setPath string) string {
	if setPath == "" {
		setPath = DefaultSetFile
	}
	return filepath.Join(filepath.Dir(setPath), LocalMapFile)
}

// LoadLocalMap reads the mapping file. A missing file is an error: the caller
// only asks for the map when --docker was given, and silently proceeding
// unmapped would dial addresses that do not route.
func LoadLocalMap(path string) (*LocalMap, error) {
	raw, err := os.ReadFile(path)
	if err != nil {
		if os.IsNotExist(err) {
			return nil, fmt.Errorf("serverset: --docker needs %s (generate it with env/docker/gen-env.sh)", path)
		}
		return nil, fmt.Errorf("serverset: read %s: %w", path, err)
	}
	var m LocalMap
	if err := yaml.Unmarshal(raw, &m); err != nil {
		return nil, fmt.Errorf("serverset: parse %s: %w", path, err)
	}
	if len(m.Hosts) == 0 {
		return nil, fmt.Errorf("serverset: %s maps no hosts", path)
	}
	return &m, nil
}

// AddrMap returns the dial-time translation. onApply, when non-nil, is told
// each translation the first time it is applied ("172.30.0.11:22 →
// 127.0.0.1:2201"), so the harness never connects somewhere the operator
// cannot see. An address the map does not know passes through unchanged.
func (m *LocalMap) AddrMap(onApply func(from, to string)) remote.AddrMap {
	var mu sync.Mutex // dials may be concurrent; the report cache must not race
	seen := map[string]bool{}
	return func(host string, port int) (string, int) {
		h, ok := m.Hosts[host]
		if !ok {
			return host, port
		}
		outHost, outPort := h.Host, port
		if mapped, found := h.Ports[port]; found {
			outPort = mapped
		}
		if onApply != nil {
			from := net.JoinHostPort(host, strconv.Itoa(port))
			mu.Lock()
			first := !seen[from]
			seen[from] = true
			mu.Unlock()
			if first {
				onApply(from, net.JoinHostPort(outHost, strconv.Itoa(outPort)))
			}
		}
		return outHost, outPort
	}
}
