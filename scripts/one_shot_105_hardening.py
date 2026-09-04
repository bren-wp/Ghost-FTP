#!/usr/bin/env python3
from pathlib import Path

ROOT = Path(__file__).resolve().parents[1]


def read(path: str) -> str:
    return (ROOT / path).read_text(encoding="utf-8")


def write(path: str, text: str) -> None:
    (ROOT / path).write_text(text, encoding="utf-8", newline="\n")


def replace_once(path: str, old: str, new: str) -> None:
    text = read(path)
    count = text.count(old)
    if count != 1:
        raise SystemExit(f"PATCH_FAILED: {path}: expected one match, got {count}")
    write(path, text.replace(old, new, 1))


def replace_all_checked(path: str, old: str, new: str, expected: int) -> None:
    text = read(path)
    count = text.count(old)
    if count != expected:
        raise SystemExit(f"PATCH_FAILED: {path}: expected {expected} matches, got {count}")
    write(path, text.replace(old, new))


# Canonical version surfaces.
replace_once("VERSION", "1.0.4\n", "1.0.5\n")
replace_once("GhostFTP WEB/VERSION", "1.0.4\n", "1.0.5\n")
replace_once('GhostFTP WEB/composer.json', '"version": "1.0.4"', '"version": "1.0.5"')
replace_once("GhostFTP WEB/service-worker.js", "ghostftp-static-v1.0.4", "ghostftp-static-v1.0.5")

# Prevent Throwable -> RuntimeException wrappers from smuggling raw internal
# messages through the intentionally user-readable RuntimeException boundary.
replace_once(
    "GhostFTP WEB/app/Operations/RemoteOperations.php",
    "throw new RuntimeException('Stavka je kopirana na odredište, ali izvor nije moguće obrisati: ' . $deleteError->getMessage(), 0, $deleteError);",
    "throw new RuntimeException('Stavka je kopirana na odredište, ali izvor nije moguće obrisati. Izvor je ostavljen radi sigurnog ponavljanja operacije.', 0, $deleteError);",
)
replace_once(
    "GhostFTP WEB/app/Storage/UserStore.php",
    """            throw new RuntimeException(
                'Korisnički račun je deaktiviran, ali workspace nije moguće u potpunosti obrisati. Provjeri dozvole i ponovi brisanje: ' . $e->getMessage(),
                0,
                $e
            );
""",
    """            throw new RuntimeException(
                'Korisnički račun je deaktiviran, ali workspace nije moguće u potpunosti obrisati. Provjeri dozvole i ponovi brisanje.',
                0,
                $e
            );
""",
)

# Extend the 1.0.4 public-error boundary to HTML form surfaces. Raw exception
# details stay in the protected server log together with the exception class.
html_catches = {
    "GhostFTP WEB/account.php": (
        """        } catch (Throwable $e) {
            $error = $e->getMessage();
        }
""",
        """        } catch (Throwable $e) {
            AppLogger::event('account.error', ['user_id' => Auth::id(), 'exception' => get_class($e), 'error' => GhostFTP_truncate($e->getMessage(), 300)]);
            $error = GhostFTP_public_error($e);
        }
""",
    ),
    "GhostFTP WEB/register.php": (
        """            } catch (Throwable $e) {
                $error = $e->getMessage();
            }
""",
        """            } catch (Throwable $e) {
                AppLogger::event('auth.registration_failed', ['exception' => get_class($e), 'error' => GhostFTP_truncate($e->getMessage(), 300)]);
                $error = GhostFTP_public_error($e);
            }
""",
    ),
    "GhostFTP WEB/users.php": (
        """        } catch (Throwable $e) {
            $error = $e->getMessage();
        }
""",
        """        } catch (Throwable $e) {
            AppLogger::event('admin.user_action_failed', ['user_id' => Auth::id(), 'action' => $action ?? '', 'exception' => get_class($e), 'error' => GhostFTP_truncate($e->getMessage(), 300)]);
            $error = GhostFTP_public_error($e);
        }
""",
    ),
    "GhostFTP WEB/settings.php": (
        """        } catch (Throwable $e) {
            $error = $e->getMessage();
        }
""",
        """        } catch (Throwable $e) {
            AppLogger::event('admin.settings_update_failed', ['user_id' => Auth::id(), 'exception' => get_class($e), 'error' => GhostFTP_truncate($e->getMessage(), 300)]);
            $error = GhostFTP_public_error($e);
        }
""",
    ),
}
for path, (old, new) in html_catches.items():
    replace_once(path, old, new)

