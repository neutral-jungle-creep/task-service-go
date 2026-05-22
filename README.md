# Task service

REST service for task management (CRUD): create, get by id, list. PostgreSQL is the system of record; an in-memory cache with memory-based eviction sits in front of it so that recent tasks are served without hitting the database.

---

## Table of contents

- [Tech stack](#tech-stack)
- [Architecture](#architecture)
- [Configuration](#configuration)
- [API](#api)
- [Local run](#local-run)
- [Tests](#tests)
- [Linter](#linter)
- [Build](#build)
- [Deploy](#deploy)
- [Swagger](#swagger)
- [Database migrations](#database-migrations)
- [CI/CD](#cicd)
- [Conventions and limitations](#conventions-and-limitations)
- [Roadmap](#roadmap)

---

## Tech stack

| Layer | Technology |
|---|---|
| Language | Go 1.26.3 |
| HTTP | stdlib `net/http` + custom router [pkg/http/server](pkg/http/server) with `{param}` support |
| Logging | custom sync + async logger [pkg/logging](pkg/logging) |
| Cache | generic in-memory cache [pkg/cache](pkg/cache) with memory-based eviction |
| DB | PostgreSQL 17 + driver [`jackc/pgx/v5/stdlib`](https://github.com/jackc/pgx) |
| Migrations | [`pressly/goose/v3`](https://github.com/pressly/goose) |
| Configuration | [`kelseyhightower/envconfig`](https://github.com/kelseyhightower/envconfig) + [`joho/godotenv`](https://github.com/joho/godotenv) (.env for local) |
| API docs | [`swaggo/swag`](https://github.com/swaggo/swag) + [`swaggo/http-swagger/v2`](https://github.com/swaggo/http-swagger) |
| Tests | stdlib `testing` + [`stretchr/testify`](https://github.com/stretchr/testify) |
| Containers | Docker (multi-stage) + docker-compose |
| Task runner | [Task](https://taskfile.dev) (`Taskfile.yml`) |
| Linter | [`golangci-lint` v2](https://golangci-lint.run) (.golangci.yml) |
| Vulnerabilities | `govulncheck` |
| CI | GitHub Actions |

## Architecture

Layered architecture with a DI container (`internal/root`):

```
┌──────────────────────────────────────────────────────────────────┐
│                          cmd/main.go                             │
│           load config → create logger → root.New → Run            │
└──────────────────────────────┬───────────────────────────────────┘
                               │
┌──────────────────────────────▼───────────────────────────────────┐
│                   internal/root  (DI container)                  │
│   observability → repositories → services → http server          │
│            background jobs + stop handlers                       │
└──┬────────────────┬───────────────────┬──────────────────┬───────┘
   │                │                   │                  │
   ▼                ▼                   ▼                  ▼
┌──────────┐  ┌──────────────┐  ┌────────────────┐  ┌──────────────┐
│  logging │  │ repositories │  │    services    │  │ server (http)│
│ (async)  │  │ (Postgres)   │  │ TaskService    │  │ API + router │
│          │  │              │  │ TaskCache      │  │              │
└──────────┘  └──────┬───────┘  └────────┬───────┘  └──────┬───────┘
                     │                   │                 │
                     └────────┬──────────┴─────────────────┘
                              │
                  ┌───────────▼────────────┐
                  │   internal/ports        │   ← interfaces between layers
                  │   internal/domain       │   ← Task entity
                  └────────────────────────┘
```

Principles:

- **Ports & Adapters** — external dependencies (DB, cache, HTTP) sit behind interfaces in `internal/ports`.
- **One-way imports** — `domain` ← `ports` ← `services` ← `adapters`/`server` ← `root` ← `cmd`.
- **Background jobs + stop handlers** — every long-lived component (HTTP server, async logger, cache memory monitor, DB pool) is registered in the DI container, started in parallel in `Run`, and stopped in `stop`.

## Configuration

Configuration is read via `kelseyhightower/envconfig`. Locally a `.env` file is convenient (it is auto-loaded); in containers values come from `environment:` in the compose files. Template: [.env.example](.env.example).

| ENV | Default | Description |
|---|---|---|
| `SERVICE_NAME` | `task-service-go` | Service name in log fields |
| `RELEASE_ID` | — | Build identifier in log fields |
| `LOG_LEVEL` | `info` | `debug` / `info` / `warn` / `error` / `fatal` |
| `ROUTE_GROUP` | `/api/v1/task-service` | HTTP route prefix |
| `HTTP_SERVER_LISTEN_PORT` | `8888` | Listening port |
| `HTTP_SERVER_KEEP_ALIVE_TIME` | `60s` | |
| `HTTP_SERVER_KEEP_ALIVE_TIMEOUT` | `10s` | |
| `HTTP_SERVER_READ_HEADER_TIMEOUT` | `10s` | |
| `HTTP_SERVER_READ_TIMEOUT` | `10s` | |
| `HTTP_SERVER_WRITE_TIMEOUT` | `10s` | |
| `DB_POSTGRES_DSN` | **required** | Postgres connection string (`host=… port=… user=… password=… dbname=… sslmode=…`) |
| `DB_POSTGRES_MAX_OPEN_CONNS` | `10` | |
| `DB_POSTGRES_MAX_IDLE_CONNS` | `5` | |
| `DB_POSTGRES_MAX_LIFETIME` | `30m` | |
| `DB_POSTGRES_QUERY_TIMEOUT` | `5s` | |
| `CACHE_MEMORY_LIMIT_MB` | `1024` | Memory budget; cleanup starts at 90% |
| `CACHE_MEMORY_MONITOR_INTERVAL` | `5s` | How often the monitor checks memory pressure |

## API

All endpoints live under the `ROUTE_GROUP` prefix (default `/api/v1/task-service`).

### `POST /tasks` — create a task

```bash
curl -s -X POST localhost:8888/api/v1/task-service/tasks \
  -H 'Content-Type: application/json' \
  -d '{"name":"task 1","body":"write tests"}'
```

```json
{ "id": 1 }
```

### `GET /tasks` — list all tasks (no pagination yet)

```bash
curl -s localhost:8888/api/v1/task-service/tasks
```

```json
{
  "items": [
    {
      "id": 1,
      "name": "task 1",
      "body": "write tests",
      "status": "NEW",
      "createdAt": "2025-08-25T13:41:16.443471+03:00",
      "updatedAt": null
    }
  ],
  "total": 1
}
```

### `GET /tasks/{id}` — get a task by id

```bash
curl -s localhost:8888/api/v1/task-service/tasks/1
```

Errors are returned as `{ "errorMessage": "...", "status": 4xx, "timestamp": "..." }`.

Full specification is available in the Swagger UI — see [Swagger](#swagger).

## Local run

Requirements: Docker, Docker Compose, [Task](https://taskfile.dev/installation/) (`brew install go-task` / `go install github.com/go-task/task/v3/cmd/task@latest`).

```bash
# Bring up Postgres + migrations + service
task deploy:local

# Smoke test
curl localhost:8888/api/v1/task-service/tasks

# Logs
task deploy:local:logs

# Stop and remove volumes
task deploy:local:down
```

Without Docker:

```bash
cp .env.example .env  # edit DB_POSTGRES_DSN to point at your local Postgres
task db:up            # apply migrations
task build            # produce the binary at ./bin/server
./bin/server          # run
```

## Tests

```bash
# Unit (fast, no DB)
task tests
task tests:coverage   # + HTML report + threshold check

# Integration (boots the test stack, hits real Postgres; HTML report + threshold check)
task integration-tests
```

Coverage thresholds (override via ENV before `task default`):

- `MIN_UNIT_COVERAGE` — defaults to **50%**
- `MIN_INTEGRATION_COVERAGE` — defaults to **30%**

`/docs/`, `main.go`, `/dto/` are excluded from the total (see `COVERAGE_EXCLUDE` in [Taskfile.yml](Taskfile.yml)).

Integration tests are guarded by the `integration` build tag and live in a separate package. They expect Postgres at `127.0.0.1:5433` (the test compose file exposes it); override via `TEST_DB_POSTGRES_DSN`.

## Linter

```bash
task lint        # run
task lint:fix    # run with autofixes
```

Config: [.golangci.yml](.golangci.yml). Enabled: `errcheck`, `staticcheck`, `revive`, `gosec`, `gocritic`, `gocyclo`, `bodyclose`, `nilerr`, `errorlint`, `prealloc`, `testifylint`, `testpackage`, `tparallel`, `forbidigo` (forbids `fmt.Print*`, `errors.Wrap`), `depguard` (forbids `pkg/errors`) and more. Formatters: `gci`, `gofmt`, `gofumpt`, `goimports`.

## Build

```bash
# Local binary (./bin/server)
task build

# Docker image (target=app)
task build:docker
```

[build/server/Dockerfile](build/server/Dockerfile) is multi-stage:

- `builder` — `golang:1.26.3-alpine`, `go build`
- `goose-builder` — installs goose in its own layer
- `app` — `alpine:3.20` with the built binary, runs as user `app`
- `migrate` — `alpine:3.20` with goose and `/migrations`

## Deploy

Two compose scenarios.

### Local (`deployment/local/docker-compose.yml`)

Brings up Postgres + migrations + service, exposes `:8888`.

```bash
task deploy:local
task deploy:local:down
```

### Test (`deployment/test/docker-compose.yml`)

Brings up only Postgres + migrations with `:5433` exposed. Used by `task integration-tests`, which boots the stack, runs `tests/integration/`, then tears it down.

```bash
task deploy:test
task deploy:test:down
```

## Swagger

Swagger UI is served at `http://localhost:8888/swagger/index.html` once the service is up.

Regenerate docs from code annotations:

```bash
task swag:gen
```

This runs `swag init -g cmd/main.go -o docs --parseInternal --parseDependency`, regenerating `docs/docs.go`, `docs/swagger.json`, `docs/swagger.yaml`. The generated files are committed so CI and local builds do not depend on `swag` being installed on the machine.

Annotations live in:

- shared meta (title / version / host / basePath) — [cmd/main.go](cmd/main.go);
- endpoints — [internal/server/task.go](internal/server/task.go) above `ListTasks`, `GetTask`, `CreateTask`.

## Database migrations

`goose` manages migrations under [db/migrations](db/migrations).

```bash
# Apply pending migrations
task db:up

# Rollback the last one
task db:down

# Status
task db:status

# Create a new sql migration
task db:create -- create_indexes
```

`GOOSE_DBSTRING` is sourced from `.env` (see [.env.example](.env.example)).

In containers, migrations are applied by a dedicated `task-service-migrate` service that runs before `task-service` (via `depends_on: condition: service_completed_successfully`).

## CI/CD

[.github/workflows/ci.yml](.github/workflows/ci.yml) triggers on `push` to `main`/`master` and `pull_request`. Jobs:

| Job | What it does |
|---|---|
| `fmt` | `task fmt:check` — fails if `gofmt`/`goimports` finds unformatted files |
| `lint` | `task lint` |
| `unit-tests` | `task tests`, coverage threshold check, coverage artifact upload |
| `integration-tests` | brings up Postgres as a `service`, applies migrations via goose, runs the `integration`-tagged tests |
| `build` | `docker buildx build` of both Dockerfile targets (`app` + `migrate`) |
| `security` | `govulncheck` |

## Conventions and limitations

- **Cache eviction.** Memory-driven — cleanup triggers at 90% of the configured limit and drops 1/5 of the oldest entries (sorted by key).
- **Unpaginated list.** `GET /tasks` returns everything (cache + tail from DB). Fine for a PoC, not for production — pagination is in [ROADMAP.md](ROADMAP.md).

## Roadmap

See [ROADMAP.md](ROADMAP.md) for the next-steps backlog (pagination, validation, metrics, tracing, auth, PATCH/DELETE, rate limits, expanded coverage, `pkg/background`, Russian README).
