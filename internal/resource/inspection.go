package resource

import (
	"context"
	"crypto/sha256"
	"encoding/hex"
	"errors"
	"path/filepath"
	"strings"

	"github.com/0xmhha/chainbench/internal/core/process"
	"github.com/0xmhha/chainbench/internal/core/remote"
)

// Inspection identifies a physical target independently of server names, DNS
// aliases and SSH ports, and resolves symlink ancestors of its declared root.
// Machine identifiers are hashed before leaving this resource boundary.
type Inspection struct {
	HostIdentity string `json:"hostIdentity"`
	DataPath     string `json:"dataPath"`
	OS           string `json:"os"`
	Architecture string `json:"architecture"`
	Transport    string `json:"transport"`
}

// Inspect performs only read operations. Linux machine-id and Darwin platform
// UUID must be available; unsupported or ambiguous identity is refused rather
// than allowing competing aliases to lock the same physical resources twice.
func (o Opener) Inspect(ctx context.Context, spec Spec) (Inspection, error) {
	var out Inspection
	if !filepath.IsAbs(spec.DataRoot) || strings.ContainsAny(spec.DataRoot, "\r\n\x00") {
		return out, errors.New("resource: inspection requires an absolute data root")
	}
	acc, err := o.Open(spec)
	if err != nil {
		return out, err
	}
	command := `set -eu
system=$(uname -s)
arch=$(uname -m)
case "$system" in
 Linux) identity=$(cat /etc/machine-id); system=linux ;;
 Darwin) identity=$(/usr/sbin/ioreg -rd1 -c IOPlatformExpertDevice | awk -F '"' '/"IOPlatformUUID" =/{print $(NF-1)}'); system=darwin ;;
 *) exit 2 ;;
esac
case "$arch" in arm64|aarch64) arch=arm64 ;; x86_64|amd64) arch=amd64 ;; *) exit 2 ;; esac
p=` + remote.ShellQuote(filepath.Clean(spec.DataRoot)) + `
suffix=''
while [ ! -e "$p" ]; do
 suffix="${p##*/}/$suffix"
 p=$(dirname "$p")
done
test -d "$p" && test -w "$p" && test -x "$p"
parent=$(cd "$p" && pwd -P)
resolved="$parent/${suffix%/}"
printf '%s\n%s\n%s\n%s\n' "$system" "$arch" "$identity" "$resolved"`
	var output string
	if acc.Runner == nil {
		out.Transport = "local"
		output, err = process.NewLocalDriver().Run(ctx, command)
	} else {
		out.Transport = "ssh"
		var result remote.ExecResult
		// A login shell can redefine cd or enable non-POSIX semantics. Run the
		// fixed inspection in its own POSIX shell without user shell functions.
		result, err = acc.Runner(ctx, "/bin/sh -c "+remote.ShellQuote(command))
		if err == nil && result.ExitCode != 0 {
			err = errors.New("inspection command failed")
		}
		output = result.Stdout
	}
	if err != nil {
		return Inspection{}, errors.New("resource: target identity, platform or data-root access could not be verified")
	}
	lines := strings.Split(strings.TrimSuffix(output, "\n"), "\n")
	if len(lines) != 4 || len(lines[2]) < 8 || !filepath.IsAbs(lines[3]) || (lines[0] != "linux" && lines[0] != "darwin") || (lines[1] != "arm64" && lines[1] != "amd64") {
		return Inspection{}, errors.New("resource: malformed target inspection")
	}
	for _, c := range lines[2] {
		if !strings.ContainsRune("0123456789abcdefABCDEF-", c) {
			return Inspection{}, errors.New("resource: unsupported machine identity")
		}
	}
	digest := sha256.Sum256([]byte(strings.ToLower(lines[0] + ":" + lines[2])))
	out.HostIdentity = "machine-sha256:" + hex.EncodeToString(digest[:])
	out.OS, out.Architecture, out.DataPath = lines[0], lines[1], filepath.Clean(lines[3])
	return out, nil
}
