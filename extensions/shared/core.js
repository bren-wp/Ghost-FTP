(function (root) {
  'use strict';

  const OFFICIAL_BRAND = 'Ghost FTP';
  const DESKTOP_SCHEME = 'ghostftp:';
  const ALLOWED_PROTOCOLS = new Set(['ftp:', 'ftps:', 'sftp:']);
  const MAX_INPUT_LENGTH = 4096;
  const CONTROL_CHARACTERS = /[\u0000-\u001f\u007f]/;

  function fail(error) {
    return Object.freeze({ ok: false, error });
  }

  function decodeForDisplay(value) {
    if (!value) {
      return '';
    }
    let decoded;
    try {
      decoded = decodeURIComponent(value);
    } catch (_error) {
      decoded = value;
    }
    if (CONTROL_CHARACTERS.test(decoded)) {
      throw new Error('The address contains encoded control characters.');
    }
    return decoded;
  }

  function parseConnectionTarget(rawValue) {
    const raw = String(rawValue || '');
    if (raw.length > MAX_INPUT_LENGTH) {
      return fail('The connection address is too long.');
    }
    if (CONTROL_CHARACTERS.test(raw)) {
      return fail('The connection address contains control characters.');
    }

    const input = raw.trim();
    if (!input) {
      return fail('Enter an FTP, FTPS or SFTP address.');
    }

    let parsed;
    try {
      parsed = new URL(input);
    } catch (_error) {
      return fail('That connection address is not a valid URL.');
    }

    if (!ALLOWED_PROTOCOLS.has(parsed.protocol)) {
      return fail('Only FTP, FTPS and SFTP addresses are supported.');
    }
    if (!parsed.hostname) {
      return fail('The connection address must contain a host.');
    }

    let username;
    let path;
    try {
      username = decodeForDisplay(parsed.username);
      path = decodeForDisplay(parsed.pathname || '/');
    } catch (error) {
      return fail(error instanceof Error ? error.message : 'The connection address contains unsafe encoded data.');
    }

    const passwordDetected = parsed.password.length > 0;
    const safePath = parsed.pathname || '/';
    const safeTarget = `${parsed.protocol}//${parsed.host}${safePath}`;

    return Object.freeze({
      ok: true,
      brand: OFFICIAL_BRAND,
      protocol: parsed.protocol.slice(0, -1).toUpperCase(),
      host: parsed.hostname,
      port: parsed.port,
      username,
      path,
      passwordDetected,
      safeTarget
    });
  }

  function buildDesktopLaunchURL(connection) {
    if (!connection || connection.ok !== true) {
      throw new Error('A validated connection target is required.');
    }

    const params = new URLSearchParams();
    params.set('protocol', String(connection.protocol || '').toLowerCase());
    params.set('host', connection.host || '');
    if (connection.port) {
      params.set('port', connection.port);
    }
    if (connection.username) {
      params.set('username', connection.username);
    }
    if (connection.path && connection.path !== '/') {
      params.set('path', connection.path);
    }

    return `${DESKTOP_SCHEME}//connect?${params.toString()}`;
  }

  root.GhostFTPConnection = Object.freeze({
    brand: OFFICIAL_BRAND,
    parseConnectionTarget,
    buildDesktopLaunchURL
  });
})(globalThis);
