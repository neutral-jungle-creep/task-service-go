#!/bin/sh
set -e

COVERAGE_FILE="${1:-.coverage/unit.out}"
THRESHOLD="${2:-60}"
# EXCLUDE_PATTERN — optional egrep regex for paths excluded from total coverage
# (e.g. "/docs/|/mocks/"). Useful for generated or trivial packages that are
# not meant to be covered by tests.
EXCLUDE_PATTERN="${3:-}"

if [ ! -f "$COVERAGE_FILE" ]; then
    echo "ERROR: coverage file not found: $COVERAGE_FILE"
    exit 1
fi

TARGET_FILE="$COVERAGE_FILE"
if [ -n "$EXCLUDE_PATTERN" ]; then
    TARGET_FILE="${COVERAGE_FILE}.filtered"
    awk -v pat="$EXCLUDE_PATTERN" 'NR==1 || $0 !~ pat' "$COVERAGE_FILE" > "$TARGET_FILE"
fi

PCT=$(go tool cover -func="$TARGET_FILE" | grep "^total:" | awk '{gsub(/%/, ""); print $NF}')

printf "Coverage: %s%%  (threshold: %s%%)\n" "$PCT" "$THRESHOLD"
if [ -n "$EXCLUDE_PATTERN" ]; then
    printf "Excluded from total: %s\n" "$EXCLUDE_PATTERN"
fi

awk -v pct="$PCT" -v thr="$THRESHOLD" 'BEGIN {
    if (pct + 0 < thr + 0) {
        printf "FAIL: coverage %.1f%% is below threshold %s%%\n", pct + 0, thr
        exit 1
    }
    printf "PASS: coverage %.1f%% >= threshold %s%%\n", pct + 0, thr
}'
