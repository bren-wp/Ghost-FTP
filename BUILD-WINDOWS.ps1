$ErrorActionPreference = 'Stop'
Set-StrictMode -Version 3.0

Set-Location -LiteralPath $PSScriptRoot

$minimumGo = [Version]'1.26.5'
$versionFile = Join-Path $PSScriptRoot 'VERSION'
$dist = Join-Path $PSScriptRoot 'dist'
$internalDist = Join-Path $dist 'internal'
$payloadDir = Join-Path $PSScriptRoot 'cmd\installer\payload'
$payloadZip = Join-Path $payloadDir 'payload.zip'
$icon = Join-Path $PSScriptRoot 'build\icon.ico'
$goMod = Join-Path $PSScriptRoot 'go.mod'

function Assert-File {
    param(
        [Parameter(Mandatory = $true)][string]$Path,
        [Parameter(Mandatory = $true)][string]$Description
    )
    if (-not (Test-Path -LiteralPath $Path -PathType Leaf)) {
        throw "$Description is missing: $Path"
    }
}

function Invoke-Native {
    param(
        [Parameter(Mandatory = $true)][string]$FilePath,
        [Parameter(Mandatory = $false)][string[]]$ArgumentList = @(),
        [Parameter(Mandatory = $true)][string]$FailureMessage
    )
    & $FilePath @ArgumentList
    $exitCode = $LASTEXITCODE
    if ($exitCode -ne 0) {
        throw "$FailureMessage (exit code $exitCode)."
    }
}

function Invoke-NativeCapture {
    param(
        [Parameter(Mandatory = $true)][string]$FilePath,
        [Parameter(Mandatory = $false)][string[]]$ArgumentList = @(),
        [Parameter(Mandatory = $true)][string]$FailureMessage
    )
    $output = & $FilePath @ArgumentList 2>&1
    $exitCode = $LASTEXITCODE
    if ($exitCode -ne 0) {
        $details = ($output | Out-String).Trim()
        if ($details) { throw "$FailureMessage (exit code $exitCode): $details" }
        throw "$FailureMessage (exit code $exitCode)."
    }
    return ($output | Out-String).Trim()
}

function Invoke-NativeTee {
    param(
        [Parameter(Mandatory = $true)][string]$FilePath,
        [Parameter(Mandatory = $false)][string[]]$ArgumentList = @(),
        [Parameter(Mandatory = $true)][string]$OutputFile,
        [Parameter(Mandatory = $true)][string]$FailureMessage
    )
    $output = & $FilePath @ArgumentList 2>&1
    $exitCode = $LASTEXITCODE
    $output | Tee-Object -FilePath $OutputFile
    if ($exitCode -ne 0) {
        throw "$FailureMessage (exit code $exitCode)."
    }
}

function Get-NormalizedVersion {
    param([Parameter(Mandatory = $true)][string]$GoVersion)
    if ($GoVersion -notmatch '^go(\d+)\.(\d+)(?:\.(\d+))?$') {
        throw "Unable to verify a stable Go version: $GoVersion"
    }
    $patch = if ($Matches[3]) { [int]$Matches[3] } else { 0 }
    return [Version]::new([int]$Matches[1], [int]$Matches[2], $patch)
}

