<?php
declare(strict_types=1);
require __DIR__ . '/app/bootstrap.php';

use GhostFTP\Security\AppLogger;
use GhostFTP\Security\Auth;
use GhostFTP\Storage\UserStore;

Auth::requireAdmin();
$store = new UserStore();
$currentUser = Auth::user() ?? [];
$error = '';
$success = '';

if ($_SERVER['REQUEST_METHOD'] === 'POST') {
    if (!GhostFTP_verify_csrf(is_string($_POST['csrf'] ?? null) ? $_POST['csrf'] : null)) {
        $error = 'Sigurnosni token nije valjan.';
    } else {
        try {
            $action = (string)($_POST['action'] ?? '');
            if ($action === 'create') {
                $password = (string)($_POST['password'] ?? '');
                if ($password !== (string)($_POST['confirm'] ?? '')) {
                    throw new RuntimeException('Lozinke se ne podudaraju.');
                }
                $created = $store->create(
                    (string)($_POST['name'] ?? ''),
                    (string)($_POST['email'] ?? ''),
                    $password,
                    (string)($_POST['role'] ?? 'user')
                );
                $success = 'Korisnik ' . $created['email'] . ' je izrađen.';
                AppLogger::event('admin.user_create', ['user_id' => $created['id']]);
            } elseif ($action === 'update') {
                $id = (string)($_POST['id'] ?? '');
                if ($id === Auth::id() && empty($_POST['active'])) {
                    throw new RuntimeException('Ne možeš deaktivirati vlastiti račun.');
                }
                $updated = $store->updateAdminFields(
                    $id,
                    (string)($_POST['role'] ?? 'user'),
                    !empty($_POST['active'])
                );
                $success = 'Prava korisnika ' . $updated['email'] . ' su spremljena.';
                AppLogger::event('admin.user_update', ['user_id' => $id]);
            } elseif ($action === 'reset_password') {
                $id = (string)($_POST['id'] ?? '');
                $password = (string)($_POST['password'] ?? '');
                if ($password !== (string)($_POST['confirm'] ?? '')) {
                    throw new RuntimeException('Lozinke se ne podudaraju.');
                }
                $store->changePassword($id, '', $password, false);
                if ($id === Auth::id()) {
                    $fresh = $store->findById($id);
                    if ($fresh) {
                        unset($fresh['password_hash']);
                        Auth::login($fresh);
                    }
                }
                $success = 'Lozinka je resetirana i postojeće sesije korisnika su opozvane.';
                AppLogger::event('admin.password_reset', ['user_id' => $id]);
            } elseif ($action === 'delete') {
                $id = (string)($_POST['id'] ?? '');
                if ($id === Auth::id()) {
                    throw new RuntimeException('Ne možeš obrisati vlastiti račun.');
                }
                $store->delete($id);
                $success = 'Korisnički račun i njegov izolirani workspace su obrisani.';
                AppLogger::event('admin.user_delete', ['user_id' => $id]);
            }
        } catch (Throwable $e) {
            $error = $e->getMessage();
        }
    }
}

