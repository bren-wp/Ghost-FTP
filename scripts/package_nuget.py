#!/usr/bin/env python3
"""Create and validate the GhostFTP NuGet package from verified portable binaries."""

from __future__ import annotations

import argparse
import hashlib
import re
import stat
import sys
import zipfile
from pathlib import Path
from xml.etree import ElementTree as ET
from xml.sax.saxutils import escape

ROOT = Path(__file__).resolve().parents[1]
PACKAGE_ID = "GhostFTP"
AUTHOR = "Brendigo"
REPOSITORY_URL = "https://github.com/bren-wp/Ghost-FTP"


def fail(message: str) -> None:
    raise SystemExit("NUGET_PACKAGE_FAILED: " + message)


def canonical_version() -> str:
    version = (ROOT / "VERSION").read_text(encoding="utf-8").strip()
    if not re.fullmatch(r"\d+\.\d+\.\d+", version):
        fail(f"invalid VERSION: {version!r}")
    return version


def sha256(path: Path) -> str:
    digest = hashlib.sha256()
    with path.open("rb") as handle:
        for chunk in iter(lambda: handle.read(1024 * 1024), b""):
            digest.update(chunk)
    return digest.hexdigest()


def validate_portable(path: Path, architecture: str) -> bytes:
    if not path.is_file():
        fail(f"missing {architecture} portable binary: {path}")
    data = path.read_bytes()
    if len(data) < 4096 or data[:2] != b"MZ":
        fail(f"{architecture} portable input is not a valid PE executable")
    return data


def write_entry(archive: zipfile.ZipFile, name: str, data: bytes, executable: bool = False) -> None:
    info = zipfile.ZipInfo(name, date_time=(1980, 1, 1, 0, 0, 0))
    info.create_system = 3
    info.compress_type = zipfile.ZIP_DEFLATED
    mode = 0o755 if executable else 0o644
    info.external_attr = (stat.S_IFREG | mode) << 16
    archive.writestr(info, data)


def nuspec(version: str) -> bytes:
    text = f'''<?xml version="1.0" encoding="utf-8"?>
<package xmlns="http://schemas.microsoft.com/packaging/2013/05/nuspec.xsd">
  <metadata>
    <id>{PACKAGE_ID}</id>
    <version>{escape(version)}</version>
    <authors>{AUTHOR}</authors>
    <owners>{AUTHOR}</owners>
    <requireLicenseAcceptance>false</requireLicenseAcceptance>
    <license type="file">LICENSE.txt</license>
    <projectUrl>https://ghostftp.com</projectUrl>
    <description>Ghost FTP portable Windows client binaries for x64 and x86.</description>
    <tags>ghostftp ftp sftp windows portable</tags>
    <repository type="git" url="{REPOSITORY_URL}" />
  </metadata>
</package>
'''
    return text.encode("utf-8")


def content_types() -> bytes:
    return b'''<?xml version="1.0" encoding="utf-8"?>
<Types xmlns="http://schemas.openxmlformats.org/package/2006/content-types">
  <Default Extension="rels" ContentType="application/vnd.openxmlformats-package.relationships+xml" />
  <Default Extension="psmdcp" ContentType="application/vnd.openxmlformats-package.core-properties+xml" />
  <Default Extension="exe" ContentType="application/octet-stream" />
  <Default Extension="txt" ContentType="text/plain" />
  <Default Extension="nuspec" ContentType="application/octet" />
</Types>
'''


def relationships() -> bytes:
    return b'''<?xml version="1.0" encoding="utf-8"?>
<Relationships xmlns="http://schemas.openxmlformats.org/package/2006/relationships">
  <Relationship Type="http://schemas.microsoft.com/packaging/2010/07/manifest" Target="/GhostFTP.nuspec" Id="R1" />
  <Relationship Type="http://schemas.openxmlformats.org/package/2006/relationships/metadata/core-properties" Target="/package/services/metadata/core-properties/ghostftp.psmdcp" Id="R2" />
</Relationships>
'''


def core_properties(version: str) -> bytes:
    text = f'''<?xml version="1.0" encoding="utf-8"?>
<coreProperties xmlns="http://schemas.openxmlformats.org/package/2006/metadata/core-properties"
 xmlns:dc="http://purl.org/dc/elements/1.1/"
 xmlns:dcterms="http://purl.org/dc/terms/"
 xmlns:xsi="http://www.w3.org/2001/XMLSchema-instance">
  <dc:creator>{AUTHOR}</dc:creator>
  <dc:description>Ghost FTP portable Windows client binaries.</dc:description>
  <dc:identifier>{PACKAGE_ID}.{escape(version)}</dc:identifier>
  <version>{escape(version)}</version>
  <keywords>ghostftp ftp sftp windows portable</keywords>
</coreProperties>
'''
    return text.encode("utf-8")


