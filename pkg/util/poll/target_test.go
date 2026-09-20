package poll

import (
	"testing"

	"gotest.tools/v3/assert"
)

func TestParseTarget(t *testing.T) {
	for _, tc := range []struct {
		in   string
		kind Kind
		addr string
	}{
		// A bare path is a socket path, absolute or relative.
		{"/run/crabswarm/default.sock", KindUnix, "/run/crabswarm/default.sock"},
		{"run/default.sock", KindUnix, "run/default.sock"},
		// The prefix is stripped literally, so the leading slash count survives.
		{"unix:///run/crabswarm/default.sock", KindUnix, "/run/crabswarm/default.sock"},
		{"unix://run/default.sock", KindUnix, "run/default.sock"},
		{"file:///var/lib/crabswarm/ready", KindFile, "/var/lib/crabswarm/ready"},
		{"file://ready", KindFile, "ready"},
		// http and https keep the whole string.
		{"http://localhost:8080/healthz", KindHTTP, "http://localhost:8080/healthz"},
		{"https://example.com/healthz", KindHTTP, "https://example.com/healthz"},
	} {
		got, err := ParseTarget(tc.in)
		assert.NilError(t, err, "input %q", tc.in)
		assert.Equal(t, got.Kind, tc.kind, "input %q", tc.in)
		assert.Equal(t, got.Addr, tc.addr, "input %q", tc.in)
	}
}

func TestParseTargetErrors(t *testing.T) {
	for _, in := range []string{
		"",
		"unix://",
		"file://",
		"http://",
		"https://",
		"tcp://localhost:8080",
		"ftp://example.com/ready",
		"://no-scheme",
	} {
		_, err := ParseTarget(in)
		assert.Assert(t, err != nil, "expected an error for %q", in)
	}
}
