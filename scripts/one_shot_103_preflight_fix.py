#!/usr/bin/env python3
from pathlib import Path


def patch_once(path: str, old: str, new: str) -> None:
    file = Path(path)
    text = file.read_text(encoding='utf-8')
    if text.count(old) != 1:
        raise SystemExit(f'PATCH_FAILED: {path}: expected one match, got {text.count(old)}')
    file.write_text(text.replace(old, new, 1), encoding='utf-8', newline='\n')


patch_once(
    'GhostFTP WEB/api.php',
    """        case 'bulk_delete':
            $items = GhostFTP_json_array((string)($_POST['items'] ?? '[]'), 200);
            $deleted = 0;
            foreach ($items as $item) {
                if (!is_array($item)) throw new RuntimeException('Popis za skupno brisanje sadrži neispravnu stavku.');
                $path = PathGuard::ensureNotRoot((string)($item['path'] ?? ''));
                $type = (string)($item['type'] ?? 'file');
                if (!in_array($type, ['file', 'dir'], true)) throw new RuntimeException('Vrsta stavke za skupno brisanje nije valjana.');
                $ops->deleteRecursive($path, $type);
                $deleted++;
            }
            AppLogger::event('item.bulk_delete', ['profile_id'=>$profileId,'count'=>$deleted]);
            GhostFTP_json(['ok'=>true,'deleted'=>$deleted]);

""",
    """        case 'bulk_delete':
            $items = GhostFTP_json_array((string)($_POST['items'] ?? '[]'), 200);
            $validatedItems = [];
            foreach ($items as $item) {
                if (!is_array($item)) throw new RuntimeException('Popis za skupno brisanje sadrži neispravnu stavku.');
                $path = PathGuard::ensureNotRoot((string)($item['path'] ?? ''));
                $type = (string)($item['type'] ?? 'file');
                if (!in_array($type, ['file', 'dir'], true)) throw new RuntimeException('Vrsta stavke za skupno brisanje nije valjana.');
                $validatedItems[] = ['path' => $path, 'type' => $type];
            }
            $deleted = 0;
            foreach ($validatedItems as $item) {
                $ops->deleteRecursive((string)$item['path'], (string)$item['type']);
                $deleted++;
            }
            AppLogger::event('item.bulk_delete', ['profile_id'=>$profileId,'count'=>$deleted]);
            GhostFTP_json(['ok'=>true,'deleted'=>$deleted]);

""",
)

patch_once(
    'GhostFTP WEB/app/Remote/SftpClient.php',
    """        try {
            @chmod($pub, 0600);
            @chmod($priv, 0600);
            $publicMaterial = $publicKey . (str_ends_with($publicKey, "\\n") ? '' : "\\n");
""",
    """        try {
            if (DIRECTORY_SEPARATOR === '/') {
                if (!@chmod($pub, 0600) || !@chmod($priv, 0600)) {
                    throw new RuntimeException('Nije moguće zaštititi privremene SFTP ključeve dozvolama 0600.');
                }
            } else {
                @chmod($pub, 0600);
                @chmod($priv, 0600);
            }
            $publicMaterial = $publicKey . (str_ends_with($publicKey, "\\n") ? '' : "\\n");
""",
)
patch_once(
    'GhostFTP WEB/app/Remote/SftpClient.php',
    """            @chmod($pub, 0600);
            @chmod($priv, 0600);
            $passphrase = (string)($this->profile['key_passphrase'] ?? '');
""",
    """            if (DIRECTORY_SEPARATOR === '/') {
                if (!@chmod($pub, 0600) || !@chmod($priv, 0600)) {
                    throw new RuntimeException('Nije moguće zadržati zaštitu privremenih SFTP ključeva dozvolama 0600.');
                }
            } else {
                @chmod($pub, 0600);
                @chmod($priv, 0600);
            }
            $passphrase = (string)($this->profile['key_passphrase'] ?? '');
""",
)

patch_once(
    'scripts/audit_web.py',
    '            "$privateWritten !== strlen($privateMaterial)",\n            "ssh2_sftp_lstat",',
    '            "$privateWritten !== strlen($privateMaterial)",\n            "DIRECTORY_SEPARATOR === \'/\'",\n            "dozvolama 0600",\n            "ssh2_sftp_lstat",',
)

api_test = Path('GhostFTP WEB/tests/api-batch-preflight.php')
if api_test.exists():
    raise SystemExit('PATCH_FAILED: api-batch-preflight.php already exists')
api_test.write_text("""<?php
declare(strict_types=1);

$source = file_get_contents(__DIR__ . '/../api.php');
if (!is_string($source)) {
    fwrite(STDERR, "FAIL: unable to inspect API batch-delete source.\\n");
    exit(1);
}
$start = strpos($source, "case 'bulk_delete':");
$end = $start !== false ? strpos($source, "case 'copy':", $start) : false;
if ($start === false || $end === false || $end <= $start) {
    fwrite(STDERR, "FAIL: unable to isolate bulk_delete API branch.\\n");
    exit(1);
}
$block = substr($source, $start, $end - $start);
$bufferPos = strpos($block, '$validatedItems = [];');
$validateLoopPos = strpos($block, 'foreach ($items as $item)');
$bufferAppendPos = strpos($block, '$validatedItems[] =');
$executeLoopPos = strpos($block, 'foreach ($validatedItems as $item)');
$deletePos = strpos($block, '$ops->deleteRecursive(', $executeLoopPos === false ? 0 : $executeLoopPos);
$malformedPos = strpos($block, 'Popis za skupno brisanje sadrži neispravnu stavku.');
$typePos = strpos($block, 'Vrsta stavke za skupno brisanje nije valjana.');
if (
    $bufferPos === false
    || $validateLoopPos === false
    || $bufferAppendPos === false
    || $executeLoopPos === false
    || $deletePos === false
    || $malformedPos === false
    || $typePos === false
    || !($bufferPos < $validateLoopPos
        && $validateLoopPos < $bufferAppendPos
        && $bufferAppendPos < $executeLoopPos
        && $executeLoopPos < $deletePos
        && $malformedPos < $executeLoopPos
        && $typePos < $executeLoopPos)
) {
    fwrite(STDERR, "FAIL: bulk_delete does not fully validate its batch before remote mutation.\\n");
    exit(1);
}

echo "WEB_API_BATCH_PREFLIGHT_TEST=PASS\\n";
""", encoding='utf-8', newline='\n')

print('GHOST_FTP_1_0_3_PREFLIGHT_FIX=APPLIED')
