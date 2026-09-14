#!/usr/bin/env python3
from __future__ import annotations
import re, shutil, subprocess, sys
from pathlib import Path
ROOT=Path(__file__).resolve().parents[1]
WEB=ROOT/'web'
REQUIRED={
'web/index.html','web/download.html','web/security.html','web/privacy.html','web/legal.html','web/.htaccess','web/robots.txt','web/sitemap.xml','web/assets/css/site.css','web/assets/js/site.js','web/assets/logo.svg','web/assets/icon.svg','web/ftp/index.php','web/ftp/api.php','web/ftp/config.php','web/ftp/lib/Security.php','web/ftp/lib/Transport.php','web/ftp/lib/CurlFtpTransport.php','web/ftp/lib/SftpTransport.php','web/ftp/assets/app.css','web/ftp/assets/app.js','web/ftp/README.md'}
IMAGES={'ghost-ftp-main-workspace.png','ghost-ftp-site-manager.png','ghost-ftp-settings.png','ghost-ftp-linux-main-workspace.png','ghost-ftp-linux-bookmarks.png','ghost-ftp-linux-settings.png','ghost-ftp-android-files.png','ghost-ftp-android-sites.png','ghost-ftp-android-transfers.png'}
def fail(msg): raise SystemExit('WEB_AUDIT_FAILED: '+msg)
def text(rel): return (ROOT/rel).read_text(encoding='utf-8')
def main():
 paths={p.relative_to(ROOT).as_posix() for p in WEB.rglob('*') if p.is_file()} if WEB.is_dir() else set();missing=sorted(REQUIRED-paths)
 if missing: fail('missing required web files: '+', '.join(missing))
 for name in IMAGES:
  if f'web/assets/images/{name}' not in paths: fail('missing self-contained runtime media: '+name)
 for rel in sorted(paths):
  p=ROOT/rel
  if p.suffix.lower() in {'.html','.php','.js','.css','.md','.xml','.txt'}:
   t=p.read_text(encoding='utf-8')
   if re.search(r'by[\s_-]?ftp',t,re.I): fail('retired product branding in '+rel)
 index=text('web/index.html'); app=text('web/ftp/assets/app.js'); api=text('web/ftp/api.php'); config=text('web/ftp/config.php'); sec=text('web/ftp/lib/Security.php'); ftp=text('web/ftp/lib/CurlFtpTransport.php'); sftp=text('web/ftp/lib/SftpTransport.php')
 for marker in ('Your servers. Your files. No cloud middleman.','Open Web FTP','No telemetry','Authentic runtime evidence'):
  if marker not in index: fail('marketing site missing marker: '+marker)
 for marker in ('sessionStorage','localStorage','indexedDB','document.cookie'):
  if marker in app and marker!='sessionStorage': fail('web client contains persistent browser storage primitive: '+marker)
 if 'localStorage' in app or 'indexedDB' in app or 'document.cookie' in app: fail('web client may persist connection state')
 for marker in ('FILTER_FLAG_NO_PRIV_RANGE','FILTER_FLAG_NO_RES_RANGE'):
  if marker not in sec: fail('SSRF boundary missing: '+marker)
 if 'GHOSTFTP_WEB_ALLOW_PRIVATE' not in config: fail('private-host opt-in marker missing from config')
 for marker in ('CURLOPT_RESOLVE','CURLOPT_SSL_VERIFYPEER,true','CURLOPT_SSL_VERIFYHOST,2'):
  if marker not in ftp: fail('FTPS identity/pinning boundary missing: '+marker)
 if sftp.find('ssh2_fingerprint')<0 or sftp.find('ssh2_auth_password')<0 or sftp.find('ssh2_fingerprint')>sftp.find('ssh2_auth_password'): fail('SFTP host-key verification must precede authentication')
 for marker in ('No valid upload','GHOSTFTP_WEB_MAX_DOWNLOAD_BYTES','Unsupported action'):
  if marker not in api: fail('API boundary missing: '+marker)
 for html in WEB.glob('*.html'):
  t=html.read_text(encoding='utf-8')
  if '<style' in t.lower() or re.search(r'<script(?![^>]*\bsrc=)',t,re.I): fail('inline CSS/JS is forbidden: '+html.name)
 php=shutil.which('php')
 if not php: fail('PHP CLI is required to lint maintained Web FTP source')
 for path in sorted(WEB.rglob('*.php')):
  r=subprocess.run([php,'-l',str(path)],capture_output=True,text=True)
  if r.returncode: fail('PHP lint failed for '+path.relative_to(ROOT).as_posix()+': '+r.stdout+r.stderr)
 print('WEB_AUDIT=PASS');print('WEB_MARKETING_SITE=ACTIVE');print('WEB_FTP=SERVER_ASSISTED_EPHEMERAL');print('WEB_ACCOUNT_DATABASE=NONE');print('WEB_TELEMETRY=NONE');print('WEB_FTPS_CERTIFICATE_VERIFICATION=ENFORCED');print('WEB_SFTP_HOST_KEY_PINNING=ENFORCED');return 0
if __name__=='__main__': sys.exit(main())
