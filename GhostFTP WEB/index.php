<?php
declare(strict_types=1);
require __DIR__ . '/app/bootstrap.php';

use GhostFTP\I18n;
use GhostFTP\Security\Auth;
use GhostFTP\Storage\PreferenceStore;

Auth::requireAuth();
$currentUser = Auth::user() ?? [];
$preferences = (new PreferenceStore(Auth::id()))->clientState();
$language = I18n::normalize((string)($preferences['language'] ?? I18n::DEFAULT_LANGUAGE));
$htmlLanguage = match ($language) {
    'zh' => 'zh-Hans',
    'no' => 'nb',
    default => $language,
};
$catalog = I18n::catalog($language);
$catalogJson = json_encode(
    $catalog,
    JSON_UNESCAPED_UNICODE | JSON_UNESCAPED_SLASHES | JSON_HEX_TAG | JSON_HEX_AMP | JSON_HEX_APOS | JSON_HEX_QUOT
);
if (!is_string($catalogJson)) {
    $catalogJson = '{}';
}
$pageTitle = I18n::t($language, 'app.name');
$initial = strtoupper(function_exists('mb_substr') ? mb_substr((string)($currentUser['name'] ?? 'U'), 0, 1) : substr((string)($currentUser['name'] ?? 'U'), 0, 1));
?>
<!doctype html>
<html lang="<?= GhostFTP_e($htmlLanguage) ?>">
<head>
<?php require __DIR__ . '/app/Views/head.php'; ?>
<meta name="csrf-token" content="<?= GhostFTP_e(GhostFTP_csrf_token()) ?>">
<meta name="api-url" content="<?= GhostFTP_e(GhostFTP_url('api')) ?>">
<meta name="login-url" content="<?= GhostFTP_e(GhostFTP_url('login')) ?>">
<meta name="download-url" content="<?= GhostFTP_e(GhostFTP_url('download')) ?>">
<meta name="download-archive-url" content="<?= GhostFTP_e(GhostFTP_url('download_archive')) ?>">
<meta name="preview-url" content="<?= GhostFTP_e(GhostFTP_url('preview')) ?>">
<meta name="upload-max-bytes" content="<?= GhostFTP_e((string)GhostFTP_upload_limit_bytes()) ?>">
<meta name="ghostftp-language" content="<?= GhostFTP_e($language) ?>">
<meta name="ghostftp-i18n" content="<?= GhostFTP_e($catalogJson) ?>">
</head>
<body class="app-page">
<a class="skip-link" href="#workspace">Skip to files</a>
<div class="app-shell" id="app">
<header class="topbar">
    <button id="sidebarToggle" class="icon-button mobile-only" type="button" aria-label="Open menu" aria-controls="sidebar" aria-expanded="false">☰</button>
    <a class="brand" href="<?= GhostFTP_e(GhostFTP_url('app')) ?>"><img src="<?= GhostFTP_e(GhostFTP_asset('images/logo.svg')) ?>" alt="Ghost FTP"></a>
    <div class="connection-summary"><strong id="connectionName">No server selected</strong><span id="connectionMeta">Choose a connection from the menu</span></div>
    <div class="topbar-search"><span aria-hidden="true">⌕</span><input id="remoteSearch" type="search" placeholder="Search server…" autocomplete="off"><button id="searchBtn" class="icon-button" type="button" aria-label="Search">↵</button></div>
    <div class="user-menu-wrap">
        <button id="userMenuBtn" class="user-menu-button" type="button" aria-haspopup="menu" aria-expanded="false"><span class="user-avatar"><?= GhostFTP_e($initial) ?></span><span class="desktop-only"><?= GhostFTP_e((string)($currentUser['name'] ?? 'User')) ?></span></button>
        <div id="userMenu" class="user-menu hidden" role="menu">
            <strong><?= GhostFTP_e((string)($currentUser['email'] ?? '')) ?></strong>
            <label class="stack tiny" for="languageSelect">Language
                <select id="languageSelect" aria-label="Language">
                    <?php foreach (I18n::languages() as $item): ?>
                        <option value="<?= GhostFTP_e((string)$item['code']) ?>" <?= $item['code'] === $language ? 'selected' : '' ?>><?= GhostFTP_e((string)$item['nativeName']) ?> · <?= GhostFTP_e((string)$item['englishName']) ?></option>
                    <?php endforeach; ?>
                </select>
            </label>
            <a href="<?= GhostFTP_e(GhostFTP_url('account')) ?>">My account</a>
            <?php if (($currentUser['role'] ?? '') === 'admin'): ?><a href="<?= GhostFTP_e(GhostFTP_url('users')) ?>">Users</a><a href="<?= GhostFTP_e(GhostFTP_url('settings')) ?>">Settings</a><?php endif; ?>
            <a href="<?= GhostFTP_e(GhostFTP_url('diagnostics')) ?>">Diagnostics</a>
            <button type="button" data-install-app>Install app</button>
            <a class="danger-text" href="<?= GhostFTP_e(GhostFTP_url('logout')) ?>">Sign out</a>
        </div>
    </div>
