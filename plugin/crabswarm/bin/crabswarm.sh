#!/usr/bin/env bash

dir=$(cd $(dirname $0) && pwd -P)

os=$(uname -s)

# well known, linux, darwin(Mac OS).
# I'm not sure I will use other than those.
case ${os} in
  "Linux")
    os="linux";;
  "Darwin")
    os="darwin";;
  *)
    echo "unsupported os: ${os}: must be Linux or Darwin"
    return 1
    ;;
esac

arch=$(uname -m)
case ${arch} in
  "x86_64")
    arch="amd64";;
  "x86_64-AT386")
    arch="amd64";;
  "aarch64_be")
    arch="arm64";;
  "aarch64")
    arch="arm64";;
  "armv8b")
    arch="arm64";;
  "armv8l")
    arch="arm64";;
  *)
    echo "unsupported arch: ${arch}: must be one of x86_64, x86_64-AT386, aarch64_be, aarch64, armv8b or armv8l"
    return 1
    ;;
esac

exec $dir/crabswarm-${os}-${arch} "$@"
