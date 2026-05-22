# Roadmap

Список доработок, запланированных после первичной стабилизации сервиса. Пункты не отсортированы по приоритету — выбирайте по контексту.

---

## Инфраструктура и качество

### 1. `BackgroundRegistrar` в `pkg/background`

**Что:** вынести логику `startBackgroundJobs` + `stop` из `internal/root/root.go` в переиспользуемый пакет `pkg/background`.

**Текущее состояние:** в [internal/root/root.go](internal/root/root.go) живут:
- `backgroundJobs []func() error`
- `stopHandlers []func()`
- `RegisterBackgroundJob`, `RegisterStopHandler`
- `startBackgroundJobs() chan error`
- `stop()` (параллельная остановка через `sync.WaitGroup`)

Это шаблон не специфичный для task-service — пригодится в любом сервисе с фоновыми задачами.

**Целевой API (примерно):**

```go
// pkg/background/registrar.go
package background

type Job func() error
type StopHandler func()

type Registrar struct { ... }

func New() *Registrar
func (r *Registrar) RegisterJob(j Job)
func (r *Registrar) RegisterStopHandler(h StopHandler)
func (r *Registrar) Run(ctx context.Context) error  // запускает все jobs, ждёт ctx.Done или первую ошибку
func (r *Registrar) Stop()                          // параллельный shutdown handler-ов
```

**Что меняется в проекте:**
- `internal/root/root.go` — содержит `*background.Registrar` вместо локальных полей; делегирует `RegisterBackgroundJob`/`RegisterStopHandler`/`Run`/`stop`.
- `internal/root/{http_server,observability,service,repository}.go` — без изменений, всё ещё вызывают `r.RegisterBackgroundJob` / `r.RegisterStopHandler`.
- Заодно фиксится мини-баг: при панике одного job-а сейчас `wg.Add(len(stopHandlers))` корректно, но `stop` запускает горутины не в `defer` — стоит пересобрать на `errgroup` или ручной `sync.WaitGroup` внутри `pkg/background`.

**Тесты:** unit-тесты в `pkg/background` на сценарии — все jobs зелёные → Run ждёт ctx; один job падает → Run возвращает ошибку и тушит handler-ы; ctx отменён → Run возвращается без ошибок; параллельность остановки.

---

### 2. Улучшить покрытие тестами

**Целевая планка:** ≥75% по всем пакетам, кроме `cmd`, `docs`, `ports`, `dto`, `internal/root` (исключены из подсчёта).

**Сделано (ветка `feature/expand-test-coverage`):**

- `internal/adapters/repositories` — unit-тесты через `go-sqlmock`: Store/Get/List, все ветки `buildListQuery`, `sql.ErrNoRows` → `ErrTaskNotFound`, ошибки query/scan/iteration.
- `pkg/http/protocol` — добавлена ветка `MarshalError` для `SendSuccessResponse` через тип с `MarshalJSON`, возвращающим ошибку.
- `pkg/http/server` router — пустой роутер, lowercase method, multi-`{param}`, разные статические сегменты, разное число сегментов, разные методы на одном пути, `RequestParams` без контекста.
- `pkg/cache` — Cleanup при `len < 5` (no-op), пустом кэше, `Store` overwrite, `Run` без интервала, `Run` cancel.
- `internal/root` — вынесен в `COVERAGE_EXCLUDE` (DI-wiring, покрывается integration-тестами).

**Остаётся открытым:**

- **`testcontainers-go` для repository unit-тестов.** Альтернатива sqlmock — поднять Postgres внутри теста без внешнего docker-compose, чтобы получить реальные query-execution планы (а не сравнение строк SQL). Минусы: 5-10 секунд старта контейнера на тест-сьют, требует Docker на машине разработчика и в CI. Решение про подключение — отдельный PR, когда станет узким местом разница между sqlmock и реальным Postgres.

---

### 3. Разобраться с миграциями goose в CI

**Что:** integration-tests job в CI падает на накатывании миграций:

```
go: downloading github.com/remyoudompheng/bigfft v0.0.0-20230129092748-24d4a6f8daec
2026/05/22 10:22:33 goose run: "postgres": no such command
exit status 1
```

