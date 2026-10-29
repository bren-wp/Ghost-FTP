#Requires -RunAsAdministrator
[CmdletBinding(SupportsShouldProcess = $true)]
param()
$ErrorActionPreference = 'Stop'
$name='GhostFTPServer'
$service=Get-Service -Name $name -ErrorAction SilentlyContinue
if (-not $service) { Write-Output 'GhostFTPServer is not installed.'; return }
if ($PSCmdlet.ShouldProcess($name, 'Stop and remove the service registration only')) {
 if ($service.Status -ne 'Stopped') {
  Stop-Service -Name $name -ErrorAction Stop
  (Get-Service -Name $name).WaitForStatus('Stopped',[TimeSpan]::FromSeconds(30))
 }
 & sc.exe delete $name
 if ($LASTEXITCODE -ne 0) {throw "SCM removal failed: $LASTEXITCODE"}
 Write-Output 'Service registration removed; files, accounts, TLS keys and data were intentionally preserved.'
}