replace_once(
    "GhostFTP WEB/login.php",
    """                } catch (Throwable $e) {
                    $error = $e->getMessage();
                }
""",
    """                } catch (Throwable $e) {
                    AppLogger::event('auth.legacy_setup_failed', ['exception' => get_class($e), 'error' => GhostFTP_truncate($e->getMessage(), 300)]);
                    $error = GhostFTP_public_error($e);
                }
""",
)
replace_once(
    "GhostFTP WEB/setup.php",
    """                $error = $e->getMessage();
            } finally {
""",
    """                AppLogger::event('install.failed', ['transaction_started' => $setupTransactionStarted, 'exception' => get_class($e), 'error' => GhostFTP_truncate($e->getMessage(), 300)]);
                $error = GhostFTP_public_error($e);
            } finally {
""",
)

# The transport-level write() API has no call site. All real editor/new-file
# writes already go through RemoteOperations::writeAtomic(), which has bounded
# staging, exact local write verification and atomic remote promotion. Remove
# the duplicate weaker contract and keep the read default aligned at 4 MiB.
replace_once(
    "GhostFTP WEB/app/Remote/RemoteClientInterface.php",
    """    public function read(string $remotePath, int $maxBytes = 2097152): string;
    public function write(string $remotePath, string $content): void;
""",
    """    public function read(string $remotePath, int $maxBytes = 4194304): string;
""",
)
replace_once(
    "GhostFTP WEB/app/Remote/FtpClient.php",
    """    public function write(string $remotePath, string $content): void
    {
        $tmp = tempnam(GhostFTP_STORAGE . '/tmp', 'write-');
        if ($tmp === false) throw new RuntimeException('Ne mogu stvoriti privremenu datoteku.');
        try {
            if (file_put_contents($tmp, $content, LOCK_EX) === false) throw new RuntimeException('Ne mogu pripremiti sadržaj za spremanje.');
            $this->upload($tmp, $remotePath);
        } finally {
            @unlink($tmp);
        }
    }

""",
    "",
)
replace_once(
    "GhostFTP WEB/app/Remote/SftpClient.php",
    """    public function write(string $remotePath, string $content): void
    {
        $this->ensureConnected();
        $fp = @fopen($this->uri($this->full($remotePath)), 'wb');
        if (!is_resource($fp)) throw new RuntimeException('Ne mogu otvoriti datoteku za spremanje.');
        $remaining = $content;
        while ($remaining !== '') {
            $written = fwrite($fp, $remaining);
            if ($written === false || $written === 0) {
                fclose($fp);
                throw new RuntimeException('Spremanje nije uspjelo.');
            }
            $remaining = substr($remaining, $written);
        }
        fclose($fp);
    }

""",
    "",
)
replace_once(
    "GhostFTP WEB/tests/unit.php",
    """    public function read(string $remotePath, int $maxBytes = 2097152): string
    {
        throw new RuntimeException('Unexpected read in batch rename test.');
    }

    public function write(string $remotePath, string $content): void
    {
        throw new RuntimeException('Unexpected write in batch rename test.');
    }

""",
    """    public function read(string $remotePath, int $maxBytes = 4194304): string
    {
        throw new RuntimeException('Unexpected read in batch rename test.');
    }

""",
)
replace_once(
    "GhostFTP WEB/tests/zip-extraction-preflight.php",
    """    public function read(string $remotePath, int $maxBytes = 2097152): string
    {
        throw new RuntimeException('Unexpected read during ZIP extraction preflight test.');
    }

    public function write(string $remotePath, string $content): void
    {
        $this->mutations[] = 'write:' . $remotePath;
    }

""",
    """    public function read(string $remotePath, int $maxBytes = 4194304): string
    {
        throw new RuntimeException('Unexpected read during ZIP extraction preflight test.');
    }

""",
)

