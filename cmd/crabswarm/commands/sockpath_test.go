package commands

import (
	"path/filepath"
	"testing"

	"github.com/spf13/cobra"
)

func TestResolveSocketPath(t *testing.T) {
	tests := []struct {
		name     string
		sockFlag string
		envSock  string
		envXDG   string
		want     string
	}{
		{
			name:     "flag takes precedence",
			sockFlag: "/custom/path.sock",
			envSock:  "/env/path.sock",
			envXDG:   "/run/user/1000",
			want:     "/custom/path.sock",
		},
		{
			name:    "env CRABSWARM_SOCK takes precedence over XDG",
			envSock: "/env/path.sock",
			envXDG:  "/run/user/1000",
			want:    "/env/path.sock",
		},
		{
			name:   "XDG_RUNTIME_DIR fallback",
			envXDG: "/run/user/1000",
			want:   filepath.Join("/run/user/1000", "crabswarm", "default.sock"),
		},
		{
			name: "default to /tmp when nothing set",
			want: filepath.Join("/tmp", "crabswarm", "default.sock"),
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Setenv("CRABSWARM_SOCK", tt.envSock)
			t.Setenv("XDG_RUNTIME_DIR", tt.envXDG)

			cmd := &cobra.Command{}
			cmd.Flags().String("sock", tt.sockFlag, "")

			got := resolveSocketPath(cmd)
			if got != tt.want {
				t.Errorf("resolveSocketPath() = %q, want %q", got, tt.want)
			}
		})
	}
}
