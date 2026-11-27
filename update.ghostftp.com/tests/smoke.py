#!/usr/bin/env python3
"""Smoke test a locally deployed Apache/PHP root; pass its base URL."""
import sys
import urllib.request
import urllib.error

base = sys.argv[1].rstrip("/")
version = sys.argv[2]
for path, status, marker in [
    ("/", 200, "Keep Ghost FTP up to date"),
    ("/?lang=hr", 200, "Redovito ažurirajte"),
    ("/?lang[]=hr", 200, "Keep Ghost FTP up to date"),
    ("/latest.json", 204, ""),
    ("/updates/latest.json", 204, ""),
    ("/.htaccess", 403, ""),
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
        if path == "/":
            assert "default-src" in response.headers["Content-Security-Policy"]
            assert "github.com" not in body.lower() and "portable" not in body.lower() and "appimage" not in body.lower()
            assert f"v{version}" in body
            assert "downloads/" in body
    print(path, status, "PASS")

import hashlib
import re
sums = urllib.request.urlopen(base + "/downloads/SHA256SUMS.txt", timeout=10).read().decode()
lines = sums.splitlines()
assert len(lines) == 4
for line in lines:
    match = re.fullmatch(r"([a-f0-9]{64})  ([A-Za-z0-9._-]+)", line)
    assert match
    name = match[2]
    assert version in name and "portable" not in name.lower()
    with urllib.request.urlopen(base + "/downloads/" + name, timeout=60) as response:
        h = hashlib.sha256()
        for chunk in iter(lambda: response.read(1024 * 1024), b""):
            h.update(chunk)
        assert h.hexdigest() == match[1], name
    print(name, "SHA-256 PASS")