**Где смотреть:** [.github/workflows/ci.yml](.github/workflows/ci.yml) job `integration-tests`, шаг `Apply migrations`:

```yaml
go run github.com/pressly/goose/v3/cmd/goose@v3.22.1 \
  -dir ./db/migrations \
  postgres "$GOOSE_DBSTRING" up
```

**Корень проблемы.** В свежих версиях goose v3 driver и DSN передаются не позиционно, а через переменные окружения `GOOSE_DRIVER`/`GOOSE_DBSTRING` (они уже выставлены в окружении job-а). После этого команда становится `goose -dir <path> up`, без слова `postgres` и без явного DSN. Старый синтаксис `goose ... postgres "dsn" up` отпал, отсюда `"postgres": no such command`.

**Что сделать:**
- В CI поменять команду на `goose -dir ./db/migrations up` (driver+DSN уже в env);
- Аналогично пройтись по таргетам `db:up`/`db:down`/`db:status` в [Taskfile.yml](Taskfile.yml) — там тоже передаём DSN позиционно (`goose ... postgres '<dsn>' up`), что сломается на той же версии. Перевести на env-mode и убрать позиционные аргументы;
- В Dockerfile (target `migrate`) ENTRYPOINT уже использует позиционный DSN — переписать на env-вариант, чтобы compose `task-service-migrate` сервис продолжал работать.
- Зафиксировать версию goose в одном месте (например, vars `GOOSE_VERSION` в Taskfile уже есть — её и тиражировать в CI и Dockerfile).

**Проверка:** `task deploy:test` + `task integration-tests` локально должны проходить без падения миграций; CI `integration-tests` job — зелёный.

---

## Roadmap фич

### 4. Пагинация для `GET /tasks` (`limit` / `offset` / `cursor`)

**Что:** добавить query-параметры `?limit=N&offset=M` (страничная навигация) или `?cursor=ID` (курсорная) для list-эндпоинта.

**Где:**
- `internal/server/dto/dto.go` — добавить `ListTasksQuery { Limit, Offset, Cursor uint64 }`, парсинг из `r.URL.Query()`.
- `internal/ports/task_repository.go` — расширить `ListTasksFilter` полями `Limit`, `Offset`, либо `Cursor`.
- `internal/services/task_service.go` — учесть в `List`: если limit задан, сначала пробуем кеш в этой границе, потом docорим из БД.
- `internal/adapters/repositories/task_repository.go` — `LIMIT $N OFFSET $M` в SQL.
- Swagger-аннотации в `internal/server/task.go`.

**Решение про default limit:** разумно 50, max — 500. Превышение → 400.

---

### 5. Валидация входных DTO через `go-playground/validator`

**Что:** заменить ручную проверку в `CreateTask` на тэги + единую функцию валидации.

**Где:**
- `internal/server/dto/dto.go` — теги `validate:"required,min=1,max=255"` на полях.
- Подключить `github.com/go-playground/validator/v10`, держать singleton.
- В `pkg/http/protocol/` добавить вспомогалку `ValidateAndBind(r *http.Request, v any) error`.
- Обновить тесты — проверить разные сценарии валидации (пустые поля, слишком длинные, нарушение формата).

**Учесть депенденси:** `validator/v10` весит ~200KB; для текущего сервиса это нормально.

---

### 6. Метрики Prometheus (`/metrics`)

**Что:** инструментация HTTP и БД, выкладывание `/metrics` эндпоинта.

**Где:**
- Новый `pkg/metrics` — обёртка над `github.com/prometheus/client_golang`, регистрирующая стандартные коллекторы и оборачивающая HTTP-handlerы middleware-ом для запросов/латентности/статусов.
- `internal/root/observability.go` — поднять отдельный HTTP-сервер на `:9090` (envconfig `METRICS_PORT`), смонтировать `promhttp.Handler()`.
- Конкретные метрики: `http_requests_total`, `http_request_duration_seconds` (histogram), `tasks_cache_size`, `tasks_cache_hits_total`, `tasks_cache_misses_total`, `db_queries_total`, `db_query_duration_seconds`.

**Тесты:** на регистрацию коллекторов и инкремент по запросу через `prometheus/testutil`.

---

### 7. Distributed tracing (OpenTelemetry)

