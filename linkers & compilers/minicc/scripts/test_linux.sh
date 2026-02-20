#!/usr/bin/env bash
# test_linux.sh — End-to-end test on Linux (Docker, ch07+)
# Same as test_macos.sh but targets linux platform.
set -euo pipefail

SCRIPT_DIR="$(cd "$(dirname "$0")" && pwd)"
PROJECT_DIR="$(dirname "$SCRIPT_DIR")"
MINICC="${PROJECT_DIR}/bin/minicc"
RUNTIME="${PROJECT_DIR}/runtime/runtime.c"
RESULTS="${PROJECT_DIR}/testdata/expected/results.txt"
TMPDIR="${TMPDIR:-/tmp}"

if [[ ! -x "$MINICC" ]]; then
    echo "ERROR: minicc not found at $MINICC — run 'make build' first"
    exit 1
fi

passed=0
failed=0
total=0

while IFS= read -r line; do
    [[ "$line" =~ ^#.*$ || -z "$line" ]] && continue

    file=$(echo "$line" | awk '{print $1}')
    expected=$(echo "$line" | awk '{print $2}')
    total=$((total + 1))

    src="${PROJECT_DIR}/testdata/valid/${file}"
    asm="${TMPDIR}/minicc_${file%.mc}.s"
    bin="${TMPDIR}/minicc_${file%.mc}"

    if ! "$MINICC" -platform linux "$src" -o "$asm" 2>/dev/null; then
        echo "FAIL  $file  (compile error)"
        failed=$((failed + 1))
        continue
    fi

    if ! gcc "$asm" "$RUNTIME" -o "$bin" -no-pie 2>/dev/null; then
        echo "FAIL  $file  (assemble/link error)"
        failed=$((failed + 1))
        continue
    fi

    set +e
    "$bin"
    actual=$?
    set -e

    if [[ "$actual" -eq "$expected" ]]; then
        echo "PASS  $file  (exit=$actual)"
        passed=$((passed + 1))
    else
        echo "FAIL  $file  (expected=$expected, got=$actual)"
        failed=$((failed + 1))
    fi
done < "$RESULTS"

echo ""
echo "Results: $passed/$total passed, $failed failed"
[[ "$failed" -eq 0 ]]
