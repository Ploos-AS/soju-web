# soju-web

`soju-web` is a small Go WebAdmin for [soju](https://soju.im/), the IRC bouncer maintained upstream at Codeberg.

This repository is a separate project from soju itself. It does not fork or bundle the soju source tree.

## Current scope

M0 established the hardened WebAdmin foundation. M1 adds real soju administration through soju's native Unix admin interface:

- authenticated WebAdmin login
- dashboard with soju backend reachability and TCP latency
- `/users` administration page
- user status via `user status`
- create users
- enable/disable users
- grant/revoke soju administrator status
- change user passwords
- CSRF protection for administrative POST actions
- health endpoint at `/healthz`
- pure-Go application with no runtime framework dependencies
- non-root, read-only Alpine container
- Docker Compose example
- amd64 and arm64 CI/release pipeline
- GHCR publication with SBOM, provenance and build attestation

IRC chat remains intentionally out of scope. `soju-web` also does not mount the Docker socket and does not modify the soju database directly.

## soju admin socket

M1 uses soju's native `unix+admin` listener, the same administrative interface used by `sojuctl`.

Add this listener to the soju configuration:

```text
listen unix+admin:///run/soju/admin
```

The soju and soju-web containers must share the runtime directory containing that socket. With the Ploos-AS images, both containers run as UID/GID 1000, so a shared host directory is a simple deployment model.

Example soju volume:

```yaml
volumes:
  - ./run/soju:/run/soju
```

The soju-web Compose example mounts the same directory read-only:

```yaml
volumes:
  - ./run/soju:/run/soju:ro
```

If the projects live in different directories, set `SOJU_RUNTIME_DIR` in the soju-web `.env` to the same absolute host directory used by soju.

## Quick start

```bash
cp .env.example .env
```

Set strong values for `SOJU_WEB_ADMIN_PASSWORD` and `SOJU_WEB_SESSION_SECRET`, configure the shared soju runtime directory, then start:

```bash
docker compose up -d
```

Open `http://HOST:8080/`.

The Compose example reaches an already-published soju IRC listener through `host.docker.internal:6667` using Docker's Linux `host-gateway`. If both containers share a Docker network, set `SOJU_ADDRESS` to the soju service name instead, for example `soju:6667`.

## Configuration

| Variable | Application default | Compose example | Purpose |
| --- | --- | --- | --- |
| `SOJU_WEB_LISTEN` | `:8080` | `:8080` | HTTP listen address |
| `SOJU_WEB_ADMIN_USER` | `admin` | `admin` | WebAdmin username |
| `SOJU_WEB_ADMIN_PASSWORD` | required | required | WebAdmin password, minimum 12 characters |
| `SOJU_WEB_SESSION_SECRET` | generated if omitted | required | HMAC key for sessions; use 32+ random characters in production |
| `SOJU_WEB_COOKIE_SECURE` | `false` | `false` | Set `true` when served over HTTPS |
| `SOJU_ADDRESS` | `soju:6667` | `host.docker.internal:6667` | soju TCP endpoint used for dashboard reachability |
| `SOJU_ADMIN_SOCKET` | `/run/soju/admin` | `/run/soju/admin` | Unix admin socket inside the container |
| `SOJU_RUNTIME_DIR` | Compose-only | `./run/soju` | Host directory mounted at `/run/soju` |

For production, terminate HTTPS at a reverse proxy and set `SOJU_WEB_COOKIE_SECURE=true`.

## Security model

The container runs as UID/GID 1000, drops all Linux capabilities, supports a read-only root filesystem and needs no Docker socket. The login cookie is HTTP-only, SameSite=Strict and HMAC-authenticated. Administrative forms carry HMAC-backed CSRF tokens. Security headers include CSP, frame denial, no-sniff and no-referrer policy.

The admin socket is powerful by design. Mount only the soju runtime directory needed for the socket, keep it read-only in soju-web, and do not expose the socket over TCP.

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
export SOJU_ADMIN_SOCKET='/run/soju/admin'
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
