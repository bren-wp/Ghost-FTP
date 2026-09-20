#!/usr/bin/env bash
set -euo pipefail
cat source.part.*.b64 | base64 -d > Ghost-FTP-v2.1.1-RC6-GITHUB-SOURCE-TEXT.tar.xz
echo "8291c7a62d446be3728a4e805974a5aa78f00e96eb07512530ac0cece7f4cb0e  Ghost-FTP-v2.1.1-RC6-GITHUB-SOURCE-TEXT.tar.xz" | sha256sum -c -
tar -xJf Ghost-FTP-v2.1.1-RC6-GITHUB-SOURCE-TEXT.tar.xz
