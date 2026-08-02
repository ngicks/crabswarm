package commands

import (
	"context"
	"log/slog"
	"os"

	"github.com/ngicks/go-common/contextkey"
	"github.com/spf13/cobra"

	"github.com/ngicks/crabswarm/crabswarm/git"
)

// gitRepoFromCwd builds a git.Service and resolves the repository that owns
// the current working directory. Worktree commands operate on the repo
// detected this way rather than on an explicit target.
func gitRepoFromCwd(cmd *cobra.Command) (git.Service, git.Repo, error) {
	ctx := cmd.Context()

	logger, _ := contextkey.ValueSlogLogger(ctx)
	if logger == nil {
		logger = slog.Default()
	}

	svc := git.Service{Logger: logger}

	wd, err := os.Getwd()
	if err != nil {
		return svc, git.Repo{}, err
	}
	repo, err := svc.ResolveRepo(ctx, wd)
	return svc, repo, err
}

// completeGitBranches completes a branch-name argument from the repository
// detected at the current directory. It is silent on failure so completion
// degrades to no suggestions rather than surfacing errors.
func completeGitBranches(
	cmd *cobra.Command,
	_ []string,
	_ string,
) ([]string, cobra.ShellCompDirective) {
	svc := git.Service{Logger: slog.New(slog.DiscardHandler)}
	repo, err := resolveQuietly(cmd.Context(), svc)
	if err != nil {
		return nil, cobra.ShellCompDirectiveNoFileComp
	}
	branches, err := svc.Branches(cmd.Context(), repo)
	if err != nil {
		return nil, cobra.ShellCompDirectiveNoFileComp
	}
	return branches, cobra.ShellCompDirectiveNoFileComp
}

// completeGitWorktrees completes a worktree handle (directory name) from the
// repository detected at the current directory, excluding the bare entry.
func completeGitWorktrees(
	cmd *cobra.Command,
	_ []string,
	_ string,
) ([]string, cobra.ShellCompDirective) {
	svc := git.Service{Logger: slog.New(slog.DiscardHandler)}
	repo, err := resolveQuietly(cmd.Context(), svc)
	if err != nil {
		return nil, cobra.ShellCompDirectiveNoFileComp
	}
	wts, err := svc.ListWorktrees(cmd.Context(), repo)
	if err != nil {
		return nil, cobra.ShellCompDirectiveNoFileComp
	}
	var names []string
	for _, wt := range wts {
		if !wt.Bare {
			names = append(names, wt.Name())
		}
	}
	return names, cobra.ShellCompDirectiveNoFileComp
}

func resolveQuietly(ctx context.Context, svc git.Service) (git.Repo, error) {
	wd, err := os.Getwd()
	if err != nil {
		return git.Repo{}, err
	}
	return svc.ResolveRepo(ctx, wd)
}
