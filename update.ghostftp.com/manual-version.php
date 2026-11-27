<?php
declare(strict_types=1);
require_once __DIR__ . '/catalog.php';
header('Cache-Control: no-store');
header('Content-Type: application/json; charset=UTF-8');
[$version, , $ready] = local_release();
if (!$ready) {
    http_response_code(204);
    exit;
}
echo json_encode(['version' => $version], JSON_THROW_ON_ERROR);
