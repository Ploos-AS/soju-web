#!/bin/sh
set -eu

tag=${1:-${GITHUB_REF_NAME:-}}
expected_sha=${2:-${GITHUB_SHA:-}}
mode=${3:-strict}

if [ -z "$tag" ] || [ -z "$expected_sha" ]; then
  echo "usage: verify-release-ref.sh <tag> <expected-commit-sha> [--allow-head-mismatch-for-audit]" >&2
  exit 2
fi

case "$mode" in
  strict|--allow-head-mismatch-for-audit) ;;
  *)
    echo "invalid verification mode: $mode" >&2
    exit 2
    ;;
esac

if ! printf '%s\n' "$tag" | grep -Eq '^v(0|[1-9][0-9]*)\.(0|[1-9][0-9]*)\.(0|[1-9][0-9]*)(-[0-9A-Za-z-]+(\.[0-9A-Za-z-]+)*)?(\+[0-9A-Za-z-]+(\.[0-9A-Za-z-]+)*)?$'; then
  echo "invalid release tag: $tag" >&2
  exit 1
fi

actual_sha=$(git rev-parse "${tag}^{commit}" 2>/dev/null) || {
  echo "cannot resolve release tag to a commit: $tag" >&2
  exit 1
}

if [ "$actual_sha" != "$expected_sha" ]; then
  echo "release tag/commit mismatch: tag=$tag actual=$actual_sha expected=$expected_sha" >&2
  exit 1
fi

if [ "$mode" = "strict" ]; then
  head_sha=$(git rev-parse HEAD)
  if [ "$head_sha" != "$expected_sha" ]; then
    echo "checkout/commit mismatch: HEAD=$head_sha expected=$expected_sha" >&2
    exit 1
  fi
fi

printf 'release ref verified: tag=%s commit=%s mode=%s\n' "$tag" "$expected_sha" "$mode"
