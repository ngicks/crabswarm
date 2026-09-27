package commands

import (
	"bytes"
	"context"
	"net"
	"path/filepath"
	"strings"
	"testing"
	"time"

	mcpsdk "github.com/modelcontextprotocol/go-sdk/mcp"
	"golang.org/x/sync/errgroup"
	"gotest.tools/v3/assert"

	"github.com/ngicks/crabswarm/crabswarm/mcp"
)

// runMCPCmd executes "mcp <args...>" through the real root command.
func runMCPCmd(ctx context.Context, args ...string) error {
	root := rootCmd()
	var outBuf, errBuf bytes.Buffer
	root.SetOut(&outBuf)
	root.SetErr(&errBuf)
	root.SetArgs(append([]string{"mcp"}, args...))
	return root.ExecuteContext(ctx)
}

// A flag combination that names no way to serve, or one the transport would
// ignore, is refused before anything is dialed or listened on, and the refusal
// names the flag to fix.
func TestMCPCmd_RefusesAMismatchedTransport(t *testing.T) {
	for _, tc := range []struct {
		name string
		args []string
		want string
	}{
		{
			name: "stdio with an address",
			args: []string{"--listen", "127.0.0.1:7801"},
			want: "--transport http",
		},
		{
			name: "http without an address",
			args: []string{"--transport", "http"},
			want: "--listen",
		},
		{
			name: "http with a token",
			args: []string{"--transport", "http", "--listen", "127.0.0.1:7801", "--token", "tok"},
			want: mcp.TokenHeader,
		},
		{
			name: "an unknown transport",
			args: []string{"--transport", "sse"},
			want: `unknown --transport "sse"`,
		},
	} {
		t.Run(tc.name, func(t *testing.T) {
			chatHermeticEnv(t)

			err := runMCPCmd(t.Context(), tc.args...)
			assert.Assert(t, err != nil)
			assert.Assert(t, strings.Contains(err.Error(), tc.want),
				"%q is missing from %q", tc.want, err.Error())
		})
	}
}

// freeAddr is a loopback address nothing listens on right now.
func freeAddr(t *testing.T) string {
	t.Helper()

	ln, err := net.Listen("tcp", "127.0.0.1:0")
	assert.NilError(t, err)
	addr := ln.Addr().String()
	assert.NilError(t, ln.Close())
	return addr
}

// --transport http serves the tools at /mcp on --listen, and a shutdown the
// caller asked for is a clean exit.
func TestMCPCmd_ServesOverHTTP(t *testing.T) {
	chatHermeticEnv(t)
	addr := freeAddr(t)

	ctx, cancel := context.WithCancel(t.Context())
	defer cancel()
	var served errgroup.Group
	served.Go(func() error {
		// The daemon is never reached: listing tools asks nothing of it.
		return runMCPCmd(ctx, "--transport", "http", "--listen", addr,
			"--sock", filepath.Join(t.TempDir(), "absent.sock"))
	})

	client := mcpsdk.NewClient(&mcpsdk.Implementation{Name: "test-harness", Version: "v0"}, nil)
	var session *mcpsdk.ClientSession
	deadline := time.Now().Add(5 * time.Second)
	for {
		var err error
		session, err = client.Connect(t.Context(),
			&mcpsdk.StreamableClientTransport{Endpoint: "http://" + addr + mcp.HTTPPath}, nil)
		if err == nil {
			break
		}
		if time.Now().After(deadline) {
			t.Fatalf("the server never answered on %s: %v", addr, err)
		}
		time.Sleep(10 * time.Millisecond)
	}

	tools, err := session.ListTools(t.Context(), nil)
	assert.NilError(t, err)
	names := make([]string, 0, len(tools.Tools))
	for _, tool := range tools.Tools {
		names = append(names, tool.Name)
	}
	assert.Assert(t, strings.Contains(strings.Join(names, " "), "chat_send"),
		"the tools served are %v", names)
	assert.NilError(t, session.Close())

	cancel()
	assert.NilError(t, served.Wait())
}