</header>

<div id="sidebarBackdrop" class="sidebar-backdrop hidden"></div>
<aside id="sidebar" class="sidebar" aria-label="Servers and favorites">
    <div class="sidebar-heading"><div><span class="eyebrow">Connections</span><strong>Servers</strong></div><button id="addProfile" class="icon-button" type="button" aria-label="Add connection">＋</button></div>
    <div id="profiles" class="profile-list"></div>
    <div class="sidebar-section"><div class="sidebar-heading small-heading"><div><span class="eyebrow">Quick access</span><strong>Favorites</strong></div></div><div id="favorites" class="favorite-list"><span class="muted tiny">Select a server.</span></div></div>
    <nav class="app-nav">
        <a href="<?= GhostFTP_e(GhostFTP_url('account')) ?>">● My account</a>
        <?php if (($currentUser['role'] ?? '') === 'admin'): ?><a href="<?= GhostFTP_e(GhostFTP_url('users')) ?>">◎ Users</a><?php endif; ?>
        <a href="<?= GhostFTP_e(GhostFTP_url('diagnostics')) ?>">◇ Diagnostics</a>
        <button type="button" data-install-app>↓ Install Ghost FTP</button>
    </nav>
    <div class="sidebar-footer"><span id="connectionStatus" class="status-dot offline"></span><span id="statusText">Not connected</span><small>v<?= GhostFTP_e(GhostFTP_VERSION) ?></small></div>
</aside>

<main id="workspace" class="workspace" tabindex="-1">
    <section id="welcome" class="welcome-card">
        <img src="<?= GhostFTP_e(GhostFTP_asset('images/mark.svg')) ?>" class="welcome-mark" alt="">
        <p class="eyebrow">FTP · FTPS · SFTP</p><h1>Files under control.</h1>
        <p><?= GhostFTP_e(I18n::t($language, 'app.subtitle')) ?>. Secure shared-hosting file management without a CDN, telemetry or an external backend.</p>
        <div class="welcome-actions"><button id="welcomeAdd" class="button primary" type="button">Add server</button><a class="button ghost" href="<?= GhostFTP_e(GhostFTP_url('diagnostics')) ?>">Check hosting</a></div>
    </section>

    <section id="fileApp" class="file-app hidden">
        <div class="toolbar">
            <div class="toolbar-group"><button id="upBtn" class="icon-button" type="button" title="Up">↑</button><button id="refreshBtn" class="icon-button" type="button" title="Refresh">↻</button><button id="favoriteBtn" class="icon-button" type="button" title="Favorite">☆</button></div>
            <div id="breadcrumbs" class="breadcrumbs" aria-label="Path"></div>
            <div class="toolbar-group toolbar-main-actions">
                <label class="button ghost compact">Upload<input id="uploadInput" type="file" multiple hidden></label>
                <label class="button ghost compact desktop-only">Folder<input id="folderUploadInput" type="file" webkitdirectory multiple hidden></label>
                <button id="newFileBtn" class="button ghost compact desktop-only" type="button">New file</button>
                <button id="newFolderBtn" class="button primary compact" type="button">New folder</button>
            </div>
        </div>
        <div class="file-controls">
            <label class="filter-box">⌕ <input id="filterInput" type="search" placeholder="Filter this folder…"></label>
            <label class="inline-check"><input id="showHidden" type="checkbox" checked> hidden files</label>
            <select id="uploadConflict" aria-label="Upload conflict policy"><option value="rename">Conflict: rename</option><option value="overwrite">Conflict: overwrite</option><option value="skip">Conflict: skip</option></select>
        </div>
        <div id="selectionBar" class="selection-bar hidden"><strong id="selectionCount">0 selected</strong><button id="bulkDownload" class="button ghost compact" type="button">ZIP download</button><button id="bulkCopy" class="button ghost compact" type="button">Copy</button><button id="bulkMove" class="button ghost compact" type="button">Move</button><button id="bulkZip" class="button ghost compact" type="button">ZIP</button><button id="bulkDelete" class="button danger compact" type="button">Delete</button><button id="clearSelection" class="icon-button" type="button" aria-label="Clear selection">×</button></div>
        <div id="dropZone" class="file-table-wrap">
            <table class="file-table"><thead><tr><th class="check-col"><input id="selectAll" type="checkbox" aria-label="Select all"></th><th><button class="sort-button" data-sort="name" type="button">Name ↕</button></th><th class="desktop-only">Type</th><th><button class="sort-button" data-sort="size" type="button">Size ↕</button></th><th class="desktop-only"><button class="sort-button" data-sort="modified" type="button">Modified ↕</button></th><th class="actions-col">Actions</th></tr></thead><tbody id="fileRows"></tbody></table>
            <div id="emptyState" class="empty-state hidden"><strong>This folder is empty.</strong><span>Upload files or create a new folder.</span></div>
            <div id="loadingState" class="loading-state hidden">Loading…</div>
        </div>
        <footer class="workspace-footer"><span id="itemSummary">0 items</span><span id="pathStatus">/</span></footer>
    </section>
