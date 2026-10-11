#!/usr/bin/env bash
# Validate the one-test Android visual capture independently of optional
# Android version-specific INSTRUMENTATION_CODE text.
set -euo pipefail

if [ "$#" -ne 2 ]; then
  echo "Usage: check-visual-instrumentation-output.sh <instrumentation.log> <adb-exit-code>" >&2
  exit 2
fi
log="$1"
code="$2"
test -s "$log" || { echo "Missing Android instrumentation result." >&2; exit 1; }
case "$code" in
  ''|*[!0-9]*) echo "Invalid Android instrumentation exit code." >&2; exit 2 ;;
esac
if [ "$code" -ne 0 ]; then
  echo "Android instrumentation exited with status $code." >&2
  exit 1
fi
if ! grep -Eq '^[[:space:]]*OK \(1 test\)[[:space:]]*$' "$log"; then
  echo "Visual instrumentation never reported exactly one passing test." >&2
  exit 1
fi
if grep -Eq '^(FAILURES!!!|INSTRUMENTATION_FAILED:|INSTRUMENTATION_ABORTED:)' "$log"; then
  echo "Visual instrumentation also reported an error or an abort." >&2
  exit 1
fi

# Some API versions print a final legacy status, others omit it. Reject
# ANY contradictory code instead of requiring a particular line to exist.
while IFS= read -r line || [ -n "$line" ]; do
  line="${line%$'\r'}"
  case "$line" in
    INSTRUMENTATION_CODE:*)
      if [ "$line" != 'INSTRUMENTATION_CODE: -1' ]; then
        echo "Visual instrumentation reported a non-success completion code." >&2
        exit 1
      fi
      ;;
  esac
done < "$log"

echo "Android visual instrumentation: one test passed and adb returned success."
