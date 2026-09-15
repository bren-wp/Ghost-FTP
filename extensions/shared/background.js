(() => {
  'use strict';

  const api = globalThis.browser || globalThis.chrome;
  const HOST_NAME = 'com.ghostftp.bridge';
  const UI_PORT_NAME = 'ghostftp-ui';
  const POLL_INTERVAL_MS = 750;
  const REQUEST_TIMEOUT_MS = 45000;
  const MAX_PENDING = 256;

  let nativePort = null;
  let pollTimer = null;
  let transferSince = 0;
  let requestCounter = 0;
  let nativeDisconnectMessage = '';
  let transferPollPending = false;

  const clients = new Set();
  const pending = new Map();
  const transfers = new Map();

  function nextRequestId(prefix = 'bg') {
    requestCounter = (requestCounter + 1) % Number.MAX_SAFE_INTEGER;
    return `${prefix}-${Date.now().toString(36)}-${requestCounter.toString(36)}`;
  }

  function activeTransferCount() {
    let active = 0;
    for (const job of transfers.values()) {
      if (job && (job.status === 'queued' || job.status === 'running')) {
        active += 1;
      }
    }
    return active;
  }

  function broadcast(message) {
    for (const client of [...clients]) {
      try {
        client.postMessage(message);
      } catch (_error) {
        clients.delete(client);
      }
    }
  }

  function updateTransferStateFromEvent(event) {
    if (!event || typeof event !== 'object') {
      return;
    }
    if (event.type === 'state' && Array.isArray(event.jobs)) {
      transfers.clear();
      for (const job of event.jobs) {
        if (job && typeof job.id === 'string') {
          transfers.set(job.id, job);
        }
      }
      return;
    }
    if (event.type === 'job' && event.job && typeof event.job.id === 'string') {
      transfers.set(event.job.id, event.job);
    }
  }

  function updateTransfersFromResponse(request, response) {
    if (!request || !response || response.ok !== true) {
      return;
    }

    if ((request.type === 'transfer.upload' || request.type === 'transfer.download') && response.result && typeof response.result.id === 'string') {
      transfers.set(response.result.id, response.result);
    }

    if (request.type === 'transfer.events' && response.result && Array.isArray(response.result.events)) {
      for (const event of response.result.events) {
        updateTransferStateFromEvent(event);
      }
      if (Number.isSafeInteger(response.result.next) && response.result.next >= 0) {
        transferSince = response.result.next;
      }
    }

    if (request.type === 'disconnect') {
      for (const [id, job] of transfers.entries()) {
        if (job && (job.status === 'queued' || job.status === 'running')) {
          transfers.set(id, {...job, status: 'cancelled'});
        }
      }
    }
  }

  function closeNativeIfIdle() {
    if (clients.size !== 0 || activeTransferCount() !== 0 || pending.size !== 0) {
      return;
    }
    stopPolling();
    if (nativePort) {
      const port = nativePort;
      nativePort = null;
      try {
        port.disconnect();
      } catch (_error) {
        // The host may already be gone. There is nothing else to release.
      }
    }
  }

  function finishPending(id) {
    const entry = pending.get(id);
    if (!entry) {
      return null;
    }
    pending.delete(id);
    if (entry.timer !== null) {
      clearTimeout(entry.timer);
      entry.timer = null;
    }
    if (entry.request.type === 'transfer.events') {
      transferPollPending = false;
    }
    return entry;
  }

  function timeoutPending(id) {
    const entry = finishPending(id);
    if (!entry) {
      return;
    }
    const error = {
      code: 'operation_timeout',
      message: 'Ghost FTP local bridge did not respond in time. Try the operation again.'
    };
    try {
      entry.client.postMessage({kind: 'response', response: {id: entry.request.id, ok: false, error}});
    } catch (_error) {
      // The popup may have closed while the native operation was pending.
    }
    closeNativeIfIdle();
  }

  function trackPending(client, request) {
    const entry = {client, request, timer: null};
    entry.timer = setTimeout(() => timeoutPending(request.id), REQUEST_TIMEOUT_MS);
    pending.set(request.id, entry);
  }

  function rejectPending(message) {
    const error = {
      code: 'native_host_unavailable',
      message: message || 'Ghost FTP desktop components are not available. Install or repair Ghost FTP and try again.'
    };
    for (const entry of pending.values()) {
      if (entry.timer !== null) {
        clearTimeout(entry.timer);
        entry.timer = null;
      }
      try {
        entry.client.postMessage({kind: 'response', response: {id: entry.request.id, ok: false, error}});
      } catch (_error) {
        // The popup may have closed while the native host was stopping.
      }
    }
    pending.clear();
    transferPollPending = false;
  }

  function handleNativeDisconnect(port) {
    if (nativePort !== port) {
      return;
    }
    nativePort = null;
    stopPolling();

    let message = '';
    try {
      message = api.runtime.lastError && api.runtime.lastError.message ? api.runtime.lastError.message : '';
    } catch (_error) {
      message = '';
    }
    nativeDisconnectMessage = message;
    rejectPending('Ghost FTP local bridge disconnected. Reopen the extension after checking the Ghost FTP installation.');
    broadcast({
      kind: 'bridge-status',
      available: false,
      message: 'Ghost FTP local bridge disconnected. Check the Ghost FTP installation and reopen the extension.'
    });
  }

  function handleNativeMessage(message) {
    if (!message || typeof message !== 'object' || typeof message.id !== 'string') {
      return;
    }
    const entry = finishPending(message.id);
    if (!entry) {
      return;
    }
    updateTransfersFromResponse(entry.request, message);

    try {
      entry.client.postMessage({kind: 'response', response: message});
    } catch (_error) {
      clients.delete(entry.client);
    }

    if (activeTransferCount() > 0) {
      startPolling();
    } else if (entry.request.type === 'transfer.events') {
      stopPolling();
    }
    closeNativeIfIdle();
  }

  function ensureNativePort() {
    if (nativePort) {
      return nativePort;
    }
    nativeDisconnectMessage = '';
    const port = api.runtime.connectNative(HOST_NAME);
    nativePort = port;
    port.onMessage.addListener(handleNativeMessage);
    port.onDisconnect.addListener(() => handleNativeDisconnect(port));
    return port;
  }

  function sendNative(client, request) {
    if (!request || typeof request !== 'object' || typeof request.id !== 'string' || typeof request.type !== 'string') {
      client.postMessage({
        kind: 'response',
        response: {
          id: request && typeof request.id === 'string' ? request.id : '',
          ok: false,
          error: {code: 'invalid_request', message: 'The browser request is invalid.'}
        }
      });
      return;
    }
    if (pending.has(request.id)) {
      client.postMessage({
        kind: 'response',
        response: {
          id: request.id,
          ok: false,
          error: {code: 'duplicate_request', message: 'Ghost FTP is already processing this browser request.'}
        }
      });
      return;
    }
    if (pending.size >= MAX_PENDING) {
      client.postMessage({
        kind: 'response',
        response: {
          id: request.id,
          ok: false,
          error: {code: 'busy', message: 'Ghost FTP is busy. Finish an existing operation and try again.'}
        }
      });
      return;
    }

    let port;
    try {
      port = ensureNativePort();
    } catch (_error) {
      client.postMessage({
        kind: 'response',
        response: {
          id: request.id,
          ok: false,
          error: {code: 'native_host_unavailable', message: 'Ghost FTP desktop components are not available. Install or repair Ghost FTP and try again.'}
        }
      });
      return;
    }

    trackPending(client, request);
    try {
      port.postMessage(request);
    } catch (_error) {
      finishPending(request.id);
      client.postMessage({
        kind: 'response',
        response: {
          id: request.id,
          ok: false,
          error: {code: 'native_host_unavailable', message: 'Ghost FTP local bridge could not accept the request.'}
        }
      });
      closeNativeIfIdle();
    }
  }

  function sendBackgroundRequest(type, params = {}) {
    if (!nativePort || pending.size >= MAX_PENDING || transferPollPending) {
      return;
    }
    const id = nextRequestId('poll');
    const request = {id, type, params};
    const syntheticClient = {
      postMessage(message) {
        if (!message || message.kind !== 'response' || !message.response) {
          return;
        }
        const response = message.response;
        if (response.ok === true && response.result && Array.isArray(response.result.events)) {
          for (const event of response.result.events) {
            updateTransferStateFromEvent(event);
          }
          if (Number.isSafeInteger(response.result.next) && response.result.next >= 0) {
            transferSince = response.result.next;
          }
          broadcast({kind: 'transfer-events', result: response.result});
        }
      }
    };
    transferPollPending = true;
    trackPending(syntheticClient, request);
    try {
      nativePort.postMessage(request);
    } catch (_error) {
      finishPending(id);
      handleNativeDisconnect(nativePort);
    }
  }

  function pollTransfers() {
    pollTimer = null;
    if (!nativePort || activeTransferCount() === 0) {
      closeNativeIfIdle();
      return;
    }
    sendBackgroundRequest('transfer.events', {since: transferSince});
    pollTimer = setTimeout(pollTransfers, POLL_INTERVAL_MS);
  }

  function startPolling() {
    if (pollTimer !== null || activeTransferCount() === 0) {
      return;
    }
    pollTimer = setTimeout(pollTransfers, POLL_INTERVAL_MS);
  }

  function stopPolling() {
    if (pollTimer !== null) {
      clearTimeout(pollTimer);
      pollTimer = null;
    }
  }

  api.runtime.onConnect.addListener((client) => {
    if (!client || client.name !== UI_PORT_NAME) {
      return;
    }
    clients.add(client);
    client.postMessage({
      kind: 'bridge-status',
      available: nativePort !== null,
      message: nativeDisconnectMessage ? 'Ghost FTP local bridge is not currently connected.' : ''
    });

    client.onMessage.addListener((message) => {
      if (!message || message.kind !== 'request') {
        return;
      }
      sendNative(client, message.request);
    });

    client.onDisconnect.addListener(() => {
      clients.delete(client);
      closeNativeIfIdle();
    });
  });
})();
