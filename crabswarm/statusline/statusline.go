// Package statusline implements `crabswarm statusline render`: it reads a
// Claude Code status line JSON payload from a reader and renders it through a
// Go text/template supplied by the caller.
//
// Claude Code feeds the status line command a single JSON object on stdin and
// shows the command's stdout as the status line. [Input] models that payload;
// [Render] parses it and renders the caller's template against it. The schema
// is intentionally lenient: unknown fields are ignored (so newer Claude Code
// versions don't break rendering) and absent fields render as their zero
// value.
package statusline

import (
	"bytes"
	"encoding/json"
	"fmt"
	"io"
	"text/template"

	"github.com/ngicks/crabswarm/internal/templateutil"
)

// Input is the status line JSON payload Claude Code writes to stdin. Field
// names are the PascalCase of the JSON keys so a template can be written by
// mechanically transforming the documented key path, e.g.
// `context_window.used_percentage` → `{{ .ContextWindow.UsedPercentage }}`.
// Abbreviations use mixed case per project convention (.Model.Id, .Pr.Url).
//
// Objects that Claude Code omits in some sessions are pointers so a template
// can guard them with `{{ with }}` / `{{ if }}`; absent scalars are the zero
// value.
type Input struct {
	// Cwd is the current working directory. Workspace.CurrentDir carries the
	// same value and is preferred for consistency with the workspace block.
	Cwd string `json:"cwd"`
	// SessionId is the unique session identifier.
	SessionId string `json:"session_id"`
	// SessionName is the custom session name set via --name or /rename. Empty
	// when no custom name has been set.
	SessionName string `json:"session_name,omitzero"`
	// TranscriptPath is the path to the conversation transcript file.
	TranscriptPath string `json:"transcript_path"`
	// Model identifies the current model.
	Model Model `json:"model"`
	// Workspace holds directory and repository context.
	Workspace Workspace `json:"workspace"`
	// Version is the Claude Code version.
	Version string `json:"version"`
	// OutputStyle names the current output style.
	OutputStyle OutputStyle `json:"output_style"`
	// Cost holds client-side cost and duration estimates.
	Cost Cost `json:"cost"`
	// ContextWindow holds token-usage figures from the most recent API
	// response.
	ContextWindow ContextWindow `json:"context_window"`
	// Exceeds200kTokens reports whether the combined token count from the most
	// recent API response exceeds the fixed 200k threshold, regardless of the
	// actual context window size.
	Exceeds200kTokens bool `json:"exceeds_200k_tokens"`
	// Effort is the current reasoning effort. Absent when the model does not
	// support the effort parameter.
	Effort *Effort `json:"effort,omitzero"`
	// Thinking reports whether extended thinking is enabled.
	Thinking Thinking `json:"thinking"`
	// RateLimits holds rate-limit consumption windows.
	RateLimits RateLimits `json:"rate_limits"`
	// Vim is the current vim mode, present only when vim mode is enabled.
	Vim *Vim `json:"vim,omitzero"`
	// Agent names the active agent, present only with --agent or agent
	// settings.
	Agent *Agent `json:"agent,omitzero"`
	// Pr is the open pull request for the current branch, absent until one is
	// found and once it merges or closes.
	Pr *Pr `json:"pr,omitzero"`
	// Worktree describes the active worktree, present only during --worktree
	// sessions.
	Worktree *Worktree `json:"worktree,omitzero"`
}

// Model identifies the current model.
type Model struct {
	Id          string `json:"id"`
	DisplayName string `json:"display_name"`
}

// Workspace holds directory and repository context.
type Workspace struct {
	// CurrentDir is the current working directory (preferred over [Input.Cwd]).
	CurrentDir string `json:"current_dir"`
	// ProjectDir is the directory Claude Code was launched in, which may
	// differ from CurrentDir if the working directory changed mid-session.
	ProjectDir string `json:"project_dir"`
	// AddedDirs holds directories added via /add-dir or --add-dir; empty when
	// none have been added.
	AddedDirs []string `json:"added_dirs"`
	// GitWorktree is the linked-worktree name when the current directory is
	// inside one. Empty in the main working tree.
	GitWorktree string `json:"git_worktree,omitzero"`
	// Repo is the repository identity parsed from the origin remote. Absent
	// outside a git repository or when no origin remote is configured.
	Repo *Repo `json:"repo,omitzero"`
}

// Repo is the repository identity parsed from the origin remote.
type Repo struct {
	Host  string `json:"host"`
	Owner string `json:"owner"`
	Name  string `json:"name"`
}

// OutputStyle names the current output style.
type OutputStyle struct {
	Name string `json:"name"`
}

