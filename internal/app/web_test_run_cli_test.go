package app

import (
	"context"
	"encoding/json"
	"errors"
	"os"
	"path/filepath"
	"slices"
	"strings"
	"testing"

	"github.com/0xmhha/chainbench/internal/dsl"
	"github.com/0xmhha/chainbench/internal/resource"
)

// runCLI records the command it was given and answers with a fixed document
// and exit error, the way `chainbench run --json` answers.
type runCLI struct {
	args []string
	out  string
	err  error
	// files keeps what each case file held while the command ran.
	files map[string]string
}

func (c *runCLI) ChainStatus(context.Context, string) (State, error) { return State{}, nil }

func (c *runCLI) Run(_ context.Context, args ...string) ([]byte, error) {
	c.args = args
	c.files = map[string]string{}
	for _, a := range args {
		if strings.HasSuffix(a, ".json") {
			b, _ := os.ReadFile(a)
			c.files[a] = string(b)
		}
	}
	return []byte(c.out), c.err
}

func flagValue(args []string, flag string) string {
	if i := slices.Index(args, flag); i >= 0 && i+1 < len(args) {
		return args[i+1]
	}
	return ""
}

// TestWebRunSuiteThroughCLIPassesTheReviewedInputs: the reviewed cases, the
// accepted read-only keys and every bound binary reach `chainbench run`, and
// the session it reports becomes the job's run.
func TestWebRunSuiteThroughCLIPassesTheReviewedInputs(t *testing.T) {
	dir := t.TempDir()
	in := RunSuiteIn{SpecContent: [][]byte{[]byte(`{"first":1}`), []byte(`{"second":2}`)}, DataDir: "/control", Chain: "stablenet",
		Binary: "/bin/gstable", BinaryOverrides: map[string]string{dsl.BinaryDefault: "/bin/gstable", "next": "/bin/gstable-next"},
		KeysDir: "/keys", ReadOnlyKeys: true, KeepUp: true, ArtifactRoot: "/sessions", WorkspaceConfigPath: "/inputs/workspace-config.yaml",
		Server: resource.ServerRef{SetPath: "/inputs/server-set.yaml", Name: "ssh"}}
	cli := &runCLI{out: `{"session":"/sessions/20261012-1-1/UTC-1","setupSteps":["keys: reused"],"tests":[],"summary":{"pass":1,"fail":0,"blocked":0,"skip":0}}`}
	out, err := runSuiteThroughCLI(context.Background(), cli, in, dir, "/job/server-set.yaml")
	if err != nil {
		t.Fatal(err)
	}
	if out.SessionRoot != "/sessions/20261012-1-1/UTC-1" || len(out.SetupSteps) != 1 || out.Summary.Summary.Pass != 1 {
		t.Fatalf("run result = %+v", out)
	}
	args := cli.args
	// --suite keeps the reviewed cases one run on one network with one session.
	if args[0] != "run" || !slices.Contains(args, "--json") || !slices.Contains(args, "--suite") || !slices.Contains(args, "--read-only-keys") || !slices.Contains(args, "--keep-up") {
		t.Fatalf("args = %v", args)
	}
	for flag, want := range map[string]string{"--workspace-dir": "/control", "--chain": "stablenet", "--binary": "/bin/gstable", "--keys": "/keys",
		"--artifact-root": "/sessions", "--workspace-config": "/inputs/workspace-config.yaml", "--server": "ssh",
		// The network records the set without secrets; the job's logins are
		// named apart so the record never points at a file removed after the run.
		"--server-set": "/inputs/server-set.yaml", "--server-logins": "/job/server-set.yaml"} {
		if got := flagValue(args, flag); got != want {
			t.Errorf("%s = %q, want %q", flag, got, want)
		}
	}
	if !slices.Contains(args, "next=/bin/gstable-next") {
		t.Errorf("named binary was not bound: %v", args)
	}
	var contents []string
	for _, c := range cli.files {
		contents = append(contents, c)
	}
	slices.Sort(contents)
	if !slices.Equal(contents, []string{`{"first":1}`, `{"second":2}`}) {
		t.Fatalf("case files held %v, want the reviewed content", contents)
	}
}

