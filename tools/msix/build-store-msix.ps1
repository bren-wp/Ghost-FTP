[CmdletBinding()]
param(
    [Parameter(Mandatory = $true)][string]$InputExe,
    [Parameter(Mandatory = $true)][string]$IdentityName,
    [Parameter(Mandatory = $true)][string]$Publisher,
    [Parameter(Mandatory = $true)][ValidatePattern('^\d+\.\d+\.\d+\.\d+$')][string]$PackageVersion,
    [Parameter(Mandatory = $true)][ValidatePattern('^\d+\.\d+\.\d+\.\d+$')][string]$MinVersion,
    [Parameter(Mandatory = $true)][ValidatePattern('^\d+\.\d+\.\d+\.\d+$')][string]$MaxVersionTested,
    [string]$PublisherDisplayName = 'Brendigo',
    [string]$OutputPath = 'dist/GhostFTP-Windows-x64-Store.msix'
)

$ErrorActionPreference = 'Stop'
Set-StrictMode -Version Latest

function Escape-XmlAttribute([string]$Value) {
    return [System.Security.SecurityElement]::Escape($Value)
}

$inputResolved = (Resolve-Path -LiteralPath $InputExe).Path
if ([IO.Path]::GetExtension($inputResolved) -ne '.exe') { throw 'InputExe must be a compiled EXE.' }

$repoRoot = (Resolve-Path (Join-Path $PSScriptRoot '..\..')).Path
$sourceIcon = Join-Path $repoRoot 'ghostftp-desktop\src-tauri\icons\source.png'
if (-not (Test-Path -LiteralPath $sourceIcon)) { throw "Ghost FTP source icon not found: $sourceIcon" }

$workRoot = Join-Path $env:RUNNER_TEMP ('ghostftp-msix-' + [Guid]::NewGuid().ToString('N'))
$stage = Join-Path $workRoot 'package'
$appDir = Join-Path $stage 'VFS\ProgramFilesX64\Ghost FTP'
$assetsDir = Join-Path $stage 'Assets'
New-Item -ItemType Directory -Path $appDir -Force | Out-Null
New-Item -ItemType Directory -Path $assetsDir -Force | Out-Null
Copy-Item -LiteralPath $inputResolved -Destination (Join-Path $appDir 'GhostFTP.exe')

Add-Type -AssemblyName System.Drawing
function New-PngAsset {
    param([string]$Source,[string]$Destination,[int]$Width,[int]$Height)
    $src = [System.Drawing.Image]::FromFile($Source)
    try {
        $bmp = New-Object System.Drawing.Bitmap($Width,$Height)
        try {
            $g = [System.Drawing.Graphics]::FromImage($bmp)
            try {
                $g.Clear([System.Drawing.Color]::Transparent)
                $g.InterpolationMode = [System.Drawing.Drawing2D.InterpolationMode]::HighQualityBicubic
                $g.SmoothingMode = [System.Drawing.Drawing2D.SmoothingMode]::HighQuality
                $scale = [Math]::Min($Width / $src.Width, $Height / $src.Height)
                $dw = [Math]::Max(1,[int][Math]::Round($src.Width * $scale))
                $dh = [Math]::Max(1,[int][Math]::Round($src.Height * $scale))
                $x = [int](($Width - $dw) / 2)
                $y = [int](($Height - $dh) / 2)
                $g.DrawImage($src,$x,$y,$dw,$dh)
            } finally { $g.Dispose() }
            $bmp.Save($Destination,[System.Drawing.Imaging.ImageFormat]::Png)
        } finally { $bmp.Dispose() }
    } finally { $src.Dispose() }
}

New-PngAsset $sourceIcon (Join-Path $assetsDir 'StoreLogo.png') 50 50
New-PngAsset $sourceIcon (Join-Path $assetsDir 'Square44x44Logo.png') 44 44
New-PngAsset $sourceIcon (Join-Path $assetsDir 'Square150x150Logo.png') 150 150

