#Requires -RunAsAdministrator
[CmdletBinding(SupportsShouldProcess = $true)]
param(
 [Parameter(Mandatory = $true)][string]$Executable,
 [Parameter(Mandatory = $true)][string]$Root,
 [Parameter(Mandatory = $true)][string]$Certificate,
 [Parameter(Mandatory = $true)][string]$PrivateKey,
 [Parameter(Mandatory = $true)][string]$PasswordFile,
 [Parameter(Mandatory = $true)][string]$Username
)
$ErrorActionPreference = 'Stop'
$serviceName = 'GhostFTPServer'
# Service credentials are not passed in process arguments or stored by SCM.
# LocalService is restricted; configure explicit ACL read access to root,
# certificate, private key and password file before attempting to start.
foreach ($value in @($Executable, $Root, $Certificate, $PrivateKey, $PasswordFile, $Username)) {
 if ([string]::IsNullOrWhiteSpace($value) -or $value.Contains('"') -or $value.Contains("`r") -or $value.Contains("`n")) {
   throw 'Invalid service configuration value.'
 }
}
foreach ($item in @($Executable, $Root, $Certificate, $PrivateKey, $PasswordFile)) {
 if (-not (Test-Path -LiteralPath $item)) { throw "Required path not found: $item" }
}
$exe = (Resolve-Path -LiteralPath $Executable).Path
$rootPath = (Resolve-Path -LiteralPath $Root).Path
$certPath = (Resolve-Path -LiteralPath $Certificate).Path
$keyPath = (Resolve-Path -LiteralPath $PrivateKey).Path
$passPath = (Resolve-Path -LiteralPath $PasswordFile).Path
$binary = ('"{0}" -service -listen 127.0.0.1:2121 -root "{1}" -cert "{2}" -key "{3}" -password-file "{4}" -user "{5}"' -f $exe, $rootPath, $certPath, $keyPath, $passPath, $Username)
if (Get-Service -Name $serviceName -ErrorAction SilentlyContinue) {
 throw 'GhostFTPServer already exists. Stop and remove it through the official uninstall script before reinstalling.'
}
if ($PSCmdlet.ShouldProcess($serviceName, 'Register manual-start LocalService Windows service')) {
 & sc.exe create $serviceName binPath= $binary start= demand obj= 'NT AUTHORITY\LocalService'
 if ($LASTEXITCODE -ne 0) { throw "SCM service registration failed: $LASTEXITCODE" }
 & sc.exe description $serviceName 'Ghost FTP Server - explicit FTPS, local read-only by default'
 if ($LASTEXITCODE -ne 0) { throw "SCM description failed: $LASTEXITCODE" }
 Write-Output 'Registered GhostFTPServer as LocalService (manual start, loopback only). Configure file ACLs, then start with: sc.exe start GhostFTPServer'
}
