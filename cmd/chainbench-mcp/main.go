// Command chainbench-mcp serves the chainbench MCP surface (requirement #14)
// over stdio: it reads newline-delimited JSON-RPC requests from stdin and writes
// responses to stdout, dispatching to the tools in pkg/mcp. Chain and test-case
// plugins are imported for registration.
package main

import (
	"bufio"
	"context"
	"fmt"
	"os"

	_ "github.com/0xmhha/chainbench/pkg/chains/all"
	_ "github.com/0xmhha/chainbench/tests/all"

	"github.com/0xmhha/chainbench/pkg/mcp"
)

var version = "0.0.0-dev"

func main() {
	if err := serve(os.Stdin, os.Stdout); err != nil {
		fmt.Fprintln(os.Stderr, "chainbench-mcp:", err)
		os.Exit(1)
	}
}

func serve(in *os.File, out *os.File) error {
	srv := mcp.Default("chainbench", version)
	ctx := context.Background()

	scanner := bufio.NewScanner(in)
	scanner.Buffer(make([]byte, 0, 64*1024), 8*1024*1024)
	w := bufio.NewWriter(out)
	defer w.Flush()

	for scanner.Scan() {
		line := scanner.Bytes()
		if len(line) == 0 {
			continue
		}
		resp := srv.Handle(ctx, line)
		if resp == nil {
			continue // notification: no reply
		}
		if _, err := w.Write(resp); err != nil {
			return err
		}
		if err := w.WriteByte('\n'); err != nil {
			return err
		}
		if err := w.Flush(); err != nil {
			return err
		}
	}
	return scanner.Err()
}
