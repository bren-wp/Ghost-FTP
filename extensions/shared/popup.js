(() => {
  'use strict';

  const api = globalThis.browser || globalThis.chrome;
  const UI_PORT_NAME = 'ghostftp-ui';
  const REQUEST_TIMEOUT_MS = 50000;

  const state = {
    port: null,
    requestCounter: 0,
    pending: new Map(),
    profiles: [],
    connected: false,
    connection: null,
    localRoot: '',
    localPath: '',
    localItems: [],
    localSelection: -1,
    remotePath: '/',
    remoteItems: [],
    remoteSelection: -1,
    transfers: new Map(),
    transferSince: 0,
    pendingTrust: null
  };

  const el = {};

  function byId(id) {
    return document.getElementById(id);
  }

  function cacheElements() {
    const ids = [
      'bridge-state', 'bridge-state-text', 'status', 'transfer-count', 'connection-badge',
      'connection-target', 'fill-address', 'profile-select', 'new-profile', 'connection-form',
      'profile-name', 'protocol', 'port', 'host', 'username', 'password', 'sftp-fields',
      'private-key', 'passphrase', 'remote-start', 'local-start', 'save-password',
      'clear-password', 'save-passphrase-row', 'save-passphrase', 'clear-passphrase-row',
      'clear-passphrase', 'connect-button', 'disconnect-button', 'save-profile', 'delete-profile',
      'trust-panel', 'trust-fingerprint', 'remember-fingerprint', 'trust-accept', 'trust-cancel',
      'choose-local-root', 'local-up', 'local-path', 'local-refresh', 'local-items', 'local-mkdir',
      'local-rename', 'local-delete', 'upload-selected', 'remote-up', 'remote-path', 'remote-go',
      'remote-refresh', 'remote-items', 'remote-mkdir', 'remote-rename', 'remote-delete',
      'download-selected', 'pause-transfers', 'resume-transfers', 'clear-transfers',
      'transfer-list', 'version-label'
    ];
    for (const id of ids) {
      el[id] = byId(id);
    }
  }

  function setStatus(message, kind = 'neutral') {
    el.status.textContent = message || '';
    el.status.dataset.kind = kind;
  }

  function setBridgeStatus(available, message = '') {
    el['bridge-state'].dataset.kind = available ? 'success' : (message ? 'error' : 'neutral');
    el['bridge-state-text'].textContent = available ? 'Local bridge ready' : (message || 'Checking local bridge…');
  }

  function nextRequestId() {
    state.requestCounter = (state.requestCounter + 1) % Number.MAX_SAFE_INTEGER;
    return `ui-${Date.now().toString(36)}-${state.requestCounter.toString(36)}`;
  }

  function rejectPending(error) {
    for (const entry of state.pending.values()) {
      clearTimeout(entry.timer);
      entry.reject(error);
    }
    state.pending.clear();
  }

  function requestNative(type, params = {}) {
    if (!state.port) {
      return Promise.reject(new Error('Ghost FTP local bridge is not connected.'));
    }
    const id = nextRequestId();
    return new Promise((resolve, reject) => {
      const timer = setTimeout(() => {
        state.pending.delete(id);
        reject(new Error('The Ghost FTP operation timed out.'));
      }, REQUEST_TIMEOUT_MS);
      state.pending.set(id, {resolve, reject, timer});
      try {
        state.port.postMessage({kind: 'request', request: {id, type, params}});
      } catch (_error) {
        clearTimeout(timer);
        state.pending.delete(id);
        reject(new Error('Ghost FTP local bridge could not accept the request.'));
      }
    });
  }

  function handleResponse(response) {
    if (!response || typeof response.id !== 'string') {
      return;
    }
    const entry = state.pending.get(response.id);
    if (!entry) {
      return;
    }
    clearTimeout(entry.timer);
    state.pending.delete(response.id);
    if (response.ok === true) {
      entry.resolve(response.result);
      return;
    }
    const message = response.error && typeof response.error.message === 'string'
      ? response.error.message
      : 'Ghost FTP could not complete the operation.';
    const error = new Error(message);
    error.code = response.error && response.error.code ? response.error.code : 'operation_failed';
    entry.reject(error);
  }

  function handlePortMessage(message) {
    if (!message || typeof message !== 'object') {
      return;
    }
    if (message.kind === 'response') {
      handleResponse(message.response);
      return;
    }
    if (message.kind === 'bridge-status') {
      if (message.available) {
        setBridgeStatus(true);
      } else if (message.message) {
        setBridgeStatus(false, message.message);
      }
      return;
    }
    if (message.kind === 'transfer-events' && message.result) {
      applyTransferEvents(message.result);
    }
  }

  function connectUiPort() {
    state.port = api.runtime.connect({name: UI_PORT_NAME});
    state.port.onMessage.addListener(handlePortMessage);
    state.port.onDisconnect.addListener(() => {
      state.port = null;
      rejectPending(new Error('Ghost FTP browser connection closed. Reopen the extension to continue.'));
      setBridgeStatus(false, 'Browser connection closed');
    });
  }

  function switchTab(name) {
    for (const tab of document.querySelectorAll('[data-tab]')) {
      const active = tab.dataset.tab === name;
      tab.classList.toggle('is-active', active);
      tab.setAttribute('aria-selected', active ? 'true' : 'false');
    }
    for (const panel of document.querySelectorAll('[data-panel]')) {
      const active = panel.dataset.panel === name;
      panel.classList.toggle('is-active', active);
      panel.hidden = !active;
    }
  }

  function defaultPort(protocol) {
    return protocol === 'sftp' ? 22 : 21;
  }

  function defaultRemotePath(protocol) {
    return protocol === 'sftp' ? '.' : '/';
  }

  function updateProtocolFields(resetPort = false) {
    const protocol = el.protocol.value;
    const isSftp = protocol === 'sftp';
    el['sftp-fields'].hidden = !isSftp;
    el['save-passphrase-row'].hidden = !isSftp;
    el['clear-passphrase-row'].hidden = !isSftp;
    if (resetPort) {
      el.port.value = String(defaultPort(protocol));
    }
    const currentRemote = el['remote-start'].value.trim();
    if (!currentRemote || currentRemote === '/' || currentRemote === '.') {
      el['remote-start'].value = defaultRemotePath(protocol);
    }
  }

  function currentProfile() {
    const id = el['profile-select'].value;
    return state.profiles.find((profile) => profile.id === id) || null;
  }

  function clearSecrets() {
    el.password.value = '';
    el.passphrase.value = '';
    el['save-password'].checked = false;
    el['save-passphrase'].checked = false;
  }

  function resetProfileForm() {
    el['profile-select'].value = '';
    el['profile-name'].value = '';
    el.protocol.value = 'ftp';
    el.port.value = '21';
    el.host.value = '';
    el.username.value = '';
    el['private-key'].value = '';
    el['remote-start'].value = '/';
    el['local-start'].value = '';
    el['clear-password'].checked = false;
    el['clear-passphrase'].checked = false;
    el.password.placeholder = 'Not saved in browser';
    el.passphrase.placeholder = 'Not saved in browser';
    clearSecrets();
    updateProtocolFields(false);
    updateProfileButtons();
  }

  function populateProfile(profile) {
    if (!profile) {
      resetProfileForm();
      return;
    }
    el['profile-select'].value = profile.id;
    el['profile-name'].value = profile.name || '';
    el.protocol.value = profile.protocol || 'ftp';
    el.port.value = String(profile.port || defaultPort(profile.protocol));
    el.host.value = profile.host || '';
    el.username.value = profile.username || '';
    el['private-key'].value = profile.privateKeyPath || '';
    el['remote-start'].value = profile.remotePath || defaultRemotePath(profile.protocol);
    el['local-start'].value = profile.localPath || '';
    el.password.value = '';
    el.passphrase.value = '';
    el.password.placeholder = profile.hasPassword ? 'Saved securely by Ghost FTP' : 'Not saved in browser';
    el.passphrase.placeholder = profile.hasPassphrase ? 'Saved securely by Ghost FTP' : 'Not saved in browser';
    el['save-password'].checked = false;
    el['clear-password'].checked = false;
    el['save-passphrase'].checked = false;
    el['clear-passphrase'].checked = false;
    updateProtocolFields(false);
    updateProfileButtons();
  }

  function updateProfileButtons() {
    const selected = Boolean(el['profile-select'].value);
    el['delete-profile'].disabled = !selected || state.connected;
  }

  function renderProfiles() {
    const selected = el['profile-select'].value;
    el['profile-select'].replaceChildren();
    const fresh = document.createElement('option');
    fresh.value = '';
    fresh.textContent = 'New connection';
    el['profile-select'].appendChild(fresh);
    for (const profile of state.profiles) {
      const option = document.createElement('option');
      option.value = profile.id;
      option.textContent = profile.name;
      el['profile-select'].appendChild(option);
    }
    if (state.profiles.some((profile) => profile.id === selected)) {
      el['profile-select'].value = selected;
    }
    updateProfileButtons();
  }

  function buildConnectionConfig() {
    const port = Number.parseInt(el.port.value, 10);
    if (!Number.isInteger(port) || port < 1 || port > 65535) {
      throw new Error('Enter a valid server port between 1 and 65535.');
    }
    return {
      protocol: el.protocol.value,
      host: el.host.value.trim(),
      port,
      username: el.username.value,
      password: el.password.value,
      privateKeyPath: el.protocol.value === 'sftp' ? el['private-key'].value.trim() : '',
      passphrase: el.protocol.value === 'sftp' ? el.passphrase.value : '',
      fingerprint: ''
    };
  }

  function buildProfileInput() {
    const profile = currentProfile();
    const port = Number.parseInt(el.port.value, 10);
    if (!Number.isInteger(port) || port < 1 || port > 65535) {
      throw new Error('Enter a valid server port between 1 and 65535.');
    }
    const input = {
      id: profile ? profile.id : '',
      name: el['profile-name'].value.trim(),
      protocol: el.protocol.value,
      host: el.host.value.trim(),
      port,
      username: el.username.value,
      password: el['save-password'].checked ? el.password.value : '',
      clearPassword: el['clear-password'].checked,
      privateKeyPath: el.protocol.value === 'sftp' ? el['private-key'].value.trim() : '',
      passphrase: el.protocol.value === 'sftp' && el['save-passphrase'].checked ? el.passphrase.value : '',
      clearPassphrase: el.protocol.value === 'sftp' && el['clear-passphrase'].checked,
      fingerprint: '',
      remotePath: el['remote-start'].value.trim() || defaultRemotePath(el.protocol.value),
      localPath: el['local-start'].value.trim()
    };
    if (!input.name) {
      throw new Error('Enter a name for the saved connection.');
    }
    if (input.clearPassword && input.password) {
      throw new Error('Choose either a new saved password or remove the saved password.');
    }
    if (input.clearPassphrase && input.passphrase) {
      throw new Error('Choose either a new saved passphrase or remove the saved passphrase.');
    }
    return input;
  }

  function updateConnectionUi() {
    el['connection-badge'].textContent = state.connected ? 'Connected' : 'Disconnected';
    el['connection-badge'].classList.toggle('is-connected', state.connected);
    el['connect-button'].disabled = state.connected;
    el['disconnect-button'].disabled = !state.connected;
    el['profile-select'].disabled = state.connected;
    el['new-profile'].disabled = state.connected;
    el['save-profile'].disabled = state.connected;
    updateProfileButtons();
    updateRemoteButtons();
  }

  async function loadProfiles() {
    state.profiles = await requestNative('profiles.list', {});
    if (!Array.isArray(state.profiles)) {
      state.profiles = [];
    }
    renderProfiles();
  }

  function matchActiveProfile(connection) {
    if (!connection) {
      return null;
    }
    return state.profiles.find((profile) =>
      profile.protocol === connection.protocol &&
      String(profile.host || '').toLowerCase() === String(connection.host || '').toLowerCase() &&
      Number(profile.port) === Number(connection.port) &&
      profile.username === connection.username
    ) || null;
  }

  async function refreshConnectionState() {
    const result = await requestNative('connection.active', {});
    state.connected = Boolean(result && result.connected);
    state.connection = state.connected ? result.connection : null;
    if (state.connected) {
      const profile = matchActiveProfile(state.connection);
      if (profile) {
        el['profile-select'].value = profile.id;
        populateProfile(profile);
        state.remotePath = profile.remotePath || defaultRemotePath(profile.protocol);
      } else {
        state.remotePath = defaultRemotePath(state.connection.protocol);
      }
      el['remote-path'].value = state.remotePath;
    }
    updateConnectionUi();
  }

  async function saveProfile() {
    const input = buildProfileInput();
    const saved = await requestNative('profiles.save', {profile: input});
    await loadProfiles();
    if (saved && saved.id) {
      el['profile-select'].value = saved.id;
      const profile = currentProfile();
      populateProfile(profile);
    }
    clearSecrets();
    setStatus('Connection saved in the protected Ghost FTP profile store.', 'success');
  }

  async function deleteProfile() {
    const profile = currentProfile();
    if (!profile) {
      return;
    }
    if (!globalThis.confirm(`Delete saved connection “${profile.name}”?`)) {
      return;
    }
    await requestNative('profiles.delete', {id: profile.id});
    await loadProfiles();
    resetProfileForm();
    setStatus('Saved connection removed.', 'success');
  }

  async function connectServer(trustFingerprint = '', rememberFingerprint = false) {
    const config = buildConnectionConfig();
    const profile = currentProfile();
    setStatus('Connecting through the local Ghost FTP engine…', 'neutral');
    const result = await requestNative('connect', {
      profileId: profile ? profile.id : '',
      config,
      trustFingerprint,
      rememberFingerprint
    });

    if (result && result.requiresTrust) {
      state.pendingTrust = {
        fingerprint: result.fingerprint,
        config,
        profileId: profile ? profile.id : ''
      };
      el['trust-fingerprint'].textContent = result.fingerprint || 'Fingerprint unavailable';
      el['remember-fingerprint'].checked = Boolean(profile);
      el['remember-fingerprint'].disabled = !profile;
      el['trust-panel'].hidden = false;
      setStatus('Verify the SFTP host key fingerprint before continuing.', 'warning');
      return;
    }

    state.pendingTrust = null;
    el['trust-panel'].hidden = true;
    state.connected = Boolean(result && result.connected);
    state.connection = config;
    state.remotePath = el['remote-start'].value.trim() || defaultRemotePath(config.protocol);
    el['remote-path'].value = state.remotePath;
    clearSecrets();
    updateConnectionUi();
    setStatus('Connected. Protocol traffic stays between this device and the server you selected.', 'success');
    await refreshRemote(state.remotePath);
    switchTab('files');
  }

  async function disconnectServer() {
    setStatus('Disconnecting safely…', 'neutral');
    await requestNative('disconnect', {});
    state.connected = false;
    state.connection = null;
    state.remoteItems = [];
    state.remoteSelection = -1;
    state.pendingTrust = null;
    el['trust-panel'].hidden = true;
    renderRemoteItems();
    updateConnectionUi();
    setStatus('Disconnected.', 'success');
  }

  async function acceptTrust() {
    if (!state.pendingTrust) {
      return;
    }
    await connectServer(state.pendingTrust.fingerprint, el['remember-fingerprint'].checked);
  }

  async function cancelTrust() {
    await requestNative('trust.cancel', {});
    state.pendingTrust = null;
    el['trust-panel'].hidden = true;
    setStatus('SFTP host-key verification cancelled.', 'neutral');
  }

  function fillFromAddress() {
    const parsed = globalThis.GhostFTPConnection.parseConnectionTarget(el['connection-target'].value);
    if (!parsed.ok) {
      throw new Error(parsed.error);
    }
    const protocol = parsed.protocol.toLowerCase();
    el.protocol.value = protocol;
    el.host.value = parsed.host;
    el.port.value = parsed.port || String(defaultPort(protocol));
    el.username.value = parsed.username || '';
    el['remote-start'].value = parsed.path || defaultRemotePath(protocol);
    el.password.value = '';
    updateProtocolFields(false);
    setStatus(
      parsed.passwordDetected
        ? 'Address filled locally. The embedded password was deliberately discarded; enter it separately if needed.'
        : 'Address filled locally. URL query and fragment data are not used.',
      parsed.passwordDetected ? 'warning' : 'success'
    );
  }

  function localSeparator(path) {
    return path.includes('\\') && !path.includes('/') ? '\\' : '/';
  }

  function joinLocal(base, name) {
    const separator = localSeparator(base);
    if (base.endsWith('/') || base.endsWith('\\')) {
      return `${base}${name}`;
    }
    return `${base}${separator}${name}`;
  }

  function parentLocal(path) {
    if (!path || path === state.localRoot) {
      return state.localRoot;
    }
    const normalized = path.replace(/[\\/]+$/, '');
    const slash = Math.max(normalized.lastIndexOf('/'), normalized.lastIndexOf('\\'));
    if (slash < 0) {
      return state.localRoot;
    }
    const parent = normalized.slice(0, slash) || normalized.slice(0, slash + 1);
    if (!parent || parent.length < state.localRoot.length) {
      return state.localRoot;
    }
    return parent;
  }

  function joinRemote(base, name) {
    const normalized = (base || '/').trim();
    if (normalized === '.') {
      return `./${name}`;
    }
    if (normalized === '/') {
      return `/${name}`;
    }
    return `${normalized.replace(/\/+$/, '')}/${name}`;
  }

  function parentRemote(path) {
    const normalized = (path || '/').trim();
    if (normalized === '.' || normalized === '/') {
      return normalized;
    }
    if (normalized.startsWith('./')) {
      const rest = normalized.slice(2).replace(/\/+$/, '');
      const slash = rest.lastIndexOf('/');
      return slash < 0 ? '.' : `./${rest.slice(0, slash)}`;
    }
    const clean = normalized.replace(/\/+$/, '');
    const slash = clean.lastIndexOf('/');
    if (slash <= 0) {
      return '/';
    }
    return clean.slice(0, slash);
  }

  function formatBytes(value) {
    const bytes = Number(value) || 0;
    if (bytes <= 0) {
      return '—';
    }
    const units = ['B', 'KiB', 'MiB', 'GiB', 'TiB'];
    let amount = bytes;
    let unit = 0;
    while (amount >= 1024 && unit < units.length - 1) {
      amount /= 1024;
      unit += 1;
    }
    return `${amount >= 10 || unit === 0 ? amount.toFixed(0) : amount.toFixed(1)} ${units[unit]}`;
  }

  function formatSpeed(value) {
    const speed = Number(value) || 0;
    return speed > 0 ? `${formatBytes(speed)}/s` : '—';
  }

  function formatModified(value) {
    if (!value) {
      return '—';
    }
    const date = new Date(value);
    if (Number.isNaN(date.getTime())) {
      return '—';
    }
    return date.toLocaleString(undefined, {dateStyle: 'short', timeStyle: 'short'});
  }

  function selectedLocal() {
    return state.localSelection >= 0 ? state.localItems[state.localSelection] || null : null;
  }

  function selectedRemote() {
    return state.remoteSelection >= 0 ? state.remoteItems[state.remoteSelection] || null : null;
  }

  function makeFileRow(item, index, side) {
    const row = document.createElement('tr');
    row.dataset.index = String(index);
    row.dataset.side = side;
    const name = document.createElement('td');
    name.className = 'name-cell';
    name.dataset.directory = item.isDirectory ? 'true' : 'false';
    name.textContent = item.name || '';
    name.title = item.name || '';
    const size = document.createElement('td');
    size.textContent = item.isDirectory ? '—' : formatBytes(item.size);
    const modified = document.createElement('td');
    modified.textContent = formatModified(item.modified);
    row.append(name, size, modified);
    return row;
  }

  function renderLocalItems() {
    el['local-items'].replaceChildren();
    if (!state.localRoot) {
      const row = document.createElement('tr');
      const cell = document.createElement('td');
      cell.colSpan = 3;
      cell.className = 'empty';
      cell.textContent = 'Choose a local folder.';
      row.appendChild(cell);
      el['local-items'].appendChild(row);
    } else if (state.localItems.length === 0) {
      const row = document.createElement('tr');
      const cell = document.createElement('td');
      cell.colSpan = 3;
      cell.className = 'empty';
      cell.textContent = 'This folder is empty.';
      row.appendChild(cell);
      el['local-items'].appendChild(row);
    } else {
      state.localItems.forEach((item, index) => {
        const row = makeFileRow(item, index, 'local');
        row.classList.toggle('is-selected', index === state.localSelection);
        el['local-items'].appendChild(row);
      });
    }
    el['local-path'].value = state.localPath || '';
    updateLocalButtons();
  }

  function renderRemoteItems() {
    el['remote-items'].replaceChildren();
    if (!state.connected) {
      const row = document.createElement('tr');
      const cell = document.createElement('td');
      cell.colSpan = 3;
      cell.className = 'empty';
      cell.textContent = 'Connect to a server.';
      row.appendChild(cell);
      el['remote-items'].appendChild(row);
    } else if (state.remoteItems.length === 0) {
      const row = document.createElement('tr');
      const cell = document.createElement('td');
      cell.colSpan = 3;
      cell.className = 'empty';
      cell.textContent = 'This server folder is empty.';
      row.appendChild(cell);
      el['remote-items'].appendChild(row);
    } else {
      state.remoteItems.forEach((item, index) => {
        const row = makeFileRow(item, index, 'remote');
        row.classList.toggle('is-selected', index === state.remoteSelection);
        el['remote-items'].appendChild(row);
      });
    }
    el['remote-path'].value = state.remotePath || '/';
    updateRemoteButtons();
  }

  function updateLocalButtons() {
    const selected = selectedLocal();
    const hasRoot = Boolean(state.localRoot);
    el['local-up'].disabled = !hasRoot || state.localPath === state.localRoot;
    el['local-refresh'].disabled = !hasRoot;
    el['local-mkdir'].disabled = !hasRoot;
    el['local-rename'].disabled = !selected;
    el['local-delete'].disabled = !selected;
    el['upload-selected'].disabled = !selected || selected.isDirectory || !state.connected;
  }

  function updateRemoteButtons() {
    const selected = selectedRemote();
    el['remote-up'].disabled = !state.connected;
    el['remote-go'].disabled = !state.connected;
    el['remote-refresh'].disabled = !state.connected;
    el['remote-mkdir'].disabled = !state.connected;
    el['remote-rename'].disabled = !state.connected || !selected;
    el['remote-delete'].disabled = !state.connected || !selected;
    el['download-selected'].disabled = !state.connected || !state.localRoot || !selected || selected.isDirectory;
    updateLocalButtons();
  }

  async function chooseLocalRoot() {
    setStatus('Choose the local folder Ghost FTP may access.', 'neutral');
    const result = await requestNative('local.chooseRoot', {});
    state.localRoot = result.root || '';
    state.localPath = result.path || state.localRoot;
    state.localItems = Array.isArray(result.items) ? result.items : [];
    state.localSelection = -1;
    renderLocalItems();
    setStatus('Local access is confined to the folder you selected.', 'success');
  }

  async function refreshLocal(path = state.localPath) {
    if (!state.localRoot) {
      throw new Error('Choose a local folder first.');
    }
    const result = await requestNative('local.list', {path});
    state.localPath = result.path || path;
    state.localItems = Array.isArray(result.items) ? result.items : [];
    state.localSelection = -1;
    renderLocalItems();
  }

  async function refreshRemote(path = el['remote-path'].value.trim()) {
    if (!state.connected) {
      throw new Error('Connect to a server first.');
    }
    const result = await requestNative('remote.list', {path});
    state.remotePath = result.path || path;
    state.remoteItems = Array.isArray(result.items) ? result.items : [];
    state.remoteSelection = -1;
    renderRemoteItems();
  }

  async function openSelectedDirectory(side, index) {
    if (side === 'local') {
      const item = state.localItems[index];
      if (item && item.isDirectory) {
        await refreshLocal(joinLocal(state.localPath, item.name));
      }
      return;
    }
    const item = state.remoteItems[index];
    if (item && item.isDirectory) {
      await refreshRemote(joinRemote(state.remotePath, item.name));
    }
  }

  async function mutateLocal(type) {
    if (!state.localRoot) {
      throw new Error('Choose a local folder first.');
    }
    if (type === 'mkdir') {
      const name = globalThis.prompt('New local folder name:');
      if (name === null) {
        return;
      }
      await requestNative('local.mkdir', {base: state.localPath, name});
    } else {
      const item = selectedLocal();
      if (!item) {
        throw new Error('Select a local item first.');
      }
      if (type === 'rename') {
        const newName = globalThis.prompt('New local name:', item.name);
        if (newName === null || newName === item.name) {
          return;
        }
        await requestNative('local.rename', {base: state.localPath, name: item.name, newName});
      } else if (type === 'delete') {
        if (!globalThis.confirm(`Delete local ${item.isDirectory ? 'folder' : 'file'} “${item.name}”?`)) {
          return;
        }
        await requestNative('local.delete', {base: state.localPath, name: item.name});
      }
    }
    await refreshLocal();
  }

  async function mutateRemote(type) {
    if (!state.connected) {
      throw new Error('Connect to a server first.');
    }
    if (type === 'mkdir') {
      const name = globalThis.prompt('New server folder name:');
      if (name === null) {
        return;
      }
      await requestNative('remote.mkdir', {base: state.remotePath, name});
    } else {
      const item = selectedRemote();
      if (!item) {
        throw new Error('Select a server item first.');
      }
      if (type === 'rename') {
        const newName = globalThis.prompt('New server name:', item.name);
        if (newName === null || newName === item.name) {
          return;
        }
        await requestNative('remote.rename', {base: state.remotePath, name: item.name, newName});
      } else if (type === 'delete') {
        if (!globalThis.confirm(`Delete server ${item.isDirectory ? 'folder' : 'file'} “${item.name}”?`)) {
          return;
        }
        await requestNative('remote.delete', {
          base: state.remotePath,
          name: item.name,
          isDirectory: Boolean(item.isDirectory)
        });
      }
    }
    await refreshRemote(state.remotePath);
  }

  async function uploadSelected() {
    const item = selectedLocal();
    if (!item || item.isDirectory) {
      throw new Error('Select a local file to upload.');
    }
    const job = await requestNative('transfer.upload', {
      localPath: joinLocal(state.localPath, item.name),
      remotePath: joinRemote(state.remotePath, item.name)
    });
    upsertTransfer(job);
    setStatus(`Upload queued: ${item.name}`, 'success');
    renderTransfers();
  }

  async function downloadSelected() {
    const item = selectedRemote();
    if (!item || item.isDirectory) {
      throw new Error('Select a server file to download.');
    }
    if (!state.localRoot) {
      throw new Error('Choose a local destination folder first.');
    }
    const job = await requestNative('transfer.download', {
      localDirectory: state.localPath,
      remotePath: joinRemote(state.remotePath, item.name),
      fileName: item.name
    });
    upsertTransfer(job);
    setStatus(`Download queued: ${item.name}`, 'success');
    renderTransfers();
  }

  function upsertTransfer(job) {
    if (job && typeof job.id === 'string') {
      state.transfers.set(job.id, job);
    }
  }

  function applyTransferEvents(result) {
    if (!result || !Array.isArray(result.events)) {
      return;
    }
    for (const event of result.events) {
      if (!event || typeof event !== 'object') {
        continue;
      }
      if (event.type === 'state' && Array.isArray(event.jobs)) {
        state.transfers.clear();
        for (const job of event.jobs) {
          upsertTransfer(job);
        }
      } else if (event.type === 'job') {
        upsertTransfer(event.job);
      }
    }
    if (Number.isSafeInteger(result.next) && result.next >= 0) {
      state.transferSince = result.next;
    }
    renderTransfers();
  }

  async function refreshTransfers(since = state.transferSince) {
    const result = await requestNative('transfer.events', {since});
    applyTransferEvents(result);
  }

  function transferFileName(job) {
    const path = job.direction === 'upload' ? job.localPath : job.remotePath;
    if (!path) {
      return 'Transfer';
    }
    const normalized = String(path).replace(/[\\/]+$/, '');
    const split = normalized.split(/[\\/]/);
    return split[split.length - 1] || normalized;
  }

  function createTransferCard(job) {
    const card = document.createElement('article');
    card.className = 'transfer-card';
    card.dataset.id = job.id;

    const top = document.createElement('div');
    top.className = 'transfer-top';
    const title = document.createElement('p');
    title.className = 'transfer-title';
    title.textContent = `${job.direction === 'upload' ? '↑' : '↓'} ${transferFileName(job)}`;
    title.title = job.direction === 'upload' ? job.localPath : job.remotePath;
    const status = document.createElement('span');
    status.className = 'transfer-status';
    status.dataset.status = job.status || 'queued';
    status.textContent = job.status || 'queued';
    top.append(title, status);

    const progress = document.createElement('div');
    progress.className = 'progress';
    const progressBar = document.createElement('span');
    const value = Math.max(0, Math.min(100, Number(job.progress) || 0));
    progressBar.style.width = `${value}%`;
    progress.appendChild(progressBar);

    const meta = document.createElement('div');
    meta.className = 'transfer-meta';
    const progressText = document.createElement('span');
    progressText.textContent = `${value.toFixed(value >= 10 ? 0 : 1)}%`;
    const bytes = document.createElement('span');
    bytes.textContent = `${formatBytes(job.bytesTransferred)} / ${formatBytes(job.bytesTotal)}`;
    const speed = document.createElement('span');
    speed.textContent = formatSpeed(job.bytesPerSecond);
    const attempts = document.createElement('span');
    attempts.textContent = `Attempt ${Number(job.attempts) || 0}`;
    meta.append(progressText, bytes, speed, attempts);

    card.append(top, progress, meta);

    if (job.error) {
      const error = document.createElement('p');
      error.className = 'transfer-error';
      error.textContent = job.error;
      card.appendChild(error);
    }

    const actions = document.createElement('div');
    actions.className = 'transfer-actions';
    if (job.status === 'queued' || job.status === 'running') {
      const cancel = document.createElement('button');
      cancel.type = 'button';
      cancel.className = 'danger compact';
      cancel.dataset.transferAction = 'cancel';
      cancel.dataset.id = job.id;
      cancel.textContent = 'Cancel';
      actions.appendChild(cancel);
    } else if (job.status === 'failed' || job.status === 'cancelled') {
      const retry = document.createElement('button');
      retry.type = 'button';
      retry.className = 'secondary compact';
      retry.dataset.transferAction = 'retry';
      retry.dataset.id = job.id;
      retry.textContent = 'Retry';
      actions.appendChild(retry);
    }
    if (actions.childElementCount > 0) {
      card.appendChild(actions);
    }
    return card;
  }

  function renderTransfers() {
    el['transfer-list'].replaceChildren();
    const jobs = [...state.transfers.values()];
    jobs.sort((left, right) => String(right.createdAt || '').localeCompare(String(left.createdAt || '')));
    if (jobs.length === 0) {
      const empty = document.createElement('div');
      empty.className = 'empty-card';
      empty.textContent = 'No transfers yet.';
      el['transfer-list'].appendChild(empty);
    } else {
      for (const job of jobs) {
        el['transfer-list'].appendChild(createTransferCard(job));
      }
    }
    const active = jobs.filter((job) => job.status === 'queued' || job.status === 'running').length;
    el['transfer-count'].hidden = active === 0;
    el['transfer-count'].textContent = String(active);
  }

  async function transferAction(action, id) {
    await requestNative(action === 'cancel' ? 'transfer.cancel' : 'transfer.retry', {id});
    await refreshTransfers(state.transferSince);
  }

  function withUiError(task) {
    return async (...args) => {
      try {
        await task(...args);
      } catch (error) {
        setStatus(error instanceof Error ? error.message : 'Ghost FTP could not complete the operation.', 'error');
      }
    };
  }

  function bindEvents() {
    document.querySelector('.tabs').addEventListener('click', (event) => {
      const button = event.target.closest('[data-tab]');
      if (button) {
        switchTab(button.dataset.tab);
      }
    });

    el.protocol.addEventListener('change', () => updateProtocolFields(true));
    el['profile-select'].addEventListener('change', () => populateProfile(currentProfile()));
    el['new-profile'].addEventListener('click', resetProfileForm);
    el['fill-address'].addEventListener('click', withUiError(async () => fillFromAddress()));
    el['save-profile'].addEventListener('click', withUiError(saveProfile));
    el['delete-profile'].addEventListener('click', withUiError(deleteProfile));

    el['connection-form'].addEventListener('submit', withUiError(async (event) => {
      event.preventDefault();
      await connectServer();
    }));
    el['disconnect-button'].addEventListener('click', withUiError(disconnectServer));
    el['trust-accept'].addEventListener('click', withUiError(acceptTrust));
    el['trust-cancel'].addEventListener('click', withUiError(cancelTrust));

    el['choose-local-root'].addEventListener('click', withUiError(chooseLocalRoot));
    el['local-refresh'].addEventListener('click', withUiError(async () => refreshLocal()));
    el['local-up'].addEventListener('click', withUiError(async () => refreshLocal(parentLocal(state.localPath))));
    el['remote-refresh'].addEventListener('click', withUiError(async () => refreshRemote(state.remotePath)));
    el['remote-go'].addEventListener('click', withUiError(async () => refreshRemote(el['remote-path'].value.trim())));
    el['remote-up'].addEventListener('click', withUiError(async () => refreshRemote(parentRemote(state.remotePath))));

    el['local-mkdir'].addEventListener('click', withUiError(async () => mutateLocal('mkdir')));
    el['local-rename'].addEventListener('click', withUiError(async () => mutateLocal('rename')));
    el['local-delete'].addEventListener('click', withUiError(async () => mutateLocal('delete')));
    el['remote-mkdir'].addEventListener('click', withUiError(async () => mutateRemote('mkdir')));
    el['remote-rename'].addEventListener('click', withUiError(async () => mutateRemote('rename')));
    el['remote-delete'].addEventListener('click', withUiError(async () => mutateRemote('delete')));
    el['upload-selected'].addEventListener('click', withUiError(uploadSelected));
    el['download-selected'].addEventListener('click', withUiError(downloadSelected));

    el['local-items'].addEventListener('click', (event) => {
      const row = event.target.closest('tr[data-index]');
      if (!row) {
        return;
      }
      state.localSelection = Number.parseInt(row.dataset.index, 10);
      renderLocalItems();
    });
    el['local-items'].addEventListener('dblclick', withUiError(async (event) => {
      const row = event.target.closest('tr[data-index]');
      if (row) {
        await openSelectedDirectory('local', Number.parseInt(row.dataset.index, 10));
      }
    }));
    el['remote-items'].addEventListener('click', (event) => {
      const row = event.target.closest('tr[data-index]');
      if (!row) {
        return;
      }
      state.remoteSelection = Number.parseInt(row.dataset.index, 10);
      renderRemoteItems();
    });
    el['remote-items'].addEventListener('dblclick', withUiError(async (event) => {
      const row = event.target.closest('tr[data-index]');
      if (row) {
        await openSelectedDirectory('remote', Number.parseInt(row.dataset.index, 10));
      }
    }));

    el['pause-transfers'].addEventListener('click', withUiError(async () => {
      await requestNative('transfer.pause', {});
      setStatus('Transfer queue paused.', 'success');
    }));
    el['resume-transfers'].addEventListener('click', withUiError(async () => {
      await requestNative('transfer.resume', {});
      setStatus('Transfer queue resumed.', 'success');
    }));
    el['clear-transfers'].addEventListener('click', withUiError(async () => {
      await requestNative('transfer.clearFinished', {});
      state.transferSince = 0;
      await refreshTransfers(0);
      setStatus('Finished transfers cleared.', 'success');
    }));
    el['transfer-list'].addEventListener('click', withUiError(async (event) => {
      const button = event.target.closest('[data-transfer-action]');
      if (button) {
        await transferAction(button.dataset.transferAction, button.dataset.id);
      }
    }));
  }

  async function initialize() {
    cacheElements();
    bindEvents();
    updateProtocolFields(false);
    renderLocalItems();
    renderRemoteItems();
    renderTransfers();
    connectUiPort();

    try {
      const hello = await requestNative('hello', {});
      setBridgeStatus(true);
      el['version-label'].textContent = hello && hello.version ? `Bridge ${hello.version}` : '';
      await loadProfiles();
      await refreshConnectionState();
      await refreshTransfers(0);
      if (state.connected) {
        await refreshRemote(state.remotePath);
      }
      setStatus('Ready. Browser data stays local; server traffic is handled by the Ghost FTP engine.', 'success');
    } catch (error) {
      setBridgeStatus(false, 'Local bridge unavailable');
      setStatus(error instanceof Error ? error.message : 'Ghost FTP local bridge is unavailable.', 'error');
    }
  }

  void initialize();
})();
