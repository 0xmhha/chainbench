// Command dockerprepare prepares only the Docker server selected by the live
// regression's server set. It never starts containers or rewrites existing data.
package main

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"io/fs"
	"os"
	"os/exec"
	"path/filepath"
	"regexp"
	"strconv"
	"strings"
	"time"

	"github.com/0xmhha/chainbench/internal/resource"
)

type container struct {
	ID              string `json:"Id"`
	State           struct{ Running bool }
	Config          struct{ Labels map[string]string }
	NetworkSettings struct {
		Networks map[string]struct{ IPAddress string }
		Ports    map[string][]struct {
			HostIP   string `json:"HostIp"`
			HostPort string
		}
	}
}

func main() {
	ctx, cancel := context.WithTimeout(context.Background(), 2*time.Minute)
	defer cancel()
	if err := prepare(ctx); err != nil {
		fmt.Fprintln(os.Stderr, err)
		os.Exit(1)
	}
}

func docker(ctx context.Context, args ...string) ([]byte, error) {
	out, err := exec.CommandContext(ctx, "docker", args...).CombinedOutput()
	if err != nil {
		return nil, fmt.Errorf("docker %s failed: %w", args[0], err)
	}
	return out, nil
}

func selectServer(ctx context.Context) (string, string, error) {
	build := os.Getenv("CHAINBENCH_DOCKER_SERVERS")
	if build == "" {
		return "", "", errors.New("CHAINBENCH_DOCKER_SERVERS must select the live regression build directory")
	}
	build, err := filepath.EvalSymlinks(build)
	if err != nil {
		return "", "", err
	}
	build, err = filepath.Abs(build)
	if err != nil {
		return "", "", err
	}
	set, err := resource.LoadSet(filepath.Join(build, "server-set.yaml"))
	if err != nil {
		return "", "", errors.New("cannot load selected server set")
	}
	srv, err := set.ByName("server1")
	if err != nil {
		return "", "", err
	}
	if !srv.IsRemote() || !srv.SSH.Sudo || !regexp.MustCompile(`^[a-z_][a-z0-9_-]*$`).MatchString(srv.SSH.User) || srv.SSH.User == "root" {
		return "", "", errors.New("server1 must have an ordinary sudo login")
	}
	lm, err := resource.LoadLocalMap(resource.LocalMapNear(filepath.Join(build, "server-set.yaml")))
	if err != nil {
		return "", "", errors.New("cannot load selected local map")
	}
	mapping, ok := lm.Hosts[srv.Host]
	port := srv.SSH.Port
	if port == 0 {
		port = 22
	}
	published, mapped := mapping.Ports[port]
	if !ok || mapping.Host != "127.0.0.1" || !mapped || published < 1 || published > 65535 {
		return "", "", errors.New("server1 needs an explicit loopback SSH mapping")
	}
	compose := filepath.Join(build, "docker-compose.yml")
	if info, err := os.Lstat(compose); err != nil || !info.Mode().IsRegular() {
		return "", "", errors.New("selected compose file must be regular")
	}
	out, err := docker(ctx, "compose", "-f", compose, "ps", "--all", "--quiet", "server1")
	if err != nil {
		return "", "", err
	}
	ids := strings.Fields(string(out))
	if len(ids) != 1 || !regexp.MustCompile(`^[0-9a-f]{64}$`).MatchString(ids[0]) {
		return "", "", errors.New("selected compose must identify exactly one server1 container")
	}
	out, err = docker(ctx, "inspect", ids[0])
	if err != nil {
		return "", "", err
	}
	var entries []container
	if err := json.Unmarshal(out, &entries); err != nil || len(entries) != 1 {
		return "", "", errors.New("invalid Docker inspection")
	}
	c := entries[0]
	if c.ID != ids[0] || !c.State.Running || c.Config.Labels["com.docker.compose.service"] != "server1" || c.Config.Labels["com.docker.compose.project.config_files"] != compose || c.Config.Labels["com.docker.compose.project.working_dir"] != build {
		return "", "", errors.New("container does not belong to selected compose server1")
	}
	addressOK := false
	for _, n := range c.NetworkSettings.Networks {
		addressOK = addressOK || n.IPAddress == srv.Host
	}
	bindings := c.NetworkSettings.Ports[strconv.Itoa(port)+"/tcp"]
	if !addressOK || len(bindings) != 1 || bindings[0].HostIP != mapping.Host || bindings[0].HostPort != strconv.Itoa(published) {
		return "", "", errors.New("container identity disagrees with server set or SSH local map")
	}
	return c.ID, srv.SSH.User, nil
}

