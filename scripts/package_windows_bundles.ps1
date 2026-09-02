param(
    [Parameter(Mandatory = $false)]
    [string]$OutputDir = 'dist'
)

$ErrorActionPreference = 'Stop'
Set-StrictMode -Version Latest

$root = (Resolve-Path (Join-Path $PSScriptRoot '..')).Path
$version = (Get-Content -LiteralPath (Join-Path $root 'VERSION') -Raw).Trim()
if ($version -notmatch '^\d+\.\d+\.\d+$') { throw "Invalid VERSION: $version" }

$output = Join-Path $root $OutputDir
New-Item -ItemType Directory -Force -Path $output | Out-Null
$notes = Join-Path $output 'RELEASE-NOTES.txt'
python (Join-Path $root 'scripts/release_notes.py') --version $version --output $notes
if ($LASTEXITCODE -ne 0) { throw 'Release-note generation failed.' }

$goVersion = (go env GOVERSION).Trim()
$telemetry = (go telemetry).Trim()
$metadata = @"
VERSION=$version
COMMIT=$env:GITHUB_SHA
SOURCE_REF=$env:GITHUB_REF
GO_VERSION=$goVersion
GO_TELEMETRY=$telemetry
BUILD_OS=windows
BUILD_ARCH=x64,x86
WINDOWS_UNINSTALLER=none
GITHUB_RUN_ID=$env:GITHUB_RUN_ID
GITHUB_RUN_ATTEMPT=$env:GITHUB_RUN_ATTEMPT
"@
$metadataPath = Join-Path $output 'BUILD-METADATA-WINDOWS.txt'
$metadata | Set-Content -LiteralPath $metadataPath -Encoding ascii

foreach ($arch in @('x64', 'x86')) {
    $portable = Join-Path $output "ByFTP-$version-Portable-$arch.exe"
    $setup = Join-Path $output "ByFTP-$version-Setup-$arch.exe"
    foreach ($required in @($portable, $setup)) {
        if (-not (Test-Path -LiteralPath $required -PathType Leaf)) { throw "Missing Windows release binary: $required" }
    }

    $bundleRoot = Join-Path $output "ByFTP-$version-Windows-$arch"
    Remove-Item -Recurse -Force $bundleRoot -ErrorAction SilentlyContinue
    $documentation = Join-Path $bundleRoot 'Documentation'
    New-Item -ItemType Directory -Force -Path $bundleRoot, $documentation | Out-Null

    Copy-Item -LiteralPath $portable -Destination $bundleRoot -Force
    Copy-Item -LiteralPath $setup -Destination $bundleRoot -Force
    Copy-Item -LiteralPath $notes -Destination $bundleRoot -Force
    Copy-Item -LiteralPath $metadataPath -Destination (Join-Path $bundleRoot 'BUILD-METADATA.txt') -Force
    foreach ($name in @('README.md', 'CHANGELOG.md', 'LICENSE')) {
        Copy-Item -LiteralPath (Join-Path $root $name) -Destination $bundleRoot -Force
    }
    Copy-Item -Path (Join-Path $root 'docs/*.md') -Destination $documentation -Force

    $hashLines = foreach ($file in Get-ChildItem -LiteralPath $bundleRoot -Recurse -File | Sort-Object FullName) {
        $hash = (Get-FileHash -LiteralPath $file.FullName -Algorithm SHA256).Hash.ToLowerInvariant()
        $relative = [IO.Path]::GetRelativePath($bundleRoot, $file.FullName).Replace('\', '/')
        "$hash  $relative"
    }
    $hashLines | Set-Content -LiteralPath (Join-Path $bundleRoot 'BUNDLE-SHA256.txt') -Encoding ascii

    $zip = Join-Path $output "ByFTP-$version-Windows-$arch.zip"
    Remove-Item -Force $zip -ErrorAction SilentlyContinue
    Compress-Archive -Path (Join-Path $bundleRoot '*') -DestinationPath $zip -CompressionLevel Optimal
    python (Join-Path $root 'scripts/verify_bundle.py') $zip --version $version --arch $arch
    if ($LASTEXITCODE -ne 0) { throw "Windows $arch ZIP verification failed." }
}

$unexpected = @(Get-ChildItem -LiteralPath $output -Recurse -File | Where-Object { $_.Name -match '(?i)uninstall' })
if ($unexpected.Count -gt 0) {
    throw "Windows release packaging unexpectedly produced an uninstaller-named file: $($unexpected.FullName -join ', ')"
}

Write-Output "WINDOWS_BUNDLES=PASS ($version)"
Write-Output 'WINDOWS_UNINSTALLER=none'
