#!/usr/bin/env python3
from pathlib import Path
import shutil, subprocess

ROOT=Path(__file__).resolve().parents[1]
TEXT_EXT={'.go','.py','.md','.yml','.yaml','.ps1','.sh','.cmd','.mod','.sum','.txt','.json','.xml'}
REPL={
 'brendigo.com/byftp':'github.com/bren-wp/by-ftp',
 'info@brendigo.com':'https://github.com/bren-wp/by-ftp/issues',
 'https://brendigo.com':'https://github.com/bren-wp/by-ftp',
 'brendigo.com':'github.com/bren-wp/by-ftp',
 'Brendigo.ByFTP':'ByFTP',
 "Brendigo's":"ByFTP's",
 'Brendigo':'ByFTP',
}
for p in ROOT.rglob('*'):
    if not p.is_file() or p.name=='LICENSE' or '.git' in p.parts or p.suffix.lower() not in TEXT_EXT: continue
    try: s=p.read_text(encoding='utf-8')
    except UnicodeDecodeError: continue
    n=s
    for a,b in REPL.items(): n=n.replace(a,b)
    if n!=s: p.write_text(n,encoding='utf-8',newline='\n')

renames={
 'ARHITEKTURA.md':'ARCHITECTURE.md','DOPRINOS.md':'CONTRIBUTING.md','INSTALACIJA.md':'INSTALLATION.md',
 'IZDAVANJE-NA-GITHUBU.md':'GITHUB-RELEASES.md','OBAVIJESTI-TRECIH-STRANA.md':'THIRD-PARTY-NOTICES.md',
 'PLAN-RAZVOJA.md':'ROADMAP.md','PODRSKA.md':'SUPPORT.md','POTPISIVANJE.md':'SIGNING.md',
 'PRIVATNOST.md':'PRIVACY.md','PROVJERA-IZDANJA.md':'RELEASE-VERIFICATION.md','SIGURNOST.md':'SECURITY.md','TESTIRANJE.md':'TESTING.md'}
for old,new in renames.items():
    a=ROOT/'docs'/old; b=ROOT/'docs'/new
    if a.exists() and not b.exists(): shutil.move(a,b)
for p in [ROOT/'README.md',ROOT/'docs'/'README.md',*list((ROOT/'docs').glob('*.md'))]:
    if not p.exists(): continue
    s=p.read_text(encoding='utf-8')
    for old,new in renames.items(): s=s.replace(old,new)
    s=s.replace('docs/slike/byftp-zaglavlje.png','docs/images/byftp-header.png')
    p.write_text(s,encoding='utf-8',newline='\n')

g=ROOT/'scripts'/'generate_brand_assets.py'
s=g.read_text(encoding='utf-8')
s=s.replace('docs" / "slike" / "byftp-zaglavlje.png','docs" / "images" / "byftp-header.png')
if '"C": [' not in s:
    s=s.replace('    "B": ["11110", "10001", "10001", "11110", "10001", "10001", "11110"],', '    "B": ["11110", "10001", "10001", "11110", "10001", "10001", "11110"],\n    "C": ["01111", "10000", "10000", "10000", "10000", "10000", "01111"],')
s=s.replace('SIGURAN PRIJENOS DATOTEKA','SECURE FILE TRANSFER')
s=s.replace('Generira i provjerava službene ByFTP slikovne resurse bez vanjskih ovisnosti.','Generate and verify official ByFTP image assets without external dependencies.')
g.write_text(s,encoding='utf-8',newline='\n')
old=ROOT/'docs'/'slike'/'byftp-zaglavlje.png'
if old.exists(): old.unlink()
if old.parent.exists():
    try: old.parent.rmdir()
    except OSError: pass
subprocess.run(['python','scripts/generate_brand_assets.py'],cwd=ROOT,check=True)
subprocess.run(['gofmt','-w',*map(str,ROOT.rglob('*.go'))],cwd=ROOT,check=True)
print('MECHANICAL_CLEANUP=PASS')
