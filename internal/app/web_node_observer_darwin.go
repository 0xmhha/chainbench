package app

import (
	"context"
	"errors"
	"fmt"
	"path/filepath"
	"strings"

	"github.com/0xmhha/chainbench/internal/core/process"
	"github.com/0xmhha/chainbench/internal/resource"
)

func webObserverFor(access *resource.Access, target resource.Inspection) webProcessObserver {
	if target.OS == "darwin" {
		commander, ok := access.Driver.(process.Commander)
		if !ok {
			return nil
		}
		return webDarwinObserver{commander}
	}
	observer, _ := access.Driver.(webProcessObserver)
	if commander, ok := access.Driver.(process.Commander); ok && observer != nil {
		return webDiscoveringObserver{webProcessObserver: observer, commander: commander}
	}
	return observer
}

// Darwin has no /proc. Use the target's read-only process table and refuse argv
// when the executable column disagrees. Whitespace-losing argv remains unable
// to match a recorded argument containing spaces; it never authorizes that case.
type webDarwinObserver struct{ commander process.Commander }

func (o webDarwinObserver) PIDAlive(ctx context.Context, pid int) (bool, error) {
	if pid <= 0 {
		return false, errors.New("invalid process identity")
	}
	command := fmt.Sprintf("if ! command -v ps >/dev/null 2>&1; then printf unavailable; elif ps -p %d -o pid= >/dev/null 2>&1; then printf alive; else printf absent; fi", pid)
	out, err := o.commander.Run(ctx, command)
	if err != nil {
		return false, err
	}
	switch strings.TrimSpace(out) {
	case "alive":
		return true, nil
	case "absent":
		return false, nil
	default:
		return false, errors.New("process inspection unavailable")
	}
}
func (o webDarwinObserver) Cmdline(ctx context.Context, pid int) ([]string, error) {
	if pid <= 0 {
		return nil, errors.New("invalid process identity")
	}
	raw, err := o.commander.Run(ctx, fmt.Sprintf("ps -ww -p %d -o args=", pid))
	if err != nil {
		return nil, err
	}
	executable, err := o.commander.Run(ctx, fmt.Sprintf("ps -ww -p %d -o comm=", pid))
	if err != nil {
		return nil, err
	}
	argv := strings.Fields(raw)
	if len(argv) == 0 || filepath.Clean(argv[0]) != filepath.Clean(strings.TrimSpace(executable)) {
		return nil, errors.New("process launch identity unavailable")
	}
	return argv, nil
}
