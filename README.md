# debian-watch

A system monitoring dashboard for a Debian homeserver. Go, `html/template` and
HTMX, no build step and no JavaScript framework. Served at
`debian-watch.syamxm.com` through nginx and a Cloudflare Tunnel.

Panels: CPU total and per core, memory, swap, load average, uptime, per-mount
disk usage, per-interface network throughput, CPU/GPU/NVMe temperatures, and
Docker containers with per-container CPU, memory, state and health.

Every collector degrades on its own. A missing Docker socket proxy disables that
panel; unreadable sensors hide the temperature panel; nothing takes the app down.

## Requirements

- Go 1.26+ to build
- Linux for the metrics: readings come from `/proc`, `/sys` and `statfs`
- Optional: a read-only Docker socket proxy for the container panels

## Local development

```fish
cp .env.example .env
```

Generate a bcrypt hash and put it in `.env` **unquoted** — the loader is literal,
and quotes would become part of the hash:

```fish
htpasswd -bnBC 12 "" 'your-password' | tr -d ':\n'
```

Set `DW_COOKIE_SECURE=false` for local plain HTTP, then:

```fish
./scripts/dev.fish
```

The app listens on `http://127.0.0.1:8111`. Run the checks with:

```fish
gofmt -l .
go vet ./...
go test ./... -race -cover
```

## Configuration

Every setting is an environment variable. There are no defaults for the
credentials.

| Variable | Default | Purpose |
| --- | --- | --- |
| `DW_ADDR` | `:8111` | Listen address |
| `DW_ADMIN_USER` | — | **Required.** Admin username |
| `DW_ADMIN_PASS_HASH` | — | **Required.** bcrypt hash of the password |
| `DW_SESSION_TTL` | `12h` | Session lifetime; sessions are in memory and do not survive a restart |
| `DW_COOKIE_SECURE` | `true` | `Secure` flag on cookies. Only set false for local plain HTTP |
| `DW_TRUST_PROXY_HEADER` | `false` | Read the client IP from `X-Real-IP`. Only safe behind a proxy that sets it |
| `DW_LOG_LEVEL` | `info` | `debug`, `info`, `warn` or `error` |
| `DW_DOCKER_HOST` | empty | Socket proxy URL. Empty disables the container panels |
| `DW_SAMPLE_INTERVAL` | `2s` | Host sampling interval and history resolution |
| `DW_DOCKER_INTERVAL` | `10s` | Container polling interval |
| `DW_HISTORY_SIZE` | `120` | Samples kept per series for sparklines |
| `DW_HOST_ROOT` | empty | Where the host filesystem is mounted in a container |
| `DW_HOST_NET_DEV` | `/proc/net/dev` | Interface counter table to read |

`DW_TRUST_PROXY_HEADER=true` while the app is directly reachable would let
anyone spoof their IP past the login rate limiter. Keep it false unless nginx is
the only route in.

## Deployment

The compose file expects two external networks that already exist on the host:
`proxy-net` for nginx, and `homeserver-dashboard_monitoring` for `docker-gate`.

```fish
cp .env.example .env    # fill in DW_ADMIN_USER and DW_ADMIN_PASS_HASH
docker compose up -d --build
docker compose logs -f
```

Deployment topology lives in `compose.yml`, not `.env`. `.env` holds credentials
only.

### Docker access

debian-watch never mounts the Docker socket. It talks to `docker-gate`, the
nginx allowlist in the `homeserver-dashboard` project, which forwards exactly
`/containers/json` and `/containers/{id}/stats` to a `tecnativa/docker-socket-proxy`
running with `POST=0` on an `internal: true` network. Everything else is a 403.

If that project is torn down, its network disappears and debian-watch will not
start. Bring it back up, or point `DW_DOCKER_HOST` elsewhere and drop the
`monitoring` network from `compose.yml`.

### Host visibility

The container mounts the host read-only so readings describe the machine rather
than the container:

- `/:/host:ro,rslave` with `DW_HOST_ROOT=/host` — disk usage, `/proc`, `/sys`, hostname
- `/proc/1/net:/hostnet:ro` with `DW_HOST_NET_DEV=/hostnet/dev` — host network
  counters, because `/proc/net` is a symlink to `self/net` and always resolves
  in the reader's network namespace

This is the same trade-off `node_exporter` makes, and the container is
unprivileged: read-only root filesystem, all capabilities dropped,
`no-new-privileges`, and a non-root user, so root-owned secrets stay unreadable.
Drop both mounts if you would rather lose the disk and network panels.

### nginx

`deploy/nginx/debian-watch.conf` is the vhost. It is **not** applied
automatically. Copy it in and reload:

```fish
cp deploy/nginx/debian-watch.conf ~/nginx/conf.d/debian-watch.conf
docker exec nginx-proxy nginx -t
docker exec nginx-proxy nginx -s reload
```

It resolves the upstream per request, so a stopped container is a 502 on this
host alone rather than a config-load failure that takes every site down. The app
sets its own security headers, so the vhost deliberately adds none.

### Cloudflare Tunnel — manual

The tunnel is managed by hand in the Cloudflare dashboard and is not in this
repo. Add a public hostname:

- **Subdomain** `debian-watch`, **domain** `syamxm.com`
- **Service** `HTTP` → `nginx-proxy:80`

TLS terminates at Cloudflare; nginx listens on plain port 80 inside
`proxy-net`, which is why `DW_COOKIE_SECURE=true` is correct in production even
though the app itself speaks HTTP.

## Security

- Single admin account, bcrypt hash from the environment, never a plaintext password
- Server-side sessions in memory, `HttpOnly` + `Secure` + `SameSite=Strict`
- CSRF double-submit cookie on every state-changing request, rotated on login
- Login rate limited per IP in the app and again at nginx
- CSP without CDNs: htmx and the fonts are served from the binary
- Request timeouts, body size limit, panic recovery, graceful shutdown
- Errors are logged server-side; responses carry no internal detail

## Layout

```
cmd/debian-watch/     entrypoint, graceful shutdown, health probe
internal/auth/        credentials, sessions, CSRF, rate limiting
internal/collect/     host metric collectors
internal/docker/      socket proxy client and background refresher
internal/history/     ring buffer for sparklines
internal/monitor/     background sampler serving every request
internal/httpx/       routes, handlers, middleware, rendering
web/                  embedded templates and static assets
deploy/nginx/         vhost to apply by hand
```
