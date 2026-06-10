package commands

import "github.com/spf13/cobra"

func gitWorktreeListCmd(parent *cobra.Command) {
	cmd := &cobra.Command{
		Use:               "list",
		Aliases:           []string{"ls"},
		Short:             "List the worktrees of the repository detected from the current directory",
		Args:              cobra.NoArgs,
		ValidArgsFunction: cobra.NoFileCompletions,
		RunE:              runGitWorktreeList,
	}

	parent.AddCommand(cmd)
}

func runGitWorktreeList(cmd *cobra.Command, _ []string) error {
	svc, repo, err := gitRepoFromCwd(cmd)
	if err != nil {
		return err
	}

	wts, err := svc.ListWorktrees(cmd.Context(), repo)
	if err != nil {
		return err
	}

	for _, wt := range wts {
		switch {
		case wt.Bare:
			cmd.Printf("%s\t(bare)\n", wt.Path)
		case wt.Branch == "":
			cmd.Printf("%s\t(detached)\n", wt.Path)
		default:
			cmd.Printf("%s\t%s\n", wt.Path, wt.Branch)
		}
	}
	return nil
}
