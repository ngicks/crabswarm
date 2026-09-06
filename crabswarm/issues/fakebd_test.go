package issues

import (
	"errors"
	"io/fs"
	"os"
	"path/filepath"
	"strings"
	"testing"

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
//   - list           -> list_children.json when --parent is present, else
//     list.json.
//   - show --id=<id> -> show.json for the one recorded issue, else
//     show_not_found.json with exit 1.
const fakeBdScript = `#!/bin/sh
sub="$1"
shift

parent=
for a in "$@"; do
  if [ "$a" = "--parent" ]; then parent=1; fi
done

{
  printf '%s\t' "$(pwd)"
  printf 'BD_JSON_ENVELOPE=%s\t' "$BD_JSON_ENVELOPE"
  printf 'FAKE_BD_EXTRA=%s\t' "$FAKE_BD_EXTRA"
  printf '%s' "$sub"
  for a in "$@"; do printf ' %s' "$a"; done
  printf '\n'
} >> "$FAKE_BD_LOG"

case "$sub" in
where)
  if [ -n "$FAKE_BD_NO_BEADS" ]; then
    cat "$FAKE_BD_TESTDATA/where_no_beads.json"
    exit 1
  fi
  cat "$FAKE_BD_TESTDATA/where.json"
  ;;
list)
  if [ -n "$parent" ]; then
    cat "$FAKE_BD_TESTDATA/list_children.json"
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
	// Inherited values must not leak into a run that does not set them.
	t.Setenv("FAKE_BD_NO_BEADS", "")
	t.Setenv("FAKE_BD_EXTRA", "")
	t.Setenv("BD_JSON_ENVELOPE", "")

	return func() []invocation {
		t.Helper()
		b, err := os.ReadFile(logPath)
		if errors.Is(err, fs.ErrNotExist) {
			return nil
		}
		assert.NilError(t, err)
		var out []invocation
		for line := range strings.SplitSeq(strings.TrimSuffix(string(b), "\n"), "\n") {
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
