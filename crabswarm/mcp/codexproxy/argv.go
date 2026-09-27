package codexproxy

import (
	"errors"
	"fmt"
	"slices"
	"strings"
)

// remoteFlag is the Codex option naming the app server a TUI attaches to.
const remoteFlag = "--remote"

// remoteArg is where the --remote address sits in a command line.
type remoteArg struct {
	// index is the position of the argument holding the address.
	index int
	// inline marks the --remote=ADDR spelling, where the argument holds the
	// flag as well.
	inline bool
	addr   string
}

// findRemote finds the one --remote address in argv. The scan stops at "--",
// past which Codex reads arguments as a prompt.
//
// More than one is refused rather than picking one: a TUI that read an
// address other than the rewritten one would reach the app server past the
// proxy, and nothing would say its threads went unstamped.
func findRemote(argv []string) (remoteArg, error) {
	if len(argv) == 0 {
		return remoteArg{}, errors.New("no command to run: give the Codex TUI command line")
	}
	var found []remoteArg
	for i := 1; i < len(argv); i++ {
		arg := argv[i]
		if arg == "--" {
			break
		}
		if arg == remoteFlag {
			if i+1 == len(argv) {
				return remoteArg{}, fmt.Errorf("%s has no address after it", remoteFlag)
			}
			found = append(found, remoteArg{index: i + 1, addr: argv[i+1]})
			i++
			continue
		}
		if addr, ok := strings.CutPrefix(arg, remoteFlag+"="); ok {
			found = append(found, remoteArg{index: i, inline: true, addr: addr})
		}
	}
	switch len(found) {
	case 0:
		return remoteArg{}, fmt.Errorf(
			"%q names no app server: give it %s unix://PATH", argv[0], remoteFlag)
	case 1:
		return found[0], nil
	}
	return remoteArg{}, fmt.Errorf("%s is given %d times; give it once", remoteFlag, len(found))
}

// unixSocketPath is the socket path a unix://PATH address names.
func unixSocketPath(addr string) (string, error) {
	path, ok := strings.CutPrefix(addr, "unix://")
	if !ok {
		return "", fmt.Errorf(
			"%s %q is no unix:// address; the proxy reaches the app server over a unix socket only",
			remoteFlag, addr)
	}
	// Codex resolves a bare unix:// to a default socket of its own, which the
	// proxy has no way to locate.
	if path == "" {
		return "", fmt.Errorf("%s unix:// names no socket path; give it as unix://PATH", remoteFlag)
	}
	return path, nil
}

// withRemote returns a copy of argv whose --remote address is addr.
func withRemote(argv []string, remote remoteArg, addr string) []string {
	out := slices.Clone(argv)
	if remote.inline {
		out[remote.index] = remoteFlag + "=" + addr
	} else {
		out[remote.index] = addr
	}
	return out
}
