package resourcecmd

import (
	"bytes"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"os"

	"github.com/0xmhha/chainbench/cmd/chainbench/surface"
	"github.com/0xmhha/chainbench/internal/app"
	"github.com/spf13/cobra"
)

func newInspectCmd() *cobra.Command {
	var request, logins string
	var jsonOut bool
	cmd := &cobra.Command{Use: "inspect", Short: "Observe a target's identity, files, logs or recorded process without changing it", RunE: func(cmd *cobra.Command, _ []string) error {
		var in app.ResourceReadIn
		if err := readRequestFile(request, &in); err != nil {
			return err
		}
		out, err := app.ResourceRead(cmd.Context(), surface.Deps(cmd).WithLogins(logins), in)
		if err != nil {
			return err
		}
		return writeResult(cmd, out, jsonOut)
	}}
	cmd.Flags().StringVar(&request, "request", "", "JSON file describing target and bounded read operations")
	cmd.Flags().StringVar(&logins, "server-logins", "", "server set supplying this call's SSH logins")
	cmd.Flags().BoolVar(&jsonOut, "json", false, "emit observations as JSON")
	_ = cmd.MarkFlagRequired("request")
	return cmd
}

// readRequestFile decodes one bounded, strictly typed JSON request.
func readRequestFile(path string, into any) error {
	file, err := os.Open(path)
	if err != nil {
		return err
	}
	defer func() { _ = file.Close() }()
	raw, err := io.ReadAll(io.LimitReader(file, (1<<20)+1))
	if err != nil {
		return err
	}
	if len(raw) > 1<<20 {
		return errors.New("resource request exceeds one MiB")
	}
	decoder := json.NewDecoder(bytes.NewReader(raw))
	decoder.DisallowUnknownFields()
	if err = decoder.Decode(into); err != nil {
		return err
	}
	if err = decoder.Decode(new(any)); !errors.Is(err, io.EOF) {
		return errors.New("resource request requires one JSON document")
	}
	return nil
}

func writeResult(cmd *cobra.Command, out any, jsonOut bool) error {
	if jsonOut {
		return json.NewEncoder(cmd.OutOrStdout()).Encode(out)
	}
	raw, err := json.MarshalIndent(out, "", "  ")
	if err == nil {
		_, err = fmt.Fprintln(cmd.OutOrStdout(), string(raw))
	}
	return err
}
