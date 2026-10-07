package commands

import (
	"bytes"
	"io"
	"net/http"
	"net/http/httptest"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"gotest.tools/v3/assert"
)

// runUtilCmd executes "util <args...>" through the real root command,
// capturing stdout and stderr separately (see runPreviewCmd).
func runUtilCmd(t *testing.T, args ...string) (stdout, stderr string, err error) {
	t.Helper()
	root := rootCmd()
	var outBuf, errBuf bytes.Buffer
	root.SetOut(&outBuf)
	root.SetErr(&errBuf)
	root.SetArgs(append([]string{"util"}, args...))
	err = root.Execute()
	return outBuf.String(), errBuf.String(), err
}

// An existing file satisfies a file:// target on the first probe, so the
// command returns at once without spending any of the retry budget.
func TestUtilPollWait_FileTargetReady(t *testing.T) {
	path := filepath.Join(t.TempDir(), "ready")
	assert.NilError(t, os.WriteFile(path, []byte("ok"), 0o600))

	_, _, err := runUtilCmd(t, "poll", "wait", "file://"+path)
	assert.NilError(t, err)
}

// The HTTP flags reach the probe: the method, every header, the body read from
// stdin through --file -, and the statuses --status accepts.
func TestUtilPollWait_HTTPFlags(t *testing.T) {
	var (
		method, probe, body string
	)
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		b, _ := io.ReadAll(r.Body)
		method, probe, body = r.Method, r.Header.Get("X-Probe"), string(b)
		w.WriteHeader(http.StatusAccepted)
	}))
	t.Cleanup(srv.Close)

	root := rootCmd()
	root.SetOut(io.Discard)
	root.SetErr(io.Discard)
	root.SetIn(strings.NewReader("from stdin"))
	root.SetArgs([]string{"util", "poll", "wait", srv.URL, "--retries", "1",
		"--method", "query", "--header", "X-Probe: yes", "--file", "-", "--status", "202"})
	assert.NilError(t, root.Execute())

	assert.Equal(t, method, "QUERY")
	assert.Equal(t, probe, "yes")
	assert.Equal(t, body, "from stdin")
}

// Flags that cannot describe a probe are refused before any probe runs.
func TestUtilPollWait_RefusesBadHTTPFlags(t *testing.T) {
	for _, tc := range []struct {
		name string
		args []string
		want string
	}{
		{
			name: "payload and file",
			args: []string{"http://127.0.0.1:1", "--payload", "a", "--file", "b"},
			want: "none of the others can be",
		},
		{name: "bad status", args: []string{"http://127.0.0.1:1", "--status", "2"}, want: "--status"},
		{name: "bad header", args: []string{"http://127.0.0.1:1", "--header", "x"}, want: "--header"},
		{
			name: "file target",
			args: []string{"file://" + filepath.Join(t.TempDir(), "x"), "--method", "HEAD"},
			want: "takes no HTTP request options",
		},
	} {
		t.Run(tc.name, func(t *testing.T) {
			_, _, err := runUtilCmd(t, append([]string{"poll", "wait"}, tc.args...)...)
			assert.ErrorContains(t, err, tc.want)
		})
	}
}

// wait takes exactly one target, so calling it with none fails before any
// probe runs.
func TestUtilPollWait_MissingArg(t *testing.T) {
	_, _, err := runUtilCmd(t, "poll", "wait")
	assert.Assert(t, err != nil)
}
