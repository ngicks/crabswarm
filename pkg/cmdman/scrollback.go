package cmdman

import "sync"

// RingBuffer is a thread-safe byte ring buffer for scrollback.
type RingBuffer struct {
	mu   sync.Mutex
	buf  []byte
	pos  int
	full bool
}

// NewRingBuffer creates a ring buffer with the given capacity in bytes.
func NewRingBuffer(capacity int) *RingBuffer {
	return &RingBuffer{buf: make([]byte, capacity)}
}

// Write appends data to the ring buffer, overwriting the oldest data if full.
func (r *RingBuffer) Write(p []byte) (int, error) {
	r.mu.Lock()
	defer r.mu.Unlock()

	n := len(p)
	cap := len(r.buf)

	if n >= cap {
		// Data is larger than buffer; keep only the last cap bytes.
		copy(r.buf, p[n-cap:])
		r.pos = 0
		r.full = true
		return n, nil
	}

	// Write data, wrapping around.
	first := cap - r.pos
	if first >= n {
		copy(r.buf[r.pos:], p)
	} else {
		copy(r.buf[r.pos:], p[:first])
		copy(r.buf, p[first:])
	}

	r.pos = (r.pos + n) % cap
	if !r.full && r.pos < n {
		// We've wrapped around at least once during this write.
		r.full = true
	}

	return n, nil
}

// Bytes returns the current contents of the ring buffer in order.
func (r *RingBuffer) Bytes() []byte {
	r.mu.Lock()
	defer r.mu.Unlock()

	if !r.full {
		out := make([]byte, r.pos)
		copy(out, r.buf[:r.pos])
		return out
	}

	out := make([]byte, len(r.buf))
	n := copy(out, r.buf[r.pos:])
	copy(out[n:], r.buf[:r.pos])
	return out
}
