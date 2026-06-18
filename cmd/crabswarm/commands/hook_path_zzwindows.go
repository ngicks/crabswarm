package commands

// This leaf lives in hook_path_zzwindows.go rather than the canonical
// hook_path_windows.go: a filename ending in _windows.go matches Go's
// implicit `*_GOOS.go` build constraint and would compile only on Windows,
// silently dropping the command everywhere else. The zz-prefixed OS token
// keeps the file a dedicated per-leaf source while dodging that constraint;
// future "hook path <goos>" leaves follow the same hook_path_zz<goos>.go
// pattern.

import (
	"github.com/spf13/cobra"

	"github.com/ngicks/crabswarm/internal/stdiopipe"
	hookpath "github.com/ngicks/crabswarm/pkg/crabswarm/hook/path"
)

func hookPathWindowsCmd(parent *cobra.Command) {
	cmd := &cobra.Command{
		Use:   "windows",
		Short: "Block edits whose path is invalid on Windows",
		Long: `windows is a PreToolUse hook that reads a Claude / Codex hook envelope
from stdin and blocks the tool call when any edited file path is not
valid on Windows.

A path is rejected when any of its components, taken relative to the
hook's cwd:
  - collides with a Windows reserved device name (CON, PRN, AUX, NUL,
    COM1-COM9, LPT1-LPT9, and the COM¹/COM²/COM³ and LPT¹/LPT²/LPT³
    superscript variants), including names with an extension such as
    NUL.txt;
  - contains a Windows-reserved character (< > : " | ? *), a backslash,
    or a control character;
  - ends with a space or period, which Windows silently trims.

The rules follow Microsoft's "Naming Files, Paths, and Namespaces"
reference:
https://learn.microsoft.com/en-us/windows/win32/fileio/naming-a-file

Wire it on PreToolUse for the file-editing tools so a repository edited
on a POSIX host stays checkout-able on Windows.`,
		Args:              cobra.NoArgs,
		ValidArgsFunction: cobra.NoFileCompletions,
		RunE:              runHookPathWindows,
	}

	parent.AddCommand(cmd)
}

func runHookPathWindows(cmd *cobra.Command, _ []string) error {
	ctx := cmd.Context()

	reader := stdiopipe.Stdin(ctx)
	defer reader.Close()

	return hookpath.Windows(ctx, reader)
}
