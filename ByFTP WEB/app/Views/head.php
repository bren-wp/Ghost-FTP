<?php
declare(strict_types=1);
/** @var string $pageTitle */
?><meta charset="utf-8">
<meta name="viewport" content="width=device-width, initial-scale=1, viewport-fit=cover">
<meta name="robots" content="noindex,nofollow,noarchive,nosnippet,noimageindex">
<meta name="googlebot" content="noindex,nofollow,noarchive,nosnippet,noimageindex">
<meta name="color-scheme" content="dark">
<meta name="theme-color" content="#090b10">
<meta name="application-name" content="<?= byftp_e(byftp_app_name()) ?>">
<meta name="mobile-web-app-capable" content="yes">
<meta name="apple-mobile-web-app-capable" content="yes">
<meta name="apple-mobile-web-app-status-bar-style" content="black-translucent">
<meta name="apple-mobile-web-app-title" content="Ghost FTP">
<meta name="service-worker-url" content="<?= byftp_e(byftp_url('service_worker')) ?>">
<title><?= byftp_e($pageTitle) ?></title>
<link rel="icon" href="<?= byftp_e(byftp_asset('images/favicon.svg')) ?>" type="image/svg+xml">
<link rel="manifest" href="<?= byftp_e(byftp_url('manifest')) ?>">
<link rel="stylesheet" href="<?= byftp_e(byftp_asset('css/app.css')) ?>">
<link rel="stylesheet" href="<?= byftp_e(byftp_asset('css/brendigo.css')) ?>">
