#!/bin/sh
set -eu

image_name=${1:-ghcr.io/ploos-as/soju-web}
digest=${2:-}

case "$digest" in
  sha256:[0-9a-f][0-9a-f]*) ;;
  *) echo "invalid digest: $digest" >&2; exit 1 ;;
esac

if [ "${#digest}" -ne 71 ]; then
  echo "digest must be sha256 plus exactly 64 lowercase hex characters" >&2
  exit 1
fi

hex=${digest#sha256:}
case "$hex" in
  *[!0-9a-f]*) echo "digest contains non-hex characters" >&2; exit 1 ;;
esac

printf '%s@%s\n' "$image_name" "$digest"
