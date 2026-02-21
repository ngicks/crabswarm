#!/usr/bin/env bash

set -euo pipefail

oses=(
  "linux"
  "darwin"
)

archs=(
  "amd64"
  "arm64"
)

for os in "${oses[@]}" ; do
  for arch in "${archs[@]}" ; do
    CGO_ENABLED=0 GOOS=${os} GOARCH=${arch} go build -trimpath -o ./plugin/crabswarm/bin/crabswarm-${os}-${arch} ./cmd/crabswarm
  done
done