# Canonical public Ghost FTP branding. Internal PHP namespace/functions,
# package IDs and compatibility identifiers intentionally remain GhostFTP.
visible_replacements = {
    "GhostFTP WEB/account.php": [
        ("<h1>Moj GhostFTP</h1>", "<h1>Moj Ghost FTP</h1>"),
    ],
    "GhostFTP WEB/index.php": [
        ('alt="GhostFTP"', 'alt="Ghost FTP"'),
        (">↓ Instaliraj GhostFTP</button>", ">↓ Instaliraj Ghost FTP</button>"),
        ("<h2>Instaliraj GhostFTP</h2>", "<h2>Instaliraj Ghost FTP</h2>"),
        ("<img src=\"<?= GhostFTP_e(GhostFTP_asset('images/mark.svg')) ?>\" alt=\"GhostFTP\">", "<img src=\"<?= GhostFTP_e(GhostFTP_asset('images/mark.svg')) ?>\" alt=\"Ghost FTP\">"),
        ("<p>Dodaj GhostFTP na početni zaslon.", "<p>Dodaj Ghost FTP na početni zaslon."),
    ],
    "GhostFTP WEB/login.php": [
        ("migracija starih GhostFTP podataka", "migracija starih Ghost FTP podataka"),
        ("GhostFTP je uspješno postavljen.", "Ghost FTP je uspješno postavljen."),
        (">Instaliraj GhostFTP</button>", ">Instaliraj Ghost FTP</button>"),
    ],
    "GhostFTP WEB/register.php": [
        ("<h1>Tvoj GhostFTP workspace.</h1>", "<h1>Tvoj Ghost FTP workspace.</h1>"),
        (">Instaliraj GhostFTP</button>", ">Instaliraj Ghost FTP</button>"),
    ],
    "GhostFTP WEB/settings.php": [
        ("($_POST['app_name'] ?? 'GhostFTP')) ?: 'GhostFTP'", "($_POST['app_name'] ?? 'Ghost FTP')) ?: 'Ghost FTP'"),
        ("($config['app_name'] ?? 'GhostFTP')", "($config['app_name'] ?? 'Ghost FTP')"),
    ],
    "GhostFTP WEB/setup.php": [
        ("postojeći GhostFTP korisnički podaci", "postojeći Ghost FTP korisnički podaci"),
        ("($_POST['app_name'] ?? 'GhostFTP')) ?: 'GhostFTP'", "($_POST['app_name'] ?? 'Ghost FTP')) ?: 'Ghost FTP'"),
        ("$pageTitle = 'Postavljanje GhostFTP';", "$pageTitle = 'Postavljanje Ghost FTP';"),
        ("($_POST['app_name'] ?? 'GhostFTP')", "($_POST['app_name'] ?? 'Ghost FTP')"),
        ("izraditi izolirani GhostFTP račun", "izraditi izolirani Ghost FTP račun"),
        ("HTTPS. GhostFTP ne sprema", "HTTPS. Ghost FTP ne sprema"),
    ],
    "GhostFTP WEB/app/Views/settings-nav.php": [
        ('alt="GhostFTP"', 'alt="Ghost FTP"'),
    ],
    "GhostFTP WEB/download-archive.php": [
        ("'GhostFTP-download.zip'", "'Ghost-FTP-download.zip'"),
    ],
}
for path, replacements in visible_replacements.items():
    for old, new in replacements:
        replace_once(path, old, new)

# Regression test pins both disclosure and dead-contract cleanup.
test_path = ROOT / "GhostFTP WEB/tests/html-error-and-interface-boundary.php"
if test_path.exists():
    raise SystemExit("PATCH_FAILED: html-error-and-interface-boundary.php already exists")
