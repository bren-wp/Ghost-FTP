<?php
declare(strict_types=1);
require __DIR__ . '/app/bootstrap.php';
ByFTP\Security\Auth::logout();
byftp_redirect('login');
