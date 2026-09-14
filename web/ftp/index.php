<?php
declare(strict_types=1);
require __DIR__ . '/config.php';
header('X-Content-Type-Options: nosniff');
header('X-Frame-Options: DENY');
header('Referrer-Policy: no-referrer');
header("Content-Security-Policy: default-src 'self'; img-src 'self'; style-src 'self'; script-src 'self'; connect-src 'self'; form-action 'self'; frame-ancestors 'none'; object-src 'none'");
?>
<!doctype html>
<html lang="en">
<head>
  <meta charset="utf-8">
  <meta name="viewport" content="width=device-width, initial-scale=1">
  <title>Ghost FTP Web</title>
  <meta name="description" content="Ghost FTP Web for FTP, explicit FTPS and verified SFTP connections.">
  <link rel="icon" href="../assets/icon.svg" type="image/svg+xml">
  <link rel="stylesheet" href="assets/app.css">
  <script defer src="assets/app.js"></script>
</head>
<body>
  <div class="app">
    <header class="bar appbar">
      <a class="app-logo" href="../"><img src="../assets/logo.svg" alt="Ghost FTP"></a>
      <span class="product-label">Web FTP <?php echo htmlspecialchars(GHOSTFTP_WEB_VERSION, ENT_QUOTES, 'UTF-8'); ?></span>
      <span class="spacer"></span>
      <a href="../security.html">Security</a>
      <a href="../privacy.html">Privacy</a>
      <button id="theme-toggle" class="icon-button" type="button" aria-label="Switch theme" title="Switch theme">◐</button>
    </header>

    <form id="connection" class="connection" autocomplete="off">
      <select id="protocol" aria-label="Protocol">
        <option value="ftps">Explicit FTPS</option>
        <option value="sftp">SFTP</option>
        <option value="ftp">FTP</option>
      </select>
      <input id="host" placeholder="Server address" aria-label="Host" required maxlength="253" autocapitalize="none" spellcheck="false">
      <input id="port" type="number" min="1" max="65535" value="21" aria-label="Port" required>
      <input id="username" placeholder="Username" aria-label="Username" required autocomplete="off" autocapitalize="none" spellcheck="false">
      <input id="password" type="password" placeholder="Password" aria-label="Password" required autocomplete="new-password">
      <input id="root" value="/" placeholder="Remote folder" aria-label="Remote root" autocapitalize="none" spellcheck="false">
      <input id="fingerprint" class="fingerprint" placeholder="SHA256:server-host-key…" aria-label="SFTP SHA-256 fingerprint" autocomplete="off" autocapitalize="none" spellcheck="false">
      <button class="primary" type="submit">Connect</button>
      <button id="disconnect" type="button" disabled>Disconnect</button>
    </form>

    <main class="workspace">
      <section class="pane" aria-label="Local files">
        <div class="pane-head">
          <strong>Local files</strong>
          <button id="upload-selected" type="button">Upload all</button>
          <label class="file-picker"><input id="file-input" type="file" multiple hidden><span class="btnlike">Choose files</span></label>
        </div>
        <div class="files">
          <table>
            <thead><tr><th>Name</th><th>Size</th><th>State</th></tr></thead>
            <tbody id="local-body"></tbody>
          </table>
          <div class="local-drop">Choose the files you want to upload. Selected files stay in this tab only.</div>
        </div>
      </section>

      <section class="pane" aria-label="Remote files">
        <div class="pane-head">
          <strong>Remote files</strong>
          <span id="remote-path" class="path">Not connected</span>
          <button id="up" type="button">Up</button>
          <button id="refresh" type="button">Refresh</button>
        </div>
        <div class="files">
          <table>
            <thead><tr><th>Name</th><th>Size</th><th>Modified</th><th>Mode</th></tr></thead>
            <tbody id="remote-body"><tr><td colspan="4" class="empty">Connect to a server to browse remote files.</td></tr></tbody>
          </table>
        </div>
        <div class="bar actionbar">
          <button id="mkdir" type="button">New folder</button>
          <button id="rename" type="button">Rename</button>
          <button id="download" type="button">Download</button>
          <button id="edit" type="button">Edit text</button>
          <button id="chmod" type="button">Permissions</button>
          <button id="delete" class="danger" type="button">Delete</button>
        </div>
      </section>
    </main>

    <section class="queue" aria-label="Activity">
      <div class="queue-head">Activity</div>
      <div id="queue" aria-live="polite"></div>
    </section>
    <footer id="status" class="status" role="status" aria-live="polite">Ready. Passwords are not saved.</footer>
  </div>

  <div id="editor" class="modal" role="dialog" aria-modal="true" aria-labelledby="editor-name">
    <div class="dialog">
      <div class="bar"><h2 id="editor-name">Edit text file</h2></div>
      <textarea id="editor-text" spellcheck="false" aria-label="Remote text file"></textarea>
      <div class="dialog-actions">
        <button id="editor-close" type="button">Close</button>
        <button id="editor-save" class="primary" type="button">Save changes</button>
      </div>
    </div>
  </div>
</body>
</html>
