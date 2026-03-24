package commands

import (
	"bytes"
	"io"
	"testing"

	"gotest.tools/v3/assert"
)

func TestParseDetachKeys(t *testing.T) {
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
			got, err := parseDetachKeys(tt.input)
			if tt.wantErr {
				assert.Assert(t, err != nil, "expected error for %q", tt.input)
				return
			}
			assert.NilError(t, err)
			assert.DeepEqual(t, got, tt.expected)
		})
	}
}

func TestEscapeReader_DetectsSequence(t *testing.T) {
	// Input: some data, then ctrl-p ctrl-q.
	input := []byte("hello\x10\x11")
	r := &escapeReader{
		r:    bytes.NewReader(input),
		keys: []byte{0x10, 0x11},
	}

	buf := make([]byte, 1024)
	var output []byte
	var detached bool
	for {
		n, err := r.Read(buf)
		if n > 0 {
			output = append(output, buf[:n]...)
		}
		if err == errDetached {
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

func TestEscapeReader_PartialMatchFlush(t *testing.T) {
	// Input: ctrl-p followed by 'a' (not ctrl-q). Should flush ctrl-p and 'a'.
	input := []byte("\x10a")
	r := &escapeReader{
		r:    bytes.NewReader(input),
		keys: []byte{0x10, 0x11},
	}

	var output []byte
	buf := make([]byte, 1024)
	for {
		n, err := r.Read(buf)
		if n > 0 {
			output = append(output, buf[:n]...)
		}
		if err == errDetached {
			t.Fatal("should not detach")
		}
		if err != nil {
			break
		}
	}

	assert.Equal(t, string(output), "\x10a")
}

func TestEscapeReader_NoSequence(t *testing.T) {
	input := []byte("hello world")
	r := &escapeReader{
		r:    bytes.NewReader(input),
		keys: []byte{0x10, 0x11},
	}

	data, err := io.ReadAll(r)
	assert.NilError(t, err)
	assert.Equal(t, string(data), "hello world")
}

func TestEscapeReader_OnlySequence(t *testing.T) {
	input := []byte{0x10, 0x11}
	r := &escapeReader{
		r:    bytes.NewReader(input),
		keys: []byte{0x10, 0x11},
	}

	buf := make([]byte, 1024)
	n, err := r.Read(buf)
	assert.Equal(t, n, 0)
	assert.Equal(t, err, errDetached)
}
