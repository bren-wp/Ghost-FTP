param(
    [Parameter(Mandatory = $false)]
    [string]$ReleaseDir = 'release'
)

$ErrorActionPreference = 'Stop'
Set-StrictMode -Version Latest

$root = (Resolve-Path (Join-Path $PSScriptRoot '..')).Path
$version = (Get-Content -LiteralPath (Join-Path $root 'VERSION') -Raw).Trim()
if ($version -notmatch '^\d+\.\d+\.\d+$') { throw "Invalid VERSION: $version" }

$release = Join-Path $root $ReleaseDir
if (-not (Test-Path -LiteralPath $release -PathType Container)) { throw "Release directory is missing: $release" }

$expected = @(
    "ByFTP-$version-Portable-x64.exe",
    "ByFTP-$version-Setup-x64.exe",
    "ByFTP-$version-Windows-x64.zip",
    "ByFTP-$version-Portable-x86.exe",
    "ByFTP-$version-Setup-x86.exe",
    "ByFTP-$version-Windows-x86.zip",
    "ByFTP-$version-Linux-amd64.deb",
    "ByFTP-$version-Linux-arm64.deb",
    "ByFTP-$version-Linux-i386.deb",
    "ByFTP-$version-macOS-Universal.pkg",
    "ByFTP-$version-Android-debug.apk",
    "ByFTP-$version-Android-release-unsigned.apk",
    "ByFTP-$version-iOS-arm64-unsigned.ipa",
    "ByFTP-$version-iOS-arm64-unsigned-app.zip",
    "ByFTP-$version-WEB-shared-hosting.zip"
)

$actual = @(Get-ChildItem -LiteralPath $release -File | ForEach-Object Name | Sort-Object)
$wanted = @($expected | Sort-Object)
if ($actual.Count -ne $wanted.Count) {
    throw "Release staging has $($actual.Count) package files instead of $($wanted.Count)."
}
$diff = @(Compare-Object -ReferenceObject $wanted -DifferenceObject $actual)
if ($diff.Count -gt 0) {
    throw "Release staging does not match the public package contract: $($diff | Out-String)"
}
if ($actual | Where-Object { $_ -match '(?i)uninstall' }) {
    throw 'Release staging contains an uninstaller-named public asset.'
}

$notes = Join-Path $release 'RELEASE-NOTES.txt'
python (Join-Path $root 'scripts/release_notes.py') --version $version --output $notes
if ($LASTEXITCODE -ne 0) { throw 'Release-note generation failed.' }

$metadata = @"
VERSION=$version
COMMIT=$env:GITHUB_SHA
SOURCE_REF=$env:GITHUB_REF
WINDOWS=x64,x86
WINDOWS_UNINSTALLER=none
LINUX=amd64,arm64,i386
MACOS=universal-amd64-arm64
ANDROID=debug-signed,release-unsigned
IOS=arm64-unsigned-ipa,arm64-unsigned-app-zip
WEB=shared-hosting-php-pwa-zip
PUBLIC_PLATFORM_ARTIFACTS=15
SHARED_METADATA_ARTIFACTS=3
PUBLIC_RELEASE_FILES=18
RELEASE_QUALITY_GATE=passed
GITHUB_RUN_ID=$env:GITHUB_RUN_ID
GITHUB_RUN_ATTEMPT=$env:GITHUB_RUN_ATTEMPT
"@
$metadata | Set-Content -LiteralPath (Join-Path $release 'BUILD-METADATA.txt') -Encoding ascii

$hashTargets = Get-ChildItem -LiteralPath $release -File | Where-Object { $_.Name -ne 'SHA256.txt' } | Sort-Object Name
$hashLines = foreach ($item in $hashTargets) {
    $hash = (Get-FileHash -LiteralPath $item.FullName -Algorithm SHA256).Hash.ToLowerInvariant()
    "$hash  $($item.Name)"
}
$hashLines | Set-Content -LiteralPath (Join-Path $release 'SHA256.txt') -Encoding ascii

$final = @(Get-ChildItem -LiteralPath $release -File | ForEach-Object Name | Sort-Object)
if ($final.Count -ne 18) { throw "Final public release must contain exactly 18 files; found $($final.Count)." }
foreach ($shared in @('BUILD-METADATA.txt', 'RELEASE-NOTES.txt', 'SHA256.txt')) {
    if ($shared -notin $final) { throw "Missing shared release metadata: $shared" }
}

Write-Output "RELEASE_STAGING=PASS ($version)"
Write-Output 'PUBLIC_PLATFORM_ARTIFACTS=15'
Write-Output 'PUBLIC_RELEASE_FILES=18'
Write-Output 'WINDOWS_UNINSTALLER=none'
