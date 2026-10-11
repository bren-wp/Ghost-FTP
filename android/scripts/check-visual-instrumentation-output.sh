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
# Some API images append this legacy line; when present, a non-success
# code must not be allowed to contradict the textual OK summary.
if grep '^INSTRUMENTATION_CODE:' "$log" |
  grep -Ev '^INSTRUMENTATION_CODE:[[:space:]]*-1[[:space:]]*
  echo "Visual instrumentation's final result code was not successful." >&2
  exit 1
fi

echo "Android visual instrumentation: one test passed and adb returned success."
 |
  grep -q .; then
  echo "Visual instrumentation's final result code was not successful." >&2
  exit 1
fi

echo "Android visual instrumentation: one test passed and adb returned success."
