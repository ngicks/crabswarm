#!/bin/sh
# Idempotent hk wiring for the one pre-commit fragment this skill ships.
#
# - The fragment is the only `reference/*.hk.pkl` beside this script; its
#   basename (`<name>.hk.pkl`) names the copy `.hk/<name>.pkl`.
# - Copies the fragment to `.hk/<name>.pkl` at the worktree root whenever
#   the two differ, so a skill update is picked up by running it again.
# - Pins the hk package version in that copy: the version `hk.pkl` or another
#   `.hk/*.pkl` already pins, else the installed hk. Every file of one
#   evaluation must import the same hk package, or Pkl sees two unrelated
#   `Config.Step` types. A pin whose major differs from the installed hk is
#   replaced, because the Pkl schema is not compatible across majors.
# - Creates `hk.pkl` when there is none. An existing `hk.pkl` is never
#   rewritten: Pkl is too free-form to edit safely with sed, so the script
#   prints the two lines to add and exits 1.
# - Validates the config and runs `hk install`.
#
# `hk.pkl` and `.hk/<name>.pkl` are meant to be committed: a worktree that
# does not have them runs no hooks.
set -eu

here=$(cd "$(dirname "$0")" && pwd -P)
set -- "$here"/../reference/*.hk.pkl
if [ $# -ne 1 ] || [ ! -f "$1" ]; then
  echo "hk-init: expected exactly one reference/*.hk.pkl in $here/.." >&2
  exit 2
fi
src=$1
name=$(basename "$src" .hk.pkl)
alias=$(printf '%s' "$name" | tr '-' '_')

if ! command -v hk >/dev/null 2>&1; then
  echo "hk-init: hk is not installed; install it (e.g. \`mise use -g hk\`) and rerun" >&2
  exit 1
fi

top=$(git rev-parse --show-toplevel)
cd "$top"

pkg_prefix='package://github.com/jdx/hk/releases/download/v'
# Prints the first hk version pinned by a `package://…/vX/hk@X#/…` URL.
pinned_version() {
  [ -f "$1" ] || return 0
  sed -n 's#.*"'"$pkg_prefix"'\([^/"]*\)/hk@[^"#]*\#/[A-Za-z]*\.pkl".*#\1#p' "$1" | head -n 1
}

installed=$(hk --version | sed 's/^hk[[:space:]]*//; s/[[:space:]].*//')
version=
for f in hk.pkl .hk/*.pkl; do
  [ -n "$version" ] || version=$(pinned_version "$f")
done
if [ -z "$version" ] || [ "${version%%.*}" != "${installed%%.*}" ]; then
  version=$installed
fi

changed=

# 1. The fragment copy, pinned to $version.
mkdir -p .hk
tmp=.hk/$name.pkl.tmp.$$
trap 'rm -f "$tmp"' EXIT
sed 's#'"$pkg_prefix"'[^/"]*/hk@[^"#]*\##'"$pkg_prefix$version"'/hk@'"$version"'\##g' "$src" > "$tmp"
if ! cmp -s "$tmp" ".hk/$name.pkl" 2>/dev/null; then
  mv "$tmp" ".hk/$name.pkl"
  changed="$changed .hk/$name.pkl"
fi

# 2. hk.pkl runs the fragment's steps on pre-commit.
if [ ! -f hk.pkl ]; then
  cat > hk.pkl <<EOF
amends "$pkg_prefix$version/hk@$version#/Config.pkl"

import ".hk/$name.pkl" as $alias

hooks {
  ["pre-commit"] {
    fix = true
    stash = "git"
    steps {
      ...$alias.steps
    }
  }
}
EOF
  changed="$changed hk.pkl"
elif ! grep -q "\.hk/$name\.pkl" hk.pkl; then
  cat >&2 <<EOF
hk-init: wire .hk/$name.pkl into hk.pkl by hand, then rerun this script:
hk-init:   after the amends line:   import ".hk/$name.pkl" as $alias
hk-init:   in the pre-commit steps: ...$alias.steps
hk-init: (add hooks { ["pre-commit"] { steps { ... } } } when hk.pkl has no pre-commit hook)
EOF
  exit 1
fi

hk validate >/dev/null

# 3. Git hooks. hk skips this on its own when hooks are installed globally.
hk install -q

if [ -n "$changed" ]; then
  echo "hk-init: wrote$changed; commit them so every worktree runs the hooks" >&2
fi
