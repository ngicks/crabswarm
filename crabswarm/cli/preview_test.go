package cli_test

import (
	"bytes"
	"errors"
	"strings"
	"testing"

	"connectrpc.com/connect"
	"gotest.tools/v3/assert"

	"github.com/ngicks/crabswarm/crabswarm/cli"
)

// A concrete host (name or IP) is kept verbatim in the printed URL, with the
// root mounted under /r/<id>/.
func TestPreviewURL_ConcreteHost(t *testing.T) {
	assert.Equal(t, cli.PreviewURL("192.168.1.5:6419", "abc"), "http://192.168.1.5:6419/r/abc/")
	assert.Equal(t,
		cli.PreviewURL("host.tailnet.ts.net:6419", "r1"),
		"http://host.tailnet.ts.net:6419/r/r1/",
	)
}

// A wildcard bind is rewritten to a routable host (this machine's hostname or
// localhost) so the URL is openable from another device; it never prints the
// wildcard, and keeps the port and /r/<id>/ path.
func TestPreviewURL_WildcardRewritten(t *testing.T) {
	for _, addr := range []string{"0.0.0.0:6419", ":6419", "[::]:6419"} {
		got := cli.PreviewURL(addr, "x")
		assert.Assert(t, strings.HasPrefix(got, "http://"), "addr=%q got=%q", addr, got)
		assert.Assert(t, strings.HasSuffix(got, ":6419/r/x/"), "addr=%q got=%q", addr, got)
		assert.Assert(t, !strings.Contains(got, "0.0.0.0"), "addr=%q got=%q", addr, got)
	}
}

// An address without a host:port shape is used verbatim rather than guessed.
func TestPreviewURL_Unparsable(t *testing.T) {
	assert.Equal(t, cli.PreviewURL("garbage", "y"), "http://garbage/r/y/")
}

// RenderRoots always prints the header and one aligned row per root.
func TestRenderRoots(t *testing.T) {
	var buf bytes.Buffer
	assert.NilError(t, cli.RenderRoots(&buf, []cli.PreviewRoot{
		{ID: "id1", Name: "docs", Path: "/home/me/docs"},
	}))
	out := buf.String()
	assert.Assert(t, strings.Contains(out, "ID"))
	assert.Assert(t, strings.Contains(out, "NAME"))
	assert.Assert(t, strings.Contains(out, "PATH"))
	assert.Assert(t, strings.Contains(out, "id1"))
	assert.Assert(t, strings.Contains(out, "docs"))
	assert.Assert(t, strings.Contains(out, "/home/me/docs"))
}

// An empty list still prints the header so the output shape is stable.
func TestRenderRoots_Empty(t *testing.T) {
	var buf bytes.Buffer
	assert.NilError(t, cli.RenderRoots(&buf, nil))
	assert.Assert(t, strings.Contains(buf.String(), "ID"))
}

// A refused connection surfaces as connect CodeUnavailable; PreviewDaemonError
// wraps it with a start-the-daemon hint that names the cmdman command.
func TestPreviewDaemonError_Unavailable(t *testing.T) {
	err := connect.NewError(connect.CodeUnavailable, errors.New("connection refused"))
	got := cli.PreviewDaemonError(err, "crabswarm-preview")
	assert.Assert(t, got != nil)
	assert.Assert(t, strings.Contains(got.Error(), "unreachable"))
	assert.Assert(t, strings.Contains(got.Error(), "crabswarm-preview"))
	assert.Assert(t, errors.Is(got, err))
}

// A non-transport error (e.g. a not-found root) is returned unchanged: no hint.
func TestPreviewDaemonError_OtherCodePassthrough(t *testing.T) {
	err := connect.NewError(connect.CodeNotFound, errors.New("root not found"))
	got := cli.PreviewDaemonError(err, "crabswarm-preview")
	assert.Equal(t, got, err)
	assert.Assert(t, !strings.Contains(got.Error(), "unreachable"))
}

// A nil error passes through as nil.
func TestPreviewDaemonError_Nil(t *testing.T) {
	assert.NilError(t, cli.PreviewDaemonError(nil, "crabswarm-preview"))
}
