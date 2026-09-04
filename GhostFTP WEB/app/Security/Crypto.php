<?php
declare(strict_types=1);

namespace GhostFTP\Security;

use RuntimeException;

final class Crypto
{
    private string $key;

    public function __construct()
    {
        $encoded = (string)(\GhostFTP_config()['secret_key'] ?? '');
        $key = base64_decode($encoded, true);
        if (!is_string($key) || strlen($key) !== 32) {
            throw new RuntimeException('GhostFTP encryption key is invalid.');
        }
        $this->key = $key;
    }

    public function encrypt(string $plain): string
    {
        if (function_exists('sodium_crypto_secretbox')) {
            $nonce = random_bytes(SODIUM_CRYPTO_SECRETBOX_NONCEBYTES);
            $cipher = sodium_crypto_secretbox($plain, $nonce, $this->key);
            return 'sodium:' . base64_encode($nonce . $cipher);
        }
        if (!function_exists('openssl_encrypt')) {
            throw new RuntimeException('Neither Sodium nor OpenSSL is available for encryption.');
        }
        $iv = random_bytes(12);
        $tag = '';
        $cipher = openssl_encrypt($plain, 'aes-256-gcm', $this->key, OPENSSL_RAW_DATA, $iv, $tag);
        if (!is_string($cipher)) {
            throw new RuntimeException('Encryption failed.');
        }
        return 'openssl:' . base64_encode($iv . $tag . $cipher);
    }

    public function decrypt(string $encoded): string
    {
        [$driver, $payload] = array_pad(explode(':', $encoded, 2), 2, '');
        $raw = base64_decode($payload, true);
        if (!is_string($raw)) {
            throw new RuntimeException('Encrypted value is invalid.');
        }
        if ($driver === 'sodium') {
            if (!function_exists('sodium_crypto_secretbox_open')) {
                throw new RuntimeException('Sodium is required to decrypt credentials created with Sodium.');
            }
            $nonceLen = SODIUM_CRYPTO_SECRETBOX_NONCEBYTES;
            $nonce = substr($raw, 0, $nonceLen);
            $cipher = substr($raw, $nonceLen);
            $plain = sodium_crypto_secretbox_open($cipher, $nonce, $this->key);
            if (!is_string($plain)) {
                throw new RuntimeException('Decryption failed.');
            }
            return $plain;
        }
        if ($driver === 'openssl') {
            if (!function_exists('openssl_decrypt')) {
                throw new RuntimeException('OpenSSL is required to decrypt these credentials.');
            }
            $iv = substr($raw, 0, 12);
            $tag = substr($raw, 12, 16);
            $cipher = substr($raw, 28);
            $plain = openssl_decrypt($cipher, 'aes-256-gcm', $this->key, OPENSSL_RAW_DATA, $iv, $tag);
            if (!is_string($plain)) {
                throw new RuntimeException('Decryption failed.');
            }
            return $plain;
        }
        throw new RuntimeException('Unsupported encryption format.');
    }
}
