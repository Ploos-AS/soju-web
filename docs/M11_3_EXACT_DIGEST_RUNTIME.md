# M11.3 exact-digest runtime qualification

Release qualification is bound to the immutable OCI digest emitted by the successful multi-platform build/push step.

The release workflow constructs only `ghcr.io/ploos-as/soju-web@sha256:<64 lowercase hex>` for qualification. It pulls that exact reference, verifies Docker resolves the same RepoDigest, checks the OCI revision and version labels against the release commit/tag, verifies UID/GID 1000, and starts the exact-digest image to require a successful `/healthz` response.

Mutable release tags are not used as the runtime qualification target. Attestation and GitHub Release creation remain downstream of this runtime gate, so a failed exact-digest qualification prevents release completion.

CI separately exercises the digest-reference validator with accepted and rejected inputs before any release tag is created.
