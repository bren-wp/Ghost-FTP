#!/usr/bin/env python3
"""Package the public web root with actual installer bytes and local checksums."""
import argparse
import hashlib
import json
import re
import shutil
import stat
import tempfile
from pathlib import Path
from zipfile import ZipFile, ZipInfo, ZIP_DEFLATED

ROOT = Path(__file__).resolve().parents[2] / "update.ghostftp.com"
PUBLIC = (".htaccess", "index.php", "catalog.php", "manual-version.php", "updates/index.php", "assets/site.css",
          "assets/icon.svg", "robots.txt")

def digest(path):
    with path.open("rb") as stream:
        return hashlib.file_digest(stream, "sha256").hexdigest() if hasattr(hashlib, "file_digest") else _digest(stream)

def _digest(stream):
    h = hashlib.sha256()
    for block in iter(lambda: stream.read(1024 * 1024), b""):
        h.update(block)
    return h.hexdigest()

def names(version):
    if not re.fullmatch(r"(0|[1-9]\d*)\.(0|[1-9]\d*)\.(0|[1-9]\d*)", version):
        raise ValueError("Invalid version")
    return (f"GhostFTP-Windows-x64-Setup-v{version}.exe",
            f"GhostFTP-Windows-x64-Setup-v{version}.msi",
            f"GhostFTP-Linux-amd64-v{version}.deb",
            f"GhostFTP-Linux-x86_64-v{version}.rpm")

def pack(output, version, assets, checksums):
    expected = names(version)
    records = {}
    for line in checksums.read_text(encoding="utf-8").splitlines():
        m = re.fullmatch(r"([0-9a-f]{64})  ([A-Za-z0-9._-]+)", line)
        if not m or m[2] in records:
            raise ValueError("Invalid or duplicate checksum")
        records[m[2]] = m[1]
    output.parent.mkdir(parents=True, exist_ok=True)
    with tempfile.TemporaryDirectory(dir=output.parent) as tmp:
        staged = Path(tmp)
        local = staged / "downloads"
        local.mkdir()
        for name, magic in zip(expected, (b"MZ", bytes.fromhex("d0cf11e0a1b11ae1"), b"!<arch>\n", bytes.fromhex("edabeedb"))):
            source = assets / name
            if source.is_symlink() or not source.is_file() or source.stat().st_size <= len(magic):
                raise ValueError("Missing or unsafe installer: " + name)
            with source.open("rb") as stream:
                if stream.read(len(magic)) != magic:
                    raise ValueError("Invalid package format: " + name)
            dest = local / name
            shutil.copyfile(source, dest)
            if digest(dest) != records.get(name):
                raise ValueError("Installer SHA-256 mismatch: " + name)
        metadata = {name: {"size": (local / name).stat().st_size, "sha256": records[name]}
                    for name in expected}
        (local / "catalog.json").write_text(
            json.dumps({"version": version, "packages": metadata}, sort_keys=True) + "\n",
            encoding="utf-8")
        (local / "SHA256SUMS.txt").write_text("".join(f"{records[n]}  {n}\n" for n in expected), encoding="utf-8")
        archive_path = staged / "root.zip"
        # Render the exact release allowlist; unlisted files left behind by a
        # non-atomic deployment must not become public downloads.
        htaccess = (ROOT / ".htaccess").read_text(encoding="utf-8")
        marker = "__GHOSTFTP_RELEASE_INSTALLERS__"
        if htaccess.count(marker) != 1:
            raise ValueError("Invalid installer allowlist template")
        rendered = staged / ".htaccess"
        rendered.write_text(htaccess.replace(marker, "|".join(re.escape(name) for name in expected)),
                            encoding="utf-8")
        entries = [(name, rendered if name == ".htaccess" else ROOT / name) for name in PUBLIC]
        entries += [("downloads/" + p.name, p) for p in local.iterdir()]
        with ZipFile(archive_path, "w", compression=ZIP_DEFLATED) as archive:
            for name, source in sorted(entries):
                info = ZipInfo(name, (1980, 1, 1, 0, 0, 0))
                info.compress_type = ZIP_DEFLATED
                info.create_system = 3
                info.external_attr = (stat.S_IFREG | 0o644) << 16
                with source.open("rb") as incoming, archive.open(info, "w") as outgoing:
                    shutil.copyfileobj(incoming, outgoing, 1024 * 1024)
        with ZipFile(archive_path) as archive:
            if set(archive.namelist()) != {name for name, _ in entries} or archive.testzip() is not None:
                raise ValueError("Invalid deployment archive")
        archive_path.replace(output)

def main():
    parser = argparse.ArgumentParser(description=__doc__)
    parser.add_argument("output", type=Path)
    parser.add_argument("--version", required=True)
    parser.add_argument("--assets", required=True, type=Path)
    parser.add_argument("--checksums", required=True, type=Path)
    args = parser.parse_args()
    pack(args.output, args.version, args.assets, args.checksums)

if __name__ == "__main__":
    main()
