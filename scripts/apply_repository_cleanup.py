#!/usr/bin/env python3
"""One-time deterministic repository normalization for ByFTP 1.0.12."""

from pathlib import Path
import shutil
import subprocess

ROOT = Path(__file__).resolve().parents[1]
TEXT_EXT = {'.go', '.py', '.md', '.yml', '.yaml', '.ps1', '.sh', '.cmd', '.mod', '.sum', '.txt', '.json', '.xml'}
REPL = {
    'brendigo.com/byftp': 'github.com/bren-wp/by-ftp',
    'info@brendigo.com': 'https://github.com/bren-wp/by-ftp/issues',
    'https://brendigo.com': 'https://github.com/bren-wp/by-ftp',
    'brendigo.com': 'github.com/bren-wp/by-ftp',
    'Brendigo.ByFTP': 'ByFTP',
    "Brendigo's": "ByFTP's",
    'Brendigo': 'ByFTP',
}

for path in ROOT.rglob('*'):
    if (
        not path.is_file()
        or path.name == 'LICENSE'
        or '.git' in path.parts
        or path.suffix.lower() not in TEXT_EXT
    ):
        continue
    try:
        current = path.read_text(encoding='utf-8')
    except UnicodeDecodeError:
        continue
    updated = current
    for old, new in REPL.items():
        updated = updated.replace(old, new)
    if updated != current:
        path.write_text(updated, encoding='utf-8', newline='\n')

# VERSION is the single production version source. 1.0.12 already has a
# changelog section and is the release line documented by this cleanup.
(ROOT / 'VERSION').write_text('1.0.12\n', encoding='utf-8', newline='\n')

renames = {
    'ARHITEKTURA.md': 'ARCHITECTURE.md',
    'DOPRINOS.md': 'CONTRIBUTING.md',
    'INSTALACIJA.md': 'INSTALLATION.md',
    'IZDAVANJE-NA-GITHUBU.md': 'GITHUB-RELEASES.md',
    'OBAVIJESTI-TRECIH-STRANA.md': 'THIRD-PARTY-NOTICES.md',
    'PLAN-RAZVOJA.md': 'ROADMAP.md',
    'PODRSKA.md': 'SUPPORT.md',
    'POTPISIVANJE.md': 'SIGNING.md',
    'PRIVATNOST.md': 'PRIVACY.md',
    'PROVJERA-IZDANJA.md': 'RELEASE-VERIFICATION.md',
    'SIGURNOST.md': 'SECURITY.md',
    'TESTIRANJE.md': 'TESTING.md',
}
for old, new in renames.items():
    source = ROOT / 'docs' / old
    target = ROOT / 'docs' / new
    if source.exists() and not target.exists():
        shutil.move(source, target)

for path in [ROOT / 'README.md', ROOT / 'docs' / 'README.md', *list((ROOT / 'docs').glob('*.md'))]:
    if not path.exists():
        continue
    text = path.read_text(encoding='utf-8')
    for old, new in renames.items():
        text = text.replace(old, new)
    text = text.replace('docs/slike/byftp-zaglavlje.png', 'docs/images/byftp-header.png')
    path.write_text(text, encoding='utf-8', newline='\n')

generator = ROOT / 'scripts' / 'generate_brand_assets.py'
if generator.exists():
    text = generator.read_text(encoding='utf-8')
    text = text.replace('docs" / "slike" / "byftp-zaglavlje.png', 'docs" / "images" / "byftp-header.png')
    if '"C": [' not in text:
        text = text.replace(
            '    "B": ["11110", "10001", "10001", "11110", "10001", "10001", "11110"],',
            '    "B": ["11110", "10001", "10001", "11110", "10001", "10001", "11110"],\n'
            '    "C": ["01111", "10000", "10000", "10000", "10000", "10000", "01111"],',
        )
    text = text.replace('SIGURAN PRIJENOS DATOTEKA', 'SECURE FILE TRANSFER')
    text = text.replace(
        'Generira i provjerava službene ByFTP slikovne resurse bez vanjskih ovisnosti.',
        'Generate and verify official ByFTP image assets without external dependencies.',
    )
    generator.write_text(text, encoding='utf-8', newline='\n')

old_header = ROOT / 'docs' / 'slike' / 'byftp-zaglavlje.png'
if old_header.exists():
    old_header.unlink()
if old_header.parent.exists():
    try:
        old_header.parent.rmdir()
    except OSError:
        pass

if generator.exists():
    subprocess.run(['python', 'scripts/generate_brand_assets.py'], cwd=ROOT, check=True)

go_files = [str(path) for path in ROOT.rglob('*.go')]
if go_files:
    subprocess.run(['gofmt', '-w', *go_files], cwd=ROOT, check=True)

# Fail the cleanup if the legacy product/vendor token remains in tracked text
# outside the legal license attribution. This is case-insensitive and catches
# comments, documentation, import paths and UI strings alike.
legacy = 'brendigo'
remaining = []
for path in ROOT.rglob('*'):
    if (
        not path.is_file()
        or path.name == 'LICENSE'
        or '.git' in path.parts
        or path.suffix.lower() not in TEXT_EXT
    ):
        continue
    try:
        text = path.read_text(encoding='utf-8')
    except UnicodeDecodeError:
        continue
    if legacy in text.lower():
        remaining.append(str(path.relative_to(ROOT)))
if remaining:
    raise SystemExit('Legacy branding remains outside LICENSE: ' + ', '.join(remaining))

print('MECHANICAL_CLEANUP=PASS')
