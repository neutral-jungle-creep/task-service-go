# Task service

REST-сервис для управления задачами (CRUD): создание, получение по id, список. Поверх PostgreSQL стоит in-memory кеш с эвикцией по памяти, чтобы свежие задачи отдавались без похода в БД.

---

## Содержание

- [Технологический стек](#технологический-стек)
- [Архитектура](#архитектура)
- [Структура директорий](#структура-директорий)
- [Конфигурация](#конфигурация)
- [API](#api)
- [Локальный запуск](#локальный-запуск)
- [Тесты](#тесты)
- [Линтер](#линтер)
- [Сборка](#сборка)
- [Деплой](#деплой)
- [Swagger](#swagger)
- [Миграции БД](#миграции-бд)
- [CI/CD](#cicd)
- [Соглашения и ограничения](#соглашения-и-ограничения)
- [Roadmap](#roadmap)

---

## Технологический стек

| Слой | Технология |
|---|---|
| Язык | Go 1.26.3 |
| HTTP | stdlib `net/http` + собственный роутер [pkg/http/server](pkg/http/server) с поддержкой `{param}` |
| Логи | собственный синхронный + асинхронный логгер [pkg/logging](pkg/logging) |
| Кеш | generic in-memory cache [pkg/cache](pkg/cache) с эвикцией по памяти |
| БД | PostgreSQL 17 + драйвер [`jackc/pgx/v5/stdlib`](https://github.com/jackc/pgx) |
| Миграции | [`pressly/goose/v3`](https://github.com/pressly/goose) |
| Конфигурация | [`kelseyhightower/envconfig`](https://github.com/kelseyhightower/envconfig) + [`joho/godotenv`](https://github.com/joho/godotenv) (.env для локалки) |
| Документация API | [`swaggo/swag`](https://github.com/swaggo/swag) + [`swaggo/http-swagger/v2`](https://github.com/swaggo/http-swagger) |
| Тесты | стандартный `testing` + [`stretchr/testify`](https://github.com/stretchr/testify) |
| Контейнеризация | Docker (multi-stage) + docker-compose |
| Оркестратор задач | [Task](https://taskfile.dev) (`Taskfile.yml`) |
| Линтер | [`golangci-lint` v2](https://golangci-lint.run) (.golangci.yml) |
| Уязвимости | `govulncheck` |
| CI | GitHub Actions |

## Архитектура

Чистая слоистая архитектура с DI-контейнером (`internal/root`):

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

Принципы:
- **Ports & Adapters** — внешние зависимости (БД, кеш, HTTP) скрываются за интерфейсами в `internal/ports`.
- **Однонаправленные импорты** — `domain` ← `ports` ← `services` ← `adapters`/`server` ← `root` ← `cmd`.
- **Background jobs + stop handlers** — каждое долгоживущее сервисное соединение (HTTP-сервер, асинхронный логгер, кеш-monitor, БД-пул) регистрируется в DI-контейнере, который параллельно поднимает их в `Run` и останавливает в `stop`.

## Структура директорий

```
task-service-go/
├── .github/workflows/ci.yml      # CI: lint, unit, integration, build, security
├── .golangci.yml                 # конфиг линтера v2
├── .env.example                  # шаблон переменных окружения
├── Taskfile.yml                  # оркестратор задач (go-task)
├── BUGS.md                       # известные баги (для сервисной/ревью-доки)
├── README.md
├── go.mod / go.sum
├── build/
│   └── server/
│       └── Dockerfile            # multi-stage: builder, app, migrate
├── deployment/
│   ├── local/docker-compose.yml  # postgres + migrate + app
│   └── test/docker-compose.yml   # postgres + migrate (для integration-тестов)
├── db/
│   └── migrations/               # SQL-миграции goose
├── docs/                         # swagger (генерируется `task swag:gen`)
├── scripts/
│   └── check_coverage.sh         # порог покрытия для CI
├── cmd/main.go                   # точка входа
├── internal/
│   ├── adapters/repositories/    # Postgres-реализация TaskRepository
│   ├── config/                   # envconfig-конфиг + .env поддержка
│   ├── domain/                   # Task, TaskStatus
│   ├── ports/                    # интерфейсы между слоями
│   ├── root/                     # DI: orchestration сервисов и фоновых задач
│   └── server/                   # HTTP API + DTO + конвертеры
├── pkg/                          # переиспользуемые модули
│   ├── cache/                    # generic in-memory cache
│   ├── http/
│   │   ├── protocol/             # JSON success/error responses
│   │   └── server/               # роутер + http.Server factory
│   ├── logging/                  # sync + async логгеры
│   └── postgres/                 # фабрика *sql.DB поверх pgx/v5/stdlib
└── tests/
    └── integration/              # HTTP-тесты вокруг реального Postgres
```

## Конфигурация

Конфигурация читается через `kelseyhightower/envconfig`. Локально удобно держать значения в `.env` (он автоматически подхватывается); в контейнерах — через `environment:` в compose-файлах. Шаблон значений: [.env.example](.env.example).

| ENV | Default | Описание |
|---|---|---|
| `SERVICE_NAME` | `task-service-go` | Имя сервиса в логах |
| `RELEASE_ID` | — | Идентификатор сборки в логах |
| `LOG_LEVEL` | `info` | `debug` / `info` / `warn` / `error` / `fatal` |
| `ROUTE_GROUP` | `/api/v1/task-service` | Префикс HTTP-роутов |
| `HTTP_SERVER_LISTEN_PORT` | `8888` | Порт прослушивания |
| `HTTP_SERVER_KEEP_ALIVE_TIME` | `60s` | |
| `HTTP_SERVER_KEEP_ALIVE_TIMEOUT` | `10s` | |
| `HTTP_SERVER_READ_HEADER_TIMEOUT` | `10s` | |
| `HTTP_SERVER_READ_TIMEOUT` | `10s` | |
| `HTTP_SERVER_WRITE_TIMEOUT` | `10s` | |
| `DB_POSTGRES_DSN` | **required** | Строка подключения к Postgres (формат `host=… port=… user=… password=… dbname=… sslmode=…`) |
| `DB_POSTGRES_MAX_OPEN_CONNS` | `10` | |
| `DB_POSTGRES_MAX_IDLE_CONNS` | `5` | |
| `DB_POSTGRES_MAX_LIFETIME` | `30m` | |
| `DB_POSTGRES_QUERY_TIMEOUT` | `5s` | |
| `CACHE_MEMORY_LIMIT_MB` | `1024` | Лимит по памяти; cleanup стартует на 90% |
| `CACHE_MEMORY_MONITOR_INTERVAL` | `5s` | Период проверки памяти |

## API

Все ручки доступны под префиксом из `ROUTE_GROUP` (по умолчанию `/api/v1/task-service`).

### `POST /tasks` — создать задачу

```bash
curl -s -X POST localhost:8888/api/v1/task-service/tasks \
  -H 'Content-Type: application/json' \
  -d '{"name":"task 1","body":"write tests"}'
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

### `GET /tasks/{id}` — получить задачу по id

```bash
curl -s localhost:8888/api/v1/task-service/tasks/1
```

Ошибки возвращаются JSON-объектом `{ "errorMessage": "...", "status": 4xx, "timestamp": "..." }`.

Полная спецификация доступна в Swagger UI — см. [Swagger](#swagger).

## Локальный запуск

Понадобится: Docker, Docker Compose, [Task](https://taskfile.dev/installation/) (`brew install go-task` / `go install github.com/go-task/task/v3/cmd/task@latest`).

```bash
# Поднять Postgres + миграции + сервис
task deploy:local

# Проверить
curl localhost:8888/api/v1/task-service/tasks

# Логи
task deploy:local:logs

# Остановить и удалить тома
task deploy:local:down
```

Или без Docker:

```bash
cp .env.example .env  # отредактируйте DB_POSTGRES_DSN под локальный Postgres
task db:up            # накатить миграции
task build            # собрать бинарник в ./bin/server
./bin/server          # запустить
```

## Тесты

```bash
# Unit (быстрые, без БД)
task tests
task tests:coverage   # + HTML-отчёт + проверка порога

# Integration (поднимает test-стек, ходит в реальный Postgres)
task integration-tests
task integration-tests:coverage
```

Целевые пороги покрытия (настраиваются через ENV перед `task default`):

- `MIN_UNIT_COVERAGE` — по умолчанию **50%**
- `MIN_INTEGRATION_COVERAGE` — по умолчанию **30%**

Из подсчёта исключаются `/docs/`, `main.go`, `/dto/` (см. `COVERAGE_EXCLUDE` в [Taskfile.yml](Taskfile.yml)).

Integration-тесты помечены build-tag'ом `integration` и запускаются отдельно. Они ожидают Postgres на `127.0.0.1:5433` (поднимается через `deployment/test/docker-compose.yml`); DSN можно переопределить через `TEST_DB_POSTGRES_DSN`.

## Линтер

```bash
task lint        # запустить
task lint:fix    # запустить с автофиксами
```

Конфиг — [.golangci.yml](.golangci.yml). Включены: `errcheck`, `staticcheck`, `revive`, `gosec`, `gocritic`, `gocyclo`, `bodyclose`, `nilerr`, `errorlint`, `prealloc`, `testifylint`, `testpackage`, `tparallel`, `forbidigo` (запрет `fmt.Print*`, `errors.Wrap`), `depguard` (запрет `pkg/errors`) и др. Форматтеры: `gci`, `gofmt`, `gofumpt`, `goimports`.

## Сборка

```bash
# Локальный бинарник (./bin/server)
task build

# Docker-образ (target=app)
task build:docker
```

Dockerfile [build/server/Dockerfile](build/server/Dockerfile) — multi-stage:

- `builder` — `golang:1.26.3-alpine`, `go build`
- `goose-builder` — устанавливает goose в отдельный layer
- `app` — `alpine:3.20` с собранным бинарником, запускается под user `app`
- `migrate` — `alpine:3.20` с goose и каталогом `/migrations`

## Деплой

Два compose-сценария:

### Local (`deployment/local/docker-compose.yml`)

Поднимает Postgres + миграции + сервис, пробрасывает `:8888`.

```bash
task deploy:local
task deploy:local:down
```

### Test (`deployment/test/docker-compose.yml`)

Поднимает только Postgres + миграции с прокинутым наружу `:5433`. Используется командой `task integration-tests`, которая поднимает стек, прогоняет `tests/integration/`, останавливает стек.

```bash
task deploy:test
task deploy:test:down
```

## Swagger

Swagger UI доступен по адресу `http://localhost:8888/swagger/index.html` после поднятия сервиса.

Чтобы обновить документацию из аннотаций в коде:

```bash
task swag:gen
```

Это вызовет `swag init -g cmd/main.go -o docs --parseInternal --parseDependency`, что регенерирует `docs/docs.go`, `docs/swagger.json`, `docs/swagger.yaml`. Сгенерированные файлы коммитятся в репозиторий, чтобы CI и локальные сборки не зависели от установки `swag` на машине.

Аннотации:

- общие (title/version/host/basePath) — в [cmd/main.go](cmd/main.go);
- эндпоинты — в [internal/server/task.go](internal/server/task.go) над методами `ListTasks`, `GetTask`, `CreateTask`.

## Миграции БД

Используется `goose`, миграции лежат в [db/migrations](db/migrations).

```bash
# Применить все pending миграции
task db:up

# Откатить последнюю
task db:down

# Статус
task db:status

# Создать новую миграцию (sql)
task db:create -- create_indexes
```

`GOOSE_DBSTRING` подхватывается из `.env` (см. [.env.example](.env.example)).

В контейнере миграции применяются отдельным сервисом `task-service-migrate` из compose-файла, который запускается перед `task-service` (через `depends_on: condition: service_completed_successfully`).

## CI/CD

[.github/workflows/ci.yml](.github/workflows/ci.yml) триггерится на `push` в `main`/`master` и `pull_request`. Jobs:

| Job | Что делает |
|---|---|
| `lint` | `golangci-lint-action` |
| `unit-tests` | `go test -race -coverprofile`, проверка порога покрытия, upload coverage artifact |
| `integration-tests` | поднимает Postgres как `service`, накатывает миграции через goose, прогоняет тесты с тегом `integration` |
| `build` | `docker buildx build` обоих target'ов Dockerfile (`app` + `migrate`) |
| `security` | `govulncheck` |

## Соглашения и ограничения

- **Постоянный кэш + сорт по id.** Эвикция работает по лимиту памяти (cleanup срабатывает на 90%, дропается 1/5 самых старых записей сорта по ключу). Кэш ожидает что ключ — auto-increment id.
- **Список без пагинации.** `GET /tasks` отдаёт всё что есть (cache + остаток в БД). Это ок для PoC, но не для прода.
- **Не-валидируемые DTO.** Тэги `binding:"required"` сейчас не используются (нет валидатора). См. [BUGS.md](BUGS.md) #10.
- **Известные баги.** Подробный список — [BUGS.md](BUGS.md).

## Roadmap

- [ ] Пагинация для `GET /tasks` (`limit`/`offset`/`cursor`).
- [ ] Валидация входных DTO через `go-playground/validator`.
- [ ] Метрики Prometheus (`/metrics`).
- [ ] Distributed tracing (OpenTelemetry).
- [ ] Аутентификация (JWT / API token).
- [ ] PATCH/DELETE задач.
- [ ] Лимиты на размер тела запроса и rate limiting.
- [ ] Использовать sentinel `ErrNotFound` вместо `task.ID == 0` (см. [BUGS.md](BUGS.md) #11).
