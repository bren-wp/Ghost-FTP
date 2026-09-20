#!/usr/bin/env bash
set -euo pipefail
cat source.part.*.b64 | base64 -d > Ghost-FTP-v2.1.1-RC6-GITHUB-SOURCE.tar.xz
echo "65fa0cab91803ee82b418a2cc17371c241f750567b550747cfb5ffb79059d098  Ghost-FTP-v2.1.1-RC6-GITHUB-SOURCE.tar.xz" | sha256sum -c -
tar -xJf Ghost-FTP-v2.1.1-RC6-GITHUB-SOURCE.tar.xz
