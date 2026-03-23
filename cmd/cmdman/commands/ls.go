package commands

import (
	"encoding/json"
	"fmt"
	"strings"
	"text/tabwriter"

	"github.com/ngicks/crabswarm/pkg/cmdman"
	"github.com/spf13/cobra"
)

func init() {
	rootCmd.AddCommand(lsCmd)
	lsCmd.Flags().StringArrayP("label", "l", nil, "Filter by label KEY=VALUE (repeatable)")
	lsCmd.Flags().BoolP("all", "a", false, "Show all (including exited)")
	lsCmd.Flags().BoolP("quiet", "q", false, "Print IDs only")
	lsCmd.Flags().String("format", "table", "Output format: table, json")
}

var lsCmd = &cobra.Command{
	Use:   "ls [flags]",
	Short: "List commands",
	RunE:  runLs,
}

func runLs(cmd *cobra.Command, args []string) error {
	labelSlice, _ := cmd.Flags().GetStringArray("label")
	allStates, _ := cmd.Flags().GetBool("all")
	quiet, _ := cmd.Flags().GetBool("quiet")
	format, _ := cmd.Flags().GetString("format")

	labels, err := parseLabels(labelSlice)
	if err != nil {
		return err
	}

	store, err := cmdman.OpenStore(cmdman.DBPath())
	if err != nil {
		return fmt.Errorf("open store: %w", err)
	}
	defer store.Close()

	// Clean stale entries.
	cmdman.CleanStaleEntries(store)

	entries, err := store.ListCommands(allStates, labels)
	if err != nil {
		return fmt.Errorf("list commands: %w", err)
	}

	if quiet {
		for _, e := range entries {
			fmt.Fprintln(cmd.OutOrStdout(), e.ID)
		}
		return nil
	}

	if format == "json" {
		enc := json.NewEncoder(cmd.OutOrStdout())
		enc.SetIndent("", "  ")
		return enc.Encode(entries)
	}

	w := tabwriter.NewWriter(cmd.OutOrStdout(), 0, 0, 2, ' ', 0)
	fmt.Fprintln(w, "ID\tNAME\tSTATE\tEXIT CODE")
	for _, e := range entries {
		ec := ""
		if e.ExitCode != nil {
			ec = fmt.Sprintf("%d", *e.ExitCode)
		}
		name := e.Name
		if name == "" {
			name = "-"
		}
		displayID := e.ID
		if len(displayID) > 12 {
			displayID = displayID[:12]
		}
		fmt.Fprintf(w, "%s\t%s\t%s\t%s\n", displayID, name, e.State, ec)
	}
	return w.Flush()
}

func parseLabels(labelSlice []string) (map[string]string, error) {
	if len(labelSlice) == 0 {
		return nil, nil
	}
	labels := make(map[string]string)
	for _, l := range labelSlice {
		k, v, ok := strings.Cut(l, "=")
		if !ok {
			return nil, fmt.Errorf("invalid label format: %s (expected KEY=VALUE)", l)
		}
		labels[k] = v
	}
	return labels, nil
}
