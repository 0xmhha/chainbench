package main

import (
	"fmt"
	"io"
	"strings"
	"text/tabwriter"
	"time"

	"github.com/spf13/cobra"

	"github.com/0xmhha/chainbench/internal/chainsetup"
)

// newChainCmd is the bring-up inspection surface: it executes the documented
// chain-construction procedure one step at a time so a human can see exactly
// which step works and which does not. It complements `setup` (which plans and
// launches in one shot) and `run` (which executes test specs).
func newChainCmd() *cobra.Command {
	cmd := &cobra.Command{
		Use:   "chain",
		Short: "Inspect and drive chain bring-up, one step at a time",
		Long: "Executes the bring-up procedure documented in docs/dev/chain-setup/ step by\n" +
			"step, reporting each step's outcome. Use it to check that the framework\n" +
			"actually supports a given way of standing a network up, and to find the\n" +
			"first step that does not.",
	}
	cmd.AddCommand(newChainCasesCmd(), newChainStepsCmd(), newChainUpCmd(),
		newChainStatusCmd(), newChainDownCmd())
	return cmd
}

// newChainCasesCmd lists the modelled cases and their measured support.
func newChainCasesCmd() *cobra.Command {
	return &cobra.Command{
		Use:   "cases",
		Short: "List the chain bring-up cases and how far each is known to work",
		RunE: func(cmd *cobra.Command, _ []string) error {
			out := cmd.OutOrStdout()
			w := tabwriter.NewWriter(out, 0, 0, 2, ' ', 0)
			fmt.Fprintln(w, "CASE\tBOOTSTRAP\tSUPPORT\tTITLE")
			for _, c := range chainsetup.Cases() {
				fmt.Fprintf(w, "%s\t%s\t%s\t%s\n", c.ID, c.Bootstrap, c.Support, c.Title)
			}
			if err := w.Flush(); err != nil {
				return err
			}
			for _, c := range chainsetup.Cases() {
				if c.Note != "" {
					fmt.Fprintf(out, "\n%s: %s\n  doc: %s\n", c.ID, c.Note, c.Doc)
				}
			}
			return nil
		},
	}
}

// newChainStepsCmd prints one case's steps and customization points.
func newChainStepsCmd() *cobra.Command {
	var caseID string
	cmd := &cobra.Command{
		Use:   "steps",
		Short: "Show a case's bring-up steps and customization points",
		RunE: func(cmd *cobra.Command, _ []string) error {
			c, ok := chainsetup.Find(caseID)
			if !ok {
				return unknownCase(caseID)
			}
			out := cmd.OutOrStdout()
			fmt.Fprintf(out, "%s — %s\n", c.ID, c.Title)
			fmt.Fprintf(out, "bootstrap: %s   support: %s\n", c.Bootstrap, c.Support)
			if c.Note != "" {
				fmt.Fprintf(out, "note:      %s\n", c.Note)
			}
			fmt.Fprintf(out, "doc:       %s\n", c.Doc)
			fmt.Fprintf(out, "binaries:  %s\n\n", strings.Join(c.Binaries, ", "))

			w := tabwriter.NewWriter(out, 0, 0, 2, ' ', 0)
			fmt.Fprintln(w, "#\tSTEP\tBUILT\tWHAT IT DOES")
			for i, s := range c.Steps {
				built := "yes"
				if !s.Implemented {
					built = "NO"
				}
				fmt.Fprintf(w, "%d\t%s\t%s\t%s\n", i+1, s.ID, built, s.Detail)
			}
			if err := w.Flush(); err != nil {
				return err
			}

			fmt.Fprintf(out, "\nCUSTOMIZATION POINTS\n")
			kw := tabwriter.NewWriter(out, 0, 0, 2, ' ', 0)
			fmt.Fprintln(kw, "WHAT\tWHERE\tEFFECT")
			for _, k := range c.Knobs {
				fmt.Fprintf(kw, "%s\t%s\t%s\n", k.Name, k.Where, k.Effect)
			}
			return kw.Flush()
		},
	}
	cmd.Flags().StringVar(&caseID, "case", "", "case id ("+strings.Join(chainsetup.IDs(), "|")+")")
	return cmd
}

