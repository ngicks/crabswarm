package commands

import (
	"fmt"
	"path/filepath"

	"github.com/spf13/cobra"

	"github.com/ngicks/crabswarm/crabswarm/issues"
	issuescli "github.com/ngicks/crabswarm/crabswarm/issues/cli"
	"github.com/ngicks/crabswarm/crabswarm/issues/mermaidlint"
)

// issuesLintLong is the `issues lint` Long help. It sits at package level so
// the command wiring below stays readable.
const issuesLintLong = `lint validates every ` + "```mermaid" + ` diagram written in the beads issue
backlog — descriptions, designs, acceptance criteria, notes and comments —
by handing the text to mermaid-lint, the same parser that guards the mermaid
diagrams in markdown files.

Each refused diagram is reported on one line:

  <issue-id> <field>[#<comment-n>]:<line>:<col>: <message>

The position is counted inside that text, from its first line. The command
exits 1 when anything was refused, so it can gate a turn as a hook.

mermaid-lint runs with the directory bd was run from as its working
directory, so a repository's own mermaid-lint configuration governs its
issue text exactly as it governs its files.`

func issuesLintCmd(parent *cobra.Command) {
	var (
		flagDir   string
		flagAll   bool
		flagLimit int
		flagJSON  bool
	)

	cmd := &cobra.Command{
		Use:   "lint",
		Short: "Validate the mermaid diagrams written in the issue backlog",
		Long:  issuesLintLong,
		Example: `  crabswarm issues lint
  crabswarm issues lint --all
  crabswarm issues lint --limit 20
  crabswarm issues lint --json
  crabswarm issues lint -C ./other-repo`,
		Args:              cobra.NoArgs,
		ValidArgsFunction: cobra.NoFileCompletions,
		RunE: func(cmd *cobra.Command, args []string) error {
			return runIssuesLint(cmd, args, flagDir, flagAll, flagLimit, flagJSON)
		},
	}

	f := cmd.Flags()
	f.StringVarP(&flagDir, "dir", "C", ".",
		"Directory to run bd from, and whose mermaid-lint configuration applies")
	f.BoolVar(&flagAll, "all", false,
		"Lint the closed issues too, not just the open backlog")
	f.IntVar(&flagLimit, "limit", 0,
		"Lint only this many issues, most recently updated first, in any status")
	f.BoolVar(&flagJSON, "json", false,
		"Print the findings as a JSON array instead of one line each")

	parent.AddCommand(cmd)
}

func runIssuesLint(
	cmd *cobra.Command, _ []string,
	dir string, all bool, limit int, asJSON bool,
) error {
	abs, err := filepath.Abs(dir)
	if err != nil {
		return err
	}

	client := issues.NewClient(abs, issues.WithLogger(commandLogger(cmd)))
	findings, err := mermaidlint.Sweep(
		cmd.Context(),
		client,
		mermaidlint.SweepOptions{All: all, Limit: limit},
		mermaidlint.WithDir(abs),
	)
	if err != nil {
		return err
	}

	render := issuescli.RenderFindings
	if asJSON {
		render = issuescli.RenderFindingsJSON
	}
	if err := render(cmd.OutOrStdout(), findings); err != nil {
		return err
	}

	if len(findings) > 0 {
		// The findings are the report; this error is only how the CLI exits
		// non-zero, which is what makes the command usable as a hook.
		return fmt.Errorf("mermaid-lint refused %d diagram(s) in the issue backlog", len(findings))
	}
	return nil
}
