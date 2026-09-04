<?php
declare(strict_types=1);
require __DIR__ . '/app/bootstrap.php';

use GhostFTP\Security\AppLogger;
use GhostFTP\Security\Auth;
use GhostFTP\Security\LoginRateLimitGate;
use GhostFTP\Security\RateLimiter;
use GhostFTP\Storage\UserStore;
use GhostFTP\Storage\UserWorkspace;

function GhostFTP_clear_login_rate_limiters(
    RateLimiter $accountLimiter,
    string $accountKey,
    RateLimiter $ipLimiter,
    string $ipKey
): void {
    foreach ([
        ['scope' => 'account', 'limiter' => $accountLimiter, 'key' => $accountKey],
        ['scope' => 'ip', 'limiter' => $ipLimiter, 'key' => $ipKey],
    ] as $entry) {
        try {
            $entry['limiter']->clear($entry['key']);
        } catch (Throwable $e) {
            // Authentication has already succeeded. A best-effort limiter reset must not
            // turn a valid login into a false legacy-migration failure or destroy the session.
            AppLogger::event('auth.rate_limit_clear_failed', [
                'scope' => $entry['scope'],
                'error' => GhostFTP_truncate($e->getMessage(), 300),
            ]);
        }
    }
}

if (!GhostFTP_is_configured()) {
    GhostFTP_redirect('setup');
}
if (Auth::check()) {
    GhostFTP_redirect('app');
}

$users = new UserStore();
$config = GhostFTP_config();
$legacy = $users->count() === 0 && !empty($config['password_hash']);
$error = '';
$accountLimiter = new RateLimiter(5, 900);
$ipLimiter = new RateLimiter(12, 900);

