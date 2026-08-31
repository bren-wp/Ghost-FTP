<?php
declare(strict_types=1);
require __DIR__ . '/app/bootstrap.php';

use ByFTP\Security\AppLogger;
use ByFTP\Security\Auth;
use ByFTP\Security\RateLimiter;
use ByFTP\Storage\UserStore;

if (!byftp_is_configured()) byftp_redirect('setup');
if (Auth::check()) byftp_redirect('app');
if (!byftp_registration_enabled()) byftp_redirect('login');

$error = '';
$limiter = new RateLimiter(4, 1800);
$key = 'register:' . byftp_client_ip();
if ($_SERVER['REQUEST_METHOD'] === 'POST') {
    $name = (string)($_POST['name'] ?? '');
    $email = (string)($_POST['email'] ?? '');
    $password = (string)($_POST['password'] ?? '');
    $confirm = (string)($_POST['confirm'] ?? '');
    if (!byftp_verify_csrf(is_string($_POST['csrf'] ?? null) ? $_POST['csrf'] : null)) {
        $error = 'Sigurnosni token nije valjan.';
    } elseif ($password !== $confirm) {
        $error = 'Lozinke se ne podudaraju.';
    } else {
        $attemptAllowed = false;
        try {
            // Reserve the attempt before validation/hash work. The limiter performs the
            // threshold check and increment under one exclusive JsonStore lock.
            $attemptAllowed = $limiter->consume($key);
        } catch (Throwable $e) {
            AppLogger::event('auth.registration_rate_limit_consume_failed', [
                'error' => byftp_truncate($e->getMessage(), 300),
            ]);
            $error = 'Sigurnosna zaštita registracije trenutačno nije dostupna. Pokušaj ponovno kasnije.';
        }

        if ($error === '' && !$attemptAllowed) {
            $error = 'Previše registracija s ove adrese. Pokušaj ponovno kasnije.';
            AppLogger::event('auth.registration_blocked');
        } elseif ($error === '' && $attemptAllowed) {
            try {
                // The attempt was already atomically counted, so validation/duplicate
                // failures cannot be retried without consuming the configured budget.
                $user = (new UserStore())->create($name, $email, $password, 'user');
                AppLogger::event('auth.register', ['user_id' => $user['id']]);
                byftp_redirect('login', ['registered' => 1]);
            } catch (Throwable $e) {
                $error = $e->getMessage();
            }
        }
    }
}
$pageTitle = 'Izradi račun · ' . byftp_app_name();
?><!doctype html>
<html lang="hr"><head>
<?php require __DIR__ . '/app/Views/head.php'; ?>
</head><body class="auth-page brendigo-auth">
<main class="auth-card auth-card-premium">
    <?php require __DIR__ . '/app/Views/auth-brand.php'; ?>
    <p class="eyebrow">Novi račun</p><h1>Tvoj ByFTP workspace.</h1>
    <p class="muted">Serveri, favoriti i postavke ostat će spremljeni uz ovaj račun i bit će dostupni nakon prijave s drugog uređaja.</p>
    <?php if ($error): ?><div class="alert error" role="alert"><?= byftp_e($error) ?></div><?php endif; ?>
    <form method="post" class="stack auth-form">
        <input type="hidden" name="csrf" value="<?= byftp_e(byftp_csrf_token()) ?>">
        <label>Ime<input name="name" maxlength="80" value="<?= byftp_e((string)($_POST['name'] ?? '')) ?>" autocomplete="name" required></label>
        <label>E-mail<input type="email" name="email" maxlength="254" value="<?= byftp_e((string)($_POST['email'] ?? '')) ?>" autocomplete="email" required></label>
        <label>Lozinka<input type="password" name="password" minlength="12" autocomplete="new-password" required></label>
        <label>Ponovi lozinku<input type="password" name="confirm" minlength="12" autocomplete="new-password" required></label>
        <button class="button primary auth-submit" type="submit">Izradi račun <span>→</span></button>
    </form>
    <div class="auth-link-row"><a href="<?= byftp_e(byftp_url('login')) ?>">Već imam račun</a><button class="text-button" data-install-app type="button">Instaliraj ByFTP</button></div>
</main>
<script src="<?= byftp_e(byftp_asset('js/pwa.js')) ?>" defer></script>
</body></html>
