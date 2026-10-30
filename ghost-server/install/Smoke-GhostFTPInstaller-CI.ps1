#Requires -RunAsAdministrator
[CmdletBinding()]
param([Parameter(Mandatory=$true)][string]$Setup)
$ErrorActionPreference='Stop'
$installer=(Resolve-Path -LiteralPath $Setup).Path
$installation=Join-Path $env:RUNNER_TEMP 'GhostFTP-Unsigned-Setup-Smoke'
$externalData=Join-Path $env:RUNNER_TEMP 'GhostFTP-Outside-Installer-Data-Keep.txt'
Set-Content -LiteralPath $externalData -Value 'must-survive-uninstall' -NoNewline
if (Get-Service -Name 'GhostFTPServer' -ErrorAction SilentlyContinue) {
 throw 'Cannot exercise unsigned setup with an existing service registration'
}
if (Test-Path -LiteralPath $installation) {
 throw "Refusing to overwrite an existing installation test folder: $installation"
}
$installed=$false
try {
 $process=Start-Process -FilePath $installer -ArgumentList @('/S',"/D=$installation") -Wait -PassThru
 if ($process.ExitCode -ne 0) {throw "NSIS silent setup failed: $($process.ExitCode)"}
 $executable=Join-Path $installation 'ghost-server.exe'
 $uninstaller=Join-Path $installation 'Uninstall.exe'
 $nativeGui=Join-Path $installation 'GhostFTPServer-ControlCenter.exe'
 if (-not (Test-Path -LiteralPath $nativeGui -PathType Leaf)) { throw 'Installed Ghost FTP Server has no native Control Center EXE' }
 $shortcut=Join-Path ([Environment]::GetFolderPath('CommonPrograms')) 'Ghost FTP Server/Ghost FTP Server Control Center.lnk'
 if (-not (Test-Path -LiteralPath $shortcut -PathType Leaf)) { throw 'Native Control Center Start Menu shortcut is missing' }
 if (-not (Test-Path -LiteralPath $executable -PathType Leaf)) {throw 'NSIS did not install real server binary'}
 if (-not (Test-Path -LiteralPath $uninstaller -PathType Leaf)) {throw 'NSIS did not install its real uninstaller'}
 if (Get-Service -Name 'GhostFTPServer' -ErrorAction SilentlyContinue) {throw 'Installer unexpectedly registered a service'}
 foreach ($file in @('README.md','EULA.txt','LICENSE.txt')) {
  if (-not (Test-Path -LiteralPath (Join-Path $installation $file) -PathType Leaf)) {
   throw "NSIS install is missing required product/legal file $file"
  }
 }
 $installed=$true
 $uninstallProcess=Start-Process -FilePath $uninstaller -ArgumentList '/S' -Wait -PassThru
 if ($uninstallProcess.ExitCode -ne 0) {throw "NSIS silent uninstall failed: $($uninstallProcess.ExitCode)"}
 if (Test-Path -LiteralPath $executable) {throw 'Uninstaller left executable after completion'}
 if (Test-Path -LiteralPath $nativeGui) {throw 'Uninstaller left native Control Center EXE'}
 if (-not (Test-Path -LiteralPath $externalData)) {throw 'Uninstaller deleted external user data'}
 if ((Get-Content -LiteralPath $externalData -Raw) -ne 'must-survive-uninstall') {throw 'Uninstaller modified external data'}
 Write-Output 'NSIS install/uninstall, no auto-service registration, and user-data preservation passed.'
} finally {
 if ($installed -and (Test-Path -LiteralPath (Join-Path $installation 'Uninstall.exe'))) {
   Write-Warning 'An NSIS test installation is still present; refusing silent destructive cleanup.'
 }
 if (Test-Path -LiteralPath $externalData) {
   Remove-Item -LiteralPath $externalData -Force
 }
}
