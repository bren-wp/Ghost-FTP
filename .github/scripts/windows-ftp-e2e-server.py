# Ghost FTP by Brendigo
"""Loopback-only FTP server for Windows upload/download integration tests."""

import os
from pathlib import Path

from pyftpdlib.authorizers import DummyAuthorizer
from pyftpdlib.handlers import FTPHandler
from pyftpdlib.servers import FTPServer


def main():
    root = Path(os.environ["GHOSTFTP_WINDOWS_FTP_ROOT"]).resolve(strict=True)
    if not root.is_dir():
        raise RuntimeError("FTP test home must be a directory")
    readonly = root / "ghostftp-e2e-readonly"
    readonly.mkdir(exist_ok=True)

    authorizer = DummyAuthorizer()
    username = os.environ["GHOSTFTP_E2E_USERNAME"]
    authorizer.add_user(
        username,
        os.environ["GHOSTFTP_E2E_PASSWORD"],
        str(root),
        perm="elradfmwMT",
    )
    # Enforce the denial at the FTP permission layer on Windows.
    authorizer.override_perm(username, str(readonly), perm="elr")

    class LoopbackFTPHandler(FTPHandler):
        pass

    LoopbackFTPHandler.authorizer = authorizer
    LoopbackFTPHandler.banner = "Ghost FTP Windows loopback E2E"
    LoopbackFTPHandler.passive_ports = range(32000, 32100)
    LoopbackFTPHandler.timeout = 30

    port = int(os.environ.get("GHOSTFTP_E2E_FTP_PORT", "2121"))
    if not 1024 <= port <= 65535:
        raise ValueError("Invalid unprivileged FTP test port")
    server = FTPServer(("127.0.0.1", port), LoopbackFTPHandler)
    server.serve_forever(timeout=0.25, blocking=True)


if __name__ == "__main__":
    main()
