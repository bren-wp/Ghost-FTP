#!/usr/bin/env python3
from pathlib import Path
import shutil
import subprocess
import sys
import unittest
ROOT=Path(__file__).resolve().parents[1]
class WebContractTests(unittest.TestCase):
 def read(self,rel): return (ROOT/rel).read_text(encoding='utf-8')
 def test_web_site_is_self_contained_and_branded(self):
  index=self.read('web/index.html'); self.assertIn('Ghost FTP 0.0.6',index); self.assertIn('assets/images/ghost-ftp-main-workspace.png',index); self.assertNotIn('../docs/',index); self.assertNotRegex(index.lower(), r'by[\s_-]?ftp')
 def test_web_ftp_is_truthful_about_transport_boundary(self):
  readme=self.read('web/ftp/README.md'); self.assertIn('browser cannot open raw FTP/FTPS/SFTP TCP sockets directly',readme); self.assertIn('ephemeral server-assisted transport',readme); self.assertIn('does not persist them',readme)
 def test_web_client_has_no_persistent_credential_storage(self):
  js=self.read('web/ftp/assets/app.js'); self.assertNotIn('localStorage',js); self.assertNotIn('indexedDB',js); self.assertNotIn('document.cookie',js)
 def test_ftps_and_sftp_identity_fail_closed(self):
  ftp=self.read('web/ftp/lib/CurlFtpTransport.php'); sftp=self.read('web/ftp/lib/SftpTransport.php'); self.assertIn('CURLOPT_SSL_VERIFYPEER,true',ftp); self.assertIn('CURLOPT_SSL_VERIFYHOST,2',ftp); self.assertLess(sftp.index('ssh2_fingerprint'),sftp.index('ssh2_auth_password'))
 def test_full_web_audit_when_php_is_available(self):
  if shutil.which('php') is None: self.skipTest('PHP CLI not available on this runner')
  r=subprocess.run([sys.executable,str(ROOT/'scripts/check_web_contract.py')],cwd=ROOT,capture_output=True,text=True)
  self.assertEqual(r.returncode,0,r.stdout+r.stderr)
if __name__=='__main__': unittest.main()
