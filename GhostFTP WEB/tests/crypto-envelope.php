<?php
declare(strict_types=1);

use GhostFTP\Security\Crypto;

function GhostFTP_config(): array
{
    return ['secret_key' => base64_encode(str_repeat('K', 32))];
}

require __DIR__ . '/../app/Security/Crypto.php';

$failed = false;

function crypto_check(bool $condition, string $label): void
{
    global $failed;
    if ($condition) {
        return;
    }
    $failed = true;
    fwrite(STDERR, "FAIL: {$label}\n");
}

function crypto_throws(callable $callback, string $label): void
{
    try {
        $callback();
        crypto_check(false, $label);
    } catch (Throwable) {
        crypto_check(true, $label);
    }
}

$crypto = new Crypto();
$encrypted = $crypto->encrypt('credential-secret');
crypto_check($crypto->decrypt($encrypted) === 'credential-secret', 'authenticated encryption round-trip');

crypto_throws(
    fn() => $crypto->decrypt('unknown:' . base64_encode('payload')),
    'unknown envelope driver is rejected before decryption'
);
crypto_throws(
    fn() => $crypto->decrypt('openssl:' . base64_encode(str_repeat('X', 27))),
    'truncated OpenSSL envelope is rejected explicitly'
);
crypto_throws(
    fn() => $crypto->decrypt('openssl:' . str_repeat('A', 65537)),
    'oversized encrypted envelope is rejected before unbounded decoding'
);

[$driver, $payload] = explode(':', $encrypted, 2);
$raw = base64_decode($payload, true);
if (is_string($raw) && $raw !== '') {
    $last = strlen($raw) - 1;
    $raw[$last] = chr(ord($raw[$last]) ^ 1);
    crypto_throws(
        fn() => $crypto->decrypt($driver . ':' . base64_encode($raw)),
        'authenticated envelope tampering is rejected'
    );
} else {
    crypto_check(false, 'generated encrypted payload is valid strict base64');
}

if ($failed) {
    fwrite(STDERR, "WEB_CRYPTO_ENVELOPE_TEST=FAIL\n");
    exit(1);
}

echo "WEB_CRYPTO_ENVELOPE_TEST=PASS\n";
