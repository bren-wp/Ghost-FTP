<?php
declare(strict_types=1);

const GHOSTFTP_WEB_VERSION = '0.0.6';
const GHOSTFTP_WEB_MAX_UPLOAD_BYTES = 268435456;
const GHOSTFTP_WEB_MAX_DOWNLOAD_BYTES = 536870912;
const GHOSTFTP_WEB_MAX_EDIT_BYTES = 2097152;
const GHOSTFTP_WEB_CONNECT_TIMEOUT = 12;
const GHOSTFTP_WEB_OPERATION_TIMEOUT = 120;

function ghostftp_allow_private_hosts(): bool
{
    return getenv('GHOSTFTP_WEB_ALLOW_PRIVATE') === '1';
}
