<?php
declare(strict_types=1);

namespace GhostFTP\Security;

use RuntimeException;

final class Crypto
{
    private const MAX_ENCRYPTED_VALUE_BYTES = 65536;
    private const OPENSSL_IV_BYTES = 12;
    private const OPENSSL_TAG_BYTES = 16;

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
            return $this->encodeEnvelope('sodium', $nonce . $cipher);
        }
        if (!function_exists('openssl_encrypt')) {
            throw new RuntimeException('Neither Sodium nor OpenSSL is available for encryption.');
        }
        $iv = random_bytes(self::OPENSSL_IV_BYTES);
        $tag = '';
        $cipher = openssl_encrypt($plain, 'aes-256-gcm', $this->key, OPENSSL_RAW_DATA, $iv, $tag);
        if (!is_string($cipher) || strlen($tag) !== self::OPENSSL_TAG_BYTES) {
            throw new RuntimeException('Encryption failed.');
        }
        return $this->encodeEnvelope('openssl', $iv . $tag . $cipher);
    }

    public function decrypt(string $encoded): string
    {
        if ($encoded === '' || strlen($encoded) > self::MAX_ENCRYPTED_VALUE_BYTES) {
            throw new RuntimeException('Encrypted value is invalid.');
        }

        [$driver, $payload] = array_pad(explode(':', $encoded, 2), 2, '');
        if ($driver !== 'sodium' && $driver !== 'openssl') {
            throw new RuntimeException('Unsupported encryption format.');
        }
        if ($payload === '' || strlen($payload) > self::MAX_ENCRYPTED_VALUE_BYTES) {
            throw new RuntimeException('Encrypted value is invalid.');
        }

        $raw = base64_decode($payload, true);
        if (!is_string($raw)) {
            throw new RuntimeException('Encrypted value is invalid.');
        }

        if ($driver === 'sodium') {
            if (!function_exists('sodium_crypto_secretbox_open')) {
                throw new RuntimeException('Sodium is required to decrypt credentials created with Sodium.');
            }
            $nonceLen = SODIUM_CRYPTO_SECRETBOX_NONCEBYTES;
            $minimum = $nonceLen + SODIUM_CRYPTO_SECRETBOX_MACBYTES;
            if (strlen($raw) < $minimum) {
                throw new RuntimeException('Encrypted value is truncated.');
            }
            $nonce = substr($raw, 0, $nonceLen);
            $cipher = substr($raw, $nonceLen);
            $plain = sodium_crypto_secretbox_open($cipher, $nonce, $this->key);
            if (!is_string($plain)) {
                throw new RuntimeException('Decryption failed.');
            }
            return $plain;
        }

        $minimum = self::OPENSSL_IV_BYTES + self::OPENSSL_TAG_BYTES;
        if (strlen($raw) < $minimum) {
            throw new RuntimeException('Encrypted value is truncated.');
        }
        if (!function_exists('openssl_decrypt')) {
            throw new RuntimeException('OpenSSL is required to decrypt these credentials.');
        }
        $iv = substr($raw, 0, self::OPENSSL_IV_BYTES);
        $tag = substr($raw, self::OPENSSL_IV_BYTES, self::OPENSSL_TAG_BYTES);
        $cipher = substr($raw, $minimum);
        $plain = openssl_decrypt($cipher, 'aes-256-gcm', $this->key, OPENSSL_RAW_DATA, $iv, $tag);
        if (!is_string($plain)) {
            throw new RuntimeException('Decryption failed.');
        }
        return $plain;
    }

    private function encodeEnvelope(string $driver, string $raw): string
    {
        $encoded = $driver . ':' . base64_encode($raw);
        if (strlen($encoded) > self::MAX_ENCRYPTED_VALUE_BYTES) {
            throw new RuntimeException('Encrypted value exceeds the supported size.');
        }
        return $encoded;
    }
}
