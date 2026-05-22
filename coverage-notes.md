# Проблема в `scripts/check_coverage.sh` и как её поправить

## Проблема

Скрипт `scripts/check_coverage.sh` печатает только итог покрытия и
вердикт `PASS/FAIL` относительно порога:

```
Coverage: 32.4%  (threshold: 50%)
FAIL: coverage 32.4% is below threshold 50%
```

Когда тест падает по порогу — из вывода **непонятно, где именно
просел coverage**. Чтобы понять, нужно либо вручную запустить:

```sh
go tool cover -func=.coverage/unit.out
```

(и при этом без учёта `EXCLUDE_PATTERN`, который применяется в
скрипте — то есть видишь не те же данные, по которым считался total),

либо открывать HTML-отчёт `.coverage/unit.html` — в CI это неудобно
или вообще невозможно.

## Решение

Добавить в скрипт печать per-function таблицы по тому же
**отфильтрованному** профилю (`TARGET_FILE`), по которому считается
total — чтобы таблица и итог были согласованы. Печатать таблицу
**до** проверки порога: при FAIL она остаётся в логе и сразу видна
как причина падения.

### Конкретные изменения в `scripts/check_coverage.sh`

Между блоком фильтрации (после `if [ -n "$EXCLUDE_PATTERN" ]`) и
строкой с подсчётом `PCT`, вместо одного вызова `go tool cover -func`
в `PCT=...` сохранить его вывод в переменную и переиспользовать:

```sh
FUNC_OUTPUT=$(go tool cover -func="$TARGET_FILE")
PCT=$(printf '%s\n' "$FUNC_OUTPUT" | grep "^total:" | awk '{gsub(/%/, ""); print $NF}')

# Per-function таблица печатается до проверки порога — чтобы при FAIL
# (awk exit 1 ниже) разработчик видел таблицу первой и сразу понимал,
# где не хватает покрытия. Источник правды один: TARGET_FILE — тот же
# профиль, по которому считается total, поэтому таблица и итог согласованы.
echo "--- per-function coverage ---"
if [ -n "$EXCLUDE_PATTERN" ]; then
    printf "(excluded from total: %s)\n" "$EXCLUDE_PATTERN"
fi
printf '%s\n' "$FUNC_OUTPUT"
echo "--- end per-function ---"
```

И **убрать** дублирующее сообщение об исключениях после строки
`Coverage:` — `EXCLUDE_PATTERN` уже выводится один раз над таблицей,
повторно показывать его после `Coverage:` не нужно:

```diff
 printf "Coverage: %s%%  (threshold: %s%%)\n" "$PCT" "$THRESHOLD"
-if [ -n "$EXCLUDE_PATTERN" ]; then
-    printf "Excluded from total: %s\n" "$EXCLUDE_PATTERN"
-fi
```

### Результат

После правки вывод `task tests:coverage` (или `task integration-tests`)
становится таким:

```
--- per-function coverage ---
(excluded from total: /docs/|/mocks/)
github.com/.../service.go:12:	NewService	100.0%
github.com/.../service.go:25:	Do		60.0%
...
total:				(statements)	32.4%
--- end per-function ---
Coverage: 32.4%  (threshold: 50%)
FAIL: coverage 32.4% is below threshold 50%
```

Сразу видно, какие функции тянут coverage вниз, без отдельного запуска
`go tool cover -func` и без открытия HTML.

## Что в Taskfile трогать не надо

В `Taskfile.yml` есть вызовы `go tool cover -html=...` — это не
дублирование `-func`, а генерация HTML-визуализации для локального
просмотра. Оставить как есть.

Сам Taskfile-блок `tests:coverage` / `integration-tests` менять не
надо — изменения локализованы внутри `scripts/check_coverage.sh`.
