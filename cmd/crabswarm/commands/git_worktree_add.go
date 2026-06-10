package commands

import "github.com/spf13/cobra"

func gitWorktreeAddCmd(parent *cobra.Command) {
	cmd := &cobra.Command{
		Use:   "add <branch> [path]",
		Short: "Add a worktree for a branch, defaulting its path to <repo>/<branch>",
		Long: `add creates a worktree for <branch> in the repository detected from the
current directory. The path defaults to <repo>/<branch> (a sibling of the
other worktrees); pass an explicit path to override it.

Branch resolution:
  - an existing local branch is checked out as-is;
  - a remote-only branch (e.g. origin/feature) becomes a new local tracking
    branch;
  - otherwise a new branch is created from the current HEAD.

The <branch> argument completes against the repository's local and
remote-tracking branches.`,
		Example: `  crabswarm git worktree add main
  crabswarm git worktree add feature/login
  crabswarm git worktree add hotfix ../hotfix-checkout`,
		Args:              cobra.RangeArgs(1, 2),
		ValidArgsFunction: gitWorktreeAddCompletion,
		RunE:              runGitWorktreeAdd,
	}

	parent.AddCommand(cmd)
}

// gitWorktreeAddCompletion completes the branch name (first arg) and leaves
// the optional path (second arg) to default file completion.
func gitWorktreeAddCompletion(
	cmd *cobra.Command,
	args []string,
	toComplete string,
) ([]string, cobra.ShellCompDirective) {
	if len(args) >= 1 {
		return nil, cobra.ShellCompDirectiveDefault
	}
	return completeGitBranches(cmd, args, toComplete)
}

func runGitWorktreeAdd(cmd *cobra.Command, args []string) error {
	svc, repo, err := gitRepoFromCwd(cmd)
	if err != nil {
		return err
	}

	var path string
	if len(args) == 2 {
		path = args[1]
	}

	created, err := svc.AddWorktree(cmd.Context(), repo, args[0], path)
	if err != nil {
		return err
	}
	cmd.Printf("Added worktree %s for %q\n", created, args[0])
	return nil
}