def validate_package(path: Path, version: str, x64_hash: str, x86_hash: str) -> None:
    if not path.is_file() or path.stat().st_size == 0:
        fail("NuGet package was not created")
    with zipfile.ZipFile(path) as archive:
        names = archive.namelist()
        if len(names) != len(set(names)):
            fail("NuGet package contains duplicate paths")
        expected = {
            "[Content_Types].xml",
            "_rels/.rels",
            "GhostFTP.nuspec",
            "LICENSE.txt",
            "package/services/metadata/core-properties/ghostftp.psmdcp",
            "tools/win-x64/GhostFTP.exe",
            "tools/win-x86/GhostFTP.exe",
            "tools/SHA256.txt",
        }
        if set(names) != expected:
            fail("NuGet package file set does not match the contract")
        for name in names:
            if name.startswith("/") or "../" in name or name.endswith("/.."):
                fail(f"unsafe NuGet path: {name}")
        try:
            root = ET.fromstring(archive.read("GhostFTP.nuspec"))
        except ET.ParseError as exc:
            fail(f"invalid nuspec XML: {exc}")
        namespace = {"n": "http://schemas.microsoft.com/packaging/2013/05/nuspec.xsd"}
        metadata = root.find("n:metadata", namespace)
        if metadata is None:
            fail("nuspec metadata is missing")
        if metadata.findtext("n:id", namespaces=namespace) != PACKAGE_ID:
            fail("nuspec package ID is not GhostFTP")
        if metadata.findtext("n:version", namespaces=namespace) != version:
            fail("nuspec version does not match VERSION")
        if hashlib.sha256(archive.read("tools/win-x64/GhostFTP.exe")).hexdigest() != x64_hash:
            fail("x64 portable binary changed during packaging")
        if hashlib.sha256(archive.read("tools/win-x86/GhostFTP.exe")).hexdigest() != x86_hash:
            fail("x86 portable binary changed during packaging")


def main() -> int:
    parser = argparse.ArgumentParser()
    parser.add_argument("--x64", required=True, type=Path)
    parser.add_argument("--x86", required=True, type=Path)
    parser.add_argument("--output-dir", required=True, type=Path)
    args = parser.parse_args()

    version = canonical_version()
    x64_data = validate_portable(args.x64, "x64")
    x86_data = validate_portable(args.x86, "x86")
    x64_hash = hashlib.sha256(x64_data).hexdigest()
    x86_hash = hashlib.sha256(x86_data).hexdigest()
    license_data = (ROOT / "LICENSE").read_bytes()
    if not license_data:
        fail("LICENSE is empty")

    output_dir = args.output_dir
    output_dir.mkdir(parents=True, exist_ok=True)
    output = output_dir / f"{PACKAGE_ID}.{version}.nupkg"
    output.unlink(missing_ok=True)
    manifest = (
        f"{x64_hash}  win-x64/GhostFTP.exe\n"
        f"{x86_hash}  win-x86/GhostFTP.exe\n"
    ).encode("ascii")

    with zipfile.ZipFile(output, "w", compression=zipfile.ZIP_DEFLATED, compresslevel=9) as archive:
        write_entry(archive, "[Content_Types].xml", content_types())
        write_entry(archive, "_rels/.rels", relationships())
        write_entry(archive, "GhostFTP.nuspec", nuspec(version))
        write_entry(archive, "LICENSE.txt", license_data)
        write_entry(
            archive,
            "package/services/metadata/core-properties/ghostftp.psmdcp",
            core_properties(version),
        )
        write_entry(archive, "tools/win-x64/GhostFTP.exe", x64_data, executable=True)
        write_entry(archive, "tools/win-x86/GhostFTP.exe", x86_data, executable=True)
        write_entry(archive, "tools/SHA256.txt", manifest)

    validate_package(output, version, x64_hash, x86_hash)
    print(f"NUGET_PACKAGE=PASS ({PACKAGE_ID} {version})")
    print(f"NUPKG={output.name} SHA256={sha256(output)}")
    return 0


if __name__ == "__main__":
    sys.exit(main())
