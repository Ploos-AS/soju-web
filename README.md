# soju-web

`soju-web` is a small Go WebAdmin for [soju](https://soju.im/), the IRC bouncer maintained upstream at Codeberg.

This repository is a separate project from soju itself. It does not fork or bundle the soju source tree.

## Current scope

M0 established the hardened WebAdmin foundation. M1 added users, M2 networks, M3 channels/autojoin, M4 network authentication and server statistics, M5 operational observability, and M6 adds structured status views:

- authenticated WebAdmin login
- dashboard health for the IRC listener and Unix admin socket
- dashboard TCP latency
- authoritative `server status` statistics: active/stored users, downstreams, upstreams, networks and channels
- user overview from native `user status`
- disabled-user and administrator counts
- operational `Attention needed` warnings derived from soju's own status data
- embedded soju-web build version and Git revision in the dashboard and startup log
- structured Users table with role, enabled/disabled state, network count and network limit
- structured Networks table with address, connected/disabled/disconnected badges, active nick and upstream detail/error
- structured Channels table with joined/parted/disconnected and attached/detached state
- `/users` administration: status, create, enable/disable, admin role and password changes
- `/networks` administration: status, create, update, enable/disable and delete
- `/channels` administration: saved-channel/autojoin status, create, update and delete
- detached-channel policy: relay-detached, reattach-on, detach-on and detach-after
- `/security` administration scoped to a selected user and network
- SASL PLAIN status, credential setup and reset
- CertFP status and certificate generation with Ed25519, ECDSA or RSA-3072
- CSRF protection for administrative POST actions
- health endpoint at `/healthz`
- pure-Go application with no runtime framework dependencies
- non-root, read-only Alpine container
- Docker Compose example
- amd64 and arm64 CI/release pipeline
- GHCR publication with SBOM, provenance and build attestation

IRC chat remains intentionally out of scope. `soju-web` does not mount the Docker socket and does not modify the soju database directly.

## soju admin socket

M1-M6 use soju's native `unix+admin` listener, the same administrative interface used by `sojuctl`.

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

## Dashboard

The dashboard combines independent health and status signals:

- TCP reachability and latency to the configured soju IRC listener
- successful access to the native admin socket
- soju `server status`
- soju `user status`
- the soju-web build version and Git revision

The primary statistics are produced by soju itself:

```text
active/stored users
connected downstreams
connected upstreams
stored networks
stored channels
```

M5 also derives a small `Attention needed` section. Examples include an unreachable IRC listener/admin socket, stored users that are not currently active, configured networks without a live upstream connection, and disabled user accounts. The configured-network warning explicitly includes disabled networks; soju reports those as stored networks without an upstream connection.

## Structured status views

M6 parses the stable human-readable status produced by the pinned soju admin commands into WebAdmin view models. The original command output is not used as the primary presentation anymore.

Users show username, administrator role, enabled/disabled state, configured network count and optional network limit. Networks show configured address, connected/disabled/disconnected state, connected nick when soju reports one, channel count for connected networks, and the last connection error/detail for disconnected networks. Channels show joined/parted/disconnected state plus detached status.

The parsers are covered by focused tests using representative upstream status lines. If a future soju release changes the status format, the parser tests are expected to catch the compatibility change rather than silently inventing state.

## Network addresses

The WebAdmin accepts normal soju network addresses such as:

```text
irc.libera.chat
irc.libera.chat:6697
ircs://irc.libera.chat:6697
irc+insecure://localhost:6667
```

When a URL scheme is supplied, M2 accepts `ircs://` and `irc+insecure://`. Plain host or `host:port` values are passed to soju unchanged.

Network administration is always executed in the selected user's context using soju's `user run` command. The web application does not impersonate users through database changes.

## Channel administration

M3 targets a selected user and network. Status uses soju's `-network` selector. Create, update and delete use soju's native global-context target syntax, for example:

```text
#soju/Libera
```

The WebAdmin exposes soju's saved-channel settings for detached state, relay-detached, reattach-on, detach-on and detach-after. Supported filter values are `default`, `none`, `highlight` and `message`; durations follow Go/soju duration syntax such as `300s` or `22h30m`.

## SASL and CertFP

M4 exposes soju's native network authentication commands. SASL PLAIN credentials are sent over the local Unix admin socket directly to soju and are never stored by soju-web. Reset requires an explicit `reset` confirmation.

CertFP generation uses soju itself to generate and store the certificate/key pair for the selected network. The UI offers Ed25519 by default, ECDSA, and RSA-3072.

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

The admin socket is powerful by design. Mount only the soju runtime directory needed for the socket, keep it read-only in soju-web, and do not expose the socket over TCP. Destructive network/channel operations and credential resets require explicit confirmation where appropriate.

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

The release workflow publishes `linux/amd64` and `linux/arm64` images with SBOM, provenance and GitHub build attestation. The release build arguments are also embedded into the Go binary as the displayed version and revision.

## License

MIT. See `LICENSE`.
