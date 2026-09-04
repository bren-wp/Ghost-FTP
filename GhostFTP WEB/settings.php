<?php
declare(strict_types=1);
require __DIR__ . '/app/bootstrap.php';

use GhostFTP\Security\AppLogger;
use GhostFTP\Security\Auth;

Auth::requireAdmin();
$currentUser = Auth::user() ?? [];
$config = GhostFTP_config();
$error = '';
$success = '';

if ($_SERVER['REQUEST_METHOD'] === 'POST') {
    if (!GhostFTP_verify_csrf(is_string($_POST['csrf'] ?? null) ? $_POST['csrf'] : null)) {
        $error = 'Sigurnosni token nije valjan.';
    } else {
        try {
            $appName = trim((string)($_POST['app_name'] ?? 'Ghost FTP')) ?: 'Ghost FTP';
            $idle = max(15, min(1440, (int)($_POST['session_idle_minutes'] ?? 120)));
            $maxHours = max(1, min(168, (int)($_POST['session_max_hours'] ?? 12)));
            $config = GhostFTP_update_config([
                'app_name' => GhostFTP_truncate($appName, 80),
                'allow_registration' => !empty($_POST['allow_registration']),
                'allow_private_hosts' => !empty($_POST['allow_private_hosts']),
                'session_idle_minutes' => $idle,
                'session_max_hours' => $maxHours,
                'version' => GhostFTP_VERSION,
            ]);
            AppLogger::event('admin.settings_update', ['user_id' => Auth::id()]);
            $success = 'Postavke aplikacije su spremljene.';
        } catch (Throwable $e) {
            AppLogger::event('admin.settings_update_failed', ['user_id' => Auth::id(), 'exception' => get_class($e), 'error' => GhostFTP_truncate($e->getMessage(), 300)]);
            $error = GhostFTP_public_error($e);
        }
    }
}

$pageTitle = 'Aplikacija · ' . GhostFTP_app_name();
$activeSettingsPage = 'settings';
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
        <h1>Postavke aplikacije</h1>
        <p>Kontroliraj registraciju, mrežnu sigurnost, trajanje sesije i naziv instalacije bez promjene izvornog koda.</p>
    </section>

    <?php if ($error): ?>
        <div class="alert error" role="alert"><?= GhostFTP_e($error) ?></div>
    <?php endif; ?>
    <?php if ($success): ?>
        <div class="alert success" role="status"><?= GhostFTP_e($success) ?></div>
    <?php endif; ?>

    <section class="settings-card settings-card-narrow">
        <form method="post" class="stack">
            <input type="hidden" name="csrf" value="<?= GhostFTP_e(GhostFTP_csrf_token()) ?>">
            <label>
                Naziv instalacije
                <input name="app_name" maxlength="80" value="<?= GhostFTP_e((string)($config['app_name'] ?? 'Ghost FTP')) ?>" required>
            </label>

            <label class="check-card">
                <input type="checkbox" name="allow_registration" value="1" <?= !empty($config['allow_registration']) ? 'checked' : '' ?>>
                <span>
                    <strong>Dopusti samostalnu registraciju</strong>
                    <small>Kada je isključeno, nove korisnike može izrađivati samo administrator.</small>
                </span>
            </label>

            <label class="check-card">
                <input type="checkbox" name="allow_private_hosts" value="1" <?= !empty($config['allow_private_hosts']) ? 'checked' : '' ?>>
                <span>
                    <strong>Dopusti privatne/lokalne server adrese</strong>
                    <small>Ostavi isključeno na javnom/shared hostingu. Uključi samo ako namjerno pristupaš serverima u vlastitoj privatnoj mreži.</small>
                </span>
            </label>

            <div class="form-grid two">
                <label>
                    Automatska odjava nakon neaktivnosti (min)
                    <input type="number" name="session_idle_minutes" min="15" max="1440" value="<?= (int)($config['session_idle_minutes'] ?? 120) ?>">
                </label>
                <label>
                    Maksimalno trajanje sesije (h)
                    <input type="number" name="session_max_hours" min="1" max="168" value="<?= (int)($config['session_max_hours'] ?? 12) ?>">
                </label>
            </div>
            <button class="button primary" type="submit">Spremi postavke</button>
        </form>
    </section>
</main>
<script src="<?= GhostFTP_e(GhostFTP_asset('js/pwa.js')) ?>" defer></script>
</body>
</html>
