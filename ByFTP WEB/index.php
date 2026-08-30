<?php
declare(strict_types=1);
require __DIR__ . '/app/bootstrap.php';

use ByFTP\Security\Auth;

Auth::requireAuth();
$currentUser = Auth::user() ?? [];
$pageTitle = byftp_app_name();
$initial = strtoupper(function_exists('mb_substr') ? mb_substr((string)($currentUser['name'] ?? 'U'), 0, 1) : substr((string)($currentUser['name'] ?? 'U'), 0, 1));
?>
<!doctype html>
<html lang="hr">
<head>
<?php require __DIR__ . '/app/Views/head.php'; ?>
<meta name="csrf-token" content="<?= byftp_e(byftp_csrf_token()) ?>">
<meta name="api-url" content="<?= byftp_e(byftp_url('api')) ?>">
<meta name="login-url" content="<?= byftp_e(byftp_url('login')) ?>">
<meta name="download-url" content="<?= byftp_e(byftp_url('download')) ?>">
<meta name="download-archive-url" content="<?= byftp_e(byftp_url('download_archive')) ?>">
<meta name="preview-url" content="<?= byftp_e(byftp_url('preview')) ?>">
<meta name="upload-max-bytes" content="<?= byftp_e((string)byftp_upload_limit_bytes()) ?>">
</head>
<body class="app-page">
<a class="skip-link" href="#workspace">Preskoči na datoteke</a>
<div class="app-shell" id="app">
<header class="topbar">
    <button id="sidebarToggle" class="icon-button mobile-only" type="button" aria-label="Otvori izbornik" aria-controls="sidebar" aria-expanded="false">☰</button>
    <a class="brand" href="<?= byftp_e(byftp_url('app')) ?>"><img src="<?= byftp_e(byftp_asset('images/logo.svg')) ?>" alt="ByFTP"></a>
    <div class="connection-summary"><strong id="connectionName">Nije odabran server</strong><span id="connectionMeta">Odaberi vezu iz izbornika</span></div>
    <div class="topbar-search"><span aria-hidden="true">⌕</span><input id="remoteSearch" type="search" placeholder="Pretraži server…" autocomplete="off"><button id="searchBtn" class="icon-button" type="button" aria-label="Pretraži">↵</button></div>
    <div class="user-menu-wrap">
        <button id="userMenuBtn" class="user-menu-button" type="button" aria-haspopup="menu" aria-expanded="false"><span class="user-avatar"><?= byftp_e($initial) ?></span><span class="desktop-only"><?= byftp_e((string)($currentUser['name'] ?? 'Korisnik')) ?></span></button>
        <div id="userMenu" class="user-menu hidden" role="menu">
            <strong><?= byftp_e((string)($currentUser['email'] ?? '')) ?></strong>
            <a href="<?= byftp_e(byftp_url('account')) ?>">Moj račun</a>
            <?php if (($currentUser['role'] ?? '') === 'admin'): ?><a href="<?= byftp_e(byftp_url('users')) ?>">Korisnici</a><a href="<?= byftp_e(byftp_url('settings')) ?>">Postavke</a><?php endif; ?>
            <a href="<?= byftp_e(byftp_url('diagnostics')) ?>">Dijagnostika</a>
            <button type="button" data-install-app>Instaliraj aplikaciju</button>
            <a class="danger-text" href="<?= byftp_e(byftp_url('logout')) ?>">Odjava</a>
        </div>
    </div>
</header>

