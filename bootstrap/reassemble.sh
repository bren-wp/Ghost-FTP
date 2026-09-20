#!/usr/bin/env bash
set -euo pipefail
cat source.part.*.b64 | base64 -d > Ghost-FTP-v2.1.1-RC7-GITHUB-SOURCE.tar.xz
echo "146c5d0a467c3789c9bb87273879950ed219b1f66616357cbf5e4aed127d8ac0  Ghost-FTP-v2.1.1-RC7-GITHUB-SOURCE.tar.xz" | sha256sum -c -
tar -xJf Ghost-FTP-v2.1.1-RC7-GITHUB-SOURCE.tar.xz