// newChainUpCmd runs a case's steps against real binaries.
func newChainUpCmd() *cobra.Command {
	var (
		caseID, binary, keysDir, dataDir string
		profile, fromBinary, toBinary    string
		template, genesisOverlay         string
		etcdTimeout                      time.Duration
		validators                       int
		stopAfter                        string
		healthTimeout, forkTimeout       time.Duration
	)
	cmd := &cobra.Command{
		Use:   "up",
		Short: "Bring a network up step by step, reporting each step",
		RunE: func(cmd *cobra.Command, _ []string) error {
			c, ok := chainsetup.Find(caseID)
			if !ok {
				return unknownCase(caseID)
			}
			out := cmd.OutOrStdout()
			fmt.Fprintf(out, "case %s (%s), data dir %s\n\n", c.ID, c.Title, dataDir)

			report := stepPrinter(out)
			var (
				run chainsetup.Run
				err error
			)
			switch c.ID {
			case "wbft", "stablenet":
				run, err = chainsetup.RunStatic(cmd.Context(), c, chainsetup.Options{
					Chain: c.ID, Binary: binary, KeysDir: keysDir, DataDir: dataDir,
					Validators: validators, HealthTimeout: healthTimeout, StopAfter: stopAfter,
				}, report)
			case "wemix":
				run, err = chainsetup.RunWemix(cmd.Context(), c, chainsetup.Options{
					Binary: binary, KeysDir: keysDir, DataDir: dataDir,
					Validators: validators, HealthTimeout: healthTimeout, StopAfter: stopAfter,
				}, report)
			default:
				run, err = chainsetup.RunHandoff(cmd.Context(), c, chainsetup.HandoffOptions{
					ProfilePath: profile, PresetDir: keysDir,
					FromBinary: fromBinary, ToBinary: toBinary,
					Template: template, GenesisOverlay: genesisOverlay,
					DataDir: dataDir, ForkTimeout: forkTimeout, EtcdTimeout: etcdTimeout, StopAfter: stopAfter,
				}, chainsetup.NewLiveHandoff(), report)
			}
			if err != nil {
				return err
			}
			return summarize(out, run)
		},
	}
	cmd.Flags().StringVar(&caseID, "case", "", "case id ("+strings.Join(chainsetup.IDs(), "|")+")")
	cmd.Flags().StringVar(&dataDir, "data-dir", "", "network data root (keep it short: node IPC sockets have a 104-char limit)")
	cmd.Flags().StringVar(&keysDir, "keys", "keys/preset", "preset key directory")
	cmd.Flags().StringVar(&stopAfter, "stop-after", "", "stop once this step completes (see `chain steps`)")
	// static cases
	cmd.Flags().StringVar(&binary, "binary", "", "node binary (static cases)")
	cmd.Flags().IntVar(&validators, "validators", 4, "validator count (static cases)")
	cmd.Flags().DurationVar(&healthTimeout, "health-timeout", 90*time.Second, "how long the health gate may take")
	// handoff case
	cmd.Flags().StringVar(&profile, "profile", "profiles/wemix-upgrade.yaml", "upgrade profile (handoff case)")
	cmd.Flags().StringVar(&fromBinary, "from-binary", "", "producer binary (handoff case)")
	cmd.Flags().StringVar(&toBinary, "to-binary", "", "successor binary (handoff case)")
	cmd.Flags().StringVar(&template, "template", "", "go-wemix's own genesis template (handoff case)")
	cmd.Flags().StringVar(&genesisOverlay, "genesis-overlay", "", "optional genesis overlay JSON (handoff case)")
	cmd.Flags().DurationVar(&etcdTimeout, "etcd-timeout", 60*time.Second, "how long to wait for the etcd cluster to form")
	cmd.Flags().DurationVar(&forkTimeout, "fork-timeout", 180*time.Second, "how long to wait for the handoff")
	return cmd
}

