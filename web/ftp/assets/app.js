(() => {
  'use strict';

  const $ = (selector, root = document) => root.querySelector(selector);
  const $$ = (selector, root = document) => [...root.querySelectorAll(selector)];

  const form = $('#connection');
  const protocol = $('#protocol');
  const fingerprint = $('#fingerprint');
  const connectButton = form.querySelector('button[type="submit"]');
  const disconnectButton = $('#disconnect');
  const remoteBody = $('#remote-body');
  const localBody = $('#local-body');
  const remotePath = $('#remote-path');
  const status = $('#status');
  const queue = $('#queue');
  const editor = $('#editor');
  const editorText = $('#editor-text');
  const editorName = $('#editor-name');
  const themeToggle = $('#theme-toggle');
  const uploadSelected = $('#upload-selected');
  const remoteActionIds = ['rename', 'download', 'edit', 'chmod', 'delete'];

  let profile = null;
  let current = '/';
  let selected = null;
  let editTarget = null;
  let localFiles = [];
  let manualTheme = false;
  let lightTheme = window.matchMedia('(prefers-color-scheme: light)').matches;

  function applyTheme() {
    document.documentElement.dataset.theme = lightTheme ? 'light' : 'dark';
    themeToggle.textContent = lightTheme ? '☾' : '☀';
    themeToggle.setAttribute('aria-label', lightTheme ? 'Switch to dark theme' : 'Switch to light theme');
    themeToggle.title = lightTheme ? 'Dark theme' : 'Light theme';
  }

  function setStatus(message) {
    status.textContent = String(message || 'Ready.');
  }

  function safeError(error) {
    const value = error instanceof Error ? error.message : String(error || '');
    if (!value || value.length > 180 || /(?:trace|exception|\/home\/|[A-Z]:\\|\.php:\d+)/i.test(value)) {
      return 'The operation could not be completed. Check the connection and try again.';
    }
    return value;
  }

  function join(base, name) {
    if (base === '/') return '/' + name;
    return base.replace(/\/$/, '') + '/' + name;
  }

  function parent(path) {
    const parts = path.split('/').filter(Boolean);
    parts.pop();
    return '/' + parts.join('/');
  }

  function esc(value) {
    return String(value ?? '').replace(/[&<>"']/g, (character) => ({
      '&': '&amp;',
      '<': '&lt;',
      '>': '&gt;',
      '"': '&quot;',
      "'": '&#39;'
    }[character]));
  }

  function addQueue(kind, name, state = 'Done') {
    const row = document.createElement('div');
    row.className = 'queue-row ' + (state === 'Done' ? 'ok' : '');
    row.innerHTML = `<span>${esc(kind)}</span><span>${esc(name)}</span><span>${esc(state)}</span>`;
    queue.prepend(row);
  }

  function updateRemoteActions() {
    const connected = Boolean(profile);
    $('#mkdir').disabled = !connected;
    $('#up').disabled = !connected;
    $('#refresh').disabled = !connected;
    for (const id of remoteActionIds) {
      const button = $('#' + id);
      const needsFile = id === 'download' || id === 'edit';
      button.disabled = !connected || !selected || (needsFile && selected.type === 'dir');
    }
  }

  async function api(data) {
    const response = await fetch('api.php', {
      method: 'POST',
      headers: {'Content-Type': 'application/json'},
      body: JSON.stringify({...data, profile})
    });
    const type = response.headers.get('content-type') || '';
    if (!type.includes('application/json')) {
      throw new Error('The server returned an unexpected response.');
    }
    const output = await response.json();
    if (!response.ok || !output.ok) {
      throw new Error(output.error || 'The operation could not be completed.');
    }
    return output;
  }

  function collectProfile() {
    const value = (id) => $('#' + id).value.trim();
    return {
      protocol: value('protocol'),
      host: value('host'),
      port: Number(value('port')),
      username: value('username'),
      password: $('#password').value,
      root: value('root') || '/',
      fingerprint: value('fingerprint')
    };
  }

  async function list(path) {
    setStatus('Loading folder…');
    const output = await api({action: 'list', path});
    current = path;
    remotePath.textContent = path;
    selected = null;
    renderRemote(output.items || []);
    setStatus(`Connected · ${profile.protocol.toUpperCase()} · ${profile.host}`);
    updateRemoteActions();
  }

  function renderRemote(items) {
    remoteBody.innerHTML = '';
    if (!items.length) {
      remoteBody.innerHTML = '<tr><td colspan="4" class="empty">This folder is empty.</td></tr>';
      updateRemoteActions();
      return;
    }

    for (const item of items) {
      const row = document.createElement('tr');
      row.dataset.kind = item.type;
      row.dataset.name = item.name;
      row.innerHTML = `<td>${item.type === 'dir' ? '▸ ' : ''}${esc(item.name)}</td><td>${item.size == null ? '—' : Number(item.size).toLocaleString()}</td><td>${esc(item.modified || '—')}</td><td>${esc(item.permissions || '—')}</td>`;
      row.addEventListener('click', () => {
        $$('tr.selected', remoteBody).forEach((element) => element.classList.remove('selected'));
        row.classList.add('selected');
        selected = item;
        updateRemoteActions();
      });
      row.addEventListener('dblclick', () => {
        if (item.type === 'dir') {
          list(join(current, item.name)).catch(showError);
        } else {
          editFile(item).catch(showError);
        }
      });
      remoteBody.append(row);
    }
    updateRemoteActions();
  }

  function renderLocal() {
    localBody.innerHTML = '';
    uploadSelected.disabled = localFiles.length === 0 || !profile;
    if (!localFiles.length) {
      localBody.innerHTML = '<tr><td colspan="3" class="empty">Choose files to prepare them for upload.</td></tr>';
      return;
    }

    for (const file of localFiles) {
      const row = document.createElement('tr');
      row.innerHTML = `<td>${esc(file.name)}</td><td>${file.size.toLocaleString()}</td><td>Ready</td>`;
      row.addEventListener('dblclick', () => upload(file).catch(showError));
      localBody.append(row);
    }
  }

  function showError(error) {
    const message = safeError(error);
    setStatus(message);
    addQueue('Error', message, 'Failed');
  }

  function closeEditor() {
    editor.classList.remove('open');
    editorText.value = '';
    editTarget = null;
  }

  form.addEventListener('submit', async (event) => {
    event.preventDefault();
    if (connectButton.disabled) return;
    profile = collectProfile();
    connectButton.disabled = true;
    setStatus('Connecting…');
    try {
      await list(profile.root || '/');
      disconnectButton.disabled = false;
      renderLocal();
    } catch (error) {
      profile = null;
      showError(error);
      updateRemoteActions();
    } finally {
      connectButton.disabled = false;
    }
  });

  protocol.addEventListener('change', () => {
    const isSftp = protocol.value === 'sftp';
    form.classList.toggle('sftp', isSftp);
    $('#port').value = isSftp ? '22' : '21';
    fingerprint.required = isSftp;
  });

  disconnectButton.addEventListener('click', () => {
    profile = null;
    selected = null;
    current = '/';
    remotePath.textContent = 'Not connected';
    remoteBody.innerHTML = '<tr><td colspan="4" class="empty">Connect to a server to browse remote files.</td></tr>';
    $('#password').value = '';
    fingerprint.value = '';
    disconnectButton.disabled = true;
    setStatus('Disconnected. Password cleared from this tab.');
    updateRemoteActions();
    renderLocal();
  });

  $('#refresh').addEventListener('click', () => profile && list(current).catch(showError));
  $('#up').addEventListener('click', () => profile && list(parent(current)).catch(showError));

  $('#file-input').addEventListener('change', (event) => {
    localFiles = [...event.target.files];
    renderLocal();
  });

  async function upload(file) {
    if (!profile) throw new Error('Connect to a server first.');
    const data = new FormData();
    data.append('action', 'upload');
    data.append('profile', JSON.stringify(profile));
    data.append('path', join(current, file.name));
    data.append('file', file, file.name);
    setStatus('Uploading ' + file.name + '…');
    const response = await fetch('api.php', {method: 'POST', body: data});
    const output = await response.json().catch(() => ({ok: false}));
    if (!response.ok || !output.ok) throw new Error(output.error || 'Upload failed.');
    addQueue('Upload', file.name);
    await list(current);
  }

  uploadSelected.addEventListener('click', async () => {
    uploadSelected.disabled = true;
    try {
      for (const file of localFiles) {
        await upload(file);
      }
    } catch (error) {
      showError(error);
    } finally {
      renderLocal();
    }
  });

  $('#mkdir').addEventListener('click', async () => {
    if (!profile) return;
    const name = prompt('New folder name');
    if (!name) return;
    try {
      await api({action: 'mkdir', target: join(current, name)});
      addQueue('New folder', name);
      await list(current);
    } catch (error) {
      showError(error);
    }
  });

  $('#rename').addEventListener('click', async () => {
    if (!selected) return setStatus('Select a remote item first.');
    const name = prompt('New name', selected.name);
    if (!name || name === selected.name) return;
    try {
      await api({action: 'rename', from: join(current, selected.name), to: join(current, name)});
      addQueue('Rename', selected.name + ' → ' + name);
      await list(current);
    } catch (error) {
      showError(error);
    }
  });

  $('#delete').addEventListener('click', async () => {
    if (!selected) return setStatus('Select a remote item first.');
    if (!confirm(`Delete ${selected.name}? This cannot be undone.`)) return;
    try {
      await api({action: 'delete', target: join(current, selected.name), directory: selected.type === 'dir'});
      addQueue('Delete', selected.name);
      await list(current);
    } catch (error) {
      showError(error);
    }
  });

  $('#chmod').addEventListener('click', async () => {
    if (!selected) return setStatus('Select a remote item first.');
    const mode = prompt('Permissions (for example 0644)', selected.permissions || '0644');
    if (!mode) return;
    try {
      await api({action: 'chmod', target: join(current, selected.name), mode});
      addQueue('Permissions', selected.name + ' · ' + mode);
      await list(current);
    } catch (error) {
      showError(error);
    }
  });

  $('#download').addEventListener('click', async () => {
    if (!selected || selected.type === 'dir') return setStatus('Select a remote file first.');
    try {
      setStatus('Downloading ' + selected.name + '…');
      const response = await fetch('api.php', {
        method: 'POST',
        headers: {'Content-Type': 'application/json'},
        body: JSON.stringify({action: 'download', profile, target: join(current, selected.name)})
      });
      if (!response.ok) {
        const output = await response.json().catch(() => ({}));
        throw new Error(output.error || 'Download failed.');
      }
      const blob = await response.blob();
      const url = URL.createObjectURL(blob);
      const anchor = document.createElement('a');
      anchor.href = url;
      anchor.download = selected.name;
      document.body.append(anchor);
      anchor.click();
      anchor.remove();
      URL.revokeObjectURL(url);
      addQueue('Download', selected.name);
      setStatus('Download complete.');
    } catch (error) {
      showError(error);
    }
  });

  async function editFile(item = selected) {
    if (!item || item.type === 'dir') throw new Error('Select a text file first.');
    const target = join(current, item.name);
    const output = await api({action: 'read', target});
    editTarget = target;
    editorName.textContent = item.name;
    editorText.value = output.content;
    editor.classList.add('open');
    editorText.focus();
  }

  $('#edit').addEventListener('click', () => editFile().catch(showError));
  $('#editor-close').addEventListener('click', closeEditor);
  $('#editor-save').addEventListener('click', async () => {
    if (!editTarget) return;
    try {
      await api({action: 'write', target: editTarget, content: editorText.value});
      addQueue('Edit', editTarget);
      closeEditor();
      await list(current);
    } catch (error) {
      showError(error);
    }
  });

  editor.addEventListener('click', (event) => {
    if (event.target === editor) closeEditor();
  });

  document.addEventListener('keydown', (event) => {
    if (event.key === 'Escape' && editor.classList.contains('open')) closeEditor();
  });

  themeToggle.addEventListener('click', () => {
    manualTheme = true;
    lightTheme = !lightTheme;
    applyTheme();
  });

  window.matchMedia('(prefers-color-scheme: light)').addEventListener('change', (event) => {
    if (manualTheme) return;
    lightTheme = event.matches;
    applyTheme();
  });

  window.addEventListener('beforeunload', () => {
    profile = null;
    $('#password').value = '';
    fingerprint.value = '';
    editorText.value = '';
  });

  applyTheme();
  renderLocal();
  protocol.dispatchEvent(new Event('change'));
  updateRemoteActions();
})();
