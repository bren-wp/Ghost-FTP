$ErrorActionPreference = 'Stop'
Set-StrictMode -Version 3.0

Set-Location -LiteralPath $PSScriptRoot

$versionFile = Join-Path $PSScriptRoot 'VERSION'
$dist = Join-Path $PSScriptRoot 'dist'
$internalDist = Join-Path $dist 'internal'
$stageBuilder = Join-Path $PSScriptRoot 'BUILD-WINDOWS-ARCH-STAGE.ps1'
$bootstrapPayload = Join-Path $PSScriptRoot 'cmd\windowsbootstrap\payload'
$icon = Join-Path $PSScriptRoot 'build\icon.ico'
$signingScript = Join-Path $PSScriptRoot 'scripts\Sign-WindowsArtifacts.ps1'

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

function Sign-UniversalTarget {
    param([Parameter(Mandatory = $true)][string]$Path)

    $pfx = [Environment]::GetEnvironmentVariable('GHOSTFTP_SIGNING_PFX_PATH')
    $passwordText = [Environment]::GetEnvironmentVariable('GHOSTFTP_SIGNING_PASSWORD')
    $timestamp = [Environment]::GetEnvironmentVariable('GHOSTFTP_SIGNING_TIMESTAMP_URL')
    $allowUntrusted = [Environment]::GetEnvironmentVariable('GHOSTFTP_ALLOW_UNTRUSTED_SIGNER')

    if ([string]::IsNullOrWhiteSpace($pfx)) {
        if (-not [string]::IsNullOrWhiteSpace($passwordText)) {
            throw 'GHOSTFTP_SIGNING_PASSWORD is set without GHOSTFTP_SIGNING_PFX_PATH.'
        }
        return
    }
    if ([string]::IsNullOrWhiteSpace($passwordText)) {
        throw 'GHOSTFTP_SIGNING_PFX_PATH is set but GHOSTFTP_SIGNING_PASSWORD is missing.'
    }

    Assert-File -Path $pfx -Description 'Authenticode signing PFX'
    Assert-File -Path $signingScript -Description 'Authenticode signing helper'

    $arguments = @{
        PfxPath = (Resolve-Path -LiteralPath $pfx).Path
        Password = (ConvertTo-SecureString $passwordText -AsPlainText -Force)
        Paths = @($Path)
    }
    if (-not [string]::IsNullOrWhiteSpace($timestamp)) {
        $arguments.TimestampUrl = $timestamp.Trim()
    }
    if ($allowUntrusted -eq '1' -or $allowUntrusted -ieq 'true') {
        $arguments.AllowUntrustedSigner = $true
    }

    & $signingScript @arguments
    if (-not $?) {
        throw "Authenticode signing failed: $Path"
    }
}

function Stage-BootstrapPayload {
    param(
        [Parameter(Mandatory = $true)][string]$X64,
        [Parameter(Mandatory = $true)][string]$X86
    )

    foreach ($arch in @('x64','x86')) {
        Remove-Item -LiteralPath (Join-Path $bootstrapPayload $arch) -Recurse -Force -ErrorAction SilentlyContinue
        New-Item -ItemType Directory -Force -Path (Join-Path $bootstrapPayload $arch) | Out-Null
    }
    Copy-Item -LiteralPath $X64 -Destination (Join-Path $bootstrapPayload 'x64\GhostFTP.exe') -Force
    Copy-Item -LiteralPath $X86 -Destination (Join-Path $bootstrapPayload 'x86\GhostFTP.exe') -Force
}

function Clear-BootstrapPayload {
    foreach ($arch in @('x64','x86')) {
        Remove-Item -LiteralPath (Join-Path $bootstrapPayload $arch) -Recurse -Force -ErrorAction SilentlyContinue
    }
}

