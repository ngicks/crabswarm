package path

import (
	"bytes"
	"context"
	"encoding/json"
	"errors"
	"strings"
	"testing"

	sdktypesv1 "github.com/ngicks/crabswarm/pkg/api/types/sdktypes/v1"
	"github.com/ngicks/crabswarm/pkg/claudehook/handler"
	"gotest.tools/v3/assert"
)

func TestCheckWindows(t *testing.T) {
	const cwd = "/work"

	tests := []struct {
		name  string
		cwd   string
		files []string
		// want is the offending component of each expected violation, in
		// order; nil means the input is clean.
		want []string
	}{
		{
			name:  "clean path",
			cwd:   cwd,
			files: []string{"/work/src/main.go"},
			want:  nil,
		},
		{
			name:  "reserved bare name",
			cwd:   cwd,
			files: []string{"/work/CON"},
			want:  []string{"CON"},
		},
		{
			name:  "reserved name with extension",
			cwd:   cwd,
			files: []string{"/work/src/nul.txt"},
			want:  []string{"nul.txt"},
		},
		{
			name:  "reserved name case-insensitive",
			cwd:   cwd,
			files: []string{"/work/Aux.tar.gz"},
			want:  []string{"Aux.tar.gz"},
		},
		{
			name:  "reserved com and lpt ports",
			cwd:   cwd,
			files: []string{"/work/com1", "/work/lpt9"},
			want:  []string{"com1", "lpt9"},
		},
		{
			name:  "reserved superscript com and lpt ports",
			cwd:   cwd,
			files: []string{"/work/COM¹", "/work/lpt³.log"},
			want:  []string{"COM¹", "lpt³.log"},
		},
		{
			name:  "com10 is not reserved",
			cwd:   cwd,
			files: []string{"/work/com10"},
			want:  nil,
		},
		{
			name:  "reserved component is a directory",
			cwd:   cwd,
			files: []string{"/work/prn/notes.txt"},
			want:  []string{"prn"},
		},
		{
			name: "reserved chars",
			cwd:  cwd,
			files: []string{
				`/work/a:b`,
				`/work/a|b`,
				`/work/a?b`,
				`/work/a*b`,
				`/work/a"b`,
				`/work/a<b`,
				`/work/a>b`,
			},
			want: []string{"a:b", "a|b", "a?b", "a*b", `a"b`, "a<b", "a>b"},
		},
		{
			name:  "backslash in component",
			cwd:   cwd,
			files: []string{`/work/a\b`},
			want:  []string{`a\b`},
		},
		{
			name:  "control character",
			cwd:   cwd,
			files: []string{"/work/a\x01b"},
			want:  []string{"a\x01b"},
		},
		{
			name:  "trailing period",
			cwd:   cwd,
			files: []string{"/work/file."},
			want:  []string{"file."},
		},
		{
			name:  "trailing space",
			cwd:   cwd,
			files: []string{"/work/file "},
			want:  []string{"file "},
		},
		{
			name:  "host prefix outside cwd is not flagged",
			cwd:   cwd,
			files: []string{"/home/aux/project/src/main.go"},
			want:  nil,
		},
		{
			name:  "outside cwd still checks base name",
			cwd:   cwd,
			files: []string{"/home/aux/project/nul.txt"},
			want:  []string{"nul.txt"},
		},
		{
			name:  "no cwd falls back to base name",
			cwd:   "",
			files: []string{"/some/aux/con.txt"},
			want:  []string{"con.txt"},
		},
		{
			name:  "multiple files and components",
			cwd:   cwd,
			files: []string{"/work/aux/con.txt", "/work/ok.go"},
			want:  []string{"aux", "con.txt"},
		},
		{
			name:  "empty files",
			cwd:   cwd,
			files: nil,
			want:  nil,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := CheckWindows(tt.cwd, tt.files)
			var gotComponents []string
			for _, v := range got {
				gotComponents = append(gotComponents, v.Component)
				assert.Assert(t, v.Reason != "", "violation %+v missing reason", v)
			}
			assert.DeepEqual(t, gotComponents, tt.want)
		})
	}
}

// writeEnvelope builds a PreToolUse + Write envelope for a single file.
func writeEnvelope(t *testing.T, cwd, filePath string) *bytes.Reader {
	t.Helper()
	in := &sdktypesv1.PreToolUseHookInput{
		BaseHookInput: sdktypesv1.BaseHookInput{SessionID: "s", Cwd: cwd},
		HookEventName: sdktypesv1.HookEventPreToolUse,
		ToolName:      sdktypesv1.ToolNameWrite,
		ToolInput: sdktypesv1.NewToolInputSchemas(
			&sdktypesv1.FileWriteInput{FilePath: filePath, Content: "x"},
		),
		ToolUseID: "t1",
	}
	data, err := json.Marshal(sdktypesv1.NewHookInput(in))
	assert.NilError(t, err)
	return bytes.NewReader(data)
}

func TestWindows_Blocks(t *testing.T) {
	err := Windows(context.Background(), writeEnvelope(t, "/work", "/work/src/nul.txt"))

	var he *handler.HandlerError
	if !errors.As(err, &he) {
		t.Fatalf("expected *handler.HandlerError, got %T: %v", err, err)
	}
	assert.Assert(t, he.Output != nil, "expected block output, got allow")
	assert.Assert(t, he.Output.Decision != nil &&
		*he.Output.Decision == sdktypesv1.HookDecisionBlock)
	assert.Assert(t, he.Output.Reason != nil)
	assert.Assert(t, strings.Contains(*he.Output.Reason, "nul.txt"),
		"reason should name the offending component: %q", *he.Output.Reason)
}

func TestWindows_Allows(t *testing.T) {
	err := Windows(context.Background(), writeEnvelope(t, "/work", "/work/src/main.go"))

	var he *handler.HandlerError
	if !errors.As(err, &he) {
		t.Fatalf("expected *handler.HandlerError, got %T: %v", err, err)
	}
	assert.Assert(t, he.Output == nil, "expected allow (nil Output), got %+v", he.Output)
}

func TestWindows_NonEditEventPassesThrough(t *testing.T) {
	in := &sdktypesv1.PreToolUseHookInput{
		BaseHookInput: sdktypesv1.BaseHookInput{Cwd: "/work"},
		HookEventName: sdktypesv1.HookEventPreToolUse,
		ToolName:      sdktypesv1.ToolNameBash,
		ToolInput:     sdktypesv1.NewToolInputSchemas(&sdktypesv1.BashInput{Command: "ls"}),
		ToolUseID:     "t1",
	}
	data, err := json.Marshal(sdktypesv1.NewHookInput(in))
	assert.NilError(t, err)

	var he *handler.HandlerError
	gotErr := Windows(context.Background(), bytes.NewReader(data))
	if !errors.As(gotErr, &he) {
		t.Fatalf("expected *handler.HandlerError, got %T: %v", gotErr, gotErr)
	}
	assert.Assert(t, he.Output == nil, "expected allow (nil Output), got %+v", he.Output)
}

func TestWindows_InvalidJSON(t *testing.T) {
	err := Windows(context.Background(), strings.NewReader("{not json"))
	var he *handler.HandlerError
	assert.Assert(t, !errors.As(err, &he), "parse failure should be a plain error, got %T", err)
	assert.Assert(t, err != nil)
}
