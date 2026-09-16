(function (root) {
  'use strict';

  const OFFICIAL_BRAND = 'Ghost FTP';
  const ALLOWED_PROTOCOLS = new Set(['ftp:', 'ftps:', 'sftp:']);
  const MAX_INPUT_LENGTH = 4096;
  const CONTROL_CHARACTERS = /[\u0000-\u001f\u007f]/;

  function fail(error) {
    return Object.freeze({ ok: false, error });
  }

  function decodeForDisplay(value) {
    if (!value) return '';
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

  function buildDesktopLaunchTarget(parsed, username, path) {
    const params = new URLSearchParams();
    params.set('protocol', parsed.protocol.slice(0, -1));
    params.set('host', parsed.hostname);
    if (parsed.port) params.set('port', parsed.port);
    if (username) params.set('username', username);
    if (path && path !== '/') params.set('path', path);
    return `ghostftp://open?${params.toString()}`;
  }

  function parseConnectionTarget(rawValue) {
    const input = String(rawValue || '').trim();
    if (!input) return fail('Enter an FTP, FTPS or SFTP address.');
    if (input.length > MAX_INPUT_LENGTH) return fail('The connection address is too long.');
    if (CONTROL_CHARACTERS.test(input)) return fail('The connection address contains control characters.');

    let parsed;
    try {
      parsed = new URL(input);
    } catch (_error) {
      return fail('That connection address is not a valid URL.');
    }

    if (!ALLOWED_PROTOCOLS.has(parsed.protocol)) {
      return fail('Only FTP, FTPS and SFTP addresses are supported.');
    }
    if (!parsed.hostname) return fail('The connection address must contain a host.');

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
    const desktopLaunchTarget = buildDesktopLaunchTarget(parsed, username, path);

    return Object.freeze({
      ok: true,
      brand: OFFICIAL_BRAND,
      protocol: parsed.protocol.slice(0, -1).toUpperCase(),
      host: parsed.hostname,
      port: parsed.port,
      username,
      path,
      passwordDetected,
      safeTarget,
      desktopLaunchTarget
    });
  }

  root.GhostFTPConnection = Object.freeze({
    brand: OFFICIAL_BRAND,
    parseConnectionTarget
  });
})(globalThis);
