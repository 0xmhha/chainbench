package chainsetup

import (
	"fmt"
	"path/filepath"
	"strings"

	"github.com/0xmhha/chainbench/internal/resource"
)

// The binary a node runs is named in up to four places, and before this file
// decided between them the last one won by default: whatever a declaration
// wrote went to exec unchecked. That is how the same chain came to be spelled
// two ways across the specs, and how a target-side absolute path ended up in a
// test definition — both spellings ran, so neither was ever wrong.
//
// The order here is the one an operator expects, most specific first:
//
//	1. what this command was given (--binary), which may be a path
//	2. what the composition recorded at `chain new`
//	3. the chain manifest's own name for it (gstable, gwbft, gwemix)
//
// The result is a NAME until it is placed. Placing is the second half, and it
// is what a workspace-config is for: binaryAliases says which file this
// environment calls that name, and paths.binaries says where the files live
// under the target's data root. Without a workspace-config a name stays a name,
// which means the target finds it on its PATH — the same thing exec would do,
// said out loud so the pre-launch check can agree with it.

// binary resolves which binary this command runs, as a target path or a name
// the target's PATH will resolve.
func (w *Workspace) binary(arg string) (string, error) {
	ref := arg
	if ref == "" {
		ref = w.state.Binary
	}
	if ref == "" {
		name, err := w.manifestBinary()
		if err != nil {
			return "", err
		}
		ref = name
	}
	return w.placeBinary(ref)
}

// manifestBinary is the chain's own name for its node binary. A declaration
// that omits one is not incomplete: the chain already says what it is called,
// and repeating it in every test definition is how two spellings of one binary
// got into the specs.
func (w *Workspace) manifestBinary() (string, error) {
	p, err := w.plugin()
	if err != nil {
		return "", fmt.Errorf("chainsetup: a node binary is required and the chain is not known either: %w", err)
	}
	name := p.Manifest().Binary
	if name == "" {
		return "", fmt.Errorf("chainsetup: chain %q names no binary, so one must be given (--binary)", p.Manifest().ID)
	}
	return name, nil
}

// placeBinary turns a reference into what the target will run.
//
// An absolute path is already placed and is used as given: it comes from an
// operator's --binary, and second-guessing an explicit path is worse than
// running it. A name is placed through the workspace-config when there is one,
// and otherwise left for the target's PATH.
func (w *Workspace) placeBinary(ref string) (string, error) {
	wc, err := w.wc()
	if err != nil {
		return "", err
	}
	return PlaceBinary(ref, wc)
}

// PlaceBinary turns a binary reference into what the target will run, under the
// environment wc describes. A nil wc is no environment file.
//
// Exported because the plan has to say the same thing. A plan that records the
// name while the launch records the placed path makes the two disagree about a
// run that is correct, and the pre-test comparison then refuses it: "asked for
// binary gstable, launched with /data/net1/bin/gstable". The check was right and
// the plan was reading one layer short — a workspace-config exists to say where
// the file actually is, so the plan has to go through it too.
func PlaceBinary(ref string, wc *resource.WorkspaceConfig) (string, error) {
	if ref == "" {
		return "", fmt.Errorf("chainsetup: a node binary is required (--binary, or set it at `chain new`)")
	}
	if filepath.IsAbs(ref) {
		return ref, nil
	}
	if wc == nil {
		// No environment file: the name is the target's to resolve. A relative
		// path with a separator is neither a name nor a placed path, so it is
		// refused rather than resolved against whatever directory happens to be
		// current when the launch runs.
		if strings.ContainsRune(ref, '/') {
			return "", fmt.Errorf("chainsetup: binary %q is a relative path — give a name the target's PATH resolves, or an absolute path, or a workspace-config that says where binaries live", ref)
		}
		return ref, nil
	}
	path, err := wc.BinaryPath(ref)
	if err != nil {
		return "", fmt.Errorf("chainsetup: binary %q: %w", ref, err)
	}
	return path, nil
}

// PlaceRequest turns every binary reference in a compose request into what the
// target will run, once, before the request is recorded, compared or launched.
//
// It exists because placing used to happen only at the last possible moment,
// inside init and start. Everything that asks a question ABOUT the launch was
// then asking it in the other language. Three things went wrong at once and
// none of them looked related:
//
//   - the plan promised "gstable" while the launch recorded /data/bin/gstable,
//     so the pre-test comparison refused a correct run;
//   - the per-node binaries map was stored as declared and handed to exec, so a
//     node the topology assigned a binary ran a bare name while every other node
//     in the same network ran the placed path;
//   - preflight compared a recorded placed path against a requested name, so an
//     identical request answered rebuild-all and the chain was recomposed every
//     single time.
//
// One form, decided once, at the edge. PlaceBinary is idempotent — an absolute
// path places to itself — so a later caller that places again is harmless, and
// no single site is load-bearing.
//
// A nil wc is no environment file, and a name then stays a name for the
// target's PATH to resolve.
func PlaceRequest(in *NetUpIn, wc *resource.WorkspaceConfig) error {
	if in == nil {
		return nil
	}
	if in.Binary != "" {
		placed, err := PlaceBinary(in.Binary, wc)
		if err != nil {
			return err
		}
		in.Binary = placed
	}
	if len(in.Binaries) == 0 {
		return nil
	}
	// A fresh map: the caller's may be shared, and a request being placed must
	// not rewrite a declaration somebody else is still reading.
	placed := make(map[string]string, len(in.Binaries))
	for name, ref := range in.Binaries {
		p, err := PlaceBinary(ref, wc)
		if err != nil {
			return fmt.Errorf("chainsetup: binaries.%s: %w", name, err)
		}
		placed[name] = p
	}
	in.Binaries = placed
	return nil
}
