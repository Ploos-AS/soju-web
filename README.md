# soju-web

`soju-web` is a small Go WebAdmin for [soju](https://soju.im/), the IRC bouncer maintained upstream at Codeberg.

This repository is a separate project from soju itself. It does not fork or bundle the soju source tree.

## M0 foundation

The first milestone intentionally keeps the control surface small:

- authenticated WebAdmin login
- dashboard with soju backend reachability and TCP latency
- health endpoint at `/healthz`
- pure-Go application with no runtime framework dependencies
- non-root, read-only Alpine container
- Docker Compose example
- amd64 and arm64 CI/release pipeline
- GHCR publication with SBOM, provenance and build attestation

IRC chat is explicitly out of scope for v0.1. User and configuration management will be added through a dedicated soju integration layer rather than by mounting the Docker socket into the web application.

## Quick start

```bash
cp .env.example .env
```

Set strong values for `SOJU_WEB_ADMIN_PASSWORD` and `SOJU_WEB_SESSION_SECRET`, then start:

```bash
docker compose up -d
```

Open `http://HOST:8080/`.

The Compose example reaches an already-published soju listener through `host.docker.internal:6667` using Docker's Linux `host-gateway`. If both containers share a Docker network, set `SOJU_ADDRESS` to the soju service name instead, for example `soju:6667`.

## Configuration

| Variable | Application default | Compose example | Purpose |
| --- | --- | --- | --- |
| `SOJU_WEB_LISTEN` | `:8080` | `:8080` | HTTP listen address |
| `SOJU_WEB_ADMIN_USER` | `admin` | `admin` | WebAdmin username |
| `SOJU_WEB_ADMIN_PASSWORD` | required | required | WebAdmin password, minimum 12 characters |
| `SOJU_WEB_SESSION_SECRET` | generated if omitted | required | HMAC key for sessions; use 32+ random characters in production |
| `SOJU_WEB_COOKIE_SECURE` | `false` | `false` | Set `true` when served over HTTPS |
| `SOJU_ADDRESS` | `soju:6667` | `host.docker.internal:6667` | soju TCP endpoint used for backend reachability |

For production, terminate HTTPS at a reverse proxy and set `SOJU_WEB_COOKIE_SECURE=true`.

## Security model

The container runs as UID/GID 1000, drops all Linux capabilities, supports a read-only root filesystem and needs no Docker socket. The login cookie is HTTP-only, SameSite=Strict and HMAC-authenticated. Security headers include CSP, frame denial, no-sniff and no-referrer policy.

M0 does not yet authenticate to or modify the soju database. Backend status is a TCP reachability check only.

## Development

Requires Go 1.24 or newer.

```bash
go test ./...
go vet ./...
go run .
```

Example development environment:

```bash
export SOJU_WEB_ADMIN_PASSWORD='a-long-development-password'
export SOJU_WEB_SESSION_SECRET='01234567890123456789012345678901'
export SOJU_ADDRESS='127.0.0.1:6667'
go run .
```

## Container image

Release images are published as:

```text
ghcr.io/ploos-as/soju-web
```

The release workflow publishes `linux/amd64` and `linux/arm64` images with SBOM, provenance and GitHub build attestation.

## License

MIT. See `LICENSE`.
