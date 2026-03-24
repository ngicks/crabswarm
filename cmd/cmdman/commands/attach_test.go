package commands

import (
	"bytes"
	"errors"
	"io"
	"testing"

	"github.com/moby/term"
	"gotest.tools/v3/assert"
)

func TestDetachKeys_Parse(t *testing.T) {
	tests := []struct {
		input    string
		expected []byte
		wantErr  bool
	}{
		{"ctrl-p,ctrl-q", []byte{0x10, 0x11}, false},
		{"ctrl-a", []byte{0x01}, false},
		{"ctrl-z", []byte{0x1a}, false},
		{"ctrl-P,ctrl-Q", []byte{0x10, 0x11}, false},
		{"a", []byte{'a'}, false},
		{"a,b,c", []byte{'a', 'b', 'c'}, false},
		{"", nil, false},
		{"ctrl-", nil, true},
		{"ctrl-ab", nil, true},
		{"ctrl-1", nil, true},
	}

	for _, tt := range tests {
		t.Run(tt.input, func(t *testing.T) {
			got, err := term.ToBytes(tt.input)
			if tt.wantErr {
				assert.Assert(t, err != nil, "expected error for %q", tt.input)
				return
			}
			assert.NilError(t, err)
			assert.DeepEqual(t, got, tt.expected)
		})
	}
}

func TestDetachKeys_ProxyDetectsSequence(t *testing.T) {
	input := []byte("hello\x10\x11")
	r := term.NewEscapeProxy(bytes.NewReader(input), []byte{0x10, 0x11})

	buf := make([]byte, 1024)
	var output []byte
	var detached bool
	for {
		n, err := r.Read(buf)
		if n > 0 {
			output = append(output, buf[:n]...)
		}
		var escapeErr term.EscapeError
		if errors.As(err, &escapeErr) {
			detached = true
			break
		}
		if err != nil {
			break
		}
	}

	assert.Assert(t, detached, "expected detach")
	assert.Equal(t, string(output), "hello")
}

func TestDetachKeys_ProxyPartialMatchFlush(t *testing.T) {
	input := []byte("\x10a")
	r := term.NewEscapeProxy(bytes.NewReader(input), []byte{0x10, 0x11})

	var output []byte
	buf := make([]byte, 1024)
	for {
		n, err := r.Read(buf)
		if n > 0 {
			output = append(output, buf[:n]...)
		}
		var escapeErr term.EscapeError
		if errors.As(err, &escapeErr) {
			t.Fatal("should not detach")
		}
		if err != nil {
			break
		}
	}

	assert.Equal(t, string(output), "\x10a")
}

func TestDetachKeys_ProxyNoSequence(t *testing.T) {
	input := []byte("hello world")
	r := term.NewEscapeProxy(bytes.NewReader(input), []byte{0x10, 0x11})

	data, err := io.ReadAll(r)
	assert.NilError(t, err)
	assert.Equal(t, string(data), "hello world")
}

func TestDetachKeys_ProxyOnlySequence(t *testing.T) {
	input := []byte{0x10, 0x11}
	r := term.NewEscapeProxy(bytes.NewReader(input), []byte{0x10, 0x11})

	buf := make([]byte, 1024)
	n, err := r.Read(buf)
	var escapeErr term.EscapeError
	assert.Equal(t, n, 0)
	assert.Assert(t, errors.As(err, &escapeErr))
}
