Unicode True
!include "MUI2.nsh"
; Shared Ghost FTP Client installer vector-derived artwork.
!define MUI_ICON "..\..\ghostftp-desktop\src-tauri\icons\icon.ico"
!define MUI_UNICON "..\..\ghostftp-desktop\src-tauri\icons\icon.ico"
!define MUI_HEADERIMAGE
!define MUI_HEADERIMAGE_BITMAP "..\..\ghostftp-desktop\src-tauri\nsis\ghostftp-header.bmp"
!define MUI_UNHEADERIMAGE_BITMAP "..\..\ghostftp-desktop\src-tauri\nsis\ghostftp-header.bmp"
!define MUI_WELCOMEFINISHPAGE_BITMAP "..\..\ghostftp-desktop\src-tauri\nsis\ghostftp-sidebar.bmp"
Name "Ghost CLI (Preview)"
OutFile "..\..\dist-tools\GhostCLI-Setup-Unsigned-Preview.exe"
InstallDir "$LOCALAPPDATA\Programs\Ghost CLI"
RequestExecutionLevel user
SetCompressor /SOLID lzma
ShowInstDetails show
ShowUninstDetails show
!define MUI_ABORTWARNING
!define MUI_WELCOMEPAGE_TITLE "Welcome to Ghost CLI"
!define MUI_WELCOMEPAGE_TEXT "The genuine command-line FTP, FTPS and SFTP client. Uses the same transfer engine as Ghost FTP.$\\r$\\n$\\r$\\nGhost FTP by Brendigo. Separate unsigned Preview component."
!define MUI_FINISHPAGE_TITLE "Ghost CLI files installed"
!insertmacro MUI_PAGE_WELCOME
!insertmacro MUI_PAGE_LICENSE "..\..\EULA.txt"
!insertmacro MUI_PAGE_DIRECTORY
!insertmacro MUI_PAGE_INSTFILES
!insertmacro MUI_PAGE_FINISH
!insertmacro MUI_UNPAGE_CONFIRM
!insertmacro MUI_UNPAGE_INSTFILES
!insertmacro MUI_LANGUAGE "English"
Section "Install Ghost CLI"
 SetOutPath "$INSTDIR"
 File "/oname=ghostftp-cli.exe" "..\..\ghostftp-desktop\src-tauri\target\release\ghostftp-cli.exe"
 File "/oname=README.md" "..\..\ghost-cli\README.md"
 File "..\..\EULA.txt"
 WriteUninstaller "$INSTDIR\Uninstall.exe"
 SetRegView 64
 WriteRegStr HKCU "Software\Microsoft\Windows\CurrentVersion\Uninstall\GhostCLIPreview" "DisplayName" "Ghost CLI (Preview)"
 WriteRegStr HKCU "Software\Microsoft\Windows\CurrentVersion\Uninstall\GhostCLIPreview" "UninstallString" '"$INSTDIR\Uninstall.exe"'
 WriteRegStr HKCU "Software\Microsoft\Windows\CurrentVersion\Uninstall\GhostCLIPreview" "InstallLocation" "$INSTDIR"
 WriteRegDWORD HKCU "Software\Microsoft\Windows\CurrentVersion\Uninstall\GhostCLIPreview" "NoModify" 1
 CreateDirectory "$SMPROGRAMS\Ghost CLI"
 CreateShortcut "$SMPROGRAMS\Ghost CLI\Ghost CLI.lnk" "$INSTDIR\ghostftp-cli.exe" "--help"
 CreateShortcut "$SMPROGRAMS\Ghost CLI\Uninstall Ghost CLI.lnk" "$INSTDIR\Uninstall.exe"
 DetailPrint "No credentials, servers, services, or firewall configuration created."
SectionEnd
Section "Uninstall"
 SetRegView 64
 Delete "$SMPROGRAMS\Ghost CLI\Ghost CLI.lnk"
 Delete "$SMPROGRAMS\Ghost CLI\Uninstall Ghost CLI.lnk"
 RMDir "$SMPROGRAMS\Ghost CLI"
 Delete "$INSTDIR\ghostftp-cli.exe"
 Delete "$INSTDIR\README.md"
 Delete "$INSTDIR\EULA.txt"
 Delete "$INSTDIR\Uninstall.exe"
 RMDir "$INSTDIR"
 DeleteRegKey HKCU "Software\Microsoft\Windows\CurrentVersion\Uninstall\GhostCLIPreview"
 ; No user-created or externally stored data is deleted.
SectionEnd
