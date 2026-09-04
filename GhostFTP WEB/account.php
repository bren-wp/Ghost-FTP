<?php
declare(strict_types=1);
require __DIR__ . '/app/bootstrap.php';

use ByFTP\Security\AppLogger;
use ByFTP\Security\Auth;
use ByFTP\Storage\UserStore;

Auth::requireAuth();
$store = new UserStore();
$currentUser = Auth::user() ?? [];
$error = '';
$success = '';

if ($_SERVER['REQUEST_METHOD'] === 'POST') {
    if (!byftp_verify_csrf(is_string($_POST['csrf'] ?? null) ? $_POST['csrf'] : null)) {
        $error = 'Sigurnosni token nije valjan.';
    } else {
        try {
            $action = (string)($_POST['action'] ?? '');
            if ($action === 'profile') {
                $currentUser = $store->updateProfile(Auth::id(), (string)($_POST['name'] ?? ''), (string)($_POST['email'] ?? ''));
                Auth::refresh();
                $success = 'Podaci računa su spremljeni.';
                AppLogger::event('account.profile_update', ['user_id' => Auth::id()]);
            } elseif ($action === 'password') {
                $new = (string)($_POST['new_password'] ?? '');
                if ($new !== (string)($_POST['confirm_password'] ?? '')) {
                    throw new RuntimeException('Nove lozinke se ne podudaraju.');
                }
                $userId = Auth::id();
                $store->changePassword($userId, (string)($_POST['current_password'] ?? ''), $new, true);
                // Password change invalidates existing sessions; immediately re-authenticate this browser.
                $fresh = $store->findById($userId);
                if ($fresh) {
                    unset($fresh['password_hash']);
                    Auth::login($fresh);
                }
                $currentUser = Auth::user() ?? $currentUser;
                $success = 'Lozinka je promijenjena. Druge aktivne sesije su opozvane.';
                AppLogger::event('account.password_change', ['user_id' => Auth::id()]);
            }
        } catch (Throwable $e) {
            $error = $e->getMessage();
        }
    }
}

$pageTitle = 'Moj račun · ' . byftp_app_name();
$activeSettingsPage = 'account';
?><!doctype html><html lang="hr"><head>
<?php require __DIR__ . '/app/Views/head.php'; ?>
</head><body class="settings-page">
<?php require __DIR__ . '/app/Views/settings-nav.php'; ?>
<main class="settings-main">
    <section class="settings-hero"><p class="eyebrow">Korisnički račun</p><h1>Moj ByFTP</h1><p>Upravljaj identitetom i sigurnošću svog računa. Spremljene FTP/SFTP veze i preference ostaju povezane s ovim korisnikom.</p></section>
    <?php if ($error): ?><div class="alert error" role="alert"><?= byftp_e($error) ?></div><?php endif; ?>
    <?php if ($success): ?><div class="alert success" role="status"><?= byftp_e($success) ?></div><?php endif; ?>
    <div class="settings-grid">
        <section class="settings-card">
            <div class="settings-card-head"><div><p class="eyebrow">Profil</p><h2>Podaci računa</h2></div><span class="role-badge"><?= ($currentUser['role'] ?? '') === 'admin' ? 'Administrator' : 'Korisnik' ?></span></div>
            <form method="post" class="stack">
                <input type="hidden" name="csrf" value="<?= byftp_e(byftp_csrf_token()) ?>"><input type="hidden" name="action" value="profile">
                <label>Ime<input name="name" maxlength="80" value="<?= byftp_e((string)($currentUser['name'] ?? '')) ?>" required autocomplete="name"></label>
                <label>E-mail<input type="email" name="email" maxlength="254" value="<?= byftp_e((string)($currentUser['email'] ?? '')) ?>" required autocomplete="email"></label>
                <button class="button primary" type="submit">Spremi promjene</button>
            </form>
            <dl class="account-meta"><div><dt>Račun izrađen</dt><dd><?= byftp_e(byftp_human_date($currentUser['created_at'] ?? null)) ?></dd></div><div><dt>Zadnja prijava</dt><dd><?= byftp_e(byftp_human_date($currentUser['last_login_at'] ?? null)) ?></dd></div></dl>
        </section>
        <section class="settings-card">
            <div class="settings-card-head"><div><p class="eyebrow">Sigurnost</p><h2>Promijeni lozinku</h2></div></div>
            <form method="post" class="stack" autocomplete="off">
                <input type="hidden" name="csrf" value="<?= byftp_e(byftp_csrf_token()) ?>"><input type="hidden" name="action" value="password">
                <label>Trenutačna lozinka<input type="password" name="current_password" autocomplete="current-password" required></label>
                <label>Nova lozinka<input type="password" name="new_password" minlength="12" autocomplete="new-password" required></label>
                <label>Ponovi novu lozinku<input type="password" name="confirm_password" minlength="12" autocomplete="new-password" required></label>
                <button class="button primary" type="submit">Promijeni lozinku</button>
                <p class="muted tiny">Promjena lozinke opoziva ostale prijavljene sesije tog računa.</p>
            </form>
        </section>
    </div>
</main>
<script src="<?= byftp_e(byftp_asset('js/pwa.js')) ?>" defer></script>
</body></html>
