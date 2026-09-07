package issues

import (
	"errors"
	"io/fs"
	"os"
	"path/filepath"
	"strings"
	"testing"
	"time"

	"gotest.tools/v3/assert"
)

// fakeBdScript is a stand-in for the real bd binary. Every invocation is
// appended to the file named by FAKE_BD_LOG as one tab-separated line —
// working directory, the two environment values the tests care about, then
// the space-joined argv — so a test can assert on the command line that was
// built. Output is replayed from the recorded fixtures in FAKE_BD_TESTDATA:
//
//   - where          -> where.json, or where_no_beads.json with exit 1 when
//     FAKE_BD_NO_BEADS is set, matching bd's behavior of printing the error
//     envelope on stdout.
//   - list           -> the next fixture of the colon-separated
//     FAKE_BD_LIST_SEQUENCE, counted in FAKE_BD_LIST_COUNTER, so successive
//     polls read a changing backlog; else list.json. The last fixture of the
//     sequence replays for every further call, so a poll past the end of the
//     sequence reports no change.
//   - show --id=<id> -> show.json for the one recorded issue, else
//     show_not_found.json with exit 1.
//
// FAKE_BD_EXCLUSIVE names a lock directory that turns the fake into a stand-in
// for the real bd's one-process-per-database rule: the invocation creates the
// directory, fails when it already exists, sleeps long enough for an
// overlapping invocation to reach that check, and removes it on the way out. A
// test that sets it therefore fails on an overlap instead of assuming there was
// none. The invocation is recorded before the check, so a run that should never
// have started still shows up in the log. A killed invocation leaves the
// directory behind and every later one then fails, which is also what a
// database lock does when the process holding it dies.
const fakeBdScript = `#!/bin/sh
sub="$1"
shift

{
  printf '%s\t' "$(pwd)"
  printf 'BD_JSON_ENVELOPE=%s\t' "$BD_JSON_ENVELOPE"
  printf 'FAKE_BD_EXTRA=%s\t' "$FAKE_BD_EXTRA"
  printf '%s' "$sub"
  for a in "$@"; do printf ' %s' "$a"; done
  printf '\n'
} >> "$FAKE_BD_LOG"

if [ -n "$FAKE_BD_EXCLUSIVE" ]; then
  if ! mkdir "$FAKE_BD_EXCLUSIVE" 2>/dev/null; then
    echo "fake bd: another bd is already running on this database" >&2
    exit 3
  fi
  trap 'rmdir "$FAKE_BD_EXCLUSIVE"' EXIT
  sleep 0.3
fi

case "$sub" in
where)
  if [ -n "$FAKE_BD_NO_BEADS" ]; then
    cat "$FAKE_BD_TESTDATA/where_no_beads.json"
    exit 1
  fi
  cat "$FAKE_BD_TESTDATA/where.json"
  ;;
list)
  if [ -n "$FAKE_BD_LIST_SEQUENCE" ]; then
    n=0
    if [ -f "$FAKE_BD_LIST_COUNTER" ]; then n=$(cat "$FAKE_BD_LIST_COUNTER"); fi
    echo "$((n + 1))" > "$FAKE_BD_LIST_COUNTER"
    IFS=:
    set -- $FAKE_BD_LIST_SEQUENCE
    unset IFS
    if [ "$n" -ge "$#" ]; then n=$(($# - 1)); fi
    shift "$n"
    cat "$FAKE_BD_TESTDATA/$1"
  else
    cat "$FAKE_BD_TESTDATA/list.json"
  fi
  ;;
show)
  id=${1#--id=}
  if [ "$id" = "scratch-uoj" ]; then
    cat "$FAKE_BD_TESTDATA/show.json"
  else
    cat "$FAKE_BD_TESTDATA/show_not_found.json"
    echo "Error fetching $id: no issue found matching \"$id\"" >&2
    exit 1
  fi
  ;;
*)
  echo "fake bd: unknown subcommand $sub" >&2
  exit 2
  ;;
esac
exit 0
`

// invocation is one recorded fake bd call.
type invocation struct {
	dir      string
	envelope string // the BD_JSON_ENVELOPE the subprocess saw
	extra    string // the FAKE_BD_EXTRA the subprocess saw
	args     string // subcommand and flags, space-joined
}

