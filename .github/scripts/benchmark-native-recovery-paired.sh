#!/usr/bin/env bash
# Candidate-only paired native benchmark. The timed code is identical to the
# ordinary end-to-end benchmark; build and session setup are outside timings.
set -euo pipefail

baseline=${1:-899459c5e91361294fa61b4335539dc5a34f396e}
optimized=${2:-f26482d87666f302a7fe655a52b6b3a24545a75c}
[[ "$baseline" =~ ^[0-9a-f]{40}$ && "$optimized" =~ ^[0-9a-f]{40}$ ]]
[[ "$baseline" != "$optimized" ]]
repo="$(git rev-parse --show-toplevel)"
evidence="$repo/build/recovery-paired-ab"
mkdir -p "$evidence"
scratch="$(mktemp -d)"
cleanup() {
  git -C "$repo" worktree remove --force "$scratch/A" 2>/dev/null || true
  git -C "$repo" worktree remove --force "$scratch/B" 2>/dev/null || true
  rm -rf -- "$scratch"
}
trap cleanup EXIT

git fetch --no-tags origin "$baseline" "$optimized"
git worktree add --detach "$scratch/A" "$baseline"
git worktree add --detach "$scratch/B" "$optimized"
test "$(git -C "$scratch/A" rev-parse HEAD)" = "$baseline"
test "$(git -C "$scratch/B" rev-parse HEAD)" = "$optimized"
printf 'baseline=%s\noptimized=%s\norder=A B B A\n' "$baseline" "$optimized" > "$evidence/revisions.txt"
rustc --version >> "$evidence/revisions.txt"
cargo --version >> "$evidence/revisions.txt"
uname -srmo >> "$evidence/revisions.txt"

export CARGO_TARGET_DIR="$scratch/target"
for variant in A B; do
  tree="$scratch/$variant"
  mkdir -p "$tree/ghostftp-desktop/dist"
  printf '<!doctype html><html><body></body></html>\n' > "$tree/ghostftp-desktop/dist/index.html"
  (
    cd "$tree/ghostftp-desktop/src-tauri"
    cargo test --locked -p ghostftp --lib native_vault_end_to_end_benchmark --no-run
  ) 2>&1 | tee "$evidence/build-$variant.log"
done

index=0
for variant in A B B A; do
  index=$((index + 1))
  (
    export XDG_RUNTIME_DIR="$(mktemp -d "$scratch/runtime.XXXXXXXX")"
    export XDG_DATA_HOME="$(mktemp -d "$scratch/data.XXXXXXXX")"
    chmod 700 "$XDG_RUNTIME_DIR" "$XDG_DATA_HOME"
    cd "$scratch/$variant/ghostftp-desktop/src-tauri"
    dbus-run-session -- bash -euo pipefail -c '
      printf "%s" "isolated-ci-keyring" | gnome-keyring-daemon --unlock --components=secrets >/dev/null
      cargo test --locked -p ghostftp --lib native_vault_end_to_end_benchmark -- --ignored --nocapture
    '
  ) 2>&1 | tee "$evidence/run-$index-$variant.log"
done

python3 - "$evidence" <<'PY'
import pathlib
import re
import statistics
import sys

root = pathlib.Path(sys.argv[1])
pattern = re.compile(
    r"recovery_e2e_benchmark platform=linux bytes=(\d+) samples=(\d+) median_us=(\d+) max_us=(\d+)"
)
lookup = re.compile(r"recovery_linux_helper operation=lookup calls=(\d+) ")
results = {"A": {}, "B": {}}
counts = {"A": [], "B": []}
for index, variant in enumerate("ABBA", 1):
    text = (root / f"run-{index}-{variant}.log").read_text()
    matches = pattern.findall(text)
    if len(matches) != 2 or {int(m[0]) for m in matches} != {1048576, 16777216}:
        raise SystemExit(f"Missing or ambiguous benchmark evidence: {index}")
    for size, samples, median, maximum in matches:
        if int(samples) != 3 or int(median) > int(maximum):
            raise SystemExit("Invalid benchmark evidence")
        results[variant].setdefault(int(size), []).append(int(median))
    observed = [int(x) for x in lookup.findall(text)]
    if len(observed) != 6:
        raise SystemExit(f"Missing helper profile evidence: {index}")
    counts[variant].extend(observed)
lines = ["# Paired Linux Recovery benchmark", "",
         "ABBA on one runner; two invocation medians per variant and size, three promotions per invocation.",
         "Median of invocation medians is descriptive, not a pooled percentile.", "",
         "| bytes | A median-of-medians (us) | B median-of-medians (us) | change in duration (%) |",
         "|---:|---:|---:|---:|"]
for size in (1048576, 16777216):
    a = statistics.median(results["A"][size])
    b = statistics.median(results["B"][size])
    lines.append(f"| {size} | {a:.1f} | {b:.1f} | {(b-a)/a*100:.2f} |")
lines += ["", f"A lookup counts per promotion: {counts['A']}",
          f"B lookup counts per promotion: {counts['B']}",
          "Per-invocation P95/P99 and category counts are retained in raw logs; not pooled.",
          "This is not a controlled dedicated physical hardware benchmark or the 128-journal test."]
summary = "\n".join(lines) + "\n"
(root / "summary.md").write_text(summary)
print(summary)
PY
cat "$evidence/summary.md" >> "$GITHUB_STEP_SUMMARY"
