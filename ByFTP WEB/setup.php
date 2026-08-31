<?php
declare(strict_types=1);
require __DIR__ . '/app/bootstrap.php';

use ByFTP\Security\AppLogger;
use ByFTP\Storage\UserStore;

if (byftp_is_configured()) {
    byftp_redirect('login');
}

$hasStoredData = static function (): bool {
    $candidates = [
        BYFTP_STORAGE . '/users.json',
        BYFTP_STORAGE . '/users.json.bak',
        BYFTP_STORAGE . '/profiles.json',
        BYFTP_STORAGE . '/profiles.json.bak',
        BYFTP_STORAGE . '/profiles.json.migrated.bak',
        BYFTP_STORAGE . '/preferences.json',
        BYFTP_STORAGE . '/preferences.json.bak',
        BYFTP_STORAGE . '/preferences.json.migrated.bak',
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
    foreach (glob(BYFTP_STORAGE . '/users/*', GLOB_ONLYDIR) ?: [] as $directory) {
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
$existingDataDetected = $hasStoredData();
$error = $existingDataDetected
    ? 'Pronađeni su postojeći ByFTP korisnički podaci, ali nedostaje konfiguracija s encryption ključem. Vrati storage/app.json ili storage/app.json.bak iz sigurnosne kopije prije nastavka.'
    : '';

if ($_SERVER['REQUEST_METHOD'] === 'POST') {
    $appName = trim((string)($_POST['app_name'] ?? 'ByFTP')) ?: 'ByFTP';
    $name = trim((string)($_POST['name'] ?? ''));
    $email = trim((string)($_POST['email'] ?? ''));
    $password = (string)($_POST['password'] ?? '');
    $confirm = (string)($_POST['confirm'] ?? '');
    $allowRegistration = !empty($_POST['allow_registration']);

    if ($existingDataDetected) {
        // Never rotate the encryption key while encrypted user/profile data still exists.
        $error = 'Postavljanje je zaključano radi zaštite postojećih podataka. Vrati storage/app.json ili storage/app.json.bak iz sigurnosne kopije.';
    } elseif (!byftp_verify_csrf(is_string($_POST['csrf'] ?? null) ? $_POST['csrf'] : null)) {
        $error = 'Sigurnosni token nije valjan. Osvježi stranicu.';
    } elseif ($password !== $confirm) {
        $error = 'Lozinke se ne podudaraju.';
    } elseif (!is_writable(BYFTP_STORAGE)) {
        $error = 'Direktorij storage nije zapisiv. Provjeri dozvole na hostingu.';
    } elseif (!function_exists('sodium_crypto_secretbox') && !function_exists('openssl_encrypt')) {
        $error = 'Hosting mora imati Sodium ili OpenSSL za šifriranje FTP vjerodajnica.';
    } else {
        $setupLock = @fopen(BYFTP_STORAGE . '/setup.lock', 'c+');
        if (!is_resource($setupLock) || !flock($setupLock, LOCK_EX)) {
            if (is_resource($setupLock)) {
                fclose($setupLock);
            }
            $error = 'Nije moguće zaključati instalaciju. Provjeri dozvole storage direktorija i pokušaj ponovno.';
        } else {
            @chmod(BYFTP_STORAGE . '/setup.lock', 0600);
            try {
                // Re-check under an exclusive lock so two simultaneous first-run requests
                // cannot create different encryption keys or competing administrator accounts.
                if (byftp_is_configured()) {
                    byftp_redirect('login');
                }

                $config = [
                    'app_name' => byftp_truncate($appName, 80),
                    'secret_key' => base64_encode(random_bytes(32)),
                    'allow_registration' => $allowRegistration,
                    'allow_private_hosts' => false,
                    'session_idle_minutes' => 120,
                    'session_max_hours' => 12,
                    'installed_at' => gmdate('c'),
                    'version' => BYFTP_VERSION,
                ];
                byftp_write_config($config);
                (new UserStore())->create($name, $email, $password, 'admin');
                AppLogger::event('install.complete', ['email' => strtolower($email)]);
                byftp_redirect('login', ['installed' => 1]);
            } catch (Throwable $e) {
                // existingDataDetected was false before this setup transaction. Therefore
                // these config/user JSON generations and lock files can only be empty
                // pre-existing scaffolding or artifacts created by this failed attempt.
                // Remove the complete JsonStore generations so a stale users.json.bak
                // cannot be mistaken for recoverable production data on the next request.
                $rollbackArtifacts = [
                    byftp_config_path(),
                    byftp_config_path() . '.bak',
                    byftp_config_path() . '.lock',
                    BYFTP_STORAGE . '/users.json',
                    BYFTP_STORAGE . '/users.json.bak',
                    BYFTP_STORAGE . '/users.json.lock',
                ];
                foreach ($rollbackArtifacts as $artifact) {
                    @unlink($artifact);
                }
                $GLOBALS['byftp_config_cache'] = [];
                unset($GLOBALS['byftp_config_error']);
                $error = $e->getMessage();
            } finally {
                flock($setupLock, LOCK_UN);
                fclose($setupLock);
            }
        }
    }
}
$pageTitle = 'Postavljanje ByFTP';
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
    <?php if ($error): ?><div class="alert error" role="alert"><?= byftp_e($error) ?></div><?php endif; ?>
    <?php if ($existingDataDetected): ?>
        <div class="stack">
            <p class="muted">Novi setup bi izradio novi encryption ključ i postojeće spremljene FTP/SFTP vjerodajnice više se ne bi mogle dešifrirati. Ne briši <code>storage/users/</code> niti postojeće JSON datoteke dok ne vratiš konfiguraciju.</p>
        </div>
    <?php else: ?>
    <form method="post" class="stack auth-form" autocomplete="off">
        <input type="hidden" name="csrf" value="<?= byftp_e(byftp_csrf_token()) ?>">
        <label>Naziv aplikacije
            <input name="app_name" value="<?= byftp_e((string)($_POST['app_name'] ?? 'ByFTP')) ?>" maxlength="80" required>
        </label>
        <div class="form-grid two">
            <label>Ime administratora
                <input name="name" value="<?= byftp_e((string)($_POST['name'] ?? '')) ?>" maxlength="80" autocomplete="name" required>
            </label>
            <label>E-mail
                <input type="email" name="email" value="<?= byftp_e((string)($_POST['email'] ?? '')) ?>" maxlength="254" autocomplete="email" required>
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
            <span><strong>Dopusti samostalnu registraciju</strong><small>Korisnici će moći sami izraditi izolirani ByFTP račun. Možeš promijeniti kasnije.</small></span>
        </label>
        <button class="button primary auth-submit" type="submit">Završi postavljanje <span aria-hidden="true">→</span></button>
    </form>
    <?php endif; ?>
    <div class="auth-security-note"><span aria-hidden="true">●</span> Za produkciju koristi HTTPS. ByFTP ne sprema FTP/SFTP lozinke kao čitljiv tekst.</div>
</main>
<script src="<?= byftp_e(byftp_asset('js/pwa.js')) ?>" defer></script>
</body>
</html>
