# w-popularity-backend

Go service for [w_popularity](https://github.com/suenot/w_popularity): one binary that
hosts both a JWT-protected REST API (gin) and a Postgres-backed cron scheduler that
fetches audience snapshots from ten social platforms.

## Architecture

- **HTTP API** (`cmd/api`) — gin, routes under `/api/v1`, JWT verified against
  `https://auth.marketmaker.cc/.well-known/jwks.json` (RS256, 1h cache).
- **Scheduler** (`cmd/scheduler`, also reachable via `MODE=both` on the main
  binary) — cron (`robfig/cron/v3`) enqueues fetch jobs daily, a worker pool
  drains them via `SELECT … FOR UPDATE SKIP LOCKED`.
- **Persistence** — Postgres via `pgx/v5`, schema embedded in
  `internal/db/migrations/`.
- **Parsers** — ten `github.com/suenot/w-popularity-parser-*` modules wired
  by `internal/parsers.Build`. YouTube has a real impl; others return
  `shared.ErrNotImplemented`.

## Run modes

`MODE` env (default `both`):

| Mode        | API | Cron | Workers |
|-------------|:---:|:----:|:-------:|
| `api`       | yes | no   | no      |
| `scheduler` | no  | yes  | yes     |
| `both`      | yes | yes  | yes     |

The `cmd/scheduler` binary is a convenience for `MODE=scheduler` deployments
that don't want the API code paths compiled in.

## Endpoints

All under `/api/v1`, all require `Authorization: Bearer <jwt>` except `GET /healthz`.

- `POST   /api/v1/channels` `{platform,handle,url}` → 201, enqueues immediate fetch
- `GET    /api/v1/channels` → list with latest stats from `v_channel_stats`
- `GET    /api/v1/channels/:id`
- `GET    /api/v1/channels/:id/snapshots?from=&to=`
- `GET    /api/v1/channels/:id/stats`
- `GET    /api/v1/channels/:id/posts`
- `DELETE /api/v1/channels/:id`
- `GET    /healthz`

## Configuration

| Env                       | Default                                                 |
|---------------------------|---------------------------------------------------------|
| `DATABASE_URL`            | (required)                                              |
| `BACKEND_PORT`            | `8080`                                                  |
| `AUTH_JWKS_URL`           | `https://auth.marketmaker.cc/.well-known/jwks.json`     |
| `AUTH_ISSUER`             | `auth.marketmaker.cc`                                   |
| `AUTH_SERVICE_NAME`       | `popularity`                                            |
| `NEXT_PUBLIC_FRONTEND_URL`| `*`                                                     |
| `FETCH_CRON`              | `0 3 * * *`                                             |
| `FETCH_WORKERS`           | `4`                                                     |
| `MODE`                    | `both`                                                  |
| `YOUTUBE_API_KEY`         | (optional; required for the YouTube parser)             |
| `*_CREDENTIAL`            | per-platform creds; missing means parser returns auth   |

## Build

```
go build ./...
go test ./...
docker build -t w-popularity-backend .
```

`TestClaimSkipLocked` is skipped unless `TEST_DATABASE_URL` is set.

## License

MIT — see [LICENSE](./LICENSE).
