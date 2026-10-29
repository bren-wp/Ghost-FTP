#Requires -RunAsAdministrator
[CmdletBinding()]
param(
 [Parameter(Mandatory = $true)][string]$Executable,
 [Parameter(Mandatory = $true)][string]$FixtureDir
)
$ErrorActionPreference='Stop'
$serviceName='GhostFTPServer'
$exePath=(Resolve-Path -LiteralPath $Executable).Path
$root=Join-Path $FixtureDir 'root'
$cert=Join-Path $FixtureDir 'server-cert.pem'
$key=Join-Path $FixtureDir 'server-key.pem'
$pass=Join-Path $FixtureDir 'password.txt'
try {
 & go run ./internal/testfixture -out $FixtureDir
 if ($LASTEXITCODE -ne 0) {throw 'Cannot generate transient QA-only TLS identity'}
 & icacls.exe $FixtureDir /grant '*S-1-5-19:(OI)(CI)RX' /T | Out-Null
 if ($LASTEXITCODE -ne 0) {throw 'Cannot grant read-only QA ACL to LocalService'}
 & .\install\Install-GhostFTPServer.ps1 -Executable $exePath -Root $root -Certificate $cert -PrivateKey $key -PasswordFile $pass -Username qa
 if (-not (Get-Service -Name $serviceName -ErrorAction SilentlyContinue)) {throw 'SCM did not register the service'}
 Start-Service -Name $serviceName -ErrorAction Stop
 (Get-Service -Name $serviceName).WaitForStatus('Running',[TimeSpan]::FromSeconds(20))
 $tcp=[Net.Sockets.TcpClient]::new()
 try {
  $tcp.Connect('127.0.0.1',2121)
  $tcp.ReceiveTimeout=4000
  $reader=[IO.StreamReader]::new($tcp.GetStream(),[Text.Encoding]::ASCII)
  $greeting=$reader.ReadLine()
  if (-not $greeting.StartsWith('220 ')) {throw "Unexpected real Windows service FTP greeting: $greeting"}
 } finally {
  $tcp.Dispose()
 }
 Stop-Service -Name $serviceName -ErrorAction Stop
 (Get-Service -Name $serviceName).WaitForStatus('Stopped',[TimeSpan]::FromSeconds(20))
 Write-Output 'Real Windows LocalService start, TCP greeting, stop and SCM lifecycle passed.'
}
finally {
 if (Get-Service -Name $serviceName -ErrorAction SilentlyContinue) {
  & .\install\Uninstall-GhostFTPServer.ps1
 }
}