<div id="sidebarBackdrop" class="sidebar-backdrop hidden"></div>
<aside id="sidebar" class="sidebar" aria-label="Serveri i favoriti">
    <div class="sidebar-heading"><div><span class="eyebrow">Veze</span><strong>Serveri</strong></div><button id="addProfile" class="icon-button" type="button" aria-label="Dodaj vezu">＋</button></div>
    <div id="profiles" class="profile-list"></div>
    <div class="sidebar-section"><div class="sidebar-heading small-heading"><div><span class="eyebrow">Brzi pristup</span><strong>Favoriti</strong></div></div><div id="favorites" class="favorite-list"><span class="muted tiny">Odaberi server.</span></div></div>
    <nav class="app-nav">
        <a href="<?= byftp_e(byftp_url('account')) ?>">● Moj račun</a>
        <?php if (($currentUser['role'] ?? '') === 'admin'): ?><a href="<?= byftp_e(byftp_url('users')) ?>">◎ Korisnici</a><?php endif; ?>
        <a href="<?= byftp_e(byftp_url('diagnostics')) ?>">◇ Dijagnostika</a>
        <button type="button" data-install-app>↓ Instaliraj ByFTP</button>
    </nav>
    <div class="sidebar-footer"><span id="connectionStatus" class="status-dot offline"></span><span id="statusText">Nije povezano</span><small>v<?= byftp_e(BYFTP_VERSION) ?></small></div>
</aside>

<main id="workspace" class="workspace" tabindex="-1">
    <section id="welcome" class="welcome-card">
        <img src="<?= byftp_e(byftp_asset('images/mark.svg')) ?>" class="welcome-mark" alt="">
        <p class="eyebrow">FTP · FTPS · SFTP</p><h1>Datoteke pod kontrolom.</h1>
        <p>Siguran web file manager za shared hosting, desktop, tablet i mobitel — bez CDN-a, telemetrije ili vanjskog backenda.</p>
        <div class="welcome-actions"><button id="welcomeAdd" class="button primary" type="button">Dodaj server</button><a class="button ghost" href="<?= byftp_e(byftp_url('diagnostics')) ?>">Provjeri hosting</a></div>
    </section>

    <section id="fileApp" class="file-app hidden">
        <div class="toolbar">
            <div class="toolbar-group"><button id="upBtn" class="icon-button" type="button" title="Gore">↑</button><button id="refreshBtn" class="icon-button" type="button" title="Osvježi">↻</button><button id="favoriteBtn" class="icon-button" type="button" title="Favorit">☆</button></div>
            <div id="breadcrumbs" class="breadcrumbs" aria-label="Putanja"></div>
            <div class="toolbar-group toolbar-main-actions">
                <label class="button ghost compact">Upload<input id="uploadInput" type="file" multiple hidden></label>
                <label class="button ghost compact desktop-only">Mapa<input id="folderUploadInput" type="file" webkitdirectory multiple hidden></label>
                <button id="newFileBtn" class="button ghost compact desktop-only" type="button">Nova datoteka</button>
                <button id="newFolderBtn" class="button primary compact" type="button">Nova mapa</button>
            </div>
        </div>
        <div class="file-controls">
            <label class="filter-box">⌕ <input id="filterInput" type="search" placeholder="Filtriraj ovu mapu…"></label>
            <label class="inline-check"><input id="showHidden" type="checkbox" checked> skrivene</label>
            <select id="uploadConflict" aria-label="Upload konflikt"><option value="rename">Konflikt: preimenuj</option><option value="overwrite">Konflikt: prepiši</option><option value="skip">Konflikt: preskoči</option></select>
        </div>
        <div id="selectionBar" class="selection-bar hidden"><strong id="selectionCount">0 označeno</strong><button id="bulkDownload" class="button ghost compact" type="button">ZIP download</button><button id="bulkCopy" class="button ghost compact" type="button">Kopiraj</button><button id="bulkMove" class="button ghost compact" type="button">Premjesti</button><button id="bulkZip" class="button ghost compact" type="button">ZIP</button><button id="bulkDelete" class="button danger compact" type="button">Obriši</button><button id="clearSelection" class="icon-button" type="button">×</button></div>
        <div id="dropZone" class="file-table-wrap">
            <table class="file-table"><thead><tr><th class="check-col"><input id="selectAll" type="checkbox" aria-label="Označi sve"></th><th><button class="sort-button" data-sort="name" type="button">Naziv ↕</button></th><th class="desktop-only">Vrsta</th><th><button class="sort-button" data-sort="size" type="button">Veličina ↕</button></th><th class="desktop-only"><button class="sort-button" data-sort="modified" type="button">Promijenjeno ↕</button></th><th class="actions-col">Radnje</th></tr></thead><tbody id="fileRows"></tbody></table>
            <div id="emptyState" class="empty-state hidden"><strong>Mapa je prazna.</strong><span>Upload datoteka ili izradi novu mapu.</span></div>
            <div id="loadingState" class="loading-state hidden">Učitavanje…</div>
        </div>
        <footer class="workspace-footer"><span id="itemSummary">0 stavki</span><span id="pathStatus">/</span></footer>
    </section>
