<?php
declare(strict_types=1);
require __DIR__ . '/app/bootstrap.php';

use GhostFTP\Security\AppLogger;
use GhostFTP\Storage\UserStore;

if (GhostFTP_is_configured()) {
    GhostFTP_redirect('login');
}

$hasStoredData = static function (): bool {
    $candidates = [
        GhostFTP_STORAGE . '/users.json',
        GhostFTP_STORAGE . '/users.json.bak',
        GhostFTP_STORAGE . '/profiles.json',
        GhostFTP_STORAGE . '/profiles.json.bak',
        GhostFTP_STORAGE . '/profiles.json.migrated.bak',
        GhostFTP_STORAGE . '/preferences.json',
        GhostFTP_STORAGE . '/preferences.json.bak',
        GhostFTP_STORAGE . '/preferences.json.migrated.bak',
    ];
    foreach ($candidates as $path) {
        if (!is_file($path)) {
            continue;
        }
        $raw = @file_get_contents($path);
        if (is_string($raw) && !in_array(trim($raw), ['', '[]', '{}'], true)) {
            return true;
        }
    }
    foreach (glob(GhostFTP_STORAGE . '/users/*', GLOB_ONLYDIR) ?: [] as $directory) {
        foreach (['profiles.json', 'profiles.json.bak', 'preferences.json', 'preferences.json.bak'] as $name) {
            $path = $directory . '/' . $name;
            if (!is_file($path)) {
                continue;
            }
            $raw = @file_get_contents($path);
            if (is_string($raw) && !in_array(trim($raw), ['', '[]', '{}'], true)) {
                return true;
            }
        }
    }
    return false;
};
$configRecoveryRequired = isset($GLOBALS['GhostFTP_config_error']);
$existingDataDetected = $configRecoveryRequired || $hasStoredData();
$error = $configRecoveryRequired
    ? 'Konfiguracija aplikacije nije čitljiva. Automatski povratak na stariji app.json.bak je blokiran radi sigurnosti. Vrati provjereni storage/app.json prije nastavka.'
    : ($existingDataDetected
        ? 'Pronađeni su postojeći GhostFTP korisnički podaci, ali nedostaje konfiguracija s encryption ključem. Vrati storage/app.json ili storage/app.json.bak iz sigurnosne kopije prije nastavka.'
        : '');