// TestWebRunSuiteThroughCLITellsFailedTestsFromABrokenRun: a test that ran and
// failed exits non-zero with a complete document, which is a verdict, not an
// error; a run that could not compose reports the CLI's own reason.
func TestWebRunSuiteThroughCLITellsFailedTestsFromABrokenRun(t *testing.T) {
	in := RunSuiteIn{SpecContent: [][]byte{[]byte(`{}`)}, DataDir: "/control"}
	failed := &runCLI{out: `{"session":"/s/x/UTC-1","tests":[],"summary":{"pass":0,"fail":1,"blocked":0,"skip":0}}`, err: errors.New("chainbench run: run: 1 test(s) failed")}
	out, err := runSuiteThroughCLI(context.Background(), failed, in, t.TempDir(), "")
	if err != nil || !out.Summary.Failed() || out.SessionRoot != "/s/x/UTC-1" {
		t.Fatalf("failed tests gave %+v %v, want a failed verdict without an error", out, err)
	}
	broken := &runCLI{out: `{"session":"","error":"chainsetup: init: node1: binary missing","failedAt":"chain/init","setupSteps":["keys: reused"]}`, err: errors.New("chainbench run: exit 2")}
	out, err = runSuiteThroughCLI(context.Background(), broken, in, t.TempDir(), "")
	if err == nil || !strings.Contains(err.Error(), "binary missing") || len(out.SetupSteps) != 1 {
		t.Fatalf("broken run gave %+v %v, want the CLI's reason and its setup steps", out, err)
	}
	crashed := &runCLI{err: errors.New("chainbench run: signal: killed")}
	if _, err = runSuiteThroughCLI(context.Background(), crashed, in, t.TempDir(), ""); err == nil || !strings.Contains(err.Error(), "killed") {
		t.Fatalf("a run with no document gave %v, want the process error", err)
	}
}

// TestWebJobServerSetCarriesEachBoundCredentialPrivately: the job's server set
// names each bound server's own login through private files, which the CLI's
// own loader accepts, and nothing secret is written into the set itself.
func TestWebJobServerSetCarriesEachBoundCredentialPrivately(t *testing.T) {
	s, a, w, c := leaseFixture(t)
	set, err := s.Document(w.Documents[0].ID)
	if err != nil {
		t.Fatal(err)
	}
	dir := filepath.Join(t.TempDir(), "secrets")
	path, err := s.writeJobServerSet(context.Background(), a, set.DeploymentDocumentInput, map[string]string{"ssh": c.ID}, dir)
	if err != nil {
		t.Fatal(err)
	}
	if info, err := os.Stat(dir); err != nil || info.Mode().Perm() != 0o700 {
		t.Fatalf("secret directory: %v %v", info, err)
	}
	raw, err := os.ReadFile(path)
	if err != nil {
		t.Fatal(err)
	}
	if strings.Contains(string(raw), "lease-secret-password") {
		t.Fatal("the password was written into the server set")
	}
	loaded, err := resource.LoadSet(path)
	if err != nil {
		t.Fatal(err)
	}
	server, err := loaded.ByName("ssh")
	if err != nil {
		t.Fatal(err)
	}
	creds, err := server.Credentials()
	if err != nil {
		t.Fatal(err)
	}
	if creds.User != "ssh" || creds.Password != "lease-secret-password" {
		t.Fatalf("the CLI would log in as %s:%s", creds.User, creds.Password)
	}
	entries, _ := os.ReadDir(dir)
	for _, e := range entries {
		info, _ := e.Info()
		if info.Mode().Perm() != 0o600 {
			t.Errorf("%s is %v, want 0600", e.Name(), info.Mode().Perm())
		}
	}
	var doc map[string]any
	if err = json.Unmarshal(set.Content, &doc); err != nil {
		t.Fatal(err)
	}
}

// TestWebAttachThroughCLIReadsTheRecordedNetworkOnly: an attach job runs the
// reviewed cases against the network the Web recorded, for reading only, with
// the accepted keys unchanged and the job's own logins.
func TestWebAttachThroughCLIReadsTheRecordedNetworkOnly(t *testing.T) {
	cli := &runCLI{out: `{"session":"/sessions/a/UTC-1","tests":[],"summary":{"pass":1,"fail":0,"blocked":0,"skip":0}}`}
	out, err := attachThroughCLI(context.Background(), cli, [][]byte{[]byte(`{"attach":1}`)}, "/control", "/sessions", t.TempDir(), "/job/server-set.yaml")
	if err != nil || out.SessionRoot != "/sessions/a/UTC-1" {
		t.Fatalf("attach = %+v %v", out, err)
	}
	args := cli.args
	for _, flag := range []string{"--attach", "--no-node-control", "--read-only-keys", "--json"} {
		if !slices.Contains(args, flag) {
			t.Errorf("%s missing from %v", flag, args)
		}
	}
	for flag, want := range map[string]string{"--workspace-dir": "/control", "--artifact-root": "/sessions", "--server-logins": "/job/server-set.yaml"} {
		if got := flagValue(args, flag); got != want {
			t.Errorf("%s = %q, want %q", flag, got, want)
		}
	}
	if slices.Contains(args, "--suite") || slices.Contains(args, "--keep-up") || slices.Contains(args, "--server-set") {
		t.Errorf("compose-only flags reached an attach: %v", args)
	}
}
