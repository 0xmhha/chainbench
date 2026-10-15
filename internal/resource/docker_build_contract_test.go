package resource_test

import (
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"testing"
)

// A successful container exit does not prove that its output crossed the
// Docker Desktop mount. An older host ELF must never stand in for that output.
func TestDockerBuildPublishesOnlyFreshELF(t *testing.T) {
	for _, mode := range []string{"missing", "invalid", "failed", "failedELF", "fresh"} {
		t.Run(mode, func(t *testing.T) {
			root := t.TempDir()
			write := func(path, body string, mode os.FileMode) {
				t.Helper()
				if err := os.MkdirAll(filepath.Dir(path), 0700); err != nil {
					t.Fatal(err)
				}
				if err := os.WriteFile(path, []byte(body), mode); err != nil {
					t.Fatal(err)
				}
			}
			body, err := os.ReadFile(filepath.Join("..", "..", "env", "docker", "setup.sh"))
			if err != nil {
				t.Fatal(err)
			}
			script := filepath.Join(root, "setup.sh")
			write(script, string(body), 0700)
			write(filepath.Join(root, "build/docker-compose.yml"), "services: {}\n", 0600)
			if err := os.MkdirAll(filepath.Join(root, "chain/go-stablenet"), 0700); err != nil {
				t.Fatal(err)
			}
			artifact := filepath.Join(root, "artifacts/gstable")
			write(artifact, "\x7fELFold", 0700)
			placed := filepath.Join(root, "placed.bin")
			write(filepath.Join(root, "tools/docker"), `#!/usr/bin/env bash
set -eu
case "$1" in
 compose) echo server1 ;;
 run)
  shift
  output=''
  bounded_cpu=0
  bounded_parallel=0
  while [ "$#" -gt 0 ]; do
   if [ "$1" = -v ]; then
    case "$2" in *:/out) output=${2%:/out} ;; esac
    shift 2
   elif [ "$1" = -e ]; then
    case "$2" in GOMAXPROCS=2) bounded_cpu=1 ;; GOFLAGS=-p=1) bounded_parallel=1 ;; esac
    shift 2
   else shift; fi
  done
  test -n "$output"
  if [ "$bounded_cpu" != 1 ] || [ "$bounded_parallel" != 1 ]; then exit 98; fi
  case "$BUILD_RESULT" in
   missing) : ;;
   invalid) printf 'MachO' > "$output/gstable" ;;
   failed) printf 'partial' > "$output/gstable"; exit 1 ;;
   failedELF) printf '\177ELFpartial' > "$output/gstable"; exit 1 ;;
   fresh) printf '\177ELFfresh' > "$output/gstable" ;;
  esac ;;
 cp) cat "$2" > "$PLACED_BINARY" ;;
 exec) : ;;
 *) echo unexpected docker command >&2; exit 2 ;;
esac
`, 0700)
			cmd := exec.Command("bash", script, "binaries", "go-stablenet")
			cmd.Env = append(os.Environ(), "PATH="+filepath.Join(root, "tools")+string(os.PathListSeparator)+os.Getenv("PATH"), "CHAIN_SRC_ROOT="+filepath.Join(root, "chain"), "ARTIFACT_DIR="+filepath.Join(root, "artifacts"), "BUILD_RESULT="+mode, "PLACED_BINARY="+placed, "BUILD=build", "GOMAXPROCS=2", "GOFLAGS=-p=1")
			output, err := cmd.CombinedOutput()
			if mode == "fresh" {
				if err != nil {
					t.Fatal(err, string(output))
				}
				b, e := os.ReadFile(placed)
				if e != nil || string(b) != "\x7fELFfresh" {
					t.Fatal("fresh output not placed", e, string(b), string(output))
				}
			} else {
				if err == nil {
					t.Error("container output missing or invalid but stale ELF was accepted", string(output))
				}
				if _, e := os.Stat(placed); !os.IsNotExist(e) {
					t.Error("unverified output was sent to a server", e)
				}
				b, e := os.ReadFile(artifact)
				if e != nil || string(b) != "\x7fELFold" {
					t.Error("previous artifact changed on failed build", e, string(b))
				}
			}
			entries, err := os.ReadDir(filepath.Dir(artifact))
			if err != nil {
				t.Fatal(err)
			}
			for _, e := range entries {
				if strings.HasPrefix(e.Name(), ".build-") {
					t.Error("owned build staging directory was not reclaimed", e.Name())
				}
			}
		})
	}
}