if ($_SERVER['REQUEST_METHOD'] === 'POST') {
    $email = strtolower(trim((string)($_POST['email'] ?? '')));
    $password = (string)($_POST['password'] ?? '');
    $accountKey = 'login-account:' . hash('sha256', GhostFTP_truncate($email, 320));
    $ipKey = 'login-ip:' . GhostFTP_client_ip();

    if (!GhostFTP_verify_csrf(is_string($_POST['csrf'] ?? null) ? $_POST['csrf'] : null)) {
        $error = 'Sigurnosni token nije valjan.';
    } else {
        $rateLimitState = 'allowed';
        try {
            // The IP budget is consumed first. A source that is already blocked must not
            // be allowed to consume account-specific budgets for arbitrary e-mail addresses.
            // Each individual consume remains an atomic check+increment transaction.
            if (!LoginRateLimitGate::consume($ipLimiter, $ipKey, $accountLimiter, $accountKey)) {
                $rateLimitState = 'blocked';
            }
        } catch (Throwable $e) {
            $rateLimitState = 'error';
            AppLogger::event('auth.rate_limit_consume_failed', [
                'error' => GhostFTP_truncate($e->getMessage(), 300),
            ]);
        }

        if ($rateLimitState === 'error') {
            $error = 'Sigurnosna zaštita prijave trenutačno nije dostupna. Pokušaj ponovno kasnije.';
        } elseif ($rateLimitState === 'blocked') {
            $error = 'Previše neuspjelih pokušaja. Pokušaj ponovno kasnije.';
            AppLogger::event('auth.blocked', ['email' => $email]);
        } elseif ($legacy) {
            if (strlen($password) > 4096 || !password_verify($password, (string)$config['password_hash'])) {
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
                    $next['version'] = GhostFTP_VERSION;
                    $next['allow_registration'] = (bool)($next['allow_registration'] ?? false);
                    GhostFTP_write_config($next);
                    if (!Auth::attempt($email, $password)) {
                        throw new RuntimeException('Račun je izrađen, ali automatska prijava nije dovršena. Pokušaj se ponovno prijaviti.');
                    }
                    GhostFTP_clear_login_rate_limiters($accountLimiter, $accountKey, $ipLimiter, $ipKey);
                    AppLogger::event('auth.legacy_migrated', ['user_id' => $user['id']]);
                    GhostFTP_redirect('app');
                } catch (Throwable $e) {
                    $error = $e->getMessage();
                }
            }
        } elseif (Auth::attempt($email, $password)) {
            $migrationFailed = false;

            // Recovery path for an interrupted legacy -> multi-user migration: the account may
            // already exist while the legacy password marker/data still remain. Only this actual
            // migration transaction is allowed to invalidate an otherwise successful login.
            if (!empty($config['password_hash']) && Auth::isAdmin()) {
                try {
                    UserWorkspace::migrateLegacy(Auth::id());
                    $next = $config;
                    unset($next['password_hash']);
                    $next['version'] = GhostFTP_VERSION;
                    $next['allow_registration'] = (bool)($next['allow_registration'] ?? false);
                    GhostFTP_write_config($next);
                    AppLogger::event('auth.legacy_migration_recovered', ['user_id' => Auth::id()]);
                } catch (Throwable $e) {
                    AppLogger::event('auth.legacy_migration_failed', [
                        'user_id' => Auth::id(),
                        'error' => GhostFTP_truncate($e->getMessage(), 300),
                    ]);
                    Auth::logout();
                    $migrationFailed = true;
                    $error = 'Prijava je valjana, ali migracija starih GhostFTP podataka nije dovršena. Provjeri dozvole storage direktorija i pokušaj ponovno.';
                }
            }

            if (!$migrationFailed) {
                GhostFTP_clear_login_rate_limiters($accountLimiter, $accountKey, $ipLimiter, $ipKey);
                AppLogger::event('auth.login', ['email' => $email, 'user_id' => Auth::id()]);
                GhostFTP_redirect('app');
            }
        } else {
            usleep(350000);
            $error = 'E-mail ili lozinka nisu točni.';
            AppLogger::event('auth.failed', ['email' => $email]);
        }
    }
}
$pageTitle = 'Prijava · ' . GhostFTP_app_name();
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
    <?php if (isset($_GET['installed'])): ?><div class="alert success" role="status">GhostFTP je uspješno postavljen. Prijavi se svojim administratorskim računom.</div><?php endif; ?>
    <?php if (isset($_GET['registered'])): ?><div class="alert success" role="status">Račun je izrađen. Možeš se prijaviti.</div><?php endif; ?>
    <?php if ($error): ?><div class="alert error" role="alert"><?= GhostFTP_e($error) ?></div><?php endif; ?>
    <form method="post" class="stack auth-form" autocomplete="on">
        <input type="hidden" name="csrf" value="<?= GhostFTP_e(GhostFTP_csrf_token()) ?>">
        <?php if ($legacy): ?>
            <label>Ime administratora
                <input name="name" maxlength="80" value="<?= GhostFTP_e((string)($_POST['name'] ?? 'Administrator')) ?>" autocomplete="name" required>
            </label>
        <?php endif; ?>
        <label>E-mail
            <input type="email" name="email" maxlength="254" value="<?= GhostFTP_e((string)($_POST['email'] ?? '')) ?>" autocomplete="email" autofocus required>
        </label>
        <label><?= $legacy ? 'Stara administratorska lozinka' : 'Lozinka' ?>
            <input type="password" name="password" autocomplete="current-password" required>
        </label>
        <button class="button primary auth-submit" type="submit"><?= $legacy ? 'Nadogradi i prijavi se' : 'Prijavi se' ?> <span aria-hidden="true">→</span></button>
    </form>
    <div class="auth-link-row">
        <?php if (!$legacy && GhostFTP_registration_enabled()): ?><a href="<?= GhostFTP_e(GhostFTP_url('register')) ?>">Izradi korisnički račun</a><?php endif; ?>
        <button class="text-button install-auth" type="button" data-install-app>Instaliraj GhostFTP</button>
    </div>
    <div class="auth-security-note"><span aria-hidden="true">●</span> Izolirani korisnički podaci · šifrirane vjerodajnice · bez indeksiranja</div>
</main>
<script src="<?= GhostFTP_e(GhostFTP_asset('js/pwa.js')) ?>" defer></script>
</body>
</html>
