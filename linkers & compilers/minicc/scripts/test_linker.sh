#!/usr/bin/env bash
# test_linker.sh — End-to-end linker test (Docker, ch11+)
# Compiles .mc → assembly → ELF .o via minicc, then links with minilink.
set -euo pipefail

SCRIPT_DIR="$(cd "$(dirname "$0")" && pwd)"
PROJECT_DIR="$(dirname "$SCRIPT_DIR")"
MINICC="${PROJECT_DIR}/bin/minicc"
MINILINK="${PROJECT_DIR}/bin/minilink"
RUNTIME="${PROJECT_DIR}/runtime/runtime.c"
RESULTS="${PROJECT_DIR}/testdata/expected/results.txt"
TMPDIR="${TMPDIR:-/tmp}"

for tool in "$MINICC" "$MINILINK"; do
    if [[ ! -x "$tool" ]]; then
        echo "ERROR: $(basename "$tool") not found — run 'make build' first"
        exit 1
    fi
done

passed=0
failed=0
total=0

# Compile runtime to .o
gcc -c "$RUNTIME" -o "${TMPDIR}/runtime.o" 2>/dev/null

while IFS= read -r line; do
    [[ "$line" =~ ^#.*$ || -z "$line" ]] && continue

    file=$(echo "$line" | awk '{print $1}')
    expected=$(echo "$line" | awk '{print $2}')
    total=$((total + 1))

    src="${PROJECT_DIR}/testdata/valid/${file}"
    obj="${TMPDIR}/minicc_${file%.mc}.o"
    bin="${TMPDIR}/minicc_${file%.mc}"

    # Compile to .o (ELF object via minicc's built-in ELF writer)
    if ! "$MINICC" -platform linux -format elf "$src" -o "$obj" 2>/dev/null; then
        echo "FAIL  $file  (compile error)"
        failed=$((failed + 1))
        continue
    fi

    # Link with minilink
    if ! "$MINILINK" "$obj" "${TMPDIR}/runtime.o" -o "$bin" 2>/dev/null; then
        echo "FAIL  $file  (link error)"
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