function Build-UniversalBootstrap {
    param(
        [Parameter(Mandatory = $true)][ValidateSet('setup','portable')][string]$Role,
        [Parameter(Mandatory = $true)][string]$X64,
        [Parameter(Mandatory = $true)][string]$X86,
        [Parameter(Mandatory = $true)][string]$Output,
        [Parameter(Mandatory = $true)][string]$OriginalFilename
    )

    Assert-File -Path $X64 -Description "$Role x64 native payload"
    Assert-File -Path $X86 -Description "$Role x86 native payload"
    Stage-BootstrapPayload -X64 $X64 -X86 $X86

    try {
        $env:GOTOOLCHAIN = 'local'
        $env:GOPROXY = 'off'
        $env:GOSUMDB = 'off'
        $env:CGO_ENABLED = '0'
        $env:GOWORK = 'off'
        $env:GOOS = 'windows'
        $env:GOARCH = '386'
        $env:GO386 = 'sse2'
        Remove-Item -LiteralPath 'Env:GOAMD64' -ErrorAction SilentlyContinue

        $ldflags = "-s -w -H=windowsgui -X main.version=$version -X main.role=$Role"
        Invoke-Native -FilePath $go -ArgumentList @(
            'build','-mod=readonly','-trimpath','-buildvcs=false','-ldflags',$ldflags,
            '-o',$Output,'./cmd/windowsbootstrap'
        ) -FailureMessage "Universal Windows $Role bootstrap build failed"
    }
    finally {
        Clear-BootstrapPayload
    }

    Invoke-Native -FilePath $python -ArgumentList @(
        'scripts/pe_resources.py',$Output,'--ico',$icon,'--version',$version,
        '--role',$Role,'--original-filename',$OriginalFilename
    ) -FailureMessage "Universal Windows $Role PE resource processing failed"

    Sign-UniversalTarget -Path $Output
}

Assert-File -Path $versionFile -Description 'VERSION file'
Assert-File -Path $stageBuilder -Description 'Native Windows staging builder'
Assert-File -Path $icon -Description 'Application icon'
Assert-File -Path $signingScript -Description 'Authenticode signing helper'

$version = (Get-Content -LiteralPath $versionFile -Raw).Trim()
if ($version -notmatch '^\d+\.\d+\.\d+$') {
    throw "Invalid Ghost FTP version in VERSION: $version"
}

Write-Host "Ghost FTP $version unified Windows packaging"
Write-Host '[1/6] Build and verify signed native x64/x86 setup and portable staging artifacts'
& $stageBuilder
if (-not $?) {
    throw 'Native Windows staging build failed.'
}

$goCommand = @(Get-Command go -CommandType Application -ErrorAction SilentlyContinue)[0]
if (-not $goCommand) { throw 'Go is not installed or is not available in PATH.' }
[string]$go = $goCommand.Source
$pythonCommand = @(Get-Command python -CommandType Application -ErrorAction SilentlyContinue)[0]
if (-not $pythonCommand) { throw 'Python 3 is not installed or is not available in PATH.' }
[string]$python = $pythonCommand.Source

New-Item -ItemType Directory -Force -Path $internalDist | Out-Null
$nativeSetupX64 = Join-Path $internalDist "Ghost-FTP-$version-Setup-x64.exe"
$nativeSetupX86 = Join-Path $internalDist "Ghost-FTP-$version-Setup-x86.exe"
$nativePortableX64 = Join-Path $internalDist "Ghost-FTP-$version-Portable-x64.exe"
$nativePortableX86 = Join-Path $internalDist "Ghost-FTP-$version-Portable-x86.exe"
foreach ($name in @(
    "Ghost-FTP-$version-Setup-x64.exe",
    "Ghost-FTP-$version-Setup-x86.exe",
    "Ghost-FTP-$version-Portable-x64.exe",
    "Ghost-FTP-$version-Portable-x86.exe"
)) {
    $source = Join-Path $dist $name
    Assert-File -Path $source -Description 'Native Windows staging artifact'
    Move-Item -LiteralPath $source -Destination (Join-Path $internalDist $name) -Force
}
Remove-Item -LiteralPath (Join-Path $dist 'SHA256.txt') -Force -ErrorAction SilentlyContinue

Write-Host '[2/6] Build universal Setup.exe with native architecture selection'
$setupName = "Ghost-FTP-$version-Setup.exe"
$setup = Join-Path $dist $setupName
Build-UniversalBootstrap -Role 'setup' -X64 $nativeSetupX64 -X86 $nativeSetupX86 -Output $setup -OriginalFilename $setupName

