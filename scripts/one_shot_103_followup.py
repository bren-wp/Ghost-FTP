#!/usr/bin/env python3
from pathlib import Path

path = Path('GhostFTP WEB/tests/zip-extraction-preflight.php')
text = path.read_text(encoding='utf-8')
old = "$executionPos = strpos($extractSource, '// Only a fully validated archive is allowed to mutate remote state.');"
new = "$executionPos = strpos($extractSource, '// Only a fully validated and materialized archive is allowed to mutate remote state.');"
if text.count(old) != 1:
    raise SystemExit(f'FOLLOWUP_PATCH_FAILED: expected one legacy execution marker, got {text.count(old)}')
path.write_text(text.replace(old, new, 1), encoding='utf-8', newline='\n')
print('ZIP_PREFLIGHT_TEST_MARKER=UPDATED')
