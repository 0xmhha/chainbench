package app

import (
	"bytes"
	"context"
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"testing"

	"github.com/0xmhha/chainbench/internal/core/registry"
)

// This is a native unit fixture, not a blockchain or native acceptance proof.
// Its source is committed in its own disposable Git repository, so Go embeds a
// real revision. It implements inspection only; any node command exits nonzero.
func nativeVCSFixture(t *testing.T, dirty, omitVCS bool) (string, string) {
	t.Helper()
	root := t.TempDir()
	repo := filepath.Join(root, "source")
	if err := os.Mkdir(repo, 0o700); err != nil {
		t.Fatal(err)
	}
	plugin, err := registry.Get("stablenet")
	if err != nil {
		t.Fatal(err)
	}
	help := string(webHelpOf(webSurfaceCatalog(t, "stablenet").Global, ""))
	program := fmt.Sprintf(`package main
import ("fmt"; "os"; "runtime")
func main() {
 if len(os.Args)!=2 { os.Exit(2) }
 switch os.Args[1] {
 case "version": fmt.Println(%q); fmt.Println("Operating System: "+runtime.GOOS); fmt.Println("Architecture: "+runtime.GOARCH)
 case "--help": fmt.Print(%q)
 default: os.Exit(2)
 }
}
`, plugin.Manifest().Build.MakeTarget, help)
	for name, body := range map[string]string{"go.mod": "module example.com/chainbench-native-fixture\n\ngo 1.26\n", "main.go": program} {
		if err := os.WriteFile(filepath.Join(repo, name), []byte(body), 0o600); err != nil {
			t.Fatal(err)
		}
	}
	run := func(args ...string) string {
		t.Helper()
		cmd := exec.Command(args[0], args[1:]...)
		cmd.Dir = repo
		cmd.Env = append(os.Environ(), "GOWORK=off")
		out, err := cmd.CombinedOutput()
		if err != nil {
			t.Fatalf("native unit fixture %s: %v: %s", args[0], err, out)
		}
		return strings.TrimSpace(string(out))
	}
	run("git", "init", "-q", "-b", "codex/native-fixture")
	run("git", "add", "go.mod", "main.go")
	run("git", "-c", "user.name=Fixture", "-c", "user.email=fixture@example.invalid", "-c", "commit.gpgSign=false", "-c", "core.hooksPath=/dev/null", "commit", "-q", "-m", "test: seed native identity fixture")
	commit := run("git", "rev-parse", "HEAD")
	if dirty {
		if err := os.WriteFile(filepath.Join(repo, "main.go"), []byte(program+"\n// local change\n"), 0o600); err != nil {
			t.Fatal(err)
		}
	}
	binary := filepath.Join(root, "native")
	vcs := "true"
	if omitVCS {
		vcs = "false"
	}
	run("go", "build", "-buildvcs="+vcs, "-o", binary, ".")
	return binary, commit
}

func TestNativeGoVCSIdentityReachesManifestAndUpload(t *testing.T) {
	binary, commit := nativeVCSFixture(t, false, false)
	raw, err := os.ReadFile(binary)
	if err != nil {
		t.Fatal(err)
	}
	t.Run("manifest", func(t *testing.T) {
		asset := ManifestBinary{ID: "fixture", Chain: "stablenet", Path: binary, SHA256: manifestHash(raw), Commit: commit}
		observed, err := asset.Verify(context.Background(), "stablenet")
		if err != nil {
			t.Fatalf("verified clean Go VCS identity rejected: %v", err)
		}
		if observed.SHA256 != asset.SHA256 || observed.Chain != "stablenet" {
			t.Fatalf("wrong observed identity: %+v", observed)
		}
	})
	t.Run("upload", func(t *testing.T) {
		store, err := OpenManifestStore(t.TempDir())
		if err != nil {
			t.Fatal(err)
		}
		uploaded, err := store.UploadAsset(context.Background(), DeploymentActor{ID: "operator", Role: "operator"}, "binary", "native", bytes.NewReader(raw))
		if err != nil {
			t.Fatalf("verified clean Go VCS upload rejected: %v", err)
		}
		if uploaded.Compatibility["commit"] != commit || uploaded.Compatibility["chain"] != "stablenet" {
			t.Fatalf("wrong registered identity: %+v", uploaded)
		}
	})
}

func TestNativeGoVCSIdentityRejectsModifiedAndMissingRevision(t *testing.T) {
	for _, test := range []struct {
		name        string
		dirty, omit bool
	}{{"modified", true, false}, {"missing", false, true}} {
		t.Run(test.name, func(t *testing.T) {
			binary, commit := nativeVCSFixture(t, test.dirty, test.omit)
			raw, err := os.ReadFile(binary)
			if err != nil {
				t.Fatal(err)
			}
			asset := ManifestBinary{ID: "fixture", Chain: "stablenet", Path: binary, SHA256: manifestHash(raw), Commit: commit}
			if _, err = asset.Verify(context.Background(), "stablenet"); err == nil {
				t.Fatal("unverified source accepted")
			}
			store, err := OpenManifestStore(t.TempDir())
			if err != nil {
				t.Fatal(err)
			}
			if _, err = store.UploadAsset(context.Background(), DeploymentActor{ID: "operator", Role: "operator"}, "binary", "native", bytes.NewReader(raw)); err == nil {
				t.Fatal("unverified upload accepted")
			}
		})
	}
}

func TestManifestLegacyStampRemainsAuthoritativeAndUnambiguous(t *testing.T) {
	commit := strings.Repeat("a", 40)
	for _, test := range []struct {
		name, version string
		valid         bool
	}{
		{"legacy", "Gstable\nGit Commit: " + commit + "\n", true},
		{"malformed", "Gstable\nGit Commit: not-a-commit\n", false},
		{"duplicate", "Git Commit: " + commit + "\nGit Commit: " + strings.Repeat("b", 40) + "\n", false},
	} {
		t.Run(test.name, func(t *testing.T) {
			got, err := manifestSourceCommit("missing-no-metadata-fallback", []byte(test.version))
			if test.valid {
				if err != nil || got != commit {
					t.Fatalf("legacy source identity changed: %q %v", got, err)
				}
			} else if err == nil {
				t.Fatalf("ambiguous identity accepted: %q", got)
			}
		})
	}
}
