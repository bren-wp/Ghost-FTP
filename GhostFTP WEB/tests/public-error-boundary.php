<?php
declare(strict_types=1);

require __DIR__ . '/../app/Remote/PathGuard.php';
require __DIR__ . '/../app/helpers.php';

$known = GhostFTP_public_error(new RuntimeException("Poznata validacija\r\nne smije curiti u više redaka"));
if ($known !== 'Poznata validacija  ne smije curiti u više redaka') {
    fwrite(STDERR, "FAIL: known application error was not normalized.\n");
    exit(1);
}
if (GhostFTP_public_error_status(new RuntimeException('x')) !== 400) {
    fwrite(STDERR, "FAIL: known application error status is not 400.\n");
    exit(1);
}
$internal = new TypeError('/srv/private/customer/secret.php: internal detail');
if (GhostFTP_public_error($internal) !== 'Neočekivana interna pogreška. Pokušaj ponovno ili provjeri zapisnik poslužitelja.') {
    fwrite(STDERR, "FAIL: unexpected Throwable detail reached public error text.\n");
    exit(1);
}
if (GhostFTP_public_error_status($internal) !== 500) {
    fwrite(STDERR, "FAIL: unexpected Throwable status is not 500.\n");
    exit(1);
}
try {
    GhostFTP_string_paths(json_encode(['/safe', ['bad']]), 10);
    fwrite(STDERR, "FAIL: malformed ZIP path-list row was silently ignored.\n");
    exit(1);
} catch (RuntimeException $e) {
    if (!str_contains($e->getMessage(), 'neispravnu stavku')) {
        throw $e;
    }
}

echo "WEB_PUBLIC_ERROR_BOUNDARY_TEST=PASS\n";