test_path.write_text("""<?php
declare(strict_types=1);

$root = dirname(__DIR__);
$publicPages = ['account.php', 'login.php', 'register.php', 'users.php', 'settings.php', 'setup.php'];
foreach ($publicPages as $page) {
    $source = file_get_contents($root . '/' . $page);
    if (!is_string($source)) {
        fwrite(STDERR, "FAIL: unable to inspect {$page}.\\n");
        exit(1);
    }
    if (str_contains($source, '$error = $e->getMessage();')) {
        fwrite(STDERR, "FAIL: {$page} still exposes raw Throwable messages.\\n");
        exit(1);
    }
    if (!str_contains($source, 'GhostFTP_public_error($e)')) {
        fwrite(STDERR, "FAIL: {$page} does not use the shared public error boundary.\\n");
        exit(1);
    }
}

$operations = file_get_contents($root . '/app/Operations/RemoteOperations.php');
$userStore = file_get_contents($root . '/app/Storage/UserStore.php');
if (!is_string($operations) || !is_string($userStore)) {
    fwrite(STDERR, "FAIL: unable to inspect exception wrappers.\\n");
    exit(1);
}
if (str_contains($operations, "$deleteError->getMessage()") || str_contains($userStore, "$e->getMessage(),\\n                0,")) {
    fwrite(STDERR, "FAIL: Throwable wrapper can still re-expose its previous raw message.\\n");
    exit(1);
}

$transportFiles = [
    'app/Remote/RemoteClientInterface.php',
    'app/Remote/FtpClient.php',
    'app/Remote/SftpClient.php',
    'tests/unit.php',
    'tests/zip-extraction-preflight.php',
];
foreach ($transportFiles as $relative) {
    $source = file_get_contents($root . '/' . $relative);
    if (!is_string($source)) {
        fwrite(STDERR, "FAIL: unable to inspect {$relative}.\\n");
        exit(1);
    }
    if (str_contains($source, 'function write(string $remotePath')) {
        fwrite(STDERR, "FAIL: dead transport write() contract remains in {$relative}.\\n");
        exit(1);
    }
}
$interface = file_get_contents($root . '/app/Remote/RemoteClientInterface.php');
if (!is_string($interface) || !str_contains($interface, 'int $maxBytes = 4194304')) {
    fwrite(STDERR, "FAIL: remote read contract is not aligned to the 4 MiB editor boundary.\\n");
    exit(1);
}

echo "WEB_HTML_ERROR_AND_INTERFACE_BOUNDARY_TEST=PASS\\n";
""", encoding="utf-8", newline="\n")

# Pin the new invariants in the fail-closed Web audit.
audit = "scripts/audit_web.py"
replace_once(
    audit,
    """    for endpoint in ("download.php", "download-archive.php", "preview.php"):
        endpoint_source = read(f"GhostFTP WEB/{endpoint}")
        require(endpoint_source, ("GhostFTP_public_error($e)", "GhostFTP_public_error_status($e)"), endpoint)
""",
    """    for endpoint in ("download.php", "download-archive.php", "preview.php"):
        endpoint_source = read(f"GhostFTP WEB/{endpoint}")
        require(endpoint_source, ("GhostFTP_public_error($e)", "GhostFTP_public_error_status($e)"), endpoint)

    for page in ("account.php", "login.php", "register.php", "users.php", "settings.php", "setup.php"):
        page_source = read(f"GhostFTP WEB/{page}")
        require(page_source, ("GhostFTP_public_error($e)",), page)
        if "$error = $e->getMessage();" in page_source:
            fail(f"HTML page exposes raw Throwable message: {page}")

    operations = read("GhostFTP WEB/app/Operations/RemoteOperations.php")
    user_store = read("GhostFTP WEB/app/Storage/UserStore.php")
    if "$deleteError->getMessage()" in operations:
        fail("move fallback re-exposes raw nested Throwable text")
    if "ponovi brisanje: ' . $e->getMessage()" in user_store:
        fail("user-delete wrapper re-exposes raw nested Throwable text")

    remote_interface = read("GhostFTP WEB/app/Remote/RemoteClientInterface.php")
    ftp_client = read("GhostFTP WEB/app/Remote/FtpClient.php")
    sftp_client = read("GhostFTP WEB/app/Remote/SftpClient.php")
    require(remote_interface, ("int $maxBytes = 4194304",), "app/Remote/RemoteClientInterface.php")
    for label, source in (("RemoteClientInterface", remote_interface), ("FtpClient", ftp_client), ("SftpClient", sftp_client)):
        if "function write(string $remotePath" in source:
            fail(f"dead transport write() contract remains in {label}")
""",
)