func stageFiles(dir string) error {
	if err := filepath.WalkDir("presets/keys", func(path string, entry fs.DirEntry, err error) error {
		if err != nil {
			return err
		}
		rel, err := filepath.Rel("presets/keys", path)
		if err != nil {
			return err
		}
		target := filepath.Join(dir, "kr-src", rel)
		if entry.IsDir() {
			return os.MkdirAll(target, 0o755)
		}
		info, err := entry.Info()
		if err != nil {
			return err
		}
		if !info.Mode().IsRegular() {
			return errors.New("key fixture contains a non-regular entry")
		}
		raw, err := os.ReadFile(path)
		if err != nil {
			return err
		}
		return os.WriteFile(target, raw, 0o600)
	}); err != nil {
		return err
	}
	var log strings.Builder
	log.WriteString("start\n")
	for i := 1; i <= 400; i++ {
		fmt.Fprintln(&log, i)
	}
	log.WriteString("last-line\n")
	if err := os.MkdirAll(filepath.Join(dir, "cb-kr-test/node1"), 0o700); err != nil {
		return err
	}
	for path, content := range map[string]string{
		"nodelog-test.log": log.String(), "cb-sudo-test.txt": "root-only-test-content-12345\n",
		"cb-kr-test/password": "pw\n", "cb-kr-test/metadata.json": "meta\n", "cb-kr-test/node1/keystore.json": "k1\n",
	} {
		if err := os.WriteFile(filepath.Join(dir, path), []byte(content), 0o600); err != nil {
			return err
		}
	}
	return nil
}

func prepare(ctx context.Context) (result error) {
	id, user, err := selectServer(ctx)
	if err != nil {
		return err
	}
	if _, err = docker(ctx, "exec", id, "test", "-x", "/data/chainbench/bin/gstable"); err != nil {
		return errors.New("selected server has no gstable binary")
	}
	local, err := os.MkdirTemp("", "chainbench-regression-")
	if err != nil {
		return err
	}
	defer func() { _ = os.RemoveAll(local) }()
	if err = stageFiles(local); err != nil {
		return err
	}
	out, err := docker(ctx, "exec", "-u", "root", id, "mktemp", "-d", "/tmp/chainbench-regression.XXXXXXXXXX")
	if err != nil {
		return err
	}
	remote := strings.TrimSpace(string(out))
	if !regexp.MustCompile(`^/tmp/chainbench-regression\.[A-Za-z0-9]+$`).MatchString(remote) {
		return errors.New("unexpected private staging path")
	}
	defer func() {
		cleanup, cancel := context.WithTimeout(context.Background(), 10*time.Second)
		defer cancel()
		if _, err := docker(cleanup, "exec", "-u", "root", id, "rm", "-rf", "--", remote); err != nil {
			result = errors.Join(result, errors.New("private Docker staging cleanup failed"))
		}
	}()
	if _, err = docker(ctx, "cp", local+"/.", id+":"+remote+"/"); err != nil {
		return err
	}
	if _, err = docker(ctx, "exec", "-u", "root", id, "sh", "-eu", "-c", installScript, "prepare", remote, user); err != nil {
		return err
	}
	fmt.Printf("prepared selected server1 container=%s login=%s\n", id, user)
	return nil
}

// Existing fixtures are compared before any destination changes. No existing
// file is overwritten, deleted, chmodded or chowned, including on refusal.
const installScript = `
stage=$1
user=$2
for parent in /data /data/chainbench /data/chainbench/keys /root; do
  [ ! -L "$parent" ] || exit 20
  if [ -e "$parent" ]; then [ -d "$parent" ] || exit 20; fi
done
chown -R "$user:" "$stage/kr-src" "$stage/nodelog-test.log"
chmod 644 "$stage/nodelog-test.log"
find "$stage/kr-src" -type d -exec chmod 755 {} +
find "$stage/kr-src" -type f -exec chmod 644 {} +
chown -R root:root "$stage/cb-kr-test" "$stage/cb-sudo-test.txt"
find "$stage/cb-kr-test" -type d -exec chmod 700 {} +
find "$stage/cb-kr-test" -type f -exec chmod 600 {} +
chmod 600 "$stage/cb-sudo-test.txt"
check() {
  src=$1 dst=$2
  [ ! -L "$dst" ] || exit 21
  if [ -e "$dst" ]; then
    diff -qr --no-dereference "$src" "$dst" >/dev/null || exit 22
    if [ -d "$src" ]; then
      [ -d "$dst" ] || exit 22
      (cd "$src"; find . -printf '%P %m %U %G %y\n' | LC_ALL=C sort) > "$stage/expected"
      (cd "$dst"; find . -printf '%P %m %U %G %y\n' | LC_ALL=C sort) > "$stage/actual"
      cmp "$stage/expected" "$stage/actual" || exit 23
    else
      [ "$(stat -c '%a %u %g %F' "$src")" = "$(stat -c '%a %u %g %F' "$dst")" ] || exit 23
    fi
  fi
}
check "$stage/kr-src" /data/chainbench/keys/kr-src
check "$stage/nodelog-test.log" /data/chainbench/nodelog-test.log
check "$stage/cb-sudo-test.txt" /root/cb-sudo-test.txt
check "$stage/cb-kr-test" /root/cb-kr-test
for parent in /data/chainbench /data/chainbench/keys; do
  if [ ! -e "$parent" ]; then mkdir "$parent"; chown "$user:" "$parent"; fi
done
install_missing() { if [ ! -e "$2" ]; then mv -T -n "$1" "$2"; fi; }
install_missing "$stage/kr-src" /data/chainbench/keys/kr-src
install_missing "$stage/nodelog-test.log" /data/chainbench/nodelog-test.log
install_missing "$stage/cb-sudo-test.txt" /root/cb-sudo-test.txt
install_missing "$stage/cb-kr-test" /root/cb-kr-test
`
