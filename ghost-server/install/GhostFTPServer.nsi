Unicode True
!include "MUI2.nsh"

; Reuse the exact Windows client installer artwork and app icon.
!define MUI_ICON "..\..\ghostftp-desktop\src-tauri\icons\icon.ico"
!define MUI_UNICON "..\..\ghostftp-desktop\src-tauri\icons\icon.ico"
!define MUI_HEADERIMAGE
!define MUI_HEADERIMAGE_BITMAP "..\..\ghostftp-desktop\src-tauri\nsis\ghostftp-header.bmp"
!define MUI_UNHEADERIMAGE_BITMAP "..\..\ghostftp-desktop\src-tauri\nsis\ghostftp-header.bmp"
!define MUI_WELCOMEFINISHPAGE_BITMAP "..\..\ghostftp-desktop\src-tauri\nsis\ghostftp-sidebar.bmp"

Name "Ghost FTP Server (Preview)"
OutFile "..\dist\ghost-server-windows-x64-setup-unsigned.exe"
InstallDir "$PROGRAMFILES64\Ghost FTP Server"
RequestExecutionLevel admin
SetCompressor /SOLID lzma
ShowInstDetails show
ShowUninstDetails show

!define MUI_ABORTWARNING
!define MUI_WELCOMEPAGE_TITLE "Welcome to Ghost FTP Server"
!define MUI_WELCOMEPAGE_TEXT "Ghost FTP Server · More Than Transfer. Total Control.$\r$\n$\r$\nThis is an unsigned Preview installer for the standalone Ghost FTP Server. It installs files only: no default users, certificates, firewall rules, public listeners or automatically started services."
!define MUI_FINISHPAGE_TITLE "Ghost FTP Server files installed"
!define MUI_FINISHPAGE_TEXT "Manual service provisioning is required before use. Read the documentation and provide your own protected credentials and TLS identity. This installer does not start the server."
!define MUI_LICENSEPAGE_TEXT_TOP "Please review the Ghost FTP commercial license and Preview component terms."
!insertmacro MUI_PAGE_WELCOME
!insertmacro MUI_PAGE_LICENSE "..\..\EULA.txt"
!insertmacro MUI_PAGE_DIRECTORY
!insertmacro MUI_PAGE_INSTFILES
!insertmacro MUI_PAGE_FINISH
!insertmacro MUI_UNPAGE_CONFIRM
!insertmacro MUI_UNPAGE_INSTFILES
!insertmacro MUI_LANGUAGE "English"

Section "Ghost FTP Server (read-only default)"
 SetShellVarContext all
 ; Refuse to replace a binary while an SCM registration may still be using
 ; it. Reinstallation is only allowed after an explicit service unregister.
 nsExec::ExecToStack '"$SYSDIR\sc.exe" query GhostFTPServer'
 Pop $0
 StrCmp $0 0 0 +3
 MessageBox MB_ICONSTOP "GhostFTPServer is registered. Stop and unregister the service before updating the Preview installer."
 Abort
 SetOutPath "$INSTDIR"
 File "/oname=ghost-server.exe" "..\dist\ghost-server-windows-amd64.exe"
 File "Install-GhostFTPServer.ps1"
 File "Uninstall-GhostFTPServer.ps1"
 File "..\README.md"
 File "..\..\EULA.txt"
 File "..\..\LICENSE.txt"
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
 Delete "$INSTDIR\EULA.txt"
 Delete "$INSTDIR\LICENSE.txt"
 Delete "$INSTDIR\Uninstall.exe"
 RMDir "$INSTDIR"
 DeleteRegKey HKLM "Software\Microsoft\Windows\CurrentVersion\Uninstall\GhostFTPServerPreview"
 ; Never remove external roots, credentials, TLS keys or user data.
SectionEnd
