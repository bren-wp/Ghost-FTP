(function () {
  'use strict';

  const form = document.getElementById('connection-form');
  const input = document.getElementById('connection-target');
  const result = document.getElementById('result');
  const status = document.getElementById('status');
  const passwordNotice = document.getElementById('password-notice');
  const clearButton = document.getElementById('clear-button');

  const outputs = {
    protocol: document.getElementById('protocol'),
    host: document.getElementById('host'),
    port: document.getElementById('port'),
    username: document.getElementById('username'),
    path: document.getElementById('path'),
    safeTarget: document.getElementById('safe-target')
  };

  function setStatus(message, kind) {
    status.textContent = message;
    status.dataset.kind = kind;
  }

  function setOutput(element, value, emptyLabel) {
    const normalized = value || emptyLabel;
    element.value = normalized;
    element.dataset.copyValue = value || '';
  }

  function resetResult() {
    result.hidden = true;
    passwordNotice.hidden = true;
    setStatus('Nothing leaves this popup.', 'neutral');
    Object.values(outputs).forEach((element) => {
      element.value = '';
      element.dataset.copyValue = '';
    });
  }

  async function copyText(value) {
    if (!value) {
      throw new Error('There is nothing to copy.');
    }

    if (navigator.clipboard && typeof navigator.clipboard.writeText === 'function') {
      await navigator.clipboard.writeText(value);
      return;
    }

    const fallback = document.createElement('textarea');
    fallback.value = value;
    fallback.setAttribute('readonly', '');
    fallback.className = 'clipboard-fallback';
    document.body.appendChild(fallback);
    fallback.select();
    const copied = document.execCommand('copy');
    fallback.remove();
    if (!copied) {
      throw new Error('The browser blocked clipboard access.');
    }
  }

  form.addEventListener('submit', (event) => {
    event.preventDefault();
    const parsed = globalThis.GhostFTPConnection.parseConnectionTarget(input.value);
    if (!parsed.ok) {
      result.hidden = true;
      passwordNotice.hidden = true;
      setStatus(parsed.error, 'error');
      input.focus();
      return;
    }

    setOutput(outputs.protocol, parsed.protocol, '—');
    setOutput(outputs.host, parsed.host, '—');
    setOutput(outputs.port, parsed.port, 'Default / server-defined');
    setOutput(outputs.username, parsed.username, 'Not supplied');
    setOutput(outputs.path, parsed.path, '/');
    setOutput(outputs.safeTarget, parsed.safeTarget, '—');

    passwordNotice.hidden = !parsed.passwordDetected;
    result.hidden = false;
    setStatus('Parsed locally. Safe target excludes URL credentials, query and fragment data.', 'success');
  });

  document.addEventListener('click', async (event) => {
    const button = event.target.closest('[data-copy-target]');
    if (!button) {
      return;
    }
    const target = document.getElementById(button.dataset.copyTarget);
    if (!target) {
      return;
    }

    try {
      await copyText(target.dataset.copyValue || target.value);
      setStatus('Copied after your explicit click.', 'success');
    } catch (error) {
      setStatus(error instanceof Error ? error.message : 'Unable to copy.', 'error');
    }
  });

  clearButton.addEventListener('click', () => {
    form.reset();
    resetResult();
    input.focus();
  });

  resetResult();
  input.focus();
})();