# Release documentation.
changelog = read("CHANGELOG.md")
if not changelog.startswith("# Changelog\n\n## 1.0.4"):
    raise SystemExit("PATCH_FAILED: unexpected CHANGELOG head")
write(
    "CHANGELOG.md",
    changelog.replace(
        "# Changelog\n\n",
        """# Changelog

## 1.0.5 - 2026-09-05

- Extended the shared Web public-error boundary to account, login migration, registration, user administration, application settings and first-run setup HTML flows so unexpected Throwable details are no longer rendered in pages.
- Removed nested Throwable-message concatenation from move fallback and user-workspace deletion wrappers, preserving the original exception only as the internal cause.
- Removed the unused transport-level `write()` interface and FTP/SFTP implementations; all real inline writes remain on the bounded, exact-write, atomic `RemoteOperations::writeAtomic()` path.
- Aligned the remote read interface default with the canonical 4 MiB browser editor limit.
- Added regression/audit coverage for HTML error disclosure, nested exception wrapping and dead transport write-contract removal.
- Completed additional visible Web/PWA branding cleanup so user-facing labels, setup/login text, install prompts, image alt text and default installation name use **Ghost FTP**.

""",
        1,
    ),
)
replace_once("README.md", "Current Ghost FTP version: **1.0.4**", "Current Ghost FTP version: **1.0.5**")
replace_once(
    "README.md",
    "Ghost FTP 1.0.4 extends Web/PWA fail-closed behavior to public error handling and multi-file uploads. Unexpected PHP/extension failures no longer expose raw internal messages to clients, and complete upload request metadata/path validation now finishes before the first remote mutation. The 1.0.3 SFTP trust, staging, key-permission and batch-mutation protections remain enforced by regression audits.",
    "Ghost FTP 1.0.5 extends the 1.0.4 safe-error boundary across HTML account/setup/admin flows and removes nested exception wrappers that could re-expose internal error text. It also removes the unused transport-level write contract so browser editor writes have one bounded, exact-write, atomic implementation path, while preserving the existing upload, SFTP trust and batch-mutation protections.",
)
replace_once(
    "README.md",
    "`ghostftp-v1.0.2`, `ghostftp-v1.0.3` and `ghostftp-v1.0.4`",
    "`ghostftp-v1.0.2`, `ghostftp-v1.0.3`, `ghostftp-v1.0.4` and `ghostftp-v1.0.5`",
)
replace_once(
    "README.md",
    "(`1.0.0` → `1.0.1` → `1.0.2` → `1.0.3` → `1.0.4`)",
    "(`1.0.0` → `1.0.1` → `1.0.2` → `1.0.3` → `1.0.4` → `1.0.5`)",
)
replace_once("docs/SECURITY.md", "**Current Ghost FTP release: 1.0.4**", "**Current Ghost FTP release: 1.0.5**")
replace_once(
    "docs/SECURITY.md",
    "- Known application validation failures use HTTP 400; unexpected internal `Throwable` failures use HTTP 500 without exposing their raw message to the client.\n",
    "- Known application validation failures use HTTP 400; unexpected internal `Throwable` failures use HTTP 500 without exposing their raw message to the client.\n- The same public-error mapping is used by account, registration, settings, user-administration, login-migration and setup HTML flows; nested internal exceptions are preserved as causes without concatenating their raw text into a user-visible `RuntimeException`.\n- Browser editor/new-file writes use only `RemoteOperations::writeAtomic()`; the unused direct transport `write()` contract has been removed to prevent a weaker duplicate write path from drifting back into use.\n",
)

print("GHOST_FTP_1_0_5_HARDENING_PATCH=APPLIED")
