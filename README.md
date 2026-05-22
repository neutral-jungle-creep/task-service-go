# Task service

REST service for task management (CRUD): create, get by id, list. PostgreSQL is the system of record; an in-memory cache with memory-based eviction sits in front of it so that recent tasks are served without hitting the database.

> [Русская версия — внизу](#русская-версия)

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

`/docs/`, `main.go`, `/dto/`, `/internal/root/` are excluded from the total (see `COVERAGE_EXCLUDE` in [Taskfile.yml](Taskfile.yml)).

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

See [ROADMAP.md](ROADMAP.md) for the next-steps backlog (pagination, validation, metrics, tracing, auth, PATCH/DELETE, rate limits, expanded coverage, `pkg/background`).

---

# Русская версия

REST-сервис для управления задачами (CRUD): создание, получение по id, список. PostgreSQL — источник правды; перед ним стоит in-memory кеш с эвикцией по памяти, чтобы свежие задачи отдавались без похода в БД.

---

## Содержание

- [Технологический стек](#технологический-стек)
- [Архитектура](#архитектура)
- [Конфигурация](#конфигурация)
- [API](#api-1)
- [Локальный запуск](#локальный-запуск)
- [Тесты](#тесты)
- [Линтер](#линтер)
- [Сборка](#сборка)
- [Деплой](#деплой)
- [Swagger](#swagger-1)
- [Миграции БД](#миграции-бд)
- [CI/CD](#cicd-1)
- [Соглашения и ограничения](#соглашения-и-ограничения)
- [Roadmap](#roadmap-1)

---

## Технологический стек

| Слой | Технология |
|---|---|
| Язык | Go 1.26.3 |
| HTTP | stdlib `net/http` + собственный роутер [pkg/http/server](pkg/http/server) с поддержкой `{param}` |
| Логирование | собственный синхронный + асинхронный логгер [pkg/logging](pkg/logging) |
| Кеш | generic in-memory кеш [pkg/cache](pkg/cache) с эвикцией по памяти |
| БД | PostgreSQL 17 + драйвер [`jackc/pgx/v5/stdlib`](https://github.com/jackc/pgx) |
| Миграции | [`pressly/goose/v3`](https://github.com/pressly/goose) |
| Конфигурация | [`kelseyhightower/envconfig`](https://github.com/kelseyhightower/envconfig) + [`joho/godotenv`](https://github.com/joho/godotenv) (`.env` для локалки) |
| Документация API | [`swaggo/swag`](https://github.com/swaggo/swag) + [`swaggo/http-swagger/v2`](https://github.com/swaggo/http-swagger) |
| Тесты | stdlib `testing` + [`stretchr/testify`](https://github.com/stretchr/testify) |
| Контейнеризация | Docker (multi-stage) + docker-compose |
| Оркестратор задач | [Task](https://taskfile.dev) (`Taskfile.yml`) |
| Линтер | [`golangci-lint` v2](https://golangci-lint.run) (`.golangci.yml`) |
| Уязвимости | `govulncheck` |
| CI | GitHub Actions |

## Архитектура

Слоистая архитектура с DI-контейнером (`internal/root`):

```
┌──────────────────────────────────────────────────────────────────┐
│                          cmd/main.go                             │
│           загрузка конфига → логгер → root.New → Run              │
└──────────────────────────────┬───────────────────────────────────┘
                               │
┌──────────────────────────────▼───────────────────────────────────┐
│                   internal/root  (DI-контейнер)                  │
│   observability → repositories → services → http server          │
│            фоновые задачи + stop-обработчики                     │
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
                  │   internal/ports        │   ← интерфейсы между слоями
                  │   internal/domain       │   ← Task
                  └────────────────────────┘
```

Принципы:

- **Ports & Adapters** — внешние зависимости (БД, кеш, HTTP) скрыты за интерфейсами в `internal/ports`.
- **Однонаправленные импорты** — `domain` ← `ports` ← `services` ← `adapters`/`server` ← `root` ← `cmd`.
- **Background jobs + stop handlers** — каждый долгоживущий компонент (HTTP-сервер, асинхронный логгер, memory monitor кеша, пул БД) регистрируется в DI-контейнере, параллельно стартует в `Run` и останавливается в `stop`.

## Конфигурация

Конфигурация читается через `kelseyhightower/envconfig`. Локально удобно держать значения в `.env` (он автоматически подгружается); в контейнерах значения берутся из `environment:` в compose-файлах. Шаблон значений — [.env.example](.env.example).

| ENV | По умолчанию | Описание |
|---|---|---|
| `SERVICE_NAME` | `task-service-go` | Имя сервиса в полях логов |
| `RELEASE_ID` | — | Идентификатор сборки в полях логов |
| `LOG_LEVEL` | `info` | `debug` / `info` / `warn` / `error` / `fatal` |
| `ROUTE_GROUP` | `/api/v1/task-service` | Префикс HTTP-роутов |
| `HTTP_SERVER_LISTEN_PORT` | `8888` | Порт прослушивания |
| `HTTP_SERVER_KEEP_ALIVE_TIME` | `60s` | |
| `HTTP_SERVER_KEEP_ALIVE_TIMEOUT` | `10s` | |
| `HTTP_SERVER_READ_HEADER_TIMEOUT` | `10s` | |
| `HTTP_SERVER_READ_TIMEOUT` | `10s` | |
| `HTTP_SERVER_WRITE_TIMEOUT` | `10s` | |
| `DB_POSTGRES_DSN` | **обязателен** | Строка подключения к Postgres (`host=… port=… user=… password=… dbname=… sslmode=…`) |
| `DB_POSTGRES_MAX_OPEN_CONNS` | `10` | |
| `DB_POSTGRES_MAX_IDLE_CONNS` | `5` | |
| `DB_POSTGRES_MAX_LIFETIME` | `30m` | |
| `DB_POSTGRES_QUERY_TIMEOUT` | `5s` | |
| `CACHE_MEMORY_LIMIT_MB` | `1024` | Бюджет памяти; cleanup стартует при 90% |
| `CACHE_MEMORY_MONITOR_INTERVAL` | `5s` | Период проверки давления на память |

## API

Все эндпоинты находятся под префиксом `ROUTE_GROUP` (по умолчанию `/api/v1/task-service`).

### `POST /tasks` — создать задачу

```bash
curl -s -X POST localhost:8888/api/v1/task-service/tasks \
  -H 'Content-Type: application/json' \
  -d '{"name":"задача 1","body":"написать тесты"}'
```

```json
{ "id": 1 }
```

### `GET /tasks` — список задач (без пагинации)

```bash
curl -s localhost:8888/api/v1/task-service/tasks
```

```json
{
  "items": [
    {
      "id": 1,
      "name": "задача 1",
      "body": "написать тесты",
      "status": "NEW",
      "createdAt": "2025-08-25T13:41:16.443471+03:00",
      "updatedAt": null
    }
  ],
  "total": 1
}
```

### `GET /tasks/{id}` — получить задачу по id

```bash
curl -s localhost:8888/api/v1/task-service/tasks/1
```

Ошибки возвращаются в виде `{ "errorMessage": "...", "status": 4xx, "timestamp": "..." }`.

Полная спецификация доступна через Swagger UI — см. раздел [Swagger](#swagger-1).

## Локальный запуск

Требуется: Docker, Docker Compose, [Task](https://taskfile.dev/installation/) (`brew install go-task` / `go install github.com/go-task/task/v3/cmd/task@latest`).

```bash
# Поднять Postgres + миграции + сервис
task deploy:local

# Smoke-тест
curl localhost:8888/api/v1/task-service/tasks

# Логи
task deploy:local:logs

# Остановить и удалить volume-ы
task deploy:local:down
```

Без Docker:

```bash
cp .env.example .env  # отредактируйте DB_POSTGRES_DSN на свой локальный Postgres
task db:up            # накатить миграции
task build            # собрать бинарник в ./bin/server
./bin/server          # запустить
```

## Тесты

```bash
# Unit (быстрые, без БД)
task tests
task tests:coverage   # + HTML-отчёт + проверка порога

# Integration (поднимает test-стек, идёт в реальный Postgres; HTML-отчёт + проверка порога)
task integration-tests
```

Целевые пороги покрытия (можно переопределить ENV перед `task default`):

- `MIN_UNIT_COVERAGE` — по умолчанию **50%**
- `MIN_INTEGRATION_COVERAGE` — по умолчанию **30%**

Из подсчёта исключаются `/docs/`, `main.go`, `/dto/`, `/internal/root/` (см. `COVERAGE_EXCLUDE` в [Taskfile.yml](Taskfile.yml)).

Integration-тесты помечены build-tag-ом `integration` и лежат в отдельном пакете. Они ожидают Postgres на `127.0.0.1:5433` (его поднимает test-compose-файл); переопределяется через `TEST_DB_POSTGRES_DSN`.

## Линтер

```bash
task lint        # запустить
task lint:fix    # запустить с автофиксами
```

Конфиг — [.golangci.yml](.golangci.yml). Включены: `errcheck`, `staticcheck`, `revive`, `gosec`, `gocritic`, `gocyclo`, `bodyclose`, `nilerr`, `errorlint`, `prealloc`, `testifylint`, `testpackage`, `tparallel`, `forbidigo` (запрещает `fmt.Print*`, `errors.Wrap`), `depguard` (запрещает `pkg/errors`) и другие. Форматтеры: `gci`, `gofmt`, `gofumpt`, `goimports`.

## Сборка

```bash
# Локальный бинарник (./bin/server)
task build

# Docker-образ (target=app)
task build:docker
```

[build/server/Dockerfile](build/server/Dockerfile) — multi-stage:

- `builder` — `golang:1.26.3-alpine`, `go build`
- `goose-builder` — устанавливает goose в отдельном слое
- `app` — `alpine:3.20` с собранным бинарником, запускается под пользователем `app`
- `migrate` — `alpine:3.20` с goose и каталогом `/migrations`

## Деплой

Два compose-сценария.

### Local (`deployment/local/docker-compose.yml`)

Поднимает Postgres + миграции + сервис, пробрасывает `:8888`.

```bash
task deploy:local
task deploy:local:down
```

### Test (`deployment/test/docker-compose.yml`)

Поднимает только Postgres + миграции с пробросом `:5433`. Используется командой `task integration-tests`, которая поднимает стек, прогоняет `tests/integration/` и тушит стек.

```bash
task deploy:test
task deploy:test:down
```

## Swagger

Swagger UI доступен по адресу `http://localhost:8888/swagger/index.html` после поднятия сервиса.

Регенерация документации по аннотациям в коде:

```bash
task swag:gen
```

Эта команда выполняет `swag init -g cmd/main.go -o docs --parseInternal --parseDependency`, регенерируя `docs/docs.go`, `docs/swagger.json`, `docs/swagger.yaml`. Сгенерированные файлы коммитятся в репозиторий, чтобы CI и локальная сборка не зависели от наличия `swag` на машине.

Аннотации располагаются:

- общие meta (title / version / host / basePath) — в [cmd/main.go](cmd/main.go);
- эндпоинты — в [internal/server/task.go](internal/server/task.go), над `ListTasks`, `GetTask`, `CreateTask`.

## Миграции БД

`goose` управляет миграциями в [db/migrations](db/migrations).

```bash
# Применить ожидающие миграции
task db:up

# Откатить последнюю
task db:down

# Статус
task db:status

# Создать новую sql-миграцию
task db:create -- create_indexes
```

`GOOSE_DBSTRING` берётся из `.env` (см. [.env.example](.env.example)).

В контейнерах миграции применяются отдельным сервисом `task-service-migrate`, который запускается перед `task-service` (через `depends_on: condition: service_completed_successfully`).

## CI/CD

[.github/workflows/ci.yml](.github/workflows/ci.yml) триггерится на `push` в `main`/`master` и `pull_request`. Jobs:

| Job | Что делает |
|---|---|
| `fmt` | `task fmt:check` — падает, если `gofmt`/`goimports` находят неотформатированные файлы |
| `lint` | `task lint` |
| `unit-tests` | `task tests`, проверка порога покрытия, загрузка артефакта coverage |
| `integration-tests` | поднимает Postgres как `service`, накатывает миграции через goose, прогоняет тесты с тегом `integration` |
| `build` | `docker buildx build` обоих target-ов Dockerfile (`app` + `migrate`) |
| `security` | `govulncheck` |

## Соглашения и ограничения

- **Эвикция кеша.** Управляется памятью — cleanup стартует при достижении 90% настроенного лимита и удаляет 1/5 самых старых записей (отсортированных по ключу).
- **Список без пагинации.** `GET /tasks` возвращает всё, что есть (cache + остаток из БД). Подходит для PoC, не для прода — пагинация в [ROADMAP.md](ROADMAP.md).

## Roadmap

См. [ROADMAP.md](ROADMAP.md) — там список доработок (пагинация, валидация, метрики, трассировка, аутентификация, PATCH/DELETE, rate limits, расширение coverage, `pkg/background`).
