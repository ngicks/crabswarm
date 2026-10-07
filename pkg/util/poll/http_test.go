package poll

import (
	"io"
	"net/http"
	"net/http/httptest"
	"path/filepath"
	"sync"
	"testing"

	"gotest.tools/v3/assert"
)

func TestParseStatus(t *testing.T) {
	for _, tc := range []struct {
		name  string
		specs []string
		want  []StatusRange
		err   string
	}{
		{name: "none"},
		{name: "a code", specs: []string{"204"}, want: []StatusRange{{204, 204}}},
		{name: "a class", specs: []string{"2xx"}, want: []StatusRange{{200, 299}}},
		{name: "upper-case class", specs: []string{"4XX"}, want: []StatusRange{{400, 499}}},
		{
			name:  "comma-separated and repeated",
			specs: []string{"2xx, 401", "503"},
			want:  []StatusRange{{200, 299}, {401, 401}, {503, 503}},
		},
		{name: "out of range", specs: []string{"600"}, err: `status "600"`},
		{name: "a class out of range", specs: []string{"6xx"}, err: `status "6xx"`},
		{name: "not a number", specs: []string{"ok"}, err: `status "ok"`},
		{name: "an empty item", specs: []string{"200,"}, err: `status ""`},
	} {
		t.Run(tc.name, func(t *testing.T) {
			got, err := ParseStatus(tc.specs)
			if tc.err != "" {
				assert.ErrorContains(t, err, tc.err)
				return
			}
			assert.NilError(t, err)
			assert.DeepEqual(t, got, tc.want)
		})
	}
}

func TestParseHeader(t *testing.T) {
	h, err := ParseHeader([]string{
		"content-type: application/json",
		"X-Probe:1",
		"X-Probe: 2",
		"Authorization: Bearer a:b",
	})
	assert.NilError(t, err)
	assert.DeepEqual(t, h, http.Header{
		"Content-Type":  {"application/json"},
		"X-Probe":       {"1", "2"},
		"Authorization": {"Bearer a:b"},
	})

	for _, bad := range []string{"no colon", ": no name", "Bad Name: v"} {
		_, err := ParseHeader([]string{bad})
		assert.ErrorContains(t, err, "Name: value", "line %q", bad)
	}
}

// probeRecord is what a recordingServer saw of one request.
type probeRecord struct {
	method string
	host   string
	header http.Header
	body   string
}

// recordingServer answers every request with status and records it.
func recordingServer(t *testing.T, status int) (url string, seen func() []probeRecord) {
	t.Helper()
	var (
		mu  sync.Mutex
		got []probeRecord
	)
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		body, _ := io.ReadAll(r.Body)
		mu.Lock()
		got = append(got, probeRecord{r.Method, r.Host, r.Header.Clone(), string(body)})
		mu.Unlock()
		w.WriteHeader(status)
	}))
	t.Cleanup(srv.Close)
	return srv.URL, func() []probeRecord {
		mu.Lock()
		defer mu.Unlock()
		return append([]probeRecord(nil), got...)
	}
}

// Every probe carries the method, headers and body HTTPOption names, and the
// status HTTPOption accepts ends the wait.
func TestWaitHTTPOption(t *testing.T) {
	url, seen := recordingServer(t, http.StatusUnauthorized)
	target, err := ParseTarget(url + "/rpc")
	assert.NilError(t, err)
	header, err := ParseHeader([]string{"X-Probe: 1", "Host: example.test"})
	assert.NilError(t, err)
	status, err := ParseStatus([]string{"2xx,401"})
	assert.NilError(t, err)

	opts := fastWaitOption()
	opts.HTTP = HTTPOption{
		Method: "QUERY",
		Header: header,
		Body:   []byte(`{"q":1}`),
		Status: status,
	}
	assert.NilError(t, Wait(t.Context(), nil, target, opts))

	got := seen()
	assert.Equal(t, len(got), 1)
	assert.Equal(t, got[0].method, "QUERY")
	assert.Equal(t, got[0].host, "example.test")
	assert.Equal(t, got[0].header.Get("X-Probe"), "1")
	assert.Equal(t, got[0].body, `{"q":1}`)
}

// The body is sent whole on every probe, not only on the first.
func TestWaitHTTPOptionResendsTheBody(t *testing.T) {
	url, seen := recordingServer(t, http.StatusServiceUnavailable)
	target, err := ParseTarget(url)
	assert.NilError(t, err)

	opts := fastWaitOption()
	opts.HTTP = HTTPOption{Method: http.MethodPost, Body: []byte("ping")}
	assert.ErrorContains(t, Wait(t.Context(), nil, target, opts), "503")

	got := seen()
	assert.Equal(t, len(got), opts.Retries)
	for _, r := range got {
		assert.Equal(t, r.body, "ping")
	}
}

// A status outside the accepted ones fails the probe even when it is 2xx.
func TestWaitHTTPOptionRefusesOtherStatuses(t *testing.T) {
	url, _ := recordingServer(t, http.StatusOK)
	target, err := ParseTarget(url)
	assert.NilError(t, err)

	opts := fastWaitOption()
	opts.HTTP = HTTPOption{Method: http.MethodHead, Status: []StatusRange{{204, 204}}}
	assert.ErrorContains(t, Wait(t.Context(), nil, target, opts), "unexpected status 200 OK")
}

// A socket or file target refuses HTTP request options before probing.
func TestWaitHTTPOptionOnOtherTargets(t *testing.T) {
	opts := fastWaitOption()
	opts.HTTP = HTTPOption{Method: http.MethodHead}
	target := Target{Kind: KindFile, Addr: filepath.Join(t.TempDir(), "never")}

	assert.ErrorContains(t, Wait(t.Context(), nil, target, opts), "takes no HTTP request options")
}
