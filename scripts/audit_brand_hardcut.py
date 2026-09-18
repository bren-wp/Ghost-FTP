#!/usr/bin/env python3
"""Fail closed on retired branding and author-identity leakage into product runtime surfaces."""
from __future__ import annotations
import re, subprocess
from pathlib import Path
ROOT=Path(__file__).resolve().parents[1]
RETIRED=re.compile(r"by[\s_-]?ftp",re.IGNORECASE)
AUTHOR_BRAND=re.compile(r"brendigo",re.IGNORECASE)
ALLOWED_AUTHOR_IDENTITY={"LICENSE","README.md","CHANGELOG.md","docs/README.md","docs/RELEASE-HISTORY.md","linux/README.md","internal/desktop/about_identity_windows.go","internal/brand/runtime_metadata_test.go","scripts/audit_brand_hardcut.py","scripts/audit_docs.py","scripts/test_about_card_release.py","scripts/test_official_destinations_contract.py","scripts/test_linux_distro_packaging_contract.py","web/legal.html","web/privacy.html"}
def fail(message:str)->None: raise SystemExit("BRAND_HARDCUT_FAILED: "+message)
def main()->int:
 raw=subprocess.check_output(["git","ls-files","-z"],cwd=ROOT);violations=[]
 for item in raw.split(b"\0"):
  if not item: continue
  rel=item.decode("utf-8","strict")
  if RETIRED.search(rel): violations.append("retired-path:"+rel);continue
  path=ROOT/rel
  if not path.is_file(): continue
  try:text=path.read_text(encoding="utf-8")
  except (UnicodeDecodeError,OSError):continue
  if RETIRED.search(text): violations.append("retired-content:"+rel)
  if rel not in ALLOWED_AUTHOR_IDENTITY and AUTHOR_BRAND.search(text): violations.append("author-identity-outside-approved-surface:"+rel)
 for rel,markers in (("internal/desktop/about_identity_windows.go",('aboutPublisher     = "BRENDIGO LTD"','aboutAuthorWebsite = "brendigo.com"','aboutSupport       = "brendigo.com/kontakt"')),):
  p=ROOT/rel
  if not p.is_file(): violations.append("missing-about-identity-source:"+rel);continue
  t=p.read_text(encoding="utf-8")
  for marker in markers:
   if marker not in t: violations.append("about-identity-contract:"+marker)
 if violations: fail("branding contract violation: "+", ".join(violations))
 print("BRAND_HARDCUT=PASS");print("PUBLIC_BRAND=Ghost FTP");print("AUTHOR_IDENTITY_SURFACES=ABOUT,LEGAL_DOCUMENTATION");print("TECHNICAL_IDENTITY=GhostFTP");return 0
if __name__=="__main__": raise SystemExit(main())
