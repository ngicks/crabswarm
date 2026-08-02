package preview

import (
	"testing"

	"gotest.tools/v3/assert"
)

func TestBaseURL_RewritesWildcardHostToLoopback(t *testing.T) {
	cases := map[string]struct {
		addr string
		want string
	}{
		"ipv4 wildcard": {"0.0.0.0:6419", "http://127.0.0.1:6419"},
		"empty host":    {":6419", "http://127.0.0.1:6419"},
		"ipv6 wildcard": {"[::]:6419", "http://[::1]:6419"},
		"loopback kept": {"127.0.0.1:6419", "http://127.0.0.1:6419"},
		"hostname kept": {"host.tailnet.ts.net:6419", "http://host.tailnet.ts.net:6419"},
		"concrete ip":   {"192.168.1.10:6419", "http://192.168.1.10:6419"},
		"no port kept":  {"localhost", "http://localhost"},
	}
	for name, tc := range cases {
		t.Run(name, func(t *testing.T) {
			assert.Equal(t, baseURL(tc.addr), tc.want)
		})
	}
}

func TestNewClient_ReturnsNonNilClient(t *testing.T) {
	// NewClient does no I/O; it only wires the transport, so this just guards
	// the constructor's wiring compiles and returns a usable value.
	c := NewClient("0.0.0.0:6419")
	assert.Assert(t, c != nil)
}
