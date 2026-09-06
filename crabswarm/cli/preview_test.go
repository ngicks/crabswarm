package cli_test

import (
	"errors"
	"strings"
	"testing"

	"connectrpc.com/connect"
	"gotest.tools/v3/assert"

	"github.com/ngicks/crabswarm/crabswarm/cli"
)

// A concrete host (name or IP) is kept verbatim in the printed URL, with the
// root routed at /roots/<id>/.
func TestPreviewURL_ConcreteHost(t *testing.T) {
	assert.Equal(t, cli.PreviewURL("192.168.1.5:6419", "abc"), "http://192.168.1.5:6419/roots/abc/")
	assert.Equal(t,
		cli.PreviewURL("host.tailnet.ts.net:6419", "r1"),
		"http://host.tailnet.ts.net:6419/roots/r1/",
	)
}

// A wildcard bind is rewritten to a routable host (this machine's hostname or
// localhost) so the URL is openable from another device; it never prints the
// wildcard, and keeps the port and /roots/<id>/ path.
func TestPreviewURL_WildcardRewritten(t *testing.T) {
	for _, addr := range []string{"0.0.0.0:6419", ":6419", "[::]:6419"} {
		got := cli.PreviewURL(addr, "x")
		assert.Assert(t, strings.HasPrefix(got, "http://"), "addr=%q got=%q", addr, got)
		assert.Assert(t, strings.HasSuffix(got, ":6419/roots/x/"), "addr=%q got=%q", addr, got)
		assert.Assert(t, !strings.Contains(got, "0.0.0.0"), "addr=%q got=%q", addr, got)
	}
}

// An address without a host:port shape is used verbatim rather than guessed.
func TestPreviewURL_Unparsable(t *testing.T) {
	assert.Equal(t, cli.PreviewURL("garbage", "y"), "http://garbage/roots/y/")
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
