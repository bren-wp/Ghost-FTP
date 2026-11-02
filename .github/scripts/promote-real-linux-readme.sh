#!/usr/bin/env bash
# Only promote checked, genuinely rendered native Linux screenshots.
set -euo pipefail
source_dir="$1"
run_id="$2"
source_sha="$3"
test -n "$source_dir"
[[ "$run_id" =~ ^[0-9]+$ ]]
[[ "$source_sha" =~ ^[a-f0-9]{40}$ ]]
mkdir -p docs/assets/screenshots/native-linux
test -f README.md
# Prevent a self-triggering CI commit loop; later source modifications require a
# distinct review for screenshot re-promotion.
if grep -Fq '<!-- BEGIN:real-linux-native-captures -->' README.md; then
  echo 'Genuine native Linux gallery already tracked in README.'
  exit 0
fi
for item in main:Files siteManager:Sites newConnection:New-Connection transferCenter:Transfers settings:Settings sync:Sync about:About; do
  IFS=: read -r view label <<< "$item"
  source_path="$source_dir/$view"
  if [ "$view" = "main" ]; then
    source_path="$source_dir"
  fi
  filename="Linux-$label-1290x852.png"
  test -s "$source_path/$filename"
  test -s "$source_path/SHA256SUMS.txt"
  actual="$(sha256sum "$source_path/$filename" | cut -d' ' -f1)"
  expected="$(cut -d' ' -f1 < "$source_path/SHA256SUMS.txt")"
  test "$actual" = "$expected"
  label_lower="$(printf '%s' "$label" | tr '[:upper:]' '[:lower:]')"
  cp "$source_path/$filename" "docs/assets/screenshots/native-linux/ghostftp-linux-$label_lower.png"
done
{
  printf '\n%s\n\n' '<!-- BEGIN:real-linux-native-captures -->'
  printf '### Linux — seven real native Ghost FTP screens\n\n'
  printf '<table>\n'
  row_index=0
  for label in Files Sites New-Connection Transfers Settings Sync About; do
    label_lower="$(printf '%s' "$label" | tr '[:upper:]' '[:lower:]')"
    src="docs/assets/screenshots/native-linux/ghostftp-linux-$label_lower.png"
    if [ $((row_index % 2)) -eq 0 ]; then printf '<tr>\n'; fi
    printf '<td align="center"><a href="%s"><img src="%s" alt="Genuine Linux Ghost FTP %s window" width="390" /></a><br /><strong>%s</strong></td>\n' "$src" "$src" "$label" "$label"
    row_index=$((row_index + 1))
    if [ $((row_index % 2)) -eq 0 ]; then printf '</tr>\n'; fi
  done
  if [ $((row_index % 2)) -ne 0 ]; then printf '</tr>\n'; fi
  printf '</table>\n'
  printf '<p>Verified <strong>real Linux Tauri/WebKit UI</strong> from <a href="https://github.com/bren-wp/Ghost-FTP/actions/runs/%s">GitHub Actions native build %s</a>; source SHA <code>%s</code>. Captured at 1290×852 in Xvfb, no fabricated FTP connections, no design mockups. Native capture does not establish pixel-perfect 1:1 design parity.</p>\n' "$run_id" "$run_id" "$source_sha"
  printf '\n%s\n' '<!-- END:real-linux-native-captures -->'
} >> README.md
echo "Promoted seven authentic Linux application screenshots for review: $source_sha"
