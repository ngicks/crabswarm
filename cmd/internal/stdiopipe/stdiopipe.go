// Package stdiopipe provides a cancellable reader backed by os.Stdin.
package stdiopipe

import (
	"context"
	"io"
	"os"
	"sync"
)

var once sync.Once

// Stdin returns an [io.ReadCloser] which is pied to [os.Stdin] through an [io.Pipe].
//
// This is necessary because Read calls on [os.Stdin] cannot be unblocked by closing it.
//
// Only one invocation is allowed per process; a second call will panic.
func Stdin(ctx context.Context) io.ReadCloser {
	var pr *io.PipeReader
	called := false
	once.Do(func() {
		called = true
		var pw *io.PipeWriter
		pr, pw = io.Pipe()
		go func() {
			<-ctx.Done()
			pr.CloseWithError(ctx.Err())
		}()
		go func() {
			_, err := io.Copy(pw, os.Stdin)
			pw.CloseWithError(err)
		}()
	})
	if !called {
		panic("stdiopipe: Stdin called more than once")
	}
	return pr
}
