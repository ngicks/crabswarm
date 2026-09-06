package cli_test

import (
	"bytes"
	"strings"
	"testing"

	"gotest.tools/v3/assert"

	"github.com/ngicks/crabswarm/crabswarm/issues/cli"
)

// roots and sources are listed in one table, each row carrying its kind, ID,
// name and path.
func TestRenderRegistrations(t *testing.T) {
	var buf bytes.Buffer
	assert.NilError(t, cli.RenderRegistrations(&buf, []cli.Registration{
		{Kind: cli.KindRoot, ID: "id1", Name: "docs", Path: "/home/me/docs"},
		{Kind: cli.KindSource, ID: "id2", Name: "crabswarm", Path: "/home/me/repo/.beads"},
	}))
	out := buf.String()
	for _, want := range []string{
		"KIND", "ID", "NAME", "PATH",
		"root", "id1", "docs", "/home/me/docs",
		"source", "id2", "crabswarm", "/home/me/repo/.beads",
	} {
		assert.Assert(t, strings.Contains(out, want), "missing %q in:\n%s", want, out)
	}
}

// An empty registry still prints the header so the output shape is stable.
func TestRenderRegistrations_Empty(t *testing.T) {
	var buf bytes.Buffer
	assert.NilError(t, cli.RenderRegistrations(&buf, nil))
	assert.Assert(t, strings.Contains(buf.String(), "KIND"))
}

// Every column a user can type — a root's ID or name, a source's ID or prefix
// — resolves to that registration.
func TestResolveRegistration(t *testing.T) {
	regs := []cli.Registration{
		{Kind: cli.KindRoot, ID: "r1", Name: "docs", Path: "/home/me/docs"},
		{Kind: cli.KindSource, ID: "s1", Name: "crabswarm", Path: "/home/me/repo/.beads"},
	}

	for _, tc := range []struct {
		arg  string
		want cli.Registration
	}{
		{arg: "r1", want: regs[0]},
		{arg: "docs", want: regs[0]},
		{arg: "s1", want: regs[1]},
		{arg: "crabswarm", want: regs[1]},
	} {
		got, err := cli.ResolveRegistration(regs, tc.arg)
		assert.NilError(t, err, "arg=%q", tc.arg)
		assert.Equal(t, got, tc.want, "arg=%q", tc.arg)
	}
}

func TestResolveRegistration_NoMatch(t *testing.T) {
	_, err := cli.ResolveRegistration([]cli.Registration{
		{Kind: cli.KindRoot, ID: "r1", Name: "docs"},
	}, "nope")
	assert.ErrorContains(t, err, `"nope"`)
}

// Two beads databases can carry the same issue-ID prefix, and a prefix can
// equal a root's name; either way the ambiguous argument is refused with the
// candidates named.
func TestResolveRegistration_Ambiguous(t *testing.T) {
	_, err := cli.ResolveRegistration([]cli.Registration{
		{Kind: cli.KindRoot, ID: "r1", Name: "shared"},
		{Kind: cli.KindSource, ID: "s1", Name: "shared"},
		{Kind: cli.KindSource, ID: "s2", Name: "shared"},
	}, "shared")
	assert.ErrorContains(t, err, "3 registrations")
	assert.ErrorContains(t, err, "root r1")
	assert.ErrorContains(t, err, "source s1")
	assert.ErrorContains(t, err, "source s2")
}