</main>

<div id="transferPanel" class="transfer-panel hidden"><div><strong id="transferTitle">Prijenos</strong><span id="transferDetail">Priprema…</span></div><progress id="transferProgress" max="100" value="0"></progress><button id="cancelUpload" class="button ghost compact hidden" type="button">Otkaži upload</button></div>
</div>

<div id="profileModal" class="modal hidden" role="dialog" aria-modal="true" aria-labelledby="profileTitle"><div class="modal-card profile-card">
    <div class="modal-head"><div><p class="eyebrow">Veza</p><h2 id="profileTitle">Novi server</h2></div><button class="icon-button close-profile" type="button">×</button></div>
    <form id="profileForm" class="stack" autocomplete="off">
        <input type="hidden" name="id"><div class="form-grid two"><label>Naziv<input name="label" maxlength="80" required></label><label>Protokol<select name="protocol"><option value="sftp">SFTP</option><option value="ftps">FTPS</option><option value="ftp">FTP</option></select></label></div>
        <div class="form-grid host-grid"><label>Host<input name="host" maxlength="255" spellcheck="false" required></label><label>Port<input name="port" inputmode="numeric" maxlength="5" value="22" required></label></div>
        <label>Početna putanja<input name="base_path" value="/" spellcheck="false" required></label>
        <div class="form-grid two"><label>Korisničko ime<input name="username" maxlength="512" autocomplete="username" required></label><label>Lozinka<input name="password" type="password" maxlength="4096" autocomplete="new-password" placeholder="ostavi prazno za postojeću"></label></div>
        <div class="form-grid two"><label>Timeout (s)<input name="timeout" inputmode="numeric" value="30" maxlength="3"></label><label>SFTP auth<select name="auth_method"><option value="password">Lozinka</option><option value="key">SSH ključ</option></select></label></div>
        <label class="sftp-only">SFTP SHA-256 host fingerprint<input name="host_fingerprint" maxlength="200" spellcheck="false" placeholder="SHA256:… ili 64-znamenkasti HEX"></label>
        <div class="key-auth stack hidden"><label>Javni ključ<textarea name="public_key" rows="3" spellcheck="false"></textarea></label><label>Privatni ključ<textarea name="private_key" rows="5" spellcheck="false"></textarea></label><label>Passphrase<input name="key_passphrase" type="password" maxlength="2048"></label></div>
        <div class="form-grid two"><label class="inline-check"><input name="passive" type="checkbox" checked> FTP PASV</label><label class="inline-check"><input name="utf8" type="checkbox" checked> FTP UTF-8</label></div>
        <div class="form-actions"><button id="deleteProfileBtn" class="button danger hidden" type="button">Obriši</button><button id="duplicateProfileBtn" class="button ghost hidden" type="button">Dupliciraj</button><button id="testProfileBtn" class="button ghost" type="button">Testiraj</button><span class="spacer"></span><button class="button ghost close-profile" type="button">Odustani</button><button class="button primary" type="submit">Spremi</button></div>
    </form>
</div></div>