$allUsers = $store->all();
$pageTitle = 'Korisnici · ' . GhostFTP_app_name();
$activeSettingsPage = 'users';
?>
<!doctype html>
<html lang="hr">
<head>
<?php require __DIR__ . '/app/Views/head.php'; ?>
</head>
<body class="settings-page">
<?php require __DIR__ . '/app/Views/settings-nav.php'; ?>
<main class="settings-main">
    <section class="settings-hero">
        <p class="eyebrow">Administracija</p>
        <h1>Korisnici</h1>
        <p>Svaki račun ima potpuno odvojene server profile, favorite i postavke. Deaktivacija odmah opoziva aktivne sesije.</p>
    </section>

    <?php if ($error): ?>
        <div class="alert error" role="alert"><?= GhostFTP_e($error) ?></div>
    <?php endif; ?>
    <?php if ($success): ?>
        <div class="alert success" role="status"><?= GhostFTP_e($success) ?></div>
    <?php endif; ?>

    <div class="settings-grid users-grid">
        <section class="settings-card">
            <p class="eyebrow">Novi korisnik</p>
            <h2>Izradi račun</h2>
            <form method="post" class="stack" autocomplete="off">
                <input type="hidden" name="csrf" value="<?= GhostFTP_e(GhostFTP_csrf_token()) ?>">
                <input type="hidden" name="action" value="create">
                <label>Ime<input name="name" maxlength="80" autocomplete="name" required></label>
                <label>E-mail<input type="email" name="email" maxlength="254" autocomplete="email" required></label>
                <label>
                    Uloga
                    <select name="role">
                        <option value="user">Korisnik</option>
                        <option value="admin">Administrator</option>
                    </select>
                </label>
                <label>Privremena lozinka<input type="password" name="password" minlength="12" autocomplete="new-password" required></label>
                <label>Ponovi lozinku<input type="password" name="confirm" minlength="12" autocomplete="new-password" required></label>
                <button class="button primary" type="submit">Izradi korisnika</button>
            </form>
        </section>

        <section class="settings-card users-list-card">
            <div class="settings-card-head">
                <div><p class="eyebrow">Računi</p><h2><?= count($allUsers) ?> korisnika</h2></div>
            </div>
            <div class="user-admin-list">
                <?php foreach ($allUsers as $user): ?>
                    <?php
                    $isSelf = (string)$user['id'] === Auth::id();
                    $initial = strtoupper(function_exists('mb_substr')
                        ? mb_substr((string)$user['name'], 0, 1)
                        : substr((string)$user['name'], 0, 1));
                    ?>
                    <article class="user-admin-row">
                        <div class="user-avatar"><?= GhostFTP_e($initial) ?></div>
                        <div class="user-admin-copy">
                            <strong><?= GhostFTP_e((string)$user['name']) ?></strong>
                            <span><?= GhostFTP_e((string)$user['email']) ?></span>
                            <small>Zadnja prijava: <?= GhostFTP_e(GhostFTP_human_date($user['last_login_at'] ?? null)) ?></small>
                        </div>

                        <form method="post" class="user-admin-controls">
                            <input type="hidden" name="csrf" value="<?= GhostFTP_e(GhostFTP_csrf_token()) ?>">
                            <input type="hidden" name="action" value="update">
                            <input type="hidden" name="id" value="<?= GhostFTP_e((string)$user['id']) ?>">
                            <select name="role" aria-label="Uloga korisnika">
                                <option value="user" <?= $user['role'] === 'user' ? 'selected' : '' ?>>Korisnik</option>
                                <option value="admin" <?= $user['role'] === 'admin' ? 'selected' : '' ?>>Administrator</option>
                            </select>
                            <label class="inline-check">
                                <input type="checkbox" name="active" value="1" <?= !empty($user['active']) ? 'checked' : '' ?> <?= $isSelf ? 'disabled' : '' ?>>
                                Aktivan
                            </label>
                            <?php if ($isSelf): ?><input type="hidden" name="active" value="1"><?php endif; ?>
                            <button class="button ghost compact" type="submit">Spremi</button>
                        </form>

                        <details class="user-admin-more">
                            <summary>Više</summary>
                            <div class="user-admin-more-body">
                                <form method="post" class="inline-form" autocomplete="off">
                                    <input type="hidden" name="csrf" value="<?= GhostFTP_e(GhostFTP_csrf_token()) ?>">
                                    <input type="hidden" name="action" value="reset_password">
                                    <input type="hidden" name="id" value="<?= GhostFTP_e((string)$user['id']) ?>">
                                    <input type="password" name="password" minlength="12" autocomplete="new-password" placeholder="Nova lozinka" required>
                                    <input type="password" name="confirm" minlength="12" autocomplete="new-password" placeholder="Ponovi" required>
                                    <button class="button ghost compact" type="submit">Resetiraj lozinku</button>
                                </form>
                                <?php if (!$isSelf): ?>
                                    <form method="post" data-confirm-delete-user data-user-label="<?= GhostFTP_e((string)$user['email']) ?>">
                                        <input type="hidden" name="csrf" value="<?= GhostFTP_e(GhostFTP_csrf_token()) ?>">
                                        <input type="hidden" name="action" value="delete">
                                        <input type="hidden" name="id" value="<?= GhostFTP_e((string)$user['id']) ?>">
                                        <button class="button danger compact" type="submit">Obriši račun</button>
                                    </form>
                                <?php endif; ?>
                            </div>
                        </details>
                    </article>
                <?php endforeach; ?>
            </div>
        </section>
    </div>
</main>
<script src="<?= GhostFTP_e(GhostFTP_asset('js/settings.js')) ?>" defer></script>
<script src="<?= GhostFTP_e(GhostFTP_asset('js/pwa.js')) ?>" defer></script>
</body>
</html>
