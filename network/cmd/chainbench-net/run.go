package main

import (
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"log/slog"
	"os"

	"github.com/spf13/cobra"

	"github.com/0xmhha/chainbench/network/internal/events"
	"github.com/0xmhha/chainbench/network/internal/types"
	"github.com/0xmhha/chainbench/network/internal/wire"
)

// newRunCmd builds the `run` subcommand. It reads a wire command envelope
// from stdin, dispatches to a handler, and emits a result NDJSON terminator
// on stdout. Structured logs go to stderr.
func newRunCmd() *cobra.Command {
	return &cobra.Command{
		Use:           "run",
		Short:         "Execute one wire command envelope from stdin",
		SilenceUsage:  true,
		SilenceErrors: true,
		RunE: func(cmd *cobra.Command, _ []string) error {
			stateDir := os.Getenv("CHAINBENCH_STATE_DIR")
			if stateDir == "" {
				stateDir = "state"
			}
			return runOnce(cmd.InOrStdin(), cmd.OutOrStdout(), cmd.ErrOrStderr(), allHandlers(stateDir))
		},
	}
}

// runOnce decodes one wire command envelope from stdin, dispatches to a
// handler, and emits a result terminator on stdout. Structured logs go to
// stderr. Returns the error (if any) so the caller can map to an exit code.
// Safe against handler panics via deferred recover.
func runOnce(stdin io.Reader, stdout, stderr io.Writer, handlers map[string]Handler) (returnErr error) {
	wire.SetupLoggerTo(stderr, slog.LevelInfo)
	emitter := wire.NewEmitter(stdout)
	bus := events.NewBus(emitter)
	defer bus.Close()

	defer func() {
		if r := recover(); r != nil {
			msg := fmt.Sprintf("panic: %v", r)
			slog.Error("handler panic", "panic", r)
			_ = emitter.EmitResultError(types.ResultErrorCode("INTERNAL"), msg)
			returnErr = NewInternal(msg, nil)
		}
	}()

	cmd, err := wire.DecodeCommand(stdin)
	if err != nil {
		_ = emitter.EmitResultError(types.ResultErrorCode("PROTOCOL_ERROR"), err.Error())
		return NewProtocolError("decode command envelope", err)
	}

	handler, ok := handlers[string(cmd.Command)]
	if !ok {
		msg := fmt.Sprintf("no handler for command %q", cmd.Command)
		_ = emitter.EmitResultError(types.ResultErrorCode("NOT_SUPPORTED"), msg)
		return NewNotSupported(msg)
	}

	raw, _ := json.Marshal(cmd.Args)

	data, err := handler(raw, bus)
	if err != nil {
		var api *APIError
		if errors.As(err, &api) {
			_ = emitter.EmitResultError(api.Code, api.Message)
			return err
		}
		_ = emitter.EmitResultError(types.ResultErrorCode("INTERNAL"), err.Error())
		return NewInternal(err.Error(), err)
	}

	if emitErr := emitter.EmitResult(true, data); emitErr != nil {
		return fmt.Errorf("emit result: %w", emitErr)
	}
	return nil
}
