#!/bin/sh
set -eu

digest="sha256:$(printf '%064d' 0)"
expected="ghcr.io/ploos-as/soju-web@$digest"
actual=$(sh scripts/verify-exact-digest-runtime.sh ghcr.io/ploos-as/soju-web "$digest")
test "$actual" = "$expected"

for invalid in \
  latest \
  v0.1.0 \
  sha256:deadbeef \
  "sha256:$(printf '%064d' 0)X" \
  "sha256:AAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAA"
do
  if sh scripts/verify-exact-digest-runtime.sh ghcr.io/ploos-as/soju-web "$invalid" >/dev/null 2>&1; then
    echo "invalid digest unexpectedly accepted: $invalid" >&2
    exit 1
  fi
done