// requireExclusiveFakeBd makes every following fake bd refuse to run beside
// another one, the way the real bd refuses a database a second process holds.
// It must be called after [installFakeBd], whose leak guard clears the
// variable.
func requireExclusiveFakeBd(t *testing.T) {
	t.Helper()
	// The fake creates the directory itself, so name one under a directory
	// that exists and nothing else uses.
	t.Setenv("FAKE_BD_EXCLUSIVE", filepath.Join(t.TempDir(), "database.lock"))
}

// waitInvocations blocks until n invocations have been recorded. It is how a
// test lands a second caller inside the window of a bd that is already
// running: the fake records itself before it sleeps.
func waitInvocations(t *testing.T, invocations func() []invocation, n int) {
	t.Helper()
	deadline := time.Now().Add(10 * time.Second)
	for {
		if got := len(invocations()); got >= n {
			return
		}
		if time.Now().After(deadline) {
			t.Fatalf("waited for %d invocations, saw %d", n, len(invocations()))
		}
		time.Sleep(time.Millisecond)
	}
}

// assertInvocationsStay fails unless exactly n invocations are recorded for
// the whole of d. It is how a test asserts a bd never started rather than
// merely not having started yet: the fake records itself before it does
// anything else, so d only has to cover spawning the process.
func assertInvocationsStay(
	t *testing.T,
	invocations func() []invocation,
	n int,
	d time.Duration,
) {
	t.Helper()
	deadline := time.Now().Add(d)
	for {
		if got := len(invocations()); got != n {
			t.Fatalf("saw %d invocations, want %d", got, n)
		}
		if time.Now().After(deadline) {
			return
		}
		time.Sleep(5 * time.Millisecond)
	}
}

// installFakeBd writes the fake bd onto a fresh dir prepended to PATH and
// returns a function reading back the invocations recorded so far. It uses
// t.Setenv, so the test cannot be parallel.
func installFakeBd(t *testing.T) (invocations func() []invocation) {
	t.Helper()
	dir := t.TempDir()
	bin := filepath.Join(dir, "bd")
	assert.NilError(t, os.WriteFile(bin, []byte(fakeBdScript), 0o755))

	testdata, err := filepath.Abs("testdata")
	assert.NilError(t, err)

	logPath := filepath.Join(dir, "invocations.log")
	t.Setenv("PATH", dir+string(os.PathListSeparator)+os.Getenv("PATH"))
	t.Setenv("FAKE_BD_LOG", logPath)
	t.Setenv("FAKE_BD_TESTDATA", testdata)
	t.Setenv("FAKE_BD_LIST_COUNTER", filepath.Join(dir, "list.counter"))
	// Inherited values must not leak into a run that does not set them.
	t.Setenv("FAKE_BD_NO_BEADS", "")
	t.Setenv("FAKE_BD_LIST_SEQUENCE", "")
	t.Setenv("FAKE_BD_EXTRA", "")
	t.Setenv("FAKE_BD_EXCLUSIVE", "")
	t.Setenv("BD_JSON_ENVELOPE", "")

	return func() []invocation {
		t.Helper()
		b, err := os.ReadFile(logPath)
		if errors.Is(err, fs.ErrNotExist) {
			return nil
		}
		assert.NilError(t, err)
		// A test reads the log while a fake bd may still be writing to it,
		// and the fake writes its line in several pieces. Everything up to
		// the last newline is a whole line; a tail without one is half of
		// the line the next read reports.
		recorded := string(b)
		if i := strings.LastIndexByte(recorded, '\n'); i >= 0 {
			recorded = recorded[:i]
		} else {
			recorded = ""
		}
		var out []invocation
		for line := range strings.SplitSeq(recorded, "\n") {
			if line == "" {
				continue
			}
			f := strings.Split(line, "\t")
			assert.Equal(t, len(f), 4, "malformed invocation line %q", line)
			out = append(out, invocation{
				dir:      f[0],
				envelope: strings.TrimPrefix(f[1], "BD_JSON_ENVELOPE="),
				extra:    strings.TrimPrefix(f[2], "FAKE_BD_EXTRA="),
				args:     f[3],
			})
		}
		return out
	}
}
