package main

import (
	"context"
	"fmt"
	"io"
	"time"

	"github.com/spf13/cobra"

	"github.com/0xmhha/chainbench/internal/consensus/upgrade"
	"github.com/0xmhha/chainbench/internal/core/node"
)

// etcdFormTimeout bounds the wait for the producer's etcd cluster to form
// after admin.etcdInit(). The call returns before the cluster exists, and a
// network whose cluster never forms stalls a few blocks later, far from the
// cause — so the run checks rather than trusts the exit code.
const etcdFormTimeout = 60 * time.Second

func newUpgradeRunCmd() *cobra.Command {
	var profilePath, presetDir, fromBinary, toBinary, template, dataDir, genesisOverlay string
	var waitFor int
	cmd := &cobra.Command{
		Use:   "run",
		Short: "Launch and bootstrap the full concurrent handoff from a golden profile",
		Long: "Composes the handoff end to end: build the producer's base genesis, " +
			"merge the successor fork section, launch the mixed binaries " +
			"concurrently, wire a full peer mesh, bootstrap governance + etcd on " +
			"the producer, and confirm the cluster formed. Requires the built " +
			"binaries, etcd, and a preset key set.",
		RunE: func(cmd *cobra.Command, _ []string) error {
			if profilePath == "" || template == "" || dataDir == "" {
				return fmt.Errorf("--profile, --template, and --data-dir are required")
			}
			ctx := cmd.Context()
			out := cmd.OutOrStdout()

			// The binaries resolve here, not in the sequence: the profile names
			// each by its command name, and the operator may point at a build.
			prof, err := upgrade.LoadProfile(profilePath)
			if err != nil {
				return err
			}
			fromBin, err := resolveBinary(fromBinary, prof.Chains.From.Binary)
			if err != nil {
				return fmt.Errorf("from binary: %w", err)
			}
			toBin, err := resolveBinary(toBinary, prof.Chains.To.Binary)
			if err != nil {
				return fmt.Errorf("to binary: %w", err)
			}
			h, err := upgrade.NewHandoff(upgrade.HandoffInputs{
				ProfilePath: profilePath, PresetDir: presetDir,
				FromBinary: fromBin, ToBinary: toBin,
				Template: template, GenesisOverlay: genesisOverlay,
				DataDir: dataDir,
			})
			if err != nil {
				return err
			}
			ns, err := runHandoff(ctx, out, h)
			if err != nil {
				return err
			}
			if waitFor > 0 {
				detail, err := h.AwaitFork(ctx, ns, time.Duration(waitFor)*time.Second)
				if err != nil {
					return err
				}
				fmt.Fprintf(out, "handoff confirmed: %s\n", detail)
			}
			return nil
		},
	}
	cmd.Flags().StringVar(&profilePath, "profile", "", "golden upgrade profile (profiles/*.yaml)")
	cmd.Flags().StringVar(&presetDir, "preset", "keys/preset", "preset key set directory")
	cmd.Flags().StringVar(&fromBinary, "from-binary", "", "from-chain (producer) binary path")
	cmd.Flags().StringVar(&toBinary, "to-binary", "", "to-chain (validator) binary path")
	cmd.Flags().StringVar(&template, "template", "", "wemix genesis template path")
	cmd.Flags().StringVar(&genesisOverlay, "genesis-overlay", "", "optional genesis overlay file ({\"genesis\":{...}}) deep-merged into the handoff genesis")
	cmd.Flags().StringVar(&dataDir, "data-dir", "", "node data root")
	cmd.Flags().IntVar(&waitFor, "wait", 0, "seconds to poll for the post-fork handoff (0=don't wait)")
	return cmd
}

// runHandoff runs the sequence up to a formed etcd cluster, reporting each
// node as it comes up. It is the same order the `chain up` case runner
// follows; the body is upgrade.Handoff's.
func runHandoff(ctx context.Context, out io.Writer, h *upgrade.Handoff) (node.NodeSet, error) {
	ns, info, err := h.Run(ctx, etcdFormTimeout)
	if err != nil {
		return ns, err
	}
	// The plan is composed inside Run, so its detail is reported afterwards.
	fmt.Fprintf(out, "handoff %s -> %s at %s block %d; %d nodes\n",
		h.Plan.From.ID, h.Plan.To.ID, h.Plan.AtFork, h.ForkBlock(), len(h.Plan.Nodes))
	for _, n := range ns.Nodes {
		fmt.Fprintf(out, "  node%d  %s  pid=%d\n", n.Index+1, n.RPCURL, n.PID)
	}
	fmt.Fprintf(out, "governance deployed at %s, etcd cluster %q, mesh wired.\n", info.Governance, info.Cluster())
	return ns, nil
}