// newChainStatusCmd probes a network a bring-up left behind.
func newChainStatusCmd() *cobra.Command {
	var dataDir string
	cmd := &cobra.Command{
		Use:   "status",
		Short: "Probe a network brought up under a data dir",
		RunE: func(cmd *cobra.Command, _ []string) error {
			sts, err := chainsetup.Status(cmd.Context(), dataDir)
			if err != nil {
				return err
			}
			w := tabwriter.NewWriter(cmd.OutOrStdout(), 0, 0, 2, ' ', 0)
			fmt.Fprintln(w, "NODE\tPID\tALIVE\tHEAD\tPEERS\tCHAIN\tRPC\tERROR")
			for _, s := range sts {
				fmt.Fprintf(w, "node%d\t%d\t%v\t%d\t%d\t%d\t%s\t%s\n",
					s.Index, s.PID, s.Alive, s.Head, s.Peers, s.ChainID, s.RPCURL, s.Err)
			}
			return w.Flush()
		},
	}
	cmd.Flags().StringVar(&dataDir, "data-dir", "", "network data root")
	return cmd
}

// newChainDownCmd stops a network and verifies nothing survived.
func newChainDownCmd() *cobra.Command {
	var dataDir string
	var removeData bool
	cmd := &cobra.Command{
		Use:   "down",
		Short: "Stop a network and verify no process survived",
		RunE: func(cmd *cobra.Command, _ []string) error {
			leaks, err := chainsetup.Down(dataDir, removeData)
			if err != nil {
				return err
			}
			out := cmd.OutOrStdout()
			if len(leaks) > 0 {
				return fmt.Errorf("orphan pids after stop: %v", leaks)
			}
			fmt.Fprintf(out, "stopped; no orphans\n")
			if removeData {
				fmt.Fprintf(out, "removed %s\n", dataDir)
			}
			return nil
		},
	}
	cmd.Flags().StringVar(&dataDir, "data-dir", "", "network data root")
	cmd.Flags().BoolVar(&removeData, "remove-data", false, "also delete the data root (separate from stopping)")
	return cmd
}

// stepPrinter returns a reporter that prints each step as it completes.
func stepPrinter(out io.Writer) chainsetup.Reporter {
	return func(r chainsetup.Result) {
		took := ""
		if r.Duration > 0 {
			took = fmt.Sprintf(" (%s)", r.Duration.Round(time.Millisecond))
		}
		fmt.Fprintf(out, "%-5s %-18s%s\n", r.Outcome, r.Step.ID, took)
		if r.Detail != "" {
			fmt.Fprintf(out, "      %s\n", r.Detail)
		}
	}
}

// summarize prints the verdict and returns a non-zero-exit error when a step
// failed, so the command is usable as a gate.
func summarize(out io.Writer, run chainsetup.Run) error {
	fmt.Fprintln(out)
	if len(run.Nodes) > 0 {
		fmt.Fprintf(out, "nodes: %s\n", strings.Join(run.Nodes, " "))
	}
	problem, bad := run.FirstProblem()
	if !bad {
		fmt.Fprintf(out, "all %d step(s) OK\n", len(run.Results))
		if run.DataDir != "" {
			fmt.Fprintf(out, "inspect: chainbench chain status --data-dir %s\n", run.DataDir)
			fmt.Fprintf(out, "stop:    chainbench chain down   --data-dir %s\n", run.DataDir)
		}
		return nil
	}
	switch problem.Outcome {
	case chainsetup.NotImplemented:
		return fmt.Errorf("step %q is not built yet: %s (see %s)",
			problem.Step.ID, problem.Step.Detail, run.Case.Doc)
	default:
		return fmt.Errorf("step %q failed: %s", problem.Step.ID, problem.Detail)
	}
}

// unknownCase reports an unrecognized --case with the valid ids.
func unknownCase(id string) error {
	if id == "" {
		return fmt.Errorf("--case is required (one of: %s)", strings.Join(chainsetup.IDs(), ", "))
	}
	return fmt.Errorf("unknown case %q (one of: %s)", id, strings.Join(chainsetup.IDs(), ", "))
}
