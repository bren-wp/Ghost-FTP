<?php
declare(strict_types=1);
/** @var string $activeSettingsPage */
/** @var array $currentUser */
?>
<header class="settings-topbar">
    <a class="brand settings-brand" href="<?= GhostFTP_e(GhostFTP_url('app')) ?>"><img src="<?= GhostFTP_e(GhostFTP_asset('images/logo.svg')) ?>" alt="Ghost FTP"></a>
    <nav class="settings-nav" aria-label="Postavke">
        <a class="<?= $activeSettingsPage === 'account' ? 'active' : '' ?>" href="<?= GhostFTP_e(GhostFTP_url('account')) ?>">Moj račun</a>
        <?php if (($currentUser['role'] ?? '') === 'admin'): ?>
            <a class="<?= $activeSettingsPage === 'users' ? 'active' : '' ?>" href="<?= GhostFTP_e(GhostFTP_url('users')) ?>">Korisnici</a>
            <a class="<?= $activeSettingsPage === 'settings' ? 'active' : '' ?>" href="<?= GhostFTP_e(GhostFTP_url('settings')) ?>">Aplikacija</a>
        <?php endif; ?>
        <a href="<?= GhostFTP_e(GhostFTP_url('diagnostics')) ?>">Dijagnostika</a>
    </nav>
    <div class="settings-topbar-actions">
        <button class="button ghost compact" type="button" data-install-app>Instaliraj</button>
        <a class="button ghost compact" href="<?= GhostFTP_e(GhostFTP_url('app')) ?>">Natrag u FTP</a>
    </div>
</header>
