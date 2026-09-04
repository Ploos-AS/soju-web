# soju-web

`soju-web` is a small Go WebAdmin for [soju](https://soju.im/), the IRC bouncer maintained upstream at Codeberg.

This repository is a separate project from soju itself. It does not fork or bundle the soju source tree.

## Current scope

M0 established the hardened WebAdmin foundation. M1 added users, M2 networks, M3 channels/autojoin, M4 network authentication and server statistics, M5 operational observability, M6 structured status views, M7 contextual drill-down navigation, M8 inline management, M9 native user deletion, and M10 adds release hardening and real soju integration qualification.

- authenticated WebAdmin login
- dashboard health for the IRC listener and Unix admin socket
- dashboard TCP latency
- authoritative `server status` statistics: active/stored users, downstreams, upstreams, networks and channels
- user overview from native `user status`
- disabled-user and administrator counts
- operational `Attention needed` warnings derived from soju's own status data
- embedded soju-web build version and Git revision in the dashboard and startup log
- structured Users, Networks and Channels tables
- contextual drill-down from Users to Networks, and from Networks to Channels or Security
- inline Manage actions that prefill the selected user/network/channel target
- `/users` administration: status, create, enable/disable, admin role, password changes and native token-confirmed deletion
- `/networks` administration: status, create, update, enable/disable and delete
- `/channels` administration: saved-channel/autojoin status, create, update and delete
- detached-channel policy: relay-detached, reattach-on, detach-on and detach-after
- `/security` administration scoped to a selected user and network
- SASL PLAIN status, credential setup and reset
- CertFP status and certificate generation with Ed25519, ECDSA or RSA-3072
- explicit confirmation before credential reset, CertFP generation and destructive deletes
- CSRF protection for administrative POST actions
- health endpoint at `/healthz`
- pure-Go application with no runtime framework dependencies
- non-root, read-only Alpine container
- Docker Compose example using a shared named runtime volume
- amd64 and arm64 CI/release pipeline
- end-to-end CI against a real Ploos-AS soju build and Unix admin socket
- GHCR publication with SBOM, provenance and build attestation

IRC chat remains intentionally out of scope. `soju-web` does not mount the Docker socket and does not modify the soju database directly.

## soju admin socket

M1-M10 use soju's native `unix+admin` listener, the same administrative interface used by `sojuctl`.

The current `Ploos-AS/soju` packaging enables:

```text
listen unix+admin:///run/soju/admin
```

The reference Compose files in both repositories use the same explicitly named Docker volume:

```text
soju-runtime
```

soju mounts it read-write at `/run/soju`; soju-web mounts the same volume read-only. This works even when the two Compose projects live in different directories.

Override the shared volume name in both projects when needed:

```text
SOJU_RUNTIME_VOLUME=<name>
```

The admin socket is never exposed as a TCP port.

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

M5 derives a small `Attention needed` section. Examples include an unreachable IRC listener/admin socket, stored users that are not currently active, configured networks without a live upstream connection, and disabled user accounts. The configured-network warning explicitly includes disabled networks; soju reports those as stored networks without an upstream connection.

## Structured status and drill-down

M6 parses the stable human-readable status produced by the pinned soju admin commands into WebAdmin view models. Users show username, administrator role, enabled/disabled state, configured network count and optional network limit. Networks show configured address, connected/disabled/disconnected state, connected nick and upstream detail/error. Channels show joined/parted/disconnected state plus detached status.

M7-M8 make these tables the primary management path:

```text
Users → Networks → Channels
                 ↘ Security
```

Manage links retain user/network/channel context with Go's `net/url` escaping and prefill the corresponding edit forms.

## User deletion

M9 follows soju's native two-step confirmation protocol instead of bypassing it.

The WebAdmin first asks soju for the deletion token using `user delete <username>`. That call does not delete the account. The final POST requires:

- an authenticated WebAdmin session
- a valid CSRF token
- the token returned by soju for that username
- the exact typed phrase `delete <username>`

Only then does soju-web send `user delete <username> <token>` to the admin socket.

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

CertFP generation uses soju itself to generate and store the certificate/key pair for the selected network. The UI offers Ed25519 by default, ECDSA, and RSA-3072. Because generating a certificate changes the selected network to SASL EXTERNAL and replaces its existing SASL credential material, M10 additionally requires the operator to type `generate` before this action is submitted.

## Quick start

Start the current `Ploos-AS/soju` Compose deployment first so the shared runtime volume and admin socket exist. Then in this repository:

```bash
cp .env.example .env
```

Set strong values for `SOJU_WEB_ADMIN_PASSWORD` and `SOJU_WEB_SESSION_SECRET`, then start:

```bash
docker compose up -d
```

Open `http://HOST:8080/`.

The default Compose example reaches the host-published soju IRC listener through `host.docker.internal:6667` using Docker's Linux `host-gateway`, while administrative traffic stays on the shared `soju-runtime` Unix socket. If both containers share a Docker network, set `SOJU_ADDRESS` to the soju service name instead, for example `soju:6667`.

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
| `SOJU_RUNTIME_VOLUME` | Compose-only | `soju-runtime` | Named Docker volume shared read-only with soju-web |

For production, terminate HTTPS at a reverse proxy and set `SOJU_WEB_COOKIE_SECURE=true`.

## Security model

The container runs as UID/GID 1000, drops all Linux capabilities, supports a read-only root filesystem and needs no Docker socket. The login cookie is HTTP-only, SameSite=Strict and HMAC-authenticated. Administrative forms carry HMAC-backed CSRF tokens. Security headers include CSP, frame denial, no-sniff and no-referrer policy.

The admin socket is powerful by design. Mount only the `soju-runtime` volume needed for the socket, keep it read-only in soju-web, and do not expose the socket over TCP. Destructive network/channel/user operations and credential changes require explicit confirmation where appropriate.

## M10 qualification

The normal CI continues to require:

```text
gofmt
go test ./...
go vet ./...
Compose validation
amd64 image build/runtime
arm64 image build/runtime
non-root UID/GID
health endpoint
```

M10 also adds a separate Integration workflow. It checks out `Ploos-AS/soju`, builds both images from source, creates a real soju administrator, starts soju with a shared Unix-socket volume, starts soju-web with that volume read-only, signs into the WebAdmin, and verifies that the dashboard can read real admin-socket state including the created soju user.

A `v0.1.0` tag should only be created after both CI and Integration are green on the same main HEAD and the companion `Ploos-AS/soju` admin-socket CI is green.

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
