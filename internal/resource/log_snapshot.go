package resource

import (
	"context"
	"encoding/hex"
	"errors"
	"fmt"
	"io"
	"os"
	"strconv"
	"strings"
	"syscall"

	"github.com/0xmhha/chainbench/internal/core/remote"
)

// ErrLogMissing reports an absent log without interpreting an empty file as absent.
var ErrLogMissing = errors.New("resource: log is absent")

const logHeadBytes = 256

func readLogSnapshot(ctx context.Context, a *Access, r LogRead) (LogSnapshot, error) {
	if a.Runner == nil {
		f, err := os.Open(r.Path)
		if errors.Is(err, os.ErrNotExist) {
			return LogSnapshot{}, ErrLogMissing
		}
		if err != nil {
			return LogSnapshot{}, err
		}
		defer func() { _ = f.Close() }()
		info, err := f.Stat()
		if err != nil {
			return LogSnapshot{}, err
		}
		stat, ok := info.Sys().(*syscall.Stat_t)
		if !ok || !info.Mode().IsRegular() {
			return LogSnapshot{}, errors.New("resource: log identity is unavailable")
		}
		out := LogSnapshot{ID: stat.Ino, Size: info.Size()}
		head := make([]byte, min(info.Size(), logHeadBytes))
		n, err := f.ReadAt(head, 0)
		if err != nil && !errors.Is(err, io.EOF) {
			return LogSnapshot{}, err
		}
		out.Head = head[:n]
		if info.Size() > r.Offset {
			data := make([]byte, r.Limit)
			n, err = f.ReadAt(data, r.Offset)
			if err != nil && !errors.Is(err, io.EOF) {
				return LogSnapshot{}, err
			}
			out.Data = data[:n]
		}
		return out, ctx.Err()
	}
	q := remote.ShellQuote(r.Path)
	command := "test -f " + q + " || exit 3; set -- $(ls -diL " + q + "); s=$(wc -c < " + q + "); h=$(head -c " + strconv.Itoa(logHeadBytes) + " " + q + " | od -An -v -tx1 | tr -d ' \\n'); echo \"$1 $s ${h:--}\"; tail -c +" + strconv.FormatInt(r.Offset+1, 10) + " " + q + " | head -c " + strconv.Itoa(r.Limit)
	res, err := a.Runner(ctx, command)
	if err != nil {
		return LogSnapshot{}, err
	}
	if res.ExitCode == 3 {
		return LogSnapshot{}, ErrLogMissing
	}
	header, data, found := strings.Cut(res.Stdout, "\n")
	fields := strings.Fields(header)
	if res.ExitCode != 0 || !found || len(fields) != 3 {
		return LogSnapshot{}, fmt.Errorf("resource: log snapshot failed (exit %d)", res.ExitCode)
	}
	id, err := strconv.ParseUint(fields[0], 10, 64)
	if err != nil {
		return LogSnapshot{}, err
	}
	size, err := strconv.ParseInt(fields[1], 10, 64)
	if err != nil || size < 0 {
		return LogSnapshot{}, errors.New("resource: invalid log size")
	}
	var head []byte
	if fields[2] != "-" {
		head, err = hex.DecodeString(fields[2])
		if err != nil {
			return LogSnapshot{}, err
		}
	}
	if len(head) > logHeadBytes || len(data) > r.Limit {
		return LogSnapshot{}, errors.New("resource: log snapshot exceeded its bounds")
	}
	return LogSnapshot{ID: id, Size: size, Head: head, Data: []byte(data)}, nil
}
