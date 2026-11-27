#!/usr/bin/env bash
set -euo pipefail
root="$(cd "$(dirname "$0")/../.." && pwd)"
work="$(mktemp -d)"
chmod 755 "$work"
cleanup() {
  if [ -f "$work/httpd.pid" ]; then apache2 -f "$work/httpd.conf" -k stop || true; fi
  rm -rf "$work"
}
trap cleanup EXIT
mkdir "$work/public"
cp "${1:?deployment ZIP required}" "$work/root.zip"
version="${2:?version required}"
unzip -q "$work/root.zip" -d "$work/public"
php -l "$work/public/index.php"
php -l "$work/public/updates/index.php"
php_module="$(find /usr/lib/apache2/modules -name 'libphp*.so' -print -quit)"
test -n "$php_module"
cat > "$work/httpd.conf" <<EOF
ServerRoot /etc/apache2
Listen 127.0.0.1:18087
ServerName localhost
PidFile $work/httpd.pid
ErrorLog $work/error.log
LoadModule mpm_prefork_module /usr/lib/apache2/modules/mod_mpm_prefork.so
LoadModule authz_core_module /usr/lib/apache2/modules/mod_authz_core.so
LoadModule dir_module /usr/lib/apache2/modules/mod_dir.so
LoadModule mime_module /usr/lib/apache2/modules/mod_mime.so
LoadModule rewrite_module /usr/lib/apache2/modules/mod_rewrite.so
LoadModule headers_module /usr/lib/apache2/modules/mod_headers.so
LoadModule php_module $php_module
TypesConfig /etc/mime.types
User www-data
Group www-data
DocumentRoot $work/public
<Directory $work/public>
AllowOverride All
Require all granted
<FilesMatch "\.php$">
SetHandler application/x-httpd-php
</FilesMatch>
</Directory>
EOF
apache2 -f "$work/httpd.conf" -k start
for attempt in $(seq 1 20); do
  if curl --silent --fail http://127.0.0.1:18087/ >/dev/null; then break; fi
  sleep 0.2
done
python3 "$root/update.ghostftp.com/tests/smoke.py" http://127.0.0.1:18087 "$version"
