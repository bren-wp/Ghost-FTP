<?php
declare(strict_types=1);
require __DIR__ . '/app/bootstrap.php';

use ByFTP\Security\AppLogger;
use ByFTP\Security\Auth;
use ByFTP\Security\RateLimiter;
use ByFTP\Storage\UserStore;
use ByFTP\Storage\UserWorkspace;

if (!byftp_is_configured()) {
    byftp_redirect('setup');
}
if (Auth::check()) {
    byftp_redirect('app');
}

$users = new UserStore();
$config = byftp_config();
$legacy = $users->count() === 0 && !empty($config['password_hash']);
$error = '';
$accountLimiter = new RateLimiter(5, 900);
$ipLimiter = new RateLimiter(12, 900);

if ($_SERVER['REQUEST_METHOD'] === 'POST') {
    $email = strtolower(trim((string)($_POST['email'] ?? '')));
    $accountKey = 'login-account:' . hash('sha256', byftp_truncate($email, 320));
    $ipKey = 'login-ip:' . byftp_client_ip();
    if (!byftp_verify_csrf(is_string($_POST['csrf'] ?? null) ? $_POST['csrf'] : null)) {
        $error = 'Sigurnosni token nije valjan.';
    } elseif ($ipLimiter->blocked($ipKey) || $accountLimiter->blocked($accountKey)) {
        $error = 'Previše neuspjelih pokušaja. Pokušaj ponovno kasnije.';
        AppLogger::event('auth.blocked', ['email' => $email]);
    } elseif ($legacy) {
        $password = (string)($_POST['password'] ?? '');
        if (strlen($password) > 4096 || !password_verify($password, (string)$config['password_hash'])) {
            $accountLimiter->hit($accountKey);
            $ipLimiter->hit($ipKey);
            usleep(350000);
            $error = 'Pogrešna administratorska lozinka.';
        } else {
            try {
                $user = $users->create((string)($_POST['name'] ?? 'Administrator'), $email, $password, 'admin');
                // Move legacy data before removing the old login marker. If migration fails,
                // the newly created account can retry the migration on the next valid login.
                UserWorkspace::migrateLegacy((string)$user['id']);
                $next = $config;
                unset($next['password_hash']);
                $next['version'] = BYFTP_VERSION;
                $next['allow_registration'] = (bool)($next['allow_registration'] ?? false);
                byftp_write_config($next);
                $accountLimiter->clear($accountKey);
                $ipLimiter->clear($ipKey);
                Auth::attempt($email, $password);
                AppLogger::event('auth.legacy_migrated', ['user_id' => $user['id']]);
                byftp_redirect('app');
            } catch (Throwable $e) {
                $error = $e->getMessage();
            }
        }
    } elseif (Auth::attempt($email, (string)($_POST['password'] ?? ''))) {
        try {
            // Recovery path for an interrupted legacy -> multi-user migration: the account may
            // already exist while the legacy password marker/data still remain.
            if (!empty($config['password_hash']) && Auth::isAdmin()) {
                UserWorkspace::migrateLegacy(Auth::id());
                $next = $config;
                unset($next['password_hash']);
                $next['version'] = BYFTP_VERSION;
                $next['allow_registration'] = (bool)($next['allow_registration'] ?? false);
                byftp_write_config($next);
                AppLogger::event('auth.legacy_migration_recovered', ['user_id' => Auth::id()]);
            }
            $accountLimiter->clear($accountKey);
            $ipLimiter->clear($ipKey);
            AppLogger::event('auth.login', ['email' => $email, 'user_id' => Auth::id()]);
            byftp_redirect('app');
        } catch (Throwable $e) {
            AppLogger::event('auth.legacy_migration_failed', ['user_id' => Auth::id(), 'error' => byftp_truncate($e->getMessage(), 300)]);
            Auth::logout();
            $error = 'Prijava je valjana, ali migracija starih ByFTP podataka nije dovršena. Provjeri dozvole storage direktorija i pokušaj ponovno.';
        }
    } else {
        $accountLimiter->hit($accountKey);
        $ipLimiter->hit($ipKey);
        usleep(350000);
        $error = 'E-mail ili lozinka nisu točni.';
        AppLogger::event('auth.failed', ['email' => $email]);
    }
}
$pageTitle = 'Prijava · ' . byftp_app_name();
?><!doctype html>
<html lang="hr">
<head>
<?php require __DIR__ . '/app/Views/head.php'; ?>
</head>
<body class="auth-page brendigo-auth">
<a class="skip-link" href="#main">Preskoči na prijavu</a>
<main id="main" class="auth-card auth-card-premium <?= $legacy ? 'auth-card-wide' : '' ?>">
    <?php require __DIR__ . '/app/Views/auth-brand.php'; ?>
    <?php if ($legacy): ?>
        <p class="eyebrow">Nadogradnja stare instalacije</p>
        <h1>Pretvori stari login u korisnički račun.</h1>
        <p class="muted">Tvoji postojeći server profili i favoriti bit će premješteni u administratorski račun.</p>
    <?php else: ?>
        <p class="eyebrow">Siguran pristup</p>
        <h1>Dobro došao natrag.</h1>
        <p class="muted">Prijavi se i nastavi točno gdje si stao — spremljene veze i postavke vezane su uz tvoj račun.</p>
    <?php endif; ?>
    <?php if (isset($_GET['installed'])): ?><div class="alert success" role="status">ByFTP je uspješno postavljen. Prijavi se svojim administratorskim računom.</div><?php endif; ?>
    <?php if (isset($_GET['registered'])): ?><div class="alert success" role="status">Račun je izrađen. Možeš se prijaviti.</div><?php endif; ?>
    <?php if ($error): ?><div class="alert error" role="alert"><?= byftp_e($error) ?></div><?php endif; ?>
    <form method="post" class="stack auth-form" autocomplete="on">
        <input type="hidden" name="csrf" value="<?= byftp_e(byftp_csrf_token()) ?>">
        <?php if ($legacy): ?>
            <label>Ime administratora
                <input name="name" maxlength="80" value="<?= byftp_e((string)($_POST['name'] ?? 'Administrator')) ?>" autocomplete="name" required>
            </label>
        <?php endif; ?>
        <label>E-mail
            <input type="email" name="email" maxlength="254" value="<?= byftp_e((string)($_POST['email'] ?? '')) ?>" autocomplete="email" autofocus required>
        </label>
        <label><?= $legacy ? 'Stara administratorska lozinka' : 'Lozinka' ?>
            <input type="password" name="password" autocomplete="current-password" required>
        </label>
        <button class="button primary auth-submit" type="submit"><?= $legacy ? 'Nadogradi i prijavi se' : 'Prijavi se' ?> <span aria-hidden="true">→</span></button>
    </form>
    <div class="auth-link-row">
        <?php if (!$legacy && byftp_registration_enabled()): ?><a href="<?= byftp_e(byftp_url('register')) ?>">Izradi korisnički račun</a><?php endif; ?>
        <button class="text-button install-auth" type="button" data-install-app>Instaliraj ByFTP</button>
    </div>
    <div class="auth-security-note"><span aria-hidden="true">●</span> Izolirani korisnički podaci · šifrirane vjerodajnice · bez indeksiranja</div>
</main>
<script src="<?= byftp_e(byftp_asset('js/pwa.js')) ?>" defer></script>
</body>
</html>
