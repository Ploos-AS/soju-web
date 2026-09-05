# M11.5 post-release audit

M11.5 adds an independent post-publication audit consumer for successful `Release` workflow runs.

The audit selects the published GitHub Release whose tag resolves to the producer workflow commit, re-validates the release tag policy, resolves the immutable OCI digest from the version tag, and requires both `linux/amd64` and `linux/arm64` manifests.

It then verifies the Cosign keyless signature against GitHub Actions OIDC and the exact `release.yml@refs/tags/v*` workflow identity. Finally, it pulls the immutable digest for amd64, verifies OCI revision/version labels, UID/GID 1000, and requires a successful `/healthz` response from that exact post-publication image.

The workflow is also manually dispatchable for an explicit existing release tag. It is fail-closed: missing release binding, malformed digest, missing architecture, bad signature identity, label mismatch, non-root mismatch, or runtime failure fails the audit.
