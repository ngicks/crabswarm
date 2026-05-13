package crabswarm

import (
	"path/filepath"
	"testing"
)

func TestSocketPath(t *testing.T) {
	tests := []struct {
		name     string
		override string
		envSock  string
		envXDG   string
		want     string
	}{
		{
			name:     "override takes precedence",
			override: "/custom/path.sock",
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
			t.Setenv(EnvSock, tt.envSock)
			t.Setenv(envXDGRuntime, tt.envXDG)

			got := SocketPath(tt.override)
			if got != tt.want {
				t.Errorf("SocketPath(%q) = %q, want %q", tt.override, got, tt.want)
			}
		})
	}
}

func TestProjectDir(t *testing.T) {
	t.Run("returns env value", func(t *testing.T) {
		t.Setenv(EnvProjectDir, "/work/proj")
		if got := ProjectDir(); got != "/work/proj" {
			t.Errorf("ProjectDir() = %q, want %q", got, "/work/proj")
		}
	})

	t.Run("returns empty when unset", func(t *testing.T) {
		t.Setenv(EnvProjectDir, "")
		if got := ProjectDir(); got != "" {
			t.Errorf("ProjectDir() = %q, want empty", got)
		}
	})
}