if ($_SERVER['REQUEST_METHOD'] === 'POST') {
    $appName = trim((string)($_POST['app_name'] ?? 'GhostFTP')) ?: 'GhostFTP';
    $name = trim((string)($_POST['name'] ?? ''));
    $email = trim((string)($_POST['email'] ?? ''));
    $password = (string)($_POST['password'] ?? '');
    $confirm = (string)($_POST['confirm'] ?? '');
    $allowRegistration = !empty($_POST['allow_registration']);

    if ($existingDataDetected) {
        // Never rotate the encryption key while encrypted user/profile data still exists
        // or while the primary configuration requires explicit operator recovery.
        $error = 'Postavljanje je zaključano radi zaštite postojećih podataka i sigurnosnih postavki. Vrati provjereni storage/app.json prije nastavka.';
    } elseif (!GhostFTP_verify_csrf(is_string($_POST['csrf'] ?? null) ? $_POST['csrf'] : null)) {
        $error = 'Sigurnosni token nije valjan. Osvježi stranicu.';
    } elseif ($password !== $confirm) {
        $error = 'Lozinke se ne podudaraju.';
    } elseif (!is_writable(GhostFTP_STORAGE)) {
        $error = 'Direktorij storage nije zapisiv. Provjeri dozvole na hostingu.';
    } elseif (!function_exists('sodium_crypto_secretbox') && !function_exists('openssl_encrypt')) {
        $error = 'Hosting mora imati Sodium ili OpenSSL za šifriranje FTP vjerodajnica.';
    } else {
        $setupLock = @fopen(GhostFTP_STORAGE . '/setup.lock', 'c+');
        if (!is_resource($setupLock) || !flock($setupLock, LOCK_EX)) {
            if (is_resource($setupLock)) {
                fclose($setupLock);
            }
            $error = 'Nije moguće zaključati instalaciju. Provjeri dozvole storage direktorija i pokušaj ponovno.';
        } else {
            @chmod(GhostFTP_STORAGE . '/setup.lock', 0600);
            $setupTransactionStarted = false;
            try {
                // Re-check under an exclusive lock so two simultaneous first-run requests
                // cannot create different encryption keys or competing administrator accounts.
                if (GhostFTP_is_configured()) {
                    GhostFTP_redirect('login');
                }
                if (isset($GLOBALS['GhostFTP_config_error'])) {
                    throw new RuntimeException('Konfiguracija aplikacije zahtijeva ručni oporavak. Novi setup nije pokrenut.');
                }

                $setupTransactionStarted = true;
                $config = [
                    'app_name' => GhostFTP_truncate($appName, 80),
                    'secret_key' => base64_encode(random_bytes(32)),
                    'allow_registration' => $allowRegistration,
                    'allow_private_hosts' => false,
                    'session_idle_minutes' => 120,
                    'session_max_hours' => 12,
                    'installed_at' => gmdate('c'),
                    'version' => GhostFTP_VERSION,
                ];
                GhostFTP_write_config($config);
                (new UserStore())->create($name, $email, $password, 'admin');
                AppLogger::event('install.complete', ['email' => strtolower($email)]);
                GhostFTP_redirect('login', ['installed' => 1]);
            } catch (Throwable $e) {
                // Only a transaction that passed the recovery guards may remove setup
                // artifacts. A pre-existing corrupt/missing primary config must be left
                // untouched so an operator can restore it from a verified backup.
                if ($setupTransactionStarted) {
                    $rollbackArtifacts = [
                        GhostFTP_config_path(),
                        GhostFTP_config_path() . '.bak',
                        GhostFTP_config_path() . '.lock',
                        GhostFTP_STORAGE . '/users.json',
                        GhostFTP_STORAGE . '/users.json.bak',
                        GhostFTP_STORAGE . '/users.json.lock',
                    ];
                    foreach ($rollbackArtifacts as $artifact) {
                        @unlink($artifact);
                    }
                    $GLOBALS['GhostFTP_config_cache'] = [];
                    unset($GLOBALS['GhostFTP_config_error']);
                }
                $error = $e->getMessage();
            } finally {
                flock($setupLock, LOCK_UN);
                fclose($setupLock);
            }
        }
    }
}
$pageTitle = 'Postavljanje GhostFTP';
?><!doctype html>
<html lang="hr">
<head>
<?php require __DIR__ . '/app/Views/head.php'; ?>
</head>
<body class="auth-page brendigo-auth">
<a class="skip-link" href="#main">Preskoči na postavljanje</a>
<main id="main" class="auth-card auth-card-premium setup-card auth-card-wide">
    <?php require __DIR__ . '/app/Views/auth-brand.php'; ?>
    <p class="eyebrow">Prvo pokretanje</p>
    <h1>Izradi administratorski račun.</h1>
    <p class="muted">Svaki korisnik dobiva vlastite server profile, favorite i postavke. FTP/SFTP vjerodajnice spremaju se šifrirano.</p>
    <?php if ($error): ?><div class="alert error" role="alert"><?= GhostFTP_e($error) ?></div><?php endif; ?>
    <?php if ($existingDataDetected): ?>
        <div class="stack">
            <p class="muted">Novi setup bi mogao izraditi novi encryption ključ ili vratiti stariju sigurnosnu politiku. Ne briši <code>storage/users/</code> niti postojeće JSON datoteke. Vrati provjereni <code>storage/app.json</code> prije nastavka.</p>
        </div>
    <?php else: ?>
    <form method="post" class="stack auth-form" autocomplete="off">
        <input type="hidden" name="csrf" value="<?= GhostFTP_e(GhostFTP_csrf_token()) ?>">
        <label>Naziv aplikacije
            <input name="app_name" value="<?= GhostFTP_e((string)($_POST['app_name'] ?? 'GhostFTP')) ?>" maxlength="80" required>
        </label>
        <div class="form-grid two">
            <label>Ime administratora
                <input name="name" value="<?= GhostFTP_e((string)($_POST['name'] ?? '')) ?>" maxlength="80" autocomplete="name" required>
            </label>
            <label>E-mail
                <input type="email" name="email" value="<?= GhostFTP_e((string)($_POST['email'] ?? '')) ?>" maxlength="254" autocomplete="email" required>
            </label>
        </div>
        <div class="form-grid two">
            <label>Lozinka
                <input type="password" name="password" minlength="12" autocomplete="new-password" required>
            </label>
            <label>Ponovi lozinku
                <input type="password" name="confirm" minlength="12" autocomplete="new-password" required>
            </label>
        </div>
        <label class="check-card">
            <input type="checkbox" name="allow_registration" value="1" <?= !empty($_POST['allow_registration']) ? 'checked' : '' ?>>
            <span><strong>Dopusti samostalnu registraciju</strong><small>Korisnici će moći sami izraditi izolirani GhostFTP račun. Možeš promijeniti kasnije.</small></span>
        </label>
        <button class="button primary auth-submit" type="submit">Završi postavljanje <span aria-hidden="true">→</span></button>
    </form>
    <?php endif; ?>
    <div class="auth-security-note"><span aria-hidden="true">●</span> Za produkciju koristi HTTPS. GhostFTP ne sprema FTP/SFTP lozinke kao čitljiv tekst.</div>
</main>
<script src="<?= GhostFTP_e(GhostFTP_asset('js/pwa.js')) ?>" defer></script>
</body>
</html>
