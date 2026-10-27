package session

import (
	"errors"
	"io"
	"io/fs"
	"os"
	"path"
	"regexp"
	"sort"
	"strings"
	"syscall"
)

// EvidenceTest is one failed test's gathered evidence: the test directory
// relative to the session ("tests/<seq>_<id>") and its evidence files.
type EvidenceTest struct {
	Dir   string
	Files []EvidenceFile
}

// EvidenceFile is one evidence file and its size in bytes.
type EvidenceFile struct {
	Name  string
	Bytes int64
}

// maxEvidenceFile bounds one evidence read; a node log larger than this is
// refused rather than read into memory whole.
const maxEvidenceFile = 64 << 20

var (
	evidenceTestDir = regexp.MustCompile(`^tests/[0-9]{3}_[A-Za-z0-9._-]+$`)
	evidenceName    = regexp.MustCompile(`^(health\.json|processes\.json|gather-problems\.txt|node[0-9]+\.log)$`)
)

// openSessionRoot opens the session ref names under root, refusing a ref that
// leaves root or passes through a link, the same rule Capture applies.
func openSessionRoot(root, ref string) (*os.Root, error) {
	parts := strings.Split(ref, "/")
	if len(parts) > 2 {
		return nil, errors.New("invalid session reference")
	}
	for _, part := range parts {
		if !fs.ValidPath(part) || part == "." || strings.Contains(part, `\`) {
			return nil, errors.New("invalid session reference")
		}
	}
	base, err := os.OpenRoot(root)
	if err != nil {
		return nil, err
	}
	defer func() { _ = base.Close() }()
	for i := range parts {
		info, err := base.Lstat(path.Join(parts[:i+1]...))
		if err != nil || !info.IsDir() || info.Mode()&os.ModeSymlink != 0 {
			return nil, errors.New("session must be an owned directory without links")
		}
	}
	return base.OpenRoot(ref)
}

// ListEvidence lists the evidence failed tests of a session gathered, test by
// test in order. Only regular files with an evidence name are listed.
func ListEvidence(root, ref string) ([]EvidenceTest, error) {
	run, err := openSessionRoot(root, ref)
	if err != nil {
		return nil, err
	}
	defer func() { _ = run.Close() }()
	if !realDir(run, dirTests) {
		return []EvidenceTest{}, nil
	}
	tests, err := fs.ReadDir(run.FS(), dirTests)
	if errors.Is(err, fs.ErrNotExist) {
		return []EvidenceTest{}, nil
	}
	if err != nil {
		return nil, err
	}
	out := []EvidenceTest{}
	for _, test := range tests {
		dir := dirTests + "/" + test.Name()
		if !test.IsDir() || !evidenceTestDir.MatchString(dir) {
			continue
		}
		if !realDir(run, dir+"/"+dirObservations) {
			continue
		}
		entries, err := fs.ReadDir(run.FS(), dir+"/"+dirObservations)
		if err != nil {
			continue
		}
		var files []EvidenceFile
		for _, entry := range entries {
			if !entry.Type().IsRegular() || !evidenceName.MatchString(entry.Name()) {
				continue
			}
			info, err := run.Lstat(dir + "/" + dirObservations + "/" + entry.Name())
			if err != nil || !info.Mode().IsRegular() || linkCount(info) != 1 {
				continue
			}
			files = append(files, EvidenceFile{Name: entry.Name(), Bytes: info.Size()})
		}
		if len(files) > 0 {
			sort.Slice(files, func(i, j int) bool { return files[i].Name < files[j].Name })
			out = append(out, EvidenceTest{Dir: dir, Files: files})
		}
	}
	return out, nil
}

// ReadEvidence reads one evidence file of one test. The test directory and
// the file name must be evidence names and the file a regular file.
func ReadEvidence(root, ref, test, name string) ([]byte, error) {
	if !evidenceTestDir.MatchString(test) || !evidenceName.MatchString(name) {
		return nil, errors.New("not an evidence file")
	}
	run, err := openSessionRoot(root, ref)
	if err != nil {
		return nil, err
	}
	defer func() { _ = run.Close() }()
	if !realDir(run, dirTests) || !realDir(run, test) || !realDir(run, test+"/"+dirObservations) {
		return nil, errors.New("evidence must sit in real directories")
	}
	file := test + "/" + dirObservations + "/" + name
	info, err := run.Lstat(file)
	if err != nil {
		return nil, err
	}
	if !info.Mode().IsRegular() || linkCount(info) != 1 {
		return nil, errors.New("evidence must be a regular file with one name")
	}
	if info.Size() > maxEvidenceFile {
		return nil, errors.New("evidence file is too large to read")
	}
	f, err := run.Open(file)
	if err != nil {
		return nil, err
	}
	defer func() { _ = f.Close() }()
	// The file opened must be the one checked: a link swapped in between
	// would otherwise be followed to another file in the session.
	opened, err := f.Stat()
	if err != nil || !os.SameFile(info, opened) {
		return nil, errors.New("evidence changed while it was opened")
	}
	b, err := io.ReadAll(io.LimitReader(f, maxEvidenceFile+1))
	if err == nil && len(b) > maxEvidenceFile {
		return nil, errors.New("evidence file grew too large while it was read")
	}
	return b, err
}

// realDir is whether name is a directory and not a link to one: a link inside
// the session could point evidence reads at files the engine never scrubbed.
func realDir(run *os.Root, name string) bool {
	info, err := run.Lstat(name)
	return err == nil && info.IsDir() && info.Mode()&os.ModeSymlink == 0
}

// linkCount is how many names a file has; a hard-linked file has more than one.
func linkCount(info fs.FileInfo) uint64 {
	if st, ok := info.Sys().(*syscall.Stat_t); ok {
		return uint64(st.Nlink)
	}
	return 1
}
