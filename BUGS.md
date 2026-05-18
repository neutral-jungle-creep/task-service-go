# Найденные баги и предложения по исправлению

Список проблем, выявленных при ревью текущего состояния `task-service-go`. Каждый пункт содержит локацию, описание, последствия и предлагаемый фикс. Severity: **High** — ломает функциональность; **Medium** — корректное поведение в нормальных случаях, но рушится при нагрузке/edge-кейсах; **Low** — стиль/устойчивость.

> **Закрыто:** пункты #1 (`protocol.go` — порядок WriteHeader/Write) и #2 (`task.go` — отсутствующий `return` в `GetTask`) — исправлено.

---

## 3. `task_cache.go` — `cleanup()` ломается при разрывах в id

**Severity:** Medium → ✅ **Исправлено**
**Файл (старый):** `internal/services/task_cache.go` — функция `cleanup()`.

**Проблема (исторически).**

```go
for key := firstStoredKey; key < cleanupCount; key++ { // works only for auto-increment ids without gaps
    if _, ok := t.tasks.Load(key); ok {
        t.tasks.Delete(key)
    }
    ...
}
```

Цикл идёт диапазоном `[firstStoredKey; cleanupCount)`. При разрывах в id (удалённые записи, миграции, неавтоинкрементные ключи) логика:
- может пропустить нужные удаления и оставить кэш переполненным;
- может выйти за пределы реальных id и не удалить ничего;
- условие `key < cleanupCount` опирается на ОБЩЕЕ число элементов как на правую границу id, что концептуально неверно.

**Последствия.** Кэш растёт неограниченно при работе с прерывистыми id; OOM при долгой работе.

**Текущий статус.** При выносе кэша в `pkg/cache` (пункт 6 плана) логика переписана: собираем все ключи, сортируем, удаляем самые старые `Len/5`. Работает с любыми id. Файл [pkg/cache/cache.go](pkg/cache/cache.go), метод `Cleanup()`.

---

## 4. Опечатка в имени файла: `tast_service.go`

**Severity:** Low → ✅ **Исправлено** (переименован в `task_service.go`)
**Файл:** `internal/services/tast_service.go` → [internal/services/task_service.go](internal/services/task_service.go).

**Проблема.** Очевидная опечатка: должно быть `task_service.go`.

**Фикс.** Просто переименовать файл — никаких других изменений не требуется (имя файла не часть API).

---

## 5. `domain.Task.Size()` — не учитывает реальный размер строк

**Severity:** Medium → ✅ **Исправлено** (Size учитывает длину строк; `fill` budget теперь в байтах)
**Файл:** [internal/domain/task.go](internal/domain/task.go), метод `Size()`.

**Проблема.**

```go
size += unsafe.Sizeof(t.Name)   // 16 байт — размер string-заголовка (ptr+len), а не данных
size += unsafe.Sizeof(t.Body)   // тоже 16 байт
size += unsafe.Sizeof(t.UpdatedAt)  // 8 байт — размер указателя, а не payload
```

`unsafe.Sizeof` на строке возвращает размер заголовка (16 байт на 64-битной платформе) независимо от того, сколько символов в строке. То же про `*time.Time` — возвращает размер указателя.

**Последствия.** `Size()` всегда выдаёт почти константу (~88 байт независимо от содержимого). Это используется в `TaskCache.fill` для решения "поместить ли запись в кэш". В результате:
- кэш считает что задачи занимают мало места;
- решения об эвикции принимаются на ложных данных;
- `cleanupStartMB` в исходном коде сравнивался с `totalSize` в байтах, но `cleanupStartMB` — это число мегабайт. Получалось например `cleanupStartMB = 90` (для 100MB лимита), и `totalSize >= 81` (после умножения на 0.9). То есть после 1 task'а fill завершался — кэш не наполнялся.

**Фикс.**

```go
func (t *Task) Size() uint64 {
    size := uint64(unsafe.Sizeof(*t))
    size += uint64(len(t.Name))
    size += uint64(len(t.Body))
    size += uint64(len(t.Status))
    if t.UpdatedAt != nil {
        size += uint64(unsafe.Sizeof(*t.UpdatedAt))
    }
    return size
}
```

