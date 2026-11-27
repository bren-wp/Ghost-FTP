#!/usr/bin/env python3
"""Smoke test a locally deployed Apache/PHP root; pass its base URL."""
import sys
import json
from pathlib import Path
import urllib.request
import urllib.error

base = sys.argv[1].rstrip("/")
version = sys.argv[2]
expected = {
    f"GhostFTP-Windows-x64-Setup-v{version}.exe",
    f"GhostFTP-Windows-x64-Setup-v{version}.msi",
    f"GhostFTP-Linux-amd64-v{version}.deb",
    f"GhostFTP-Linux-x86_64-v{version}.rpm",
}
for path, status, marker in [
    ("/", 200, "Keep Ghost FTP up to date"),
    ("/?lang=hr", 200, "Redovito ažurirajte"),
    ("/?lang[]=hr", 200, "Keep Ghost FTP up to date"),
    ("/manual-version.json", 200, version),
    ("/latest.json", 204, ""),
    ("/updates/latest.json", 204, ""),
    ("/.htaccess", 403, ""),
    ("/downloads/", 403, ""),
    ("/downloads/catalog.json", 404, ""),
    (f"/downloads/GhostFTP-Windows-x64-Portable-v{version}.exe", 404, ""),
    (f"/downloads/GhostFTP-Linux-x86_64-v{version}.AppImage", 404, ""),
    ("/assets/site.css", 200, "color-scheme"),
    ("/assets/icon.svg", 200, "<svg"),
]:
    try:
        response = urllib.request.urlopen(base + path, timeout=5)
    except urllib.error.HTTPError as error:
        response = error
    with response:
        body = response.read().decode()
        assert response.status == status, (path, response.status)
        assert marker in body, path
        if status == 204:
            assert not body and response.headers["Cache-Control"] == "no-store"
        if path == "/manual-version.json":
            assert json.loads(body) == {"version": version}
            assert response.headers["Cache-Control"] == "no-store"
        if path == "/":
            assert "default-src" in response.headers["Content-Security-Policy"]
            assert "github.com" not in body.lower() and "portable" not in body.lower() and "appimage" not in body.lower()
            assert f"v{version}" in body
            for name in expected:
                assert f'href="downloads/{name}" download' in body, name
    print(path, status, "PASS")

import hashlib
import re
sums = urllib.request.urlopen(base + "/downloads/SHA256SUMS.txt", timeout=10).read().decode()
lines = sums.splitlines()
assert len(lines) == len(expected)
seen = set()
for line in lines:
    match = re.fullmatch(r"([a-f0-9]{64})  ([A-Za-z0-9._-]+)", line)
    assert match
    name = match[2]
    assert name in expected and name not in seen, name
    seen.add(name)
    with urllib.request.urlopen(base + "/downloads/" + name, timeout=60) as response:
        assert response.geturl() == base + "/downloads/" + name, "Installer must be served locally without redirect"
        assert response.status == 200
        h = hashlib.sha256()
        for chunk in iter(lambda: response.read(1024 * 1024), b""):
            h.update(chunk)
        assert h.hexdigest() == match[1], name
    print(name, "SHA-256 PASS")
assert seen == expected

# Verify partial deployments hide the manual offer rather than expose dead links.
if len(sys.argv) > 3:
    installer = Path(sys.argv[3]) / "downloads" / sorted(expected)[0]
    moved = installer.with_suffix(installer.suffix + ".test-hidden")
    installer.rename(moved)
    try:
        with urllib.request.urlopen(base + "/manual-version.json", timeout=5) as response:
            assert response.status == 204 and not response.read()
        with urllib.request.urlopen(base + "/", timeout=5) as response:
            assert "Installation packages are temporarily unavailable" in response.read().decode()
    finally:
        moved.rename(installer)
    print("Partial deployment: PASS")

    def assert_unavailable():
        with urllib.request.urlopen(base + "/manual-version.json", timeout=5) as response:
            assert response.status == 204 and not response.read()
        with urllib.request.urlopen(base + "/", timeout=5) as response:
            page = response.read().decode()
            assert "Installation packages are temporarily unavailable" in page
            assert not any(f'href="downloads/{name}" download' in page for name in expected)

    root = Path(sys.argv[3]) / "downloads"
    # Simulate files accidentally left on the host. A 404 must be returned
    # even when the file exists, not just because the clean ZIP omits it.
    stale_version = "0.0.0" if version != "0.0.0" else "0.0.1"
    for filename in [
        f"GhostFTP-Windows-x64-Portable-v{version}.exe",
        f"GhostFTP-Linux-x86_64-v{version}.AppImage",
        f"GhostFTP-Windows-x64-Setup-v{stale_version}.exe",
        "SHA256SUMS.txt.bak",
        "backup.zip",
    ]:
        path = root / filename
        with path.open("xb") as fixture:
            fixture.write(b"synthetic stale file, never distributed")
        try:
            try:
                response = urllib.request.urlopen(base + "/downloads/" + filename, timeout=5)
            except urllib.error.HTTPError as error:
                response = error
            with response:
                assert response.status == 404, (filename, response.status)
        finally:
            path.unlink()
    print("Unexpected and stale download routes: PASS")
    for filename, replacement in [
        ("SHA256SUMS.txt", b"invalid checksum catalog\n"),
        ("SHA256SUMS.txt", (sums + sums.splitlines()[0] + "\n").encode()),
        ("SHA256SUMS.txt", sums.replace(sorted(expected)[0], "unexpected.exe").encode()),
        ("catalog.json", b"not JSON"),
        ("catalog.json", json.dumps({"version": "4294967296.0.0", "packages": {}}).encode()),
        ("SHA256SUMS.txt", sums.replace(sums[0], "1" if sums[0] != "1" else "0", 1).encode()),
        ("catalog.json", json.dumps({"version": version}).encode()),
    ]:
        path = root / filename
        original = path.read_bytes()
        try:
            path.write_bytes(replacement)
            assert_unavailable()
        finally:
            path.write_bytes(original)

    installer.rename(moved)
    try:
        # Non-empty but truncated files must not be advertised.
        installer.write_bytes(b"MZ")
        assert_unavailable()
    finally:
        installer.unlink()
        moved.rename(installer)
    with urllib.request.urlopen(base + "/manual-version.json", timeout=5) as response:
        assert response.status == 200 and json.loads(response.read()) == {"version": version}
    print("Corrupt catalog, altered checksum and truncated package: PASS")
