#!/usr/bin/env bash
# Offline regression cases for the actual emulator output validator.
set -euo pipefail

checker="$(dirname "$0")/check-visual-instrumentation-output.sh"
temp="$(mktemp -d)"
trap 'rm -rf "$temp"' EXIT
log="$temp/instrumentation.log"

expect_success() {
  if ! bash "$checker" "$log" "$1" > "$temp/check.log" 2>&1; then
    echo "Expected valid instrumentation result, got failure:" >&2
    cat "$log" "$temp/check.log" >&2
    exit 1
  fi
}
expect_failure() {
  if bash "$checker" "$log" "$1" > "$temp/check.log" 2>&1; then
    echo "Expected the invalid Android instrumentation result to fail." >&2
    cat "$log" "$temp/check.log" >&2
    exit 1
  fi
}

# Real API 35 runner log: OK appears with NO INSTRUMENTATION_CODE line.
printf 'com.ghostftp.android.PremiumVisualReferenceTest:.\n\nTime: 23.298\n\nOK (1 test)\n' > "$log"
expect_success 0

# Other Android API versions still emit the optional successful final code.
printf 'Time: 5.000\n\nOK (1 test)\nINSTRUMENTATION_CODE: -1\n' > "$log"
expect_success 0
expect_failure 1

printf 'Time: 5.000\n\nFAILURES!!!\nTests run: 1, Failures: 1\n' > "$log"
expect_failure 0

printf 'OK (1 test)\nINSTRUMENTATION_CODE: 0\n' > "$log"
expect_failure 0

printf 'OK (1 test)\nINSTRUMENTATION_CODE: -1\nINSTRUMENTATION_CODE: 0\n' > "$log"
expect_failure 0

printf 'OK (2 tests)\n' > "$log"
expect_failure 0

printf 'INSTRUMENTATION_ABORTED: process crash\nOK (1 test)\n' > "$log"
expect_failure 0

echo "Android visual instrumentation result contract: all positive/negative fixtures passed."
