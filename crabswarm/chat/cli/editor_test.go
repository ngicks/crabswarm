package cli

import (
	"os"
	"testing"

	"gotest.tools/v3/assert"
)

func TestEditorFromEnv_Precedence(t *testing.T) {
	for _, tc := range []struct {
		name string
		// env holds only the variables that are set at all; the two the
		// lookup reads are cleared first, so an absent key is an unset
		// variable rather than an empty one.
		env  map[string]string
		want string
	}{
		{
			name: "VISUAL wins over EDITOR",
			env:  map[string]string{VisualEnvVar: "nvim", EditorEnvVar: "ed"},
			want: "nvim",
		},
		{
			name: "EDITOR answers when VISUAL is unset",
			env:  map[string]string{EditorEnvVar: "ed"},
			want: "ed",
		},
		{
			name: "an empty VISUAL falls through to EDITOR",
			env:  map[string]string{VisualEnvVar: "", EditorEnvVar: "ed"},
			want: "ed",
		},
		{
			// A variable holding only spaces is a variable holding nothing:
			// splitting it yields no argv and there is nothing to run.
			name: "a blank VISUAL falls through to EDITOR",
			env:  map[string]string{VisualEnvVar: "   ", EditorEnvVar: "ed"},
			want: "ed",
		},
		{
			name: "neither set is no editor",
			want: "",
		},
		{
			name: "the command line is kept whole",
			env:  map[string]string{VisualEnvVar: "code -w"},
			want: "code -w",
		},
		{
			name: "surrounding blanks are not part of the command",
			env:  map[string]string{VisualEnvVar: "  nvim  "},
			want: "nvim",
		},
	} {
		t.Run(tc.name, func(t *testing.T) {
			for _, name := range []string{VisualEnvVar, EditorEnvVar} {
				// t.Setenv registers the restore; unsetting after it is what
				// makes "not set at all" a case this table can spell.
				t.Setenv(name, "")
				if v, ok := tc.env[name]; ok {
					t.Setenv(name, v)
					continue
				}
				assert.NilError(t, os.Unsetenv(name))
			}
			assert.Equal(t, EditorFromEnv(), tc.want)
		})
	}
}