**Что:** трассировка запросов через OTEL SDK с экспортом в Jaeger/Tempo.

**Где:**
- `pkg/tracing` — фабрика `TracerProvider`, конфиг (`OTEL_EXPORTER_OTLP_ENDPOINT`, `OTEL_SERVICE_NAME`, sampling rate).
- HTTP middleware из `go.opentelemetry.io/contrib/instrumentation/net/http/otelhttp` или собственный.
- Обёртка `sql.DB` через `github.com/XSAM/otelsql` для span-ов на каждом запросе.
- Спаны вокруг business-логики в `TaskService.Create/Get/List`.

**Учесть:** ENV-флаг `TRACING_ENABLED` чтобы можно было полностью отключить в локальной разработке.

---

### 8. Аутентификация (JWT / API token)

**Что:** middleware проверки токена; неавторизованные запросы → 401.

**Где:**
- `pkg/auth` — парсинг и валидация JWT (`github.com/golang-jwt/jwt/v5`), middleware HTTP.
- Альтернатива (проще) — статический API-token через ENV `SERVICE_AUTH_TOKEN`, как в data-service.
- `internal/root/http_server.go` — оборачивает router в middleware. Исключения: `/swagger/*`, `/metrics`, `/health` остаются открытыми.
- Конфиг: `AUTH_ENABLED`, `AUTH_SECRET`, `AUTH_TOKEN_TTL`.

**Решение:** для PoC хватит статического токена. JWT — когда появятся пользователи.

---

### 9. PATCH / DELETE для задач

**Что:** добавить ручки `PATCH /tasks/{id}` и `DELETE /tasks/{id}`.

**PATCH:**
- Принимает любые из полей `name`, `body`, `status`. Поля должны быть pointer'ами или sentinel для "не менять".
- Валидация переходов статуса (`NEW → IN_PROCESS → COMPLETE`, и т.д.).
- Обновление `updated_at` в БД.
- Инвалидировать запись в кэше.

**DELETE:**
- Hard delete или soft delete (`deleted_at TIMESTAMPTZ`) — решить.
- Удалить из кэша.

**Где:**
- `internal/server/task.go` — два новых хендлера, регистрация в `Api.InitRoutes`.
- `internal/ports/task_repository.go` — методы `Update`, `Delete`.
- `internal/adapters/repositories/task_repository.go` — SQL `UPDATE tasks SET ... WHERE id = $1`, `DELETE FROM tasks WHERE id = $1`.
- `internal/services/task_service.go` — `Update`, `Delete` с учётом кэша.
- `internal/ports/task_cache.go` — метод `Delete(id uint64)`.
- `pkg/cache/cache.go` — метод `Delete(key K)` (сейчас только Store/Get).
- Тесты unit + integration.

---

### 10. Лимит размера тела запроса и rate limiting

**Размер тела:**
- В `internal/server/task.go` — `r.Body = http.MaxBytesReader(w, r.Body, maxBodyBytes)` перед `io.ReadAll`.
- Конфиг: `HTTP_MAX_REQUEST_BODY_BYTES` (default 1MB).

**Rate limiting:**
- IP-based через `golang.org/x/time/rate.Limiter`, словарь по IP, периодическая чистка.
- Либо `github.com/didip/tollbooth/v7` — готовое решение.
- Конфиг: `HTTP_IP_RATE_LIMIT` (req/s), `HTTP_IP_RATE_BURST`.
- Middleware в `pkg/http/server/` или новом `pkg/http/middleware/`.

**Тесты:** проверить что 1024 запроса в секунду с лимитом 10 RPS → большинство получает 429.

---

## Уже выполнено (контекст)

- Sentinel `domain.ErrTaskNotFound` для not-found семантики Task (можно расширить аналогичной обработкой на другие ресурсы, когда они появятся).

---

## Как пользоваться этим документом

1. Не обязательно делать пункты по порядку — берите тот, что блокирует текущую задачу.
2. Каждый пункт — кандидат на отдельный PR с понятным scope. Разделяйте «инфраструктура» и «фичи» — это разный объём ревью.
3. Перед стартом крупных фич (метрики, трассировка, auth) — короткий design-doc в `docs/` (не путать с swagger `docs/`) c обоснованием выбора библиотеки.