<div id="editorModal" class="modal hidden" role="dialog" aria-modal="true"><div class="modal-card editor-card"><div class="modal-head"><div><p class="eyebrow">Remote editor</p><h2 id="editorTitle">Datoteka</h2><span id="editorInfo" class="muted tiny"></span></div><button class="icon-button close-editor" type="button">×</button></div><textarea id="editorContent" class="code-editor" spellcheck="false"></textarea><div class="editor-footer"><span>Ctrl/Cmd + S spremi</span><span id="editorCursor">Red 1, stupac 1</span><button id="saveEditor" class="button primary compact" type="button">Spremi</button></div></div></div>

<div id="promptModal" class="modal hidden" role="dialog" aria-modal="true"><div class="modal-card prompt-card"><div class="modal-head"><div><p class="eyebrow" id="promptEyebrow">Radnja</p><h2 id="promptTitle">Unos</h2></div><button class="icon-button close-prompt" type="button">×</button></div><form id="promptForm" class="stack"><p id="promptDescription" class="muted hidden"></p><label id="promptInputLabel">Vrijednost<input id="promptInput" autocomplete="off"></label><div id="promptExtra"></div><div class="form-actions"><span class="spacer"></span><button class="button ghost close-prompt" type="button">Odustani</button><button class="button primary" type="submit">Potvrdi</button></div></form></div></div>

<div id="searchModal" class="modal hidden" role="dialog" aria-modal="true"><div class="modal-card search-card"><div class="modal-head"><div><p class="eyebrow">Rekurzivna pretraga</p><h2>Rezultati</h2></div><button class="icon-button close-search" type="button">×</button></div><div id="searchStatus" class="muted"></div><div id="searchResults" class="search-results"></div></div></div>

<div id="previewModal" class="modal hidden" role="dialog" aria-modal="true"><div class="modal-card preview-card"><div class="modal-head"><h2 id="previewTitle">Pregled</h2><button class="icon-button close-preview" type="button">×</button></div><div class="preview-stage"><img id="previewImage" alt="Pregled udaljene slike"></div></div></div>

<nav id="contextMenu" class="context-menu hidden"><button data-action="open" type="button">Otvori</button><button data-action="download" type="button">Preuzmi</button><button data-action="edit" type="button">Uredi</button><button data-action="preview" type="button">Pregled slike</button><hr><button data-action="duplicate" type="button">Dupliciraj</button><button data-action="copy" type="button">Kopiraj u…</button><button data-action="move" type="button">Premjesti u…</button><button data-action="rename" type="button">Preimenuj</button><button data-action="chmod" type="button">Dozvole</button><button data-action="extract" type="button">Raspakiraj ZIP</button><hr><button class="danger-text" data-action="delete" type="button">Obriši</button></nav>

<div id="installModal" class="modal hidden" role="dialog" aria-modal="true"><div class="modal-card install-card"><div class="modal-head"><div><p class="eyebrow">PWA</p><h2>Instaliraj ByFTP</h2></div><button class="icon-button" type="button" data-close-install>×</button></div><div class="install-visual"><img src="<?= byftp_e(byftp_asset('images/mark.svg')) ?>" alt="ByFTP"><p>Dodaj ByFTP na početni zaslon. Autentificirane stranice i FTP podaci ne spremaju se u offline cache.</p></div><ol data-ios-install hidden><li>U Safariju odaberi <strong>Dijeli</strong>.</li><li>Odaberi <strong>Dodaj na početni zaslon</strong>.</li></ol><p data-generic-install>U izborniku preglednika odaberi <strong>Instaliraj aplikaciju</strong>.</p></div></div>

<div id="toastHost" class="toast-host" aria-live="polite"></div>
<script src="<?= byftp_e(byftp_asset('js/pwa.js')) ?>" defer></script>
<script type="module" src="<?= byftp_e(byftp_asset('js/app.js')) ?>"></script>
</body>
</html>
