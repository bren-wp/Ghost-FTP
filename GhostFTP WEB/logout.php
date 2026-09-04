<?php
declare(strict_types=1);
require __DIR__ . '/app/bootstrap.php';
GhostFTP\Security\Auth::logout();
GhostFTP_redirect('login');
