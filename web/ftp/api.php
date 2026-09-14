<?php
declare(strict_types=1);

require __DIR__ . '/config.php';
require __DIR__ . '/lib/Security.php';
require __DIR__ . '/lib/Transport.php';
require __DIR__ . '/lib/CurlFtpTransport.php';
require __DIR__ . '/lib/SftpTransport.php';

use GhostFTP\Web\CurlFtpTransport;
use GhostFTP\Web\Security;
use GhostFTP\Web\SftpTransport;
use GhostFTP\Web\Transport;

Security::sendHeaders();

if ($_SERVER['REQUEST_METHOD'] !== 'POST') {
    http_response_code(405);
    header('Allow: POST');
    exit;
}

function json_out(array $payload, int $status = 200): never
{
    $json = json_encode(
        $payload,
        JSON_UNESCAPED_SLASHES | JSON_UNESCAPED_UNICODE | JSON_INVALID_UTF8_SUBSTITUTE
    );
    if ($json === false) {
        $status = 500;
        $json = '{"ok":false,"error":"Response encoding failed."}';
    }

    http_response_code($status);
    header('Content-Type: application/json; charset=utf-8');
    echo $json;
    exit;
}

function payload(): array
{
    $type = strtolower((string)($_SERVER['CONTENT_TYPE'] ?? ''));
    if (str_starts_with($type, 'multipart/form-data')) {
        $profile = json_decode((string)($_POST['profile'] ?? ''), true);
        if (!is_array($profile)) {
            throw new RuntimeException('Invalid profile payload.');
        }
        return [
            'action' => (string)($_POST['action'] ?? ''),
            'profile' => $profile,
            'path' => (string)($_POST['path'] ?? ''),
        ];
    }

    $raw = file_get_contents('php://input');
    $data = json_decode(is_string($raw) ? $raw : '', true);
    if (!is_array($data)) {
        throw new RuntimeException('Invalid JSON request.');
    }
    return $data;
}

function transport(array $profile): Transport
{
    $protocol = (string)($profile['protocol'] ?? '');
    return match ($protocol) {
        'ftp', 'ftps' => new CurlFtpTransport($profile),
        'sftp' => new SftpTransport($profile),
        default => throw new RuntimeException('Unsupported protocol.'),
    };
}

function json_success(?Transport &$transport, array $payload = ['ok' => true]): never
{
    if ($transport instanceof Transport) {
        $transport->close();
    }
    $transport = null;
    json_out($payload);
}

$t = null;

try {
    $data = payload();
    $action = (string)($data['action'] ?? '');
    $profile = $data['profile'] ?? null;
    if (!is_array($profile)) {
        throw new RuntimeException('Connection profile is required.');
    }

    $t = transport($profile);
    $path = (string)($data['path'] ?? ($profile['root'] ?? '/'));

    switch ($action) {
        case 'list':
            json_success($t, [
                'ok' => true,
                'path' => $path,
                'items' => $t->list($path),
                'version' => GHOSTFTP_WEB_VERSION,
            ]);

        case 'mkdir':
            $t->mkdir((string)($data['target'] ?? ''));
            json_success($t);

        case 'rename':
            $t->rename((string)($data['from'] ?? ''), (string)($data['to'] ?? ''));
            json_success($t);

        case 'delete':
            $t->delete((string)($data['target'] ?? ''), (bool)($data['directory'] ?? false));
            json_success($t);

        case 'chmod':
            $raw = (string)($data['mode'] ?? '');
            if (!preg_match('/^[0-7]{3,4}$/', $raw)) {
                throw new RuntimeException('Permission mode must be octal.');
            }
            $t->chmod((string)($data['target'] ?? ''), octdec($raw));
            json_success($t);

        case 'read':
            $content = $t->read(
                (string)($data['target'] ?? ''),
                GHOSTFTP_WEB_MAX_EDIT_BYTES
            );
            json_success($t, ['ok' => true, 'content' => $content]);

        case 'write':
            $t->write(
                (string)($data['target'] ?? ''),
                (string)($data['content'] ?? '')
            );
            json_success($t);

        case 'upload':
            if (!isset($_FILES['file']) || !is_uploaded_file($_FILES['file']['tmp_name'])) {
                throw new RuntimeException('No valid upload was supplied.');
            }
            if ((int)$_FILES['file']['size'] > GHOSTFTP_WEB_MAX_UPLOAD_BYTES) {
                throw new RuntimeException('Upload exceeds the configured limit.');
            }
            $t->upload($_FILES['file']['tmp_name'], (string)($_POST['path'] ?? ''));
            json_success($t);

        case 'download':
            $tmp = tempnam(sys_get_temp_dir(), 'gftp-download-');
            if ($tmp === false) {
                throw new RuntimeException('Could not create a temporary download.');
            }

            try {
                $t->download(
                    (string)($data['target'] ?? ''),
                    $tmp,
                    GHOSTFTP_WEB_MAX_DOWNLOAD_BYTES
                );
                $t->close();
                $t = null;

                $size = filesize($tmp);
                if ($size === false) {
                    throw new RuntimeException('Could not determine download size.');
                }

                $name = basename((string)($data['target'] ?? 'download.bin'));
                $safeName = preg_replace('/[^A-Za-z0-9._-]/', '_', $name) ?: 'download.bin';

                header('Content-Type: application/octet-stream');
                header('Content-Length: ' . $size);
                header('Content-Disposition: attachment; filename="' . $safeName . '"');

                if (readfile($tmp) === false) {
                    throw new RuntimeException('Could not stream the temporary download.');
                }
            } finally {
                @unlink($tmp);
            }
            exit;

        default:
            throw new RuntimeException('Unsupported action.');
    }
} catch (Throwable $e) {
    if ($t instanceof Transport) {
        $t->close();
        $t = null;
    }
    json_out(['ok' => false, 'error' => $e->getMessage()], 400);
} finally {
    if ($t instanceof Transport) {
        $t->close();
    }
}