function Test-SamePath {
    param(
        [Parameter(Mandatory = $true)][string]$A,
        [Parameter(Mandatory = $true)][string]$B
    )
    try {
        $left = [IO.Path]::GetFullPath($A).TrimEnd('\')
        $right = [IO.Path]::GetFullPath($B).TrimEnd('\')
        return [string]::Equals($left, $right, [StringComparison]::OrdinalIgnoreCase)
    }
    catch { return $false }
}

Assert-File -Path $versionFile -Description 'VERSION file'
Assert-File -Path $goMod -Description 'go.mod'
Assert-File -Path $icon -Description 'Application icon'

$version = (Get-Content -LiteralPath $versionFile -Raw).Trim()
if ($version -notmatch '^\d+\.\d+\.\d+$') {
    throw "Invalid Ghost FTP version in VERSION: $version"
}

# Production builds are offline and must not silently change toolchains/modules.
$env:GOTOOLCHAIN = 'local'
$env:GOPROXY = 'off'
$env:GOSUMDB = 'off'
$env:CGO_ENABLED = '0'
$env:GOWORK = 'off'
foreach ($name in @('GOOS','GOARCH','GOAMD64','GO386','GOARM64','GOEXPERIMENT','GOFLAGS')) {
    Remove-Item -LiteralPath "Env:$name" -ErrorAction SilentlyContinue
}

$goCommand = @(Get-Command go -CommandType Application -ErrorAction SilentlyContinue)[0]
if (-not $goCommand) { throw 'Go is not installed or is not available in PATH.' }
[string]$go = $goCommand.Source

$pythonCommand = @(Get-Command python -CommandType Application -ErrorAction SilentlyContinue)[0]
if (-not $pythonCommand) { throw 'Python 3 is not installed or is not available in PATH.' }
[string]$python = $pythonCommand.Source

$pythonVersionText = Invoke-NativeCapture -FilePath $python -ArgumentList @('--version') -FailureMessage 'Unable to verify Python version'
if ($pythonVersionText -notmatch '^Python\s+3(?:\.|$)') {
    throw "Python 3 is required. Current: $pythonVersionText"
}

$rawGoVersion = Invoke-NativeCapture -FilePath $go -ArgumentList @('env','GOVERSION') -FailureMessage 'Unable to verify Go version'
$goVersion = Get-NormalizedVersion -GoVersion $rawGoVersion
if ($goVersion -lt $minimumGo) {
    throw "Ghost FTP production builds require Go $minimumGo or newer. Current: $rawGoVersion"
}

$activeGoMod = Invoke-NativeCapture -FilePath $go -ArgumentList @('env','GOMOD') -FailureMessage 'Unable to determine the active go.mod'
if (-not (Test-SamePath -A $activeGoMod -B $goMod)) {
    throw "Unexpected Go module root. Expected: $goMod; active: $activeGoMod"
}

$moduleGraph = Invoke-NativeCapture -FilePath $go -ArgumentList @('list','-m','-mod=readonly','all') -FailureMessage 'Unable to verify the Go module graph'
$moduleLines = @($moduleGraph -split '\r?\n' | Where-Object { -not [string]::IsNullOrWhiteSpace($_) })
if ($moduleLines.Count -ne 1) {
    throw "Production build contract permits no external Go modules. Found $($moduleLines.Count) modules."
}

$telemetryMode = Invoke-NativeCapture `
    -FilePath $go `
    -ArgumentList @('telemetry') `
    -FailureMessage 'Unable to verify Go telemetry mode'
if ($telemetryMode -ne 'off') {
    throw "Go telemetry must be disabled before a production build. Run: go telemetry off (current: $telemetryMode)"
}

Write-Host "Ghost FTP $version"
Write-Host "Go: $rawGoVersion | Python: $pythonVersionText | telemetry=$telemetryMode"

Write-Host '[1/8] Brand, localization and canonical version'
Invoke-Native -FilePath $python -ArgumentList @('scripts/generate_brand_assets.py','--check') -FailureMessage 'Brand asset verification failed'
Invoke-Native -FilePath $python -ArgumentList @('scripts/audit_localization.py') -FailureMessage 'Localization audit failed'
Invoke-Native -FilePath $python -ArgumentList @('scripts/audit_version.py') -FailureMessage 'Version consistency audit failed'

Write-Host '[2/8] Documentation, security, privacy and release contract'
Invoke-Native -FilePath $python -ArgumentList @('scripts/audit_docs.py') -FailureMessage 'Documentation audit failed'
Invoke-Native -FilePath $python -ArgumentList @('scripts/audit_security.py') -FailureMessage 'Security audit failed'
Invoke-Native -FilePath $python -ArgumentList @('scripts/audit_privacy.py') -FailureMessage 'Privacy audit failed'
Invoke-Native -FilePath $python -ArgumentList @('scripts/audit_release.py') -FailureMessage 'Release audit failed'

Write-Host '[3/8] Regression tests and Go static analysis'
Invoke-Native -FilePath $python -ArgumentList @('-m','unittest','discover','-s','scripts','-p','test_*.py') -FailureMessage 'Python regression tests failed'
Invoke-Native -FilePath $go -ArgumentList @('test','-count=1','-mod=readonly','./...') -FailureMessage 'Go tests failed'
Invoke-Native -FilePath $go -ArgumentList @('vet','-mod=readonly','./...') -FailureMessage 'Go vet failed'

Write-Host '[4/8] Clean Windows output directories'
Remove-Item -LiteralPath $dist -Recurse -Force -ErrorAction SilentlyContinue
New-Item -ItemType Directory -Force -Path $dist | Out-Null
New-Item -ItemType Directory -Force -Path $internalDist | Out-Null
New-Item -ItemType Directory -Force -Path $payloadDir | Out-Null
Remove-Item -LiteralPath $payloadZip -Force -ErrorAction SilentlyContinue

$ldflags = "-s -w -H=windowsgui -X main.version=$version"
$publicFiles = [System.Collections.Generic.List[string]]::new()
$verificationFiles = [System.Collections.Generic.List[string]]::new()

function Build-GhostFTPArchitecture {
    param(
        [Parameter(Mandatory = $true)][ValidateSet('amd64','386')][string]$GoArch,
        [Parameter(Mandatory = $true)][ValidateSet('x64','x86')][string]$Label
    )

    $env:GOOS = 'windows'
    $env:GOARCH = $GoArch
    if ($GoArch -eq 'amd64') {
        $env:GOAMD64 = 'v1'
        Remove-Item -LiteralPath 'Env:GO386' -ErrorAction SilentlyContinue
    }
    else {
        $env:GO386 = 'sse2'
        Remove-Item -LiteralPath 'Env:GOAMD64' -ErrorAction SilentlyContinue
    }

    $portable = Join-Path $dist "Ghost-FTP-$version-Portable-$Label.exe"
    $setup = Join-Path $dist "Ghost-FTP-$version-Setup-$Label.exe"
    $verification = Join-Path $internalDist "verification-$Label.txt"

    Write-Host "      [$Label] Ghost FTP client"
    Invoke-Native -FilePath $go -ArgumentList @(
        'build','-mod=readonly','-trimpath','-buildvcs=false','-ldflags',$ldflags,
        '-o',$portable,'./cmd/GhostFTP'
    ) -FailureMessage "Client $Label build failed"

    Invoke-Native -FilePath $python -ArgumentList @(
        'scripts/pe_resources.py',$portable,'--ico',$icon,'--version',$version,
        '--role','portable','--original-filename',"Ghost-FTP-$version-Portable-$Label.exe"
    ) -FailureMessage "Client $Label PE resource processing failed"

    Write-Host "      [$Label] Verified installer payload"
    try {
        # make_payload.py intentionally stores the inner executable as GhostFTP.exe:
        # that filename is a legacy installed-app compatibility boundary only.
        Invoke-Native -FilePath $python -ArgumentList @(
            'scripts/make_payload.py','--app',$portable,'--output',$payloadZip
        ) -FailureMessage "$Label installer payload compression failed"
        Assert-File -Path $payloadZip -Description "$Label installer payload"

        Invoke-Native -FilePath $go -ArgumentList @(
            'build','-mod=readonly','-trimpath','-buildvcs=false','-ldflags',$ldflags,
            '-o',$setup,'./cmd/installer'
        ) -FailureMessage "Setup $Label build failed"
    }
    finally {
        Remove-Item -LiteralPath $payloadZip -Force -ErrorAction SilentlyContinue
    }

    Invoke-Native -FilePath $python -ArgumentList @(
        'scripts/pe_resources.py',$setup,'--ico',$icon,'--version',$version,
        '--role','setup','--original-filename',"Ghost-FTP-$version-Setup-$Label.exe"
    ) -FailureMessage "Setup $Label PE resource processing failed"

    Invoke-NativeTee -FilePath $python -ArgumentList @(
        'scripts/verify_release.py',$setup,$portable,'--arch',$Label
    ) -OutputFile $verification -FailureMessage "$Label release verification failed"

    $script:publicFiles.Add($portable)
    $script:publicFiles.Add($setup)
    $script:verificationFiles.Add($verification)
}

Write-Host '[5/8] Windows x64 and x86 builds'
Build-GhostFTPArchitecture -GoArch 'amd64' -Label 'x64'
Build-GhostFTPArchitecture -GoArch '386' -Label 'x86'
foreach ($name in @('GOOS','GOARCH','GOAMD64','GO386')) {
    Remove-Item -LiteralPath "Env:$name" -ErrorAction SilentlyContinue
}

Write-Host '[6/8] SHA-256 manifest'
if ($publicFiles.Count -ne 4) {
    throw "Unexpected Windows binary count: $($publicFiles.Count); expected 4."
}
$hashLines = foreach ($file in ($publicFiles | Sort-Object)) {
    Assert-File -Path $file -Description 'Windows production binary'
    $item = Get-Item -LiteralPath $file
    $hash = (Get-FileHash -LiteralPath $item.FullName -Algorithm SHA256).Hash.ToLowerInvariant()
    "$hash  $($item.Name)"
}
$shaFile = Join-Path $dist 'SHA256.txt'
$hashLines | Set-Content -LiteralPath $shaFile -Encoding ascii

Write-Host '[7/8] Signing status'
$unsigned = $false
foreach ($verification in $verificationFiles) {
    Assert-File -Path $verification -Description 'Release verification report'
    $text = Get-Content -LiteralPath $verification -Raw
    if ($text -match '(?m)^(SETUP|PORTABLE)_AUTHENTICODE_SIGNED=NO\s*$') { $unsigned = $true }
}
if ($unsigned) {
    Write-Warning 'Binaries are not Authenticode-signed. Verified Publisher requires a valid Ghost FTP code-signing certificate.'
}

Write-Host '[8/8] Final output verification'
$expectedNames = @(
    "Ghost-FTP-$version-Portable-x64.exe",
    "Ghost-FTP-$version-Setup-x64.exe",
    "Ghost-FTP-$version-Portable-x86.exe",
    "Ghost-FTP-$version-Setup-x86.exe",
    'SHA256.txt'
)
$actualNames = @(Get-ChildItem -LiteralPath $dist -File | Select-Object -ExpandProperty Name | Sort-Object)
$missingNames = @($expectedNames | Where-Object { $_ -notin $actualNames })
if ($missingNames.Count -ne 0) {
    throw "Missing final output(s): $($missingNames -join ', ')"
}
if (Get-ChildItem -LiteralPath $dist -Recurse -File | Where-Object { $_.Name -match '(?i)uninstall' }) {
    throw 'Windows build unexpectedly produced an uninstaller binary.'
}
if (Test-Path -LiteralPath $payloadZip) {
    throw 'Temporary installer payload was not removed.'
}

Write-Host 'UNINSTALLER_BINARY=ABSENT'
Write-Host "Ghost FTP $version Windows x64+x86 build completed: $dist"