Write-Host '[3/6] Build universal Portable.exe with native architecture selection'
$portableName = "Ghost-FTP-$version-Portable.exe"
$portable = Join-Path $dist $portableName
Build-UniversalBootstrap -Role 'portable' -X64 $nativePortableX64 -X86 $nativePortableX86 -Output $portable -OriginalFilename $portableName

foreach ($name in @('GOOS','GOARCH','GOAMD64','GO386')) {
    Remove-Item -LiteralPath "Env:$name" -ErrorAction SilentlyContinue
}

Write-Host '[4/6] Verify public universal Windows artifact contract'
$verification = Join-Path $internalDist 'verification-universal.txt'
Invoke-NativeTee -FilePath $python -ArgumentList @(
    'scripts/verify_release.py',$setup,$portable,'--arch','universal'
) -OutputFile $verification -FailureMessage 'Universal Windows release verification failed'

$verificationText = Get-Content -LiteralPath $verification -Raw
$signingConfigured = -not [string]::IsNullOrWhiteSpace([Environment]::GetEnvironmentVariable('GHOSTFTP_SIGNING_PFX_PATH'))
$reportsUnsigned = $verificationText -match '(?m)^(SETUP|PORTABLE)_AUTHENTICODE_SIGNED=NO\s*$'
if ($signingConfigured -and $reportsUnsigned) {
    throw 'Signing was configured, but a public universal Windows executable is unsigned.'
}
if (-not $signingConfigured -and -not $reportsUnsigned) {
    throw 'Universal Windows verification unexpectedly reports signed executables without a configured signing identity.'
}

Write-Host '[5/6] Write public SHA-256 manifest'
$publicFiles = @($setup, $portable)
if ($publicFiles.Count -ne 2) {
    throw "Unexpected Windows public binary count: $($publicFiles.Count); expected 2."
}
$hashLines = foreach ($file in ($publicFiles | Sort-Object)) {
    Assert-File -Path $file -Description 'Windows public production binary'
    $item = Get-Item -LiteralPath $file
    $hash = (Get-FileHash -LiteralPath $item.FullName -Algorithm SHA256).Hash.ToLowerInvariant()
    "$hash  $($item.Name)"
}
$hashLines | Set-Content -LiteralPath (Join-Path $dist 'SHA256.txt') -Encoding ascii

Write-Host '[6/6] Final public-output and temporary-payload verification'
$expectedNames = @($setupName, $portableName, 'SHA256.txt')
$actualNames = @(Get-ChildItem -LiteralPath $dist -File | Select-Object -ExpandProperty Name | Sort-Object)
$missingNames = @($expectedNames | Where-Object { $_ -notin $actualNames })
if ($missingNames.Count -ne 0) {
    throw "Missing final output(s): $($missingNames -join ', ')"
}
if ($actualNames.Count -ne 3) {
    throw "Unexpected public Windows output count: $($actualNames.Count); expected 3 including SHA256.txt."
}
if (Get-ChildItem -LiteralPath $dist -File | Where-Object { $_.Name -match '(?i)-(?:x64|x86|x32)\.exe$' }) {
    throw 'Architecture-specific Windows executables leaked into the public artifact directory.'
}
if (Get-ChildItem -LiteralPath $dist -Recurse -File | Where-Object { $_.Name -match '(?i)uninstall' }) {
    throw 'Windows build unexpectedly produced an uninstaller binary.'
}
if (Test-Path -LiteralPath (Join-Path $bootstrapPayload 'x64')) {
    throw 'Temporary x64 universal bootstrap payload was not removed.'
}
if (Test-Path -LiteralPath (Join-Path $bootstrapPayload 'x86')) {
    throw 'Temporary x86 universal bootstrap payload was not removed.'
}

Write-Host 'WINDOWS_PUBLIC_SETUP=UNIVERSAL_X86_X64'
Write-Host 'WINDOWS_PUBLIC_PORTABLE=UNIVERSAL_X86_X64'
Write-Host 'WINDOWS_NATIVE_PAYLOADS=x64,x86'
Write-Host 'WINDOWS_PUBLIC_EXECUTABLES=2'
Write-Host 'UNINSTALLER_BINARY=ABSENT'
Write-Host "Ghost FTP $version unified Windows build completed: $dist"
