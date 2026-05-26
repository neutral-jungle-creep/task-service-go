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

### 2. `testcontainers-go` для repository unit-тестов

**Что:** альтернатива sqlmock — поднять реальный Postgres внутри теста через `testcontainers-go` (без зависимости от внешнего docker-compose), чтобы получить реальные query-execution планы вместо сравнения строк SQL.

**Минусы:** 5-10 секунд старта контейнера на тест-сьют; требует Docker на машине разработчика и в CI. Решение про подключение — отдельный PR, когда станет узким местом разница между sqlmock и реальным Postgres.

**Целевая планка покрытия:** ≥75% по всем пакетам, кроме `cmd`, `docs`, `ports`, `dto`, `internal/root` (исключены из подсчёта).

---

## Roadmap фич

### 3. Метрики Prometheus (`/metrics`)

**Что:** инструментация HTTP и БД, выкладывание `/metrics` эндпоинта.

**Где:**
- Новый `pkg/metrics` — обёртка над `github.com/prometheus/client_golang`, регистрирующая стандартные коллекторы и оборачивающая HTTP-handlerы middleware-ом для запросов/латентности/статусов.
- `internal/root/observability.go` — поднять отдельный HTTP-сервер на `:9090` (envconfig `METRICS_PORT`), смонтировать `promhttp.Handler()`.
- Конкретные метрики: `http_requests_total`, `http_request_duration_seconds` (histogram), `tasks_cache_size`, `tasks_cache_hits_total`, `tasks_cache_misses_total`, `db_queries_total`, `db_query_duration_seconds`.

**Тесты:** на регистрацию коллекторов и инкремент по запросу через `prometheus/testutil`.

---

### 4. Distributed tracing (OpenTelemetry)

**Что:** трассировка запросов через OTEL SDK с экспортом в Jaeger/Tempo.

**Где:**
- `pkg/tracing` — фабрика `TracerProvider`, конфиг (`OTEL_EXPORTER_OTLP_ENDPOINT`, `OTEL_SERVICE_NAME`, sampling rate).
- HTTP middleware из `go.opentelemetry.io/contrib/instrumentation/net/http/otelhttp` или собственный.
- Обёртка `sql.DB` через `github.com/XSAM/otelsql` для span-ов на каждом запросе.
- Спаны вокруг business-логики в `TaskService.Create/Get/List`.

**Учесть:** ENV-флаг `TRACING_ENABLED` чтобы можно было полностью отключить в локальной разработке.

---

### 5. Аутентификация (JWT / API token)

**Что:** middleware проверки токена; неавторизованные запросы → 401.

**Где:**
- `pkg/auth` — парсинг и валидация JWT (`github.com/golang-jwt/jwt/v5`), middleware HTTP.
- Альтернатива (проще) — статический API-token через ENV `SERVICE_AUTH_TOKEN`, как в data-service.
- `internal/root/http_server.go` — оборачивает router в middleware. Исключения: `/swagger/*`, `/metrics`, `/health` остаются открытыми.
- Конфиг: `AUTH_ENABLED`, `AUTH_SECRET`, `AUTH_TOKEN_TTL`.

**Решение:** для PoC хватит статического токена. JWT — когда появятся пользователи.

---

### 6. Лимит размера тела запроса и rate limiting

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
