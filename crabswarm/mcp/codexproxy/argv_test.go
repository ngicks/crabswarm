package codexproxy

import (
	"strings"
	"testing"

	"gotest.tools/v3/assert"
)

// The --remote address is replaced in whichever spelling the command line
// gives it, and every other argument stays where it was.
func TestRemote_RewritesTheAddressInPlace(t *testing.T) {
	for _, tc := range []struct {
		name     string
		argv     []string
		wantPath string
		want     []string
	}{
		{
			name:     "split",
			argv:     []string{"codex", "-m", "gpt", "--remote", "unix:///tmp/app.sock", "hi"},
			wantPath: "/tmp/app.sock",
			want:     []string{"codex", "-m", "gpt", "--remote", "unix:///run/p.sock", "hi"},
		},
		{
			name:     "inline",
			argv:     []string{"codex", "--remote=unix:///tmp/app.sock", "-c", "x=1"},
			wantPath: "/tmp/app.sock",
			want:     []string{"codex", "--remote=unix:///run/p.sock", "-c", "x=1"},
		},
		{
			name:     "beside the token flag",
			argv:     []string{"codex", "--remote-auth-token-env", "T", "--remote", "unix://rel.sock"},
			wantPath: "rel.sock",
			want:     []string{"codex", "--remote-auth-token-env", "T", "--remote", "unix:///run/p.sock"},
		},
		{
			name:     "before a prompt that mentions the flag",
			argv:     []string{"codex", "--remote", "unix:///a.sock", "--", "--remote", "ws://x"},
			wantPath: "/a.sock",
			want:     []string{"codex", "--remote", "unix:///run/p.sock", "--", "--remote", "ws://x"},
		},
	} {
		t.Run(tc.name, func(t *testing.T) {
			orig := strings.Join(tc.argv, " ")

			remote, err := findRemote(tc.argv)
			assert.NilError(t, err)
			path, err := unixSocketPath(remote.addr)
			assert.NilError(t, err)
			assert.Equal(t, path, tc.wantPath)
			assert.DeepEqual(t, withRemote(tc.argv, remote, "unix:///run/p.sock"), tc.want)
			assert.Equal(t, strings.Join(tc.argv, " "), orig, "the input was modified")
		})
	}
}

// A command line the proxy cannot put itself in front of is refused, and the
// refusal says what to give instead.
func TestRemote_RefusesWhatItCannotRelay(t *testing.T) {
	for _, tc := range []struct {
		name string
		argv []string
		want string
	}{
		{name: "no command", argv: nil, want: "no command to run"},
		{name: "no remote", argv: []string{"codex", "-m", "gpt"}, want: "names no app server"},
		{
			name: "remote only past --",
			argv: []string{"codex", "--", "--remote", "unix:///a.sock"},
			want: "names no app server",
		},
		{name: "remote without a value", argv: []string{"codex", "--remote"}, want: "no address"},
		{
			name: "remote twice",
			argv: []string{"codex", "--remote", "unix:///a.sock", "--remote=unix:///b.sock"},
			want: "given 2 times",
		},
		{name: "websocket", argv: []string{"codex", "--remote", "ws://127.0.0.1:1"}, want: "unix://"},
		{name: "default socket", argv: []string{"codex", "--remote", "unix://"}, want: "unix://PATH"},
	} {
		t.Run(tc.name, func(t *testing.T) {
			remote, err := findRemote(tc.argv)
			if err == nil {
				_, err = unixSocketPath(remote.addr)
			}
			assert.ErrorContains(t, err, tc.want)
		})
	}
}
