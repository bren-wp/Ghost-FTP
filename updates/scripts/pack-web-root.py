#!/usr/bin/env python3
"""Build a bootstrap deployment ZIP with only explicitly public files."""
import argparse
from pathlib import Path
from zipfile import ZipFile, ZipInfo, ZIP_DEFLATED

ROOT = Path(__file__).resolve().parents[2] / "update.ghostftp.com"
PUBLIC = (".htaccess", "index.php", "updates/index.php", "assets/site.css",
          "assets/icon.svg", "robots.txt")

def main():
    parser = argparse.ArgumentParser(description=__doc__)
    parser.add_argument("output", type=Path)
    args = parser.parse_args()
    args.output.parent.mkdir(parents=True, exist_ok=True)
    with ZipFile(args.output, "w", compression=ZIP_DEFLATED) as archive:
        for name in sorted(PUBLIC):
            entry = ZipInfo(name, (1980, 1, 1, 0, 0, 0))
            entry.compress_type = ZIP_DEFLATED
            entry.create_system = 3
            entry.external_attr = 0o100644 << 16
            archive.writestr(entry, (ROOT / name).read_bytes())
    with ZipFile(args.output) as archive:
        assert set(archive.namelist()) == set(PUBLIC)
        assert archive.testzip() is None

if __name__ == "__main__":
    main()