// Cost holds client-side cost and duration estimates for the session. The
// values are computed client-side and may differ from the actual bill.
type Cost struct {
	TotalCostUsd       float64 `json:"total_cost_usd"`
	TotalDurationMs    int64   `json:"total_duration_ms"`
	TotalApiDurationMs int64   `json:"total_api_duration_ms"`
	TotalLinesAdded    int     `json:"total_lines_added"`
	TotalLinesRemoved  int     `json:"total_lines_removed"`
}

// ContextWindow holds token-usage figures from the most recent API response.
type ContextWindow struct {
	TotalInputTokens  int `json:"total_input_tokens"`
	TotalOutputTokens int `json:"total_output_tokens"`
	ContextWindowSize int `json:"context_window_size"`
	// UsedPercentage and RemainingPercentage are pre-calculated by Claude
	// Code. They are float64 (not int) so a fractional value parses; an
	// integer like 8 still renders as "8".
	UsedPercentage      float64      `json:"used_percentage"`
	RemainingPercentage float64      `json:"remaining_percentage"`
	CurrentUsage        CurrentUsage `json:"current_usage"`
}

// CurrentUsage holds the token counts from the last API call.
type CurrentUsage struct {
	InputTokens              int `json:"input_tokens"`
	OutputTokens             int `json:"output_tokens"`
	CacheCreationInputTokens int `json:"cache_creation_input_tokens"`
	CacheReadInputTokens     int `json:"cache_read_input_tokens"`
}

// Effort is the current reasoning effort (low, medium, high, xhigh, or max).
type Effort struct {
	Level string `json:"level"`
}

// Thinking reports whether extended thinking is enabled.
type Thinking struct {
	Enabled bool `json:"enabled"`
}

// RateLimits holds the rate-limit consumption windows. A window is a pointer
// so a template can guard it with `{{ with }}` when only one is reported.
type RateLimits struct {
	FiveHour *RateLimitWindow `json:"five_hour,omitzero"`
	SevenDay *RateLimitWindow `json:"seven_day,omitzero"`
}

// RateLimitWindow is one rate-limit window.
type RateLimitWindow struct {
	// UsedPercentage is the consumed fraction of the window, from 0 to 100.
	UsedPercentage float64 `json:"used_percentage"`
	// ResetsAt is the Unix epoch seconds when the window resets.
	ResetsAt int64 `json:"resets_at"`
}

// Vim is the current vim mode (NORMAL, INSERT, VISUAL, or VISUAL LINE).
type Vim struct {
	Mode string `json:"mode"`
}

// Agent names the active agent.
type Agent struct {
	Name string `json:"name"`
}

// Pr is the open pull request for the current branch.
type Pr struct {
	Number int    `json:"number"`
	Url    string `json:"url"`
	// ReviewState is approved, pending, changes_requested, or draft. May be
	// absent even when the rest of Pr is present.
	ReviewState string `json:"review_state,omitzero"`
}

// Worktree describes the active worktree during a --worktree session.
type Worktree struct {
	Name string `json:"name"`
	Path string `json:"path"`
	// Branch is the worktree's git branch. Absent for hook-based worktrees.
	Branch string `json:"branch,omitzero"`
	// OriginalCwd is the directory Claude was in before entering the worktree.
	OriginalCwd string `json:"original_cwd"`
	// OriginalBranch is the branch checked out before entering the worktree.
	// Absent for hook-based worktrees.
	OriginalBranch string `json:"original_branch,omitzero"`
}

// Render reads a status line JSON payload from r, renders tmpl against the
// parsed [Input], and writes the rendered output to w. The template is
// rendered to a buffer first, so a template-execution error leaves nothing on
// w. No trailing newline is added; the template controls its own output.
func Render(r io.Reader, w io.Writer, tmpl string) error {
	raw, err := io.ReadAll(r)
	if err != nil {
		return fmt.Errorf("reading statusline input: %w", err)
	}

	var in Input
	if err := json.Unmarshal(raw, &in); err != nil {
		return fmt.Errorf("parsing statusline input: %w", err)
	}

	t, err := template.New("statusline").
		Option("missingkey=zero").
		Funcs(templateutil.FuncMap()).
		Parse(tmpl)
	if err != nil {
		return fmt.Errorf("parsing template: %w", err)
	}

	var buf bytes.Buffer
	if err := t.Execute(&buf, in); err != nil {
		return fmt.Errorf("rendering template: %w", err)
	}
	if _, err := w.Write(buf.Bytes()); err != nil {
		return fmt.Errorf("writing rendered statusline: %w", err)
	}
	return nil
}
