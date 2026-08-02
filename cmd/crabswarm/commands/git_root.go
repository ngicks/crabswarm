package commands

import (
	"fmt"
	"log/slog"
	"os"

	"github.com/ngicks/go-common/contextkey"
	"github.com/spf13/cobra"

	"github.com/ngicks/crabswarm/crabswarm/git"
)

func gitRootCmd(parent *cobra.Command) {
	cmd := &cobra.Command{
		Use:   "root",
		Short: "Print the bare-plus-worktree repository root (the directory holding .bare)",
		Long: `root walks from the current directory up to the filesystem root and prints the
nearest crabswarm repository root — the directory that holds the ".bare" bare
clone, the layout produced by "crabswarm git clone". A worktree placed outside
its repository directory is resolved through git instead. The path is printed
to stdout so it can be consumed directly, e.g. cd "$(crabswarm git root)".

When no git repository is found, or the resolved one is an ordinary clone
rather than a --bare clone (it has no ".bare" directory), a message is printed
to stderr and the command exits non-zero.`,
		Example: `  crabswarm git root
  cd "$(crabswarm git root)"`,
		Args:              cobra.NoArgs,
		ValidArgsFunction: cobra.NoFileCompletions,
		RunE:              runGitRoot,
	}

	parent.AddCommand(cmd)
}

func runGitRoot(cmd *cobra.Command, _ []string) error {
	ctx := cmd.Context()

	logger, _ := contextkey.ValueSlogLogger(ctx)
	if logger == nil {
		logger = slog.Default()
	}

	svc := git.Service{Logger: logger}

	wd, err := os.Getwd()
	if err != nil {
		return err
	}
	root, err := svc.FindRoot(ctx, wd)
	if err != nil {
		return err
	}
	// Print to stdout (not cmd.Println, which cobra routes to stderr) so the
	// path can be captured, e.g. cd "$(crabswarm git root)".
	fmt.Fprintln(cmd.OutOrStdout(), root)
	return nil
}
