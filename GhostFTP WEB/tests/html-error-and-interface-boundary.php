<?php
declare(strict_types=1);

$root = dirname(__DIR__);
$publicPages = ['account.php', 'login.php', 'register.php', 'users.php', 'settings.php', 'setup.php'];
foreach ($publicPages as $page) {
    $source = file_get_contents($root . '/' . $page);
    if (!is_string($source)) {
        fwrite(STDERR, "FAIL: unable to inspect {$page}.\n");
        exit(1);
    }
    if (str_contains($source, '$error = $e->getMessage();')) {
        fwrite(STDERR, "FAIL: {$page} still exposes raw Throwable messages.\n");
        exit(1);
    }
    if (!str_contains($source, 'GhostFTP_public_error($e)')) {
        fwrite(STDERR, "FAIL: {$page} does not use the shared public error boundary.\n");
        exit(1);
    }
}

$operations = file_get_contents($root . '/app/Operations/RemoteOperations.php');
$userStore = file_get_contents($root . '/app/Storage/UserStore.php');
if (!is_string($operations) || !is_string($userStore)) {
    fwrite(STDERR, "FAIL: unable to inspect exception wrappers.\n");
    exit(1);
}
if (str_contains($operations, '$deleteError->getMessage()') || str_contains($userStore, '$e->getMessage()')) {
    fwrite(STDERR, "FAIL: Throwable wrapper can still re-expose its previous raw message.\n");
    exit(1);
}

$transportFiles = [
    'app/Remote/RemoteClientInterface.php',
    'app/Remote/FtpClient.php',
    'app/Remote/SftpClient.php',
    'tests/unit.php',
    'tests/zip-extraction-preflight.php',
];
foreach ($transportFiles as $relative) {
    $source = file_get_contents($root . '/' . $relative);
    if (!is_string($source)) {
        fwrite(STDERR, "FAIL: unable to inspect {$relative}.\n");
        exit(1);
    }
    if (str_contains($source, 'function write(string $remotePath')) {
        fwrite(STDERR, "FAIL: dead transport write() contract remains in {$relative}.\n");
        exit(1);
    }
}
$interface = file_get_contents($root . '/app/Remote/RemoteClientInterface.php');
if (!is_string($interface) || !str_contains($interface, 'int $maxBytes = 4194304')) {
    fwrite(STDERR, "FAIL: remote read contract is not aligned to the 4 MiB editor boundary.\n");
    exit(1);
}

echo "WEB_HTML_ERROR_AND_INTERFACE_BOUNDARY_TEST=PASS\n";
