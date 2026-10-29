Unicode True
!include "MUI2.nsh"

Name "Ghost FTP Server (Preview)"
OutFile "dist\ghost-server-windows-x64-setup-unsigned.exe"
InstallDir "$PROGRAMFILES64\Ghost FTP Server"
RequestExecutionLevel admin
SetCompressor /SOLID lzma
ShowInstDetails show
ShowUninstDetails show

!define MUI_ABORTWARNING
!insertmacro MUI_PAGE_WELCOME
!insertmacro MUI_PAGE_DIRECTORY
!insertmacro MUI_PAGE_INSTFILES
!insertmacro MUI_PAGE_FINISH
!insertmacro MUI_UNPAGE_CONFIRM
!insertmacro MUI_UNPAGE_INSTFILES
!insertmacro MUI_LANGUAGE "English"

Section "Ghost FTP Server (read-only default)"
 SetShellVarContext all
 SetOutPath "$INSTDIR"
 File "/oname=ghost-server.exe" "dist\ghost-server-windows-amd64.exe"
 File "install\Install-GhostFTPServer.ps1"
 File "install\Uninstall-GhostFTPServer.ps1"
 File "README.md"
 WriteUninstaller "$INSTDIR\Uninstall.exe"
 SetRegView 64
 WriteRegStr HKLM "Software\Microsoft\Windows\CurrentVersion\Uninstall\GhostFTPServerPreview" "DisplayName" "Ghost FTP Server (Preview)"
 WriteRegStr HKLM "Software\Microsoft\Windows\CurrentVersion\Uninstall\GhostFTPServerPreview" "UninstallString" '"$INSTDIR\Uninstall.exe"'
 WriteRegStr HKLM "Software\Microsoft\Windows\CurrentVersion\Uninstall\GhostFTPServerPreview" "InstallLocation" "$INSTDIR"
 WriteRegDWORD HKLM "Software\Microsoft\Windows\CurrentVersion\Uninstall\GhostFTPServerPreview" "NoModify" 1
 CreateDirectory "$SMPROGRAMS\Ghost FTP Server"
 CreateShortcut "$SMPROGRAMS\Ghost FTP Server\Uninstall Ghost FTP Server.lnk" "$INSTDIR\Uninstall.exe"
 DetailPrint "Files installed. No service registered, no credentials created, no firewall ports opened."
SectionEnd

Section "Uninstall"
 SetShellVarContext all
 SetRegView 64
 ; Do not leave an orphaned registered service behind.
 nsExec::ExecToStack '"$SYSDIR\sc.exe" query GhostFTPServer'
 Pop $0
 StrCmp $0 0 service_registered service_absent
 service_registered:
 MessageBox MB_ICONSTOP "GhostFTPServer remains registered. Unregister it using Uninstall-GhostFTPServer.ps1 before uninstalling."
 Abort
 service_absent:
 Delete "$SMPROGRAMS\Ghost FTP Server\Uninstall Ghost FTP Server.lnk"
 RMDir "$SMPROGRAMS\Ghost FTP Server"
 Delete "$INSTDIR\ghost-server.exe"
 Delete "$INSTDIR\Install-GhostFTPServer.ps1"
 Delete "$INSTDIR\Uninstall-GhostFTPServer.ps1"
 Delete "$INSTDIR\README.md"
 Delete "$INSTDIR\Uninstall.exe"
 RMDir "$INSTDIR"
 DeleteRegKey HKLM "Software\Microsoft\Windows\CurrentVersion\Uninstall\GhostFTPServerPreview"
 ; Never remove external roots, credentials, TLS keys or user data.
SectionEnd