И в [internal/services/task_cache.go](internal/services/task_cache.go) `fill` — budget уже считается в байтах (`memoryLimitMB * 1024 * 1024 * 0.9`) после рефакторинга в пункте 6 плана.

---

## 6. `cmd/main.go` — `panic` на init-ошибках

**Severity:** Medium
**Файл:** [cmd/main.go](cmd/main.go).

**Проблема.**

```go
cfg, err := config.NewConfigFromFile(...)
if err != nil {
    panic(err)
}

logger, err := logging.NewLogger(...)
if err != nil {
    panic(err)
}
```

`panic` с `*errors.errorString` выводит `goroutine 1 [running]:` со стеком — нечитабельно для оператора, не пишется в стандартный лог-формат, не позволяет добавить контекст.

**Последствия.** Stacktrace без понятного сообщения об ошибке загрузки конфига или логгера. Проблема диагностики.

**Фикс.** Заменить на `log.Fatalf("load config: %v", err)` / `log.Fatalf("init logger: %v", err)`. Это даёт чистое сообщение и `os.Exit(1)`.

---

## 7. `root.go` — небуферизованный канал ошибок теряет ошибки

**Severity:** Medium
**Файл:** [internal/root/root.go](internal/root/root.go), функция `startBackgroundJobs`.

**Проблема.**

```go
errors := make(chan error)  // буфер 0

for _, job := range r.backgroundJobs {
    go func() {
        errors <- job()  // блокируется, пока кто-то не прочитает
    }()
}
```

`Run` читает из этого канала ровно ОДИН раз. Если несколько job'ов падают одновременно (или одна падает быстро, а другая чуть позже) — первая ошибка попадает в `Run`, вторая горутина зависает на отправке навсегда (goroutine leak).

**Последствия.**
- Утечка горутин на каждой второй и далее упавшей фоновой задаче.
- При shutdown по контексту (case `<-r.ctx.Done()`) канал не вычитывается совсем — ВСЕ горутины зависают на send и не освобождаются до завершения процесса.

**Фикс.** Буферизовать канал размером со список jobs:

```go
errors := make(chan error, len(r.backgroundJobs))
```

---

## 8. `router.go` — некорректное различение 404 vs 405

**Severity:** Medium
**Файл:** [pkg/http/server/router.go](pkg/http/server/router.go), `ServeHTTP`.

**Проблема.**

```go
routesForMethod, ok := r.routes[method]
if ok {
    for _, ro := range routesForMethod {
        if matched := ...; matched { ... return }
    }
    w.WriteHeader(http.StatusMethodNotAllowed)  // 405 для любого ненайденного пути
    return
}
w.WriteHeader(http.StatusNotFound)
```

Если для метода (`GET`) есть хотя бы один зарегистрированный роут, любой несовпавший URL по этому методу получает 405. Корректная семантика: 405 — путь существует, но не для этого метода; 404 — путь вообще не зарегистрирован.

Пример: GET `/unknown` → должен быть 404, сейчас 405.

**Дополнительный баг в `matchPattern`.** Когда pattern и path не содержат `{}` и они не равны, функция всё равно идёт в split и при равном количестве сегментов возвращает `(true, {})` — даже если статические сегменты разные. Например `pattern=/x/y`, `path=/a/b` → ошибочно матчится.

**Фикс.**
- В `matchPattern`: при равенстве числа сегментов добавить явное сравнение статических сегментов (не-параметрических).
- В `ServeHTTP`: после неудачи в текущем методе пройтись по другим методам — если путь известен другому методу, отдать 405; иначе 404.

---

## 9. Race condition: чтение `firstTaskKey` в `TaskService.List`

**Severity:** Medium
**Файл:** [internal/services/tast_service.go](internal/services/tast_service.go), метод `List`.

**Проблема.**

```go
tasksFromCache, firstTaskKey := s.cache.List()
if firstTaskKey == 1 {
    return tasksFromCache, nil
}
tasksFromDb, err := s.repository.List(&ports.ListTasksFilter{ ToID: firstTaskKey })
```

