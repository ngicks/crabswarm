package commands

import (
	"errors"
	"fmt"
	"path/filepath"

	"github.com/spf13/cobra"

	"github.com/ngicks/crabswarm/crabswarm"
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
exits 1 when anything was refused, so it can gate a commit as a pre-commit
hook.

A directory governed by no beads database holds no backlog to lint: the
command exits 0 without running mermaid-lint, so the hook passes in a
repository that keeps no issues.

mermaid-lint runs with the directory bd was run from as its working
directory, so a repository's own mermaid-lint configuration governs its
issue text exactly as it governs its files.

--changed skips the lint when the backlog has not changed since the last
clean lint. bd commits every backlog write under its default dolt.auto-commit
policy. The command reads the beads commit with "bd vc status" and compares
it with the record of the last clean lint of that database. The record lives
in

  $XDG_STATE_HOME/crabswarm/issues-lint/<source id>

and $XDG_STATE_HOME defaults to ~/.local/state. The source id is a hash of
the .beads path, and every worktree of a repository shares one record. When
the commit and the scope (--all, --limit) match the record, the command
exits 0 and prints nothing. It lists no issue and runs no mermaid-lint then.
A lint that finds nothing records the commit. A lint that finds something
records nothing, and the next run reports the findings again. The record
does not cover the mermaid-lint configuration. Run without --changed after
changing that configuration. Without --changed the command always lints and
never touches the record.`

func issuesLintCmd(parent *cobra.Command) {
	var (
		flagDir     string
		flagAll     bool
		flagLimit   int
		flagJSON    bool
		flagChanged bool
	)

	cmd := &cobra.Command{
		Use:   "lint",
		Short: "Validate the mermaid diagrams written in the issue backlog",
		Long:  issuesLintLong,
		Example: `  crabswarm issues lint
  crabswarm issues lint --changed
  crabswarm issues lint --all
  crabswarm issues lint --limit 20
  crabswarm issues lint --json
  crabswarm issues lint -C ./other-repo`,
		Args:              cobra.NoArgs,
		ValidArgsFunction: cobra.NoFileCompletions,
		RunE: func(cmd *cobra.Command, args []string) error {
			return runIssuesLint(cmd, args, flagDir, flagAll, flagLimit, flagJSON, flagChanged)
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
	f.BoolVar(&flagChanged, "changed", false,
		"Skip the lint when the beads commit and scope match the last clean lint")

	parent.AddCommand(cmd)
}

func runIssuesLint(
	cmd *cobra.Command, _ []string,
	dir string, all bool, limit int, asJSON bool, changed bool,
) error {
	abs, err := filepath.Abs(dir)
	if err != nil {
		return err
	}

	logger := commandLogger(cmd)
	// The lint is wired as a pre-commit step, and a repository may spread that
	// step without keeping issues, so a project with no beads database is the
	// ordinary shape of "nothing to lint" rather than a failure that would
	// refuse every commit there.
	loc, err := issues.Where(cmd.Context(), abs)
	if err != nil {
		if errors.Is(err, issues.ErrNoBeads) {
			logger.Debug("issues lint: no beads database, nothing to lint", "dir", abs)
			return nil
		}
		return err
	}

	client := issues.NewClient(abs, issues.WithLogger(logger))
	scope := mermaidlint.SweepOptions{All: all, Limit: limit}
	var findings []mermaidlint.Finding
	if changed {
		ledger := mermaidlint.NewLedger(crabswarm.IssuesLintRecordDir(), logger)
		var skipped bool
		findings, skipped, err = ledger.Sweep(
			cmd.Context(), client, loc.BeadsPath, scope, mermaidlint.WithDir(abs))
		if err != nil {
			return err
		}
		if skipped {
			return nil
		}
	} else {
		findings, err = mermaidlint.Sweep(cmd.Context(), client, scope, mermaidlint.WithDir(abs))
		if err != nil {
			return err
		}
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
