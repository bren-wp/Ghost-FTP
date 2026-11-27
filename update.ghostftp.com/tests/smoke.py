#!/usr/bin/env python3
"""Smoke test a locally deployed Apache/PHP root; pass its base URL."""
import sys
import urllib.request
import urllib.error

base = sys.argv[1].rstrip("/")
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
    print(path, status, "PASS")