Между чтением `firstTaskKey` и отправкой запроса в репозиторий фоновый memory-monitor может вызвать `cleanup()` и сдвинуть `firstKey` вперёд. В итоге:
- БД отдаст записи с id < старого firstKey, но не отдаст записи в диапазоне `[oldFirstKey; newFirstKey)`, которые после cleanup уже не в кэше.
- В результирующем списке появляется "дыра".

**Последствия.** Периодически часть task'ов может пропадать из ответа `GET /tasks` под нагрузкой и cleanup-событиями.

**Фикс.** Сделать `List()` атомарным со стороны кэша: возвращать одновременно snapshot tasks + firstKey под одной блокировкой. Либо дать кэшу метод `ListWithRepoFallback(repo)`, который принимает решение под локом. Либо использовать версионирование (epoch) и retry.

---

## 10. Валидация DTO: `binding:"required"` без валидатора

**Severity:** Low
**Файл:** [internal/server/dto/dto.go](internal/server/dto/dto.go).

**Проблема.**

```go
type CreateTaskRequest struct {
    Name string `json:"name" binding:"required"`
    Body string `json:"body" binding:"required"`
}
```

Тэг `binding` распознаётся фреймворками вроде Gin/Echo. У нас собственный роутер и `json.Unmarshal` — этот тэг ничего не валидирует. Можно прислать `{"name": "", "body": ""}` — пройдёт.

**Фикс.** Либо подключить `go-playground/validator` и явно валидировать в хендлере, либо убрать ввод в заблуждение тэг и валидировать вручную:

```go
if params.Name == "" || params.Body == "" {
    protocol.SendErrorResponse(w, http.StatusBadRequest, ...)
    return
}
```

---

## 11. Потенциальный баг при пустой записи в БД (`Get` возвращает `&Task{}` вместо ошибки)

**Severity:** Low
**Файл:** [internal/adapters/repositories/task_repository.go](internal/adapters/repositories/task_repository.go), метод `Get`.

**Проблема.** При `sql.ErrNoRows` репозиторий возвращает `&domain.Task{}, nil`. Хендлер `GetTask` в `task.go` опирается на `task.ID == 0` чтобы понять "не нашли". Это работает, но семантически "0 = не найдено" — соглашение, не контракт. Если кто-то поменяет логику генерации id (например, репозиторий начнёт возвращать 0 как валидный id или для сообщения о другой ошибке) — клиенты получат 404 вместо реальной ошибки.

**Фикс.** Возвращать `nil, ErrNotFound` (sentinel error в `internal/domain` или `ports`) и обрабатывать его явно через `errors.Is` в хендлере.

---

## Сводная таблица

| № | Файл | Severity | Что сломано | Кратко исправление | Статус |
|---|---|---|---|---|---|
| 1 | pkg/http/protocol/protocol.go | High | WriteHeader после Write | поменять порядок | ✅ исправлено |
| 2 | internal/server/task.go | High | нет return после 400 | добавить `return` | ✅ исправлено |
| 3 | pkg/cache/cache.go | Medium | cleanup ломается на разрывах; len не трекается | sort + drop oldest N; LoadOrStore inc len | ✅ исправлено |
| 4 | internal/services/tast_service.go | Low | опечатка имени файла | переименовать | ✅ исправлено |
| 5 | internal/domain/task.go | Medium | Size без длины строк | `len(Name)+len(Body)+...` | ✅ исправлено |
| 6 | cmd/main.go | Medium | panic на init | `log.Fatalf` |
| 7 | internal/root/root.go | Medium | leak горутин при множественных ошибках | буферизованный канал |
| 8 | pkg/http/server/router.go | Medium | 405 вместо 404 + статические сегменты | пройти по другим методам, сравнить сегменты |
| 9 | internal/services/tast_service.go | Medium | гонка `firstTaskKey` vs cleanup | snapshot под локом |
| 10 | internal/server/dto/dto.go | Low | `binding:"required"` без валидатора | validator или ручная проверка |
| 11 | internal/adapters/repositories/task_repository.go | Low | not-found через `ID == 0` | sentinel ErrNotFound |