</main>

<div id="transferPanel" class="transfer-panel hidden"><div><strong id="transferTitle">Transfer</strong><span id="transferDetail">Preparing…</span></div><progress id="transferProgress" max="100" value="0"></progress><button id="cancelUpload" class="button ghost compact hidden" type="button">Cancel upload</button></div>
</div>

<div id="profileModal" class="modal hidden" role="dialog" aria-modal="true" aria-labelledby="profileTitle"><div class="modal-card profile-card">
    <div class="modal-head"><div><p class="eyebrow"><?= GhostFTP_e(I18n::t($language, 'connection.title')) ?></p><h2 id="profileTitle">New server</h2></div><button class="icon-button close-profile" type="button" aria-label="Close">×</button></div>
    <form id="profileForm" class="stack" autocomplete="off">
        <input type="hidden" name="id"><div class="form-grid two"><label>Name<input name="label" maxlength="80" required></label><label><?= GhostFTP_e(I18n::t($language, 'connection.protocol')) ?><select name="protocol"><option value="sftp">SFTP</option><option value="ftps">FTPS</option><option value="ftp">FTP</option></select></label></div>
        <div class="form-grid host-grid"><label><?= GhostFTP_e(I18n::t($language, 'connection.host')) ?><input name="host" maxlength="255" spellcheck="false" required></label><label>Port<input name="port" inputmode="numeric" maxlength="5" value="22" required></label></div>
        <label>Initial path<input name="base_path" value="/" spellcheck="false" required></label>
        <div class="form-grid two"><label><?= GhostFTP_e(I18n::t($language, 'connection.username')) ?><input name="username" maxlength="512" autocomplete="username" required></label><label><?= GhostFTP_e(I18n::t($language, 'connection.password')) ?><input name="password" type="password" maxlength="4096" autocomplete="new-password" placeholder="leave empty to keep the existing secret"></label></div>
        <div class="form-grid two"><label>Timeout (s)<input name="timeout" inputmode="numeric" value="30" maxlength="3"></label><label>SFTP auth<select name="auth_method"><option value="password">Password</option><option value="key">SSH key</option></select></label></div>
        <label class="sftp-only">SFTP SHA-256 host fingerprint<input name="host_fingerprint" maxlength="200" spellcheck="false" placeholder="SHA256:… or 64-digit HEX"></label>
        <div class="key-auth stack hidden"><label>Public key<textarea name="public_key" rows="3" spellcheck="false"></textarea></label><label>Private key<textarea name="private_key" rows="5" spellcheck="false"></textarea></label><label>Passphrase<input name="key_passphrase" type="password" maxlength="2048"></label></div>
        <div class="form-grid two"><label class="inline-check"><input name="passive" type="checkbox" checked> FTP PASV</label><label class="inline-check"><input name="utf8" type="checkbox" checked> FTP UTF-8</label></div>
        <div class="form-actions"><button id="deleteProfileBtn" class="button danger hidden" type="button">Delete</button><button id="duplicateProfileBtn" class="button ghost hidden" type="button">Duplicate</button><button id="testProfileBtn" class="button ghost" type="button">Test</button><span class="spacer"></span><button class="button ghost close-profile" type="button">Cancel</button><button class="button primary" type="submit">Save</button></div>
    </form>