$identityEscaped = Escape-XmlAttribute $IdentityName
$publisherEscaped = Escape-XmlAttribute $Publisher
$publisherDisplayEscaped = Escape-XmlAttribute $PublisherDisplayName
$manifest = @"
<?xml version="1.0" encoding="utf-8"?>
<Package
  xmlns="http://schemas.microsoft.com/appx/manifest/foundation/windows10"
  xmlns:uap="http://schemas.microsoft.com/appx/manifest/uap/windows10"
  xmlns:rescap="http://schemas.microsoft.com/appx/manifest/foundation/windows10/restrictedcapabilities"
  IgnorableNamespaces="uap rescap">
  <Identity Name="$identityEscaped" Publisher="$publisherEscaped" Version="$PackageVersion" ProcessorArchitecture="x64" />
  <Properties>
    <DisplayName>Ghost FTP</DisplayName>
    <PublisherDisplayName>$publisherDisplayEscaped</PublisherDisplayName>
    <Logo>Assets\StoreLogo.png</Logo>
  </Properties>
  <Resources><Resource Language="en-us" /></Resources>
  <Dependencies>
    <TargetDeviceFamily Name="Windows.Desktop" MinVersion="$MinVersion" MaxVersionTested="$MaxVersionTested" />
  </Dependencies>
  <Applications>
    <Application Id="GhostFTP" Executable="VFS\ProgramFilesX64\Ghost FTP\GhostFTP.exe" EntryPoint="Windows.FullTrustApplication">
      <uap:VisualElements DisplayName="Ghost FTP" Description="Ghost FTP secure FTP, FTPS and SFTP client" BackgroundColor="transparent" Square150x150Logo="Assets\Square150x150Logo.png" Square44x44Logo="Assets\Square44x44Logo.png" />
    </Application>
  </Applications>
  <Capabilities><rescap:Capability Name="runFullTrust" /></Capabilities>
</Package>
"@

$manifestPath = Join-Path $stage 'AppxManifest.xml'
[IO.File]::WriteAllText($manifestPath,$manifest,(New-Object Text.UTF8Encoding($false)))

$programFilesX86 = [Environment]::GetFolderPath('ProgramFilesX86')
$makeAppx = Get-ChildItem -Path (Join-Path $programFilesX86 'Windows Kits\10\bin\*\x64\makeappx.exe') -ErrorAction SilentlyContinue | Sort-Object FullName -Descending | Select-Object -First 1
if (-not $makeAppx) { throw 'MakeAppx.exe was not found in the Windows SDK.' }

$outputFullPath = [IO.Path]::GetFullPath((Join-Path (Get-Location) $OutputPath))
New-Item -ItemType Directory -Path ([IO.Path]::GetDirectoryName($outputFullPath)) -Force | Out-Null
& $makeAppx.FullName pack /d $stage /p $outputFullPath /o
if ($LASTEXITCODE -ne 0) { throw "MakeAppx failed with exit code $LASTEXITCODE." }

$verifyDir = Join-Path $workRoot 'verify'
& $makeAppx.FullName unpack /p $outputFullPath /d $verifyDir /o
if ($LASTEXITCODE -ne 0) { throw "MSIX verification unpack failed with exit code $LASTEXITCODE." }
if (-not (Test-Path (Join-Path $verifyDir 'AppxManifest.xml'))) { throw 'Verified MSIX manifest is missing.' }
if (-not (Test-Path (Join-Path $verifyDir 'VFS\ProgramFilesX64\Ghost FTP\GhostFTP.exe'))) { throw 'Verified MSIX executable is missing.' }

$hash = (Get-FileHash -LiteralPath $outputFullPath -Algorithm SHA256).Hash.ToLowerInvariant()
Write-Output "MSIX=$outputFullPath"
Write-Output "SHA256=$hash"
Write-Output 'Store submission package is intentionally not Authenticode-signed by this helper.'
