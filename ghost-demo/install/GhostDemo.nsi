Unicode True
!include "MUI2.nsh"
; Shared Ghost FTP Client installer vector-derived artwork.
!define MUI_ICON "ghostftp-desktop\src-tauri\icons\icon.ico"
!define MUI_UNICON "ghostftp-desktop\src-tauri\icons\icon.ico"
!define MUI_HEADERIMAGE
!define MUI_HEADERIMAGE_BITMAP "ghostftp-desktop\src-tauri\nsis\ghostftp-header.bmp"
!define MUI_UNHEADERIMAGE_BITMAP "ghostftp-desktop\src-tauri\nsis\ghostftp-header.bmp"
!define MUI_WELCOMEFINISHPAGE_BITMAP "ghostftp-desktop\src-tauri\nsis\ghostftp-sidebar.bmp"
Name "Ghost FTP Demo (Preview)"
OutFile "dist-tools\GhostFTP-Demo-Setup-Unsigned-Preview.exe"
InstallDir "$LOCALAPPDATA\Programs\Ghost FTP Demo"
RequestExecutionLevel user
SetCompressor /SOLID lzma
ShowInstDetails show
ShowUninstDetails show
!define MUI_ABORTWARNING
!define MUI_WELCOMEPAGE_TITLE "Welcome to Ghost FTP Demo"
!define MUI_WELCOMEPAGE_TEXT "Local interactive product demonstration. All accounts, transfers and files are simulated.$\\r$\\n$\\r$\\nGhost FTP by Brendigo. Separate unsigned Preview component."
!define MUI_FINISHPAGE_TITLE "Ghost FTP Demo files installed"
!insertmacro MUI_PAGE_WELCOME
!insertmacro MUI_PAGE_LICENSE "EULA.txt"
!insertmacro MUI_PAGE_DIRECTORY
!insertmacro MUI_PAGE_INSTFILES
!insertmacro MUI_PAGE_FINISH
!insertmacro MUI_UNPAGE_CONFIRM
!insertmacro MUI_UNPAGE_INSTFILES
!insertmacro MUI_LANGUAGE "English"
Section "Install Ghost FTP Demo"
 SetOutPath "$INSTDIR"
 File "/oname=ghostftp-demo.exe" "ghost-server\dist\GhostFTP-Demo-Windows-x64-Portable.exe"
 File "/oname=README.md" "ghost-demo\README.md"
 File "EULA.txt"
 WriteUninstaller "$INSTDIR\Uninstall.exe"
 SetRegView 64
 WriteRegStr HKCU "Software\Microsoft\Windows\CurrentVersion\Uninstall\GhostFTPDemoPreview" "DisplayName" "Ghost FTP Demo (Preview)"
 WriteRegStr HKCU "Software\Microsoft\Windows\CurrentVersion\Uninstall\GhostFTPDemoPreview" "UninstallString" '"$INSTDIR\Uninstall.exe"'
 WriteRegStr HKCU "Software\Microsoft\Windows\CurrentVersion\Uninstall\GhostFTPDemoPreview" "InstallLocation" "$INSTDIR"
 WriteRegDWORD HKCU "Software\Microsoft\Windows\CurrentVersion\Uninstall\GhostFTPDemoPreview" "NoModify" 1
 CreateDirectory "$SMPROGRAMS\Ghost FTP Demo"
 CreateShortcut "$SMPROGRAMS\Ghost FTP Demo\Ghost FTP Demo.lnk" "$INSTDIR\ghostftp-demo.exe"
 CreateShortcut "$SMPROGRAMS\Ghost FTP Demo\Uninstall Ghost FTP Demo.lnk" "$INSTDIR\Uninstall.exe"
 DetailPrint "No credentials, servers, services, or firewall configuration created."
SectionEnd
Section "Uninstall"
 SetRegView 64
 Delete "$SMPROGRAMS\Ghost FTP Demo\Ghost FTP Demo.lnk"
 Delete "$SMPROGRAMS\Ghost FTP Demo\Uninstall Ghost FTP Demo.lnk"
 RMDir "$SMPROGRAMS\Ghost FTP Demo"
 Delete "$INSTDIR\ghostftp-demo.exe"
 Delete "$INSTDIR\README.md"
 Delete "$INSTDIR\EULA.txt"
 Delete "$INSTDIR\Uninstall.exe"
 RMDir "$INSTDIR"
 DeleteRegKey HKCU "Software\Microsoft\Windows\CurrentVersion\Uninstall\GhostFTPDemoPreview"
 ; No user-created or externally stored data is deleted.
SectionEnd
