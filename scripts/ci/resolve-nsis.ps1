# Resolve the actual NSIS compiler used by the Ghost CLI Windows installer.
# Version and archive hash are deliberately pinned; do not use a moving "latest" download.
# Official archive: https://sourceforge.net/projects/nsis/files/NSIS%203/3.13/nsis-3.13.zip/download
# SHA-256 published by upstream SourceForge for nsis-3.13.zip.
[CmdletBinding()]
param()

Set-StrictMode -Version Latest
$ErrorActionPreference = 'Stop'
$nsisVersion = '3.13'
$expectedSha256 = 'ba63dffc4410ee89193e1cb5a41989991bd77c61068da17e3156d136b7b0b3d8'

function Test-NsisCompiler {
    param([Parameter(Mandatory)][string]$Path)
    if (-not (Test-Path -LiteralPath $Path -PathType Leaf)) { return $false }
    try {
        $banner = (@(& $Path /VERSION 2>&1) -join ' ').Trim()
        if ($LASTEXITCODE -ne 0) {
            Write-Host "NSIS compiler returned exit code $LASTEXITCODE at $Path"
            return $false
        }
        if ($banner -notmatch '(^|[^\d])3\.13(?:\.0)?([^\d]|$)') {
            Write-Host "Ignoring NSIS compiler with unexpected version '$banner': $Path"
            return $false
        }
        Write-Host "Verified NSIS compiler $banner at $Path"
        return $true
    }
    catch {
        Write-Host "NSIS compiler is unusable at $Path : $_"
        return $false
    }
}

function Find-NsisCompiler {
    param([string]$ExtraRoot = '')
    $candidates = [System.Collections.Generic.List[string]]::new()
    if ($ExtraRoot -and (Test-Path -LiteralPath $ExtraRoot -PathType Container)) {
        Get-ChildItem -LiteralPath $ExtraRoot -Filter makensis.exe -Recurse -File -ErrorAction Stop |
            ForEach-Object { $candidates.Add($_.FullName) }
    }
    # Tauri just created the desktop NSIS package and verifies its cached toolchain.
    # Reuse it before downloading from SourceForge or contacting Chocolatey.
    if (-not [string]::IsNullOrWhiteSpace($env:LOCALAPPDATA)) {
        $candidates.Add((Join-Path $env:LOCALAPPDATA 'tauri\NSIS\makensis.exe'))
    }
    $candidates.Add('C:\Program Files (x86)\NSIS\makensis.exe')
    $candidates.Add('C:\Program Files\NSIS\makensis.exe')
    $fromPath = Get-Command makensis.exe -ErrorAction SilentlyContinue
    if ($fromPath) { $candidates.Add($fromPath.Source) }
    foreach ($candidate in ($candidates | Select-Object -Unique)) {
        if (Test-NsisCompiler -Path $candidate) {
            return (Resolve-Path -LiteralPath $candidate).Path
        }
    }
    return $null
}

$compiler = Find-NsisCompiler
if ($compiler) {
    Write-Output $compiler
    exit 0
}

$tempRoot = if ($env:RUNNER_TEMP) { $env:RUNNER_TEMP } else { [IO.Path]::GetTempPath() }
$toolsDir = Join-Path $tempRoot 'ghostftp-nsis-3.13'
$archive = Join-Path $toolsDir 'nsis-3.13.zip'
$unpacked = Join-Path $toolsDir 'unpacked'
New-Item -ItemType Directory -Path $toolsDir -Force | Out-Null
$urls = @(
    'https://downloads.sourceforge.net/project/nsis/NSIS%203/3.13/nsis-3.13.zip',
    'https://sourceforge.net/projects/nsis/files/NSIS%203/3.13/nsis-3.13.zip/download'
)
$errors = [System.Collections.Generic.List[string]]::new()
foreach ($url in $urls) {
    for ($attempt = 1; $attempt -le 2; $attempt++) {
        try {
            Write-Host "Fetching pinned NSIS $nsisVersion archive from $url (attempt $attempt/2)"
            if (Test-Path -LiteralPath $archive) { Remove-Item -LiteralPath $archive -Force }
            Invoke-WebRequest -Uri $url -OutFile $archive -TimeoutSec 90 -MaximumRedirection 10 -ErrorAction Stop
            $actualSha256 = (Get-FileHash -LiteralPath $archive -Algorithm SHA256).Hash.ToLowerInvariant()
            if ($actualSha256 -ne $expectedSha256) {
                throw "NSIS ZIP SHA-256 mismatch (mirror returned different content, possibly an HTML download page); expected $expectedSha256, got $actualSha256"
            }
            if (Test-Path -LiteralPath $unpacked) {
                Remove-Item -LiteralPath $unpacked -Force -Recurse
            }
            Expand-Archive -LiteralPath $archive -DestinationPath $unpacked -Force -ErrorAction Stop
            $compiler = Find-NsisCompiler -ExtraRoot $unpacked
            if (-not $compiler) {
                throw 'Verified NSIS ZIP did not contain a working NSIS 3.13 makensis.exe'
            }
            Write-Host "Verified official NSIS ZIP SHA-256: $actualSha256"
            Write-Output $compiler
            exit 0
        }
        catch {
            $message = "NSIS archive source failed ($url, attempt $attempt): $_"
            Write-Warning $message
            $errors.Add($message)
        }
    }
}

# Independent package-registry fallback, with exact version selection AND binary validation.
# A successful package-manager exit alone is never considered sufficient.
if (Get-Command choco -ErrorAction SilentlyContinue) {
    try {
        Write-Host 'Official portable ZIP unavailable; trying Chocolatey with pinned NSIS 3.13.0.'
        # Chocolatey writes status lines to stdout. Never leak those lines into
        # the command-substitution output consumed as the compiler path.
        & choco install nsis --version=3.13.0 --no-progress --yes | Out-Host
        if ($LASTEXITCODE -ne 0) { throw "Chocolatey returned exit code $LASTEXITCODE" }
        $compiler = Find-NsisCompiler
        if (-not $compiler) { throw 'Chocolatey finished without a valid NSIS 3.13 makensis.exe' }
        Write-Output $compiler
        exit 0
    }
    catch {
        $errors.Add("Pinned Chocolatey installation failed: $_")
    }
}

throw "No verified NSIS $nsisVersion compiler is available. $($errors -join ' | ')"
