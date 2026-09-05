# M11.4 signed release evidence

Release publication now produces keyless Sigstore/Cosign evidence for the exact OCI digest emitted by the multi-platform build.

The release workflow installs Cosign from an immutable action commit, signs `ghcr.io/ploos-as/soju-web@sha256:<digest>` with the GitHub Actions OIDC identity, and immediately verifies that signature before creating the GitHub Release.

Verification is fail-closed and binds the certificate to:

- issuer: `https://token.actions.githubusercontent.com`
- workflow identity: `https://github.com/Ploos-AS/soju-web/.github/workflows/release.yml@refs/tags/v*`
- subject: the exact immutable release digest

The GitHub Release is downstream of exact-digest runtime qualification, provenance attestation, signing, and signature verification. A missing or invalid signature therefore prevents release completion.