</div></div>

<div id="editorModal" class="modal hidden" role="dialog" aria-modal="true"><div class="modal-card editor-card"><div class="modal-head"><div><p class="eyebrow">Remote editor</p><h2 id="editorTitle">File</h2><span id="editorInfo" class="muted tiny"></span></div><button class="icon-button close-editor" type="button" aria-label="Close">×</button></div><textarea id="editorContent" class="code-editor" spellcheck="false"></textarea><div class="editor-footer"><span>Ctrl/Cmd + S to save</span><span id="editorCursor">Line 1, column 1</span><button id="saveEditor" class="button primary compact" type="button">Save</button></div></div></div>

<div id="promptModal" class="modal hidden" role="dialog" aria-modal="true"><div class="modal-card prompt-card"><div class="modal-head"><div><p class="eyebrow" id="promptEyebrow">Action</p><h2 id="promptTitle">Input</h2></div><button class="icon-button close-prompt" type="button" aria-label="Close">×</button></div><form id="promptForm" class="stack"><p id="promptDescription" class="muted hidden"></p><label id="promptInputLabel">Value<input id="promptInput" autocomplete="off"></label><div id="promptExtra"></div><div class="form-actions"><span class="spacer"></span><button class="button ghost close-prompt" type="button">Cancel</button><button class="button primary" type="submit">Confirm</button></div></form></div></div>

<div id="searchModal" class="modal hidden" role="dialog" aria-modal="true"><div class="modal-card search-card"><div class="modal-head"><div><p class="eyebrow">Recursive search</p><h2>Results</h2></div><button class="icon-button close-search" type="button" aria-label="Close">×</button></div><div id="searchStatus" class="muted"></div><div id="searchResults" class="search-results"></div></div></div>

<div id="previewModal" class="modal hidden" role="dialog" aria-modal="true"><div class="modal-card preview-card"><div class="modal-head"><h2 id="previewTitle">Preview</h2><button class="icon-button close-preview" type="button" aria-label="Close">×</button></div><div class="preview-stage"><img id="previewImage" alt="Remote image preview"></div></div></div>

<nav id="contextMenu" class="context-menu hidden" aria-label="File actions"><button data-action="open" type="button">Open</button><button data-action="download" type="button">Download</button><button data-action="edit" type="button">Edit</button><button data-action="preview" type="button">Image preview</button><hr><button data-action="duplicate" type="button">Duplicate</button><button data-action="copy" type="button">Copy to…</button><button data-action="move" type="button">Move to…</button><button data-action="rename" type="button">Rename</button><button data-action="chmod" type="button">Permissions</button><button data-action="extract" type="button">Extract ZIP</button><hr><button class="danger-text" data-action="delete" type="button">Delete</button></nav>

<div id="installModal" class="modal hidden" role="dialog" aria-modal="true"><div class="modal-card install-card"><div class="modal-head"><div><p class="eyebrow">PWA</p><h2>Install Ghost FTP</h2></div><button class="icon-button" type="button" data-close-install aria-label="Close">×</button></div><div class="install-visual"><img src="<?= GhostFTP_e(GhostFTP_asset('images/mark.svg')) ?>" alt="Ghost FTP"><p>Add Ghost FTP to your home screen. Authenticated pages and FTP data are never stored in the offline cache.</p></div><ol data-ios-install hidden><li>In Safari choose <strong>Share</strong>.</li><li>Choose <strong>Add to Home Screen</strong>.</li></ol><p data-generic-install>Choose <strong>Install app</strong> from your browser menu.</p></div></div>

<div id="toastHost" class="toast-host" aria-live="polite"></div>
<script src="<?= GhostFTP_e(GhostFTP_asset('js/pwa.js')) ?>" defer></script>
<script type="module" src="<?= GhostFTP_e(GhostFTP_asset('js/language.js')) ?>"></script>
<script type="module" src="<?= GhostFTP_e(GhostFTP_asset('js/app.js')) ?>"></script>
</body>
</html>
