(function (root) {
  'use strict';

  const ALLOWED_PROTOCOLS = new Set(['ftp:', 'ftps:', 'sftp:']);

  function decodeComponent(value) {
    if (!value) {
      return '';
    }
    try {
      return decodeURIComponent(value);
    } catch (_error) {
      return value;
    }
  }

  function parseConnectionTarget(rawValue) {
    const input = String(rawValue || '').trim();
    if (!input) {
      return { ok: false, error: 'Enter an FTP, FTPS or SFTP address.' };
    }

    let parsed;
    try {
      parsed = new URL(input);
    } catch (_error) {
      return { ok: false, error: 'That connection address is not a valid URL.' };
    }

    if (!ALLOWED_PROTOCOLS.has(parsed.protocol)) {
      return { ok: false, error: 'Only FTP, FTPS and SFTP addresses are supported.' };
    }
    if (!parsed.hostname) {
      return { ok: false, error: 'The connection address must contain a host.' };
    }

    const passwordDetected = parsed.password.length > 0;
    const path = parsed.pathname || '/';
    const safeTarget = `${parsed.protocol}//${parsed.host}${path}`;

    return {
      ok: true,
      protocol: parsed.protocol.slice(0, -1).toUpperCase(),
      host: parsed.hostname,
      port: parsed.port,
      username: decodeComponent(parsed.username),
      path: decodeComponent(path),
      passwordDetected,
      safeTarget
    };
  }

  root.GhostFTPConnection = Object.freeze({
    parseConnectionTarget
  });
})(globalThis);
