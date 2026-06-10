// Package git implements crabswarm's git helpers: a ghq-style bare clone
// that lays out a repository under a base directory and checks out its
// default branch as a worktree, plus a thin worktree wrapper that resolves
// the repository from the current directory.
//
// The bare-plus-worktree layout it creates is:
//
//	<base>/<host>/<owner>/<repo>/
//	  ├── .bare/            the bare clone (git clone --bare)
//	  ├── .git              file: "gitdir: ./.bare"
//	  └── <default-branch>/ worktree of the default branch
//
// The ".git" file makes <repo> itself a usable git directory, and every
// branch lives in its own sibling worktree directory. All operations shell
// out to the system git binary.
package git

import (
	"bytes"
	"context"
	"fmt"
	"io"
	"log/slog"
	"os/exec"
	"strings"
)

// bareDirName is the directory holding the bare clone inside a repository
// directory; gitDirFile is the pointer file that makes the repository
// directory itself a git directory.
const (
	bareDirName = ".bare"
	gitDirFile  = ".git"
)

// Service performs git operations for crabswarm. The zero value is usable
// for worktree operations (which resolve their repository from a directory);
// Clone additionally requires BaseDir.
type Service struct {
	// BaseDir is the root under which Clone materializes repositories
	// (the resolved ${CRABSWARM_GIT_REPO_BASE_DIR:-$HOME/gitrepo}).
	BaseDir string
	// Logger receives debug records for each git invocation. When nil,
	// [slog.Default] is used.
	Logger *slog.Logger
	// Progress, when non-nil, receives the live stdout+stderr of
	// long-running commands (clone, fetch) so the user sees progress.
	// Short metadata queries are always captured regardless.
	Progress io.Writer
}

func (s Service) logger() *slog.Logger {
	if s.Logger != nil {
		return s.Logger
	}
	return slog.Default()
}

// run executes "git args..." in dir (cwd when empty), capturing stdout. On
// failure the returned error carries the trimmed stderr.
func (s Service) run(ctx context.Context, dir string, args ...string) (string, error) {
	cmd := exec.CommandContext(ctx, "git", args...)
	cmd.Dir = dir
	var stdout, stderr bytes.Buffer
	cmd.Stdout = &stdout
	cmd.Stderr = &stderr
	s.logger().Debug("running git", "args", args, "dir", dir)
	if err := cmd.Run(); err != nil {
		if msg := strings.TrimSpace(stderr.String()); msg != "" {
			return stdout.String(), fmt.Errorf("git %s: %w: %s", strings.Join(args, " "), err, msg)
		}
		return stdout.String(), fmt.Errorf("git %s: %w", strings.Join(args, " "), err)
	}
	return stdout.String(), nil
}

// runProgress executes "git args..." with stdout+stderr streamed to
// s.Progress (discarded when nil). Used for clone/fetch where live output
// matters more than capturing.
func (s Service) runProgress(ctx context.Context, dir string, args ...string) error {
	cmd := exec.CommandContext(ctx, "git", args...)
	cmd.Dir = dir
	out := s.Progress
	if out == nil {
		out = io.Discard
	}
	cmd.Stdout = out
	cmd.Stderr = out
	s.logger().Debug("running git", "args", args, "dir", dir)
	if err := cmd.Run(); err != nil {
		return fmt.Errorf("git %s: %w", strings.Join(args, " "), err)
	}
	return nil
}
