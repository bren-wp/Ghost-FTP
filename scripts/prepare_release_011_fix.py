#!/usr/bin/env python3
from pathlib import Path
import re

ROOT = Path(__file__).resolve().parents[1]

for path in [ROOT / "README.md", *(ROOT / "docs").glob("*.md")]:
    text = path.read_text(encoding="utf-8")
    text = re.sub(r"(?im)(\bCurrent release\s*:\s*)0\.1\.0\b", r"\g<1>0.1.1", text)
    text = re.sub(r"(?im)(\bTrenutačno izdanje\s*:\s*)0\.1\.0\b", r"\g<1>0.1.1", text)
    path.write_text(text, encoding="utf-8")

print("CURRENT_RELEASE_MARKERS=0.1.1")
