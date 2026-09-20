// Package poll waits until a target becomes ready: a unix socket accepts a
// connection, a file exists on disk, or an HTTP endpoint answers 2xx.
package poll

import (
	"errors"
	"fmt"
	"strings"
)

// Kind tells Wait how to probe a Target.
type Kind int

const (
	// KindUnix is a unix domain socket that accepts a connection.
	KindUnix Kind = iota
	// KindFile is a path that exists on disk.
	KindFile
	// KindHTTP is a URL that answers 2xx to a GET.
	KindHTTP
)

// Target is what Wait polls.
type Target struct {
	Kind Kind
	// Addr is a socket path (KindUnix), a file path (KindFile) or a full URL
	// (KindHTTP).
	Addr string
}

// Scheme prefixes that ParseTarget strips and describe puts back in front of
// a path.
const (
	unixPrefix = "unix://"
	filePrefix = "file://"
)

// describe renders the target the way ParseTarget accepts it back, for error
// messages and log records.
func (t Target) describe() string {
	switch t.Kind {
	case KindUnix:
		return unixPrefix + t.Addr
	case KindFile:
		return filePrefix + t.Addr
	case KindHTTP:
		return t.Addr
	}
	return fmt.Sprintf("kind(%d) %s", int(t.Kind), t.Addr)
}

// ParseTarget resolves a target string to a Target.
//
// A string carrying no "://" is a bare unix socket path. "unix://" and
// "file://" prefixes are stripped literally instead of being parsed as URLs:
// url.Parse reads "unix://rel/sock" as host "rel" plus path "/sock", which
// then looks the same as the absolute "unix:///sock". http and https targets
// keep the whole string, because the probe hands it straight back to net/http.
func ParseTarget(s string) (Target, error) {
	scheme, rest, ok := strings.Cut(s, "://")
	if !ok {
		if s == "" {
			return Target{}, errors.New("empty target")
		}
		return Target{Kind: KindUnix, Addr: s}, nil
	}

	switch scheme {
	case "unix":
		if rest == "" {
			return Target{}, fmt.Errorf("target %q has no socket path", s)
		}
		return Target{Kind: KindUnix, Addr: rest}, nil
	case "file":
		if rest == "" {
			return Target{}, fmt.Errorf("target %q has no file path", s)
		}
		return Target{Kind: KindFile, Addr: rest}, nil
	case "http", "https":
		if rest == "" {
			return Target{}, fmt.Errorf("target %q has no host", s)
		}
		return Target{Kind: KindHTTP, Addr: s}, nil
	}
	return Target{}, fmt.Errorf("target %q has unsupported scheme %q", s, scheme)
}
