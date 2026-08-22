$ErrorActionPreference = 'Stop'
Set-Location $PSScriptRoot

$versionFile = Join-Path $PSScriptRoot 'VERSION'
if (-not (Test-Path -LiteralPath $versionFile -PathType Leaf)) {
    throw 'VERSION file is missing.'
}
$version = (Get-Content -LiteralPath $versionFile -Raw).Trim()
if ($version -notmatch '^\d+\.\d+\.\d+$') {
    throw "Invalid ByFTP version in VERSION: $version"
}

$minimumGo = [Version]'1.26.5'
$dist = Join-Path $PWD 'dist'
$internalDist = Join-Path $dist 'internal'
$payload = Join-Path $PWD 'cmd\installer\payload'
$icon = Join-Path $PWD 'build\icon.ico'

# The production build is intentionally offline: ByFTP has no external Go
# modules and must never fetch a toolchain or module during production builds.
$env:GOTOOLCHAIN = 'local'
$env:GOPROXY = 'off'
$env:GOSUMDB = 'off'
$env:CGO_ENABLED = '0'

if (-not (Get-Command go -ErrorAction SilentlyContinue)) { throw 'Go is not installed.' }
if (-not (Get-Command python -ErrorAction SilentlyContinue)) { throw 'Python 3 is not installed.' }

$rawGoVersion = (go env GOVERSION).Trim()
if ($rawGoVersion -notmatch '^go(\d+)\.(\d+)(?:\.(\d+))?$') {
    throw "Unable to verify Go version: $rawGoVersion"
}
$patch = if ($Matches[3]) { [int]$Matches[3] } else { 0 }
$goVersion = [Version]::new([int]$Matches[1], [int]$Matches[2], $patch)
if ($goVersion -lt $minimumGo) {
    throw "ByFTP production builds require Go $minimumGo or a newer security patch. Current: $rawGoVersion"
}

# GOTELEMETRY is read-only through go env. CI runs `go telemetry off` before
# this script; local builders must do the same explicitly so the build does not
# mutate global Go settings behind the user's back.
$telemetryMode = (go telemetry).Trim()
if ($LASTEXITCODE -ne 0 -or $telemetryMode -ne 'off') {
    throw "Go telemetry must be disabled before a production build. Run: go telemetry off (current: $telemetryMode)"
}

Write-Host "ByFTP $version"
Write-Host '[1/10] Assets, localization and version'
python scripts/generate_brand_assets.py --check
if ($LASTEXITCODE -ne 0) { throw 'Brand asset verification failed.' }
python scripts/audit_localization.py
if ($LASTEXITCODE -ne 0) { throw 'Localization audit failed.' }
python scripts/audit_version.py
if ($LASTEXITCODE -ne 0) { throw 'Version consistency audit failed.' }

Write-Host '[2/10] Documentation, security, privacy and release contract'
python scripts/audit_docs.py
if ($LASTEXITCODE -ne 0) { throw 'Documentation audit failed.' }
python scripts/audit_security.py
if ($LASTEXITCODE -ne 0) { throw 'Security audit failed.' }
python scripts/audit_privacy.py
if ($LASTEXITCODE -ne 0) { throw 'Privacy audit failed.' }
python scripts/audit_release.py
if ($LASTEXITCODE -ne 0) { throw 'Release audit failed.' }

Write-Host '[3/10] Python release-tool regressions'
python -m unittest discover -s scripts -p 'test_*.py'
if ($LASTEXITCODE -ne 0) { throw 'Python release-tool regressions failed.' }

Write-Host "[4/10] Go tests and static analysis ($rawGoVersion, telemetry=$telemetryMode)"
go test ./...
if ($LASTEXITCODE -ne 0) { throw 'Go tests failed.' }
go vet ./...
if ($LASTEXITCODE -ne 0) { throw 'Go vet failed.' }

Write-Host '[5/10] Clean output directories'
Remove-Item -Recurse -Force $dist -ErrorAction SilentlyContinue
New-Item -ItemType Directory -Force -Path $dist, $internalDist, $payload | Out-Null
Remove-Item "$payload\payload.zip" -Force -ErrorAction SilentlyContinue

$ldflags = "-s -w -H=windowsgui -X main.version=$version"
$publicFiles = New-Object System.Collections.Generic.List[string]
$verificationFiles = New-Object System.Collections.Generic.List[string]

function Build-ByFTPArchitecture {
    param(
        [Parameter(Mandatory = $true)][string]$GoArch,
        [Parameter(Mandatory = $true)][ValidateSet('x64','x86')][string]$Label
    )

    $env:GOOS = 'windows'
    $env:GOARCH = $GoArch

    $portable = Join-Path $dist "ByFTP-$version-Portable-$Label.exe"
    $uninstall = Join-Path $internalDist "ByFTP-$version-Uninstall-$Label.exe"
    $setup = Join-Path $dist "ByFTP-$version-Setup-$Label.exe"
    $verification = Join-Path $internalDist "verification-$Label.txt"

    Write-Host "      [$Label] Portable"
    go build -trimpath -buildvcs=false -ldflags $ldflags -o $portable ./cmd/byftp
    if ($LASTEXITCODE -ne 0) { throw "Portable $Label build failed." }
    python scripts/pe_resources.py $portable --ico $icon --version $version --role portable --original-filename "ByFTP-$version-Portable-$Label.exe"
    if ($LASTEXITCODE -ne 0) { throw "Portable $Label PE resource processing failed." }

    Write-Host "      [$Label] Internal uninstaller"
    go build -trimpath -buildvcs=false -ldflags $ldflags -o $uninstall ./cmd/uninstaller
    if ($LASTEXITCODE -ne 0) { throw "Internal $Label uninstaller build failed." }
    python scripts/pe_resources.py $uninstall --ico $icon --version $version --role uninstaller --original-filename 'Uninstall.exe'
    if ($LASTEXITCODE -ne 0) { throw "Internal $Label uninstaller PE resource processing failed." }

    Write-Host "      [$Label] Installer payload"
    python scripts/make_payload.py --app $portable --uninstaller $uninstall --output "$payload\payload.zip"
    if ($LASTEXITCODE -ne 0) { throw "$Label installer payload compression failed." }

    Write-Host "      [$Label] Setup"
    try {
        go build -trimpath -buildvcs=false -ldflags $ldflags -o $setup ./cmd/installer
        if ($LASTEXITCODE -ne 0) { throw "Setup $Label build failed." }
    } finally {
        Remove-Item "$payload\payload.zip" -Force -ErrorAction SilentlyContinue
    }
    python scripts/pe_resources.py $setup --ico $icon --version $version --role setup --original-filename "ByFTP-$version-Setup-$Label.exe"
    if ($LASTEXITCODE -ne 0) { throw "Setup $Label PE resource processing failed." }

    Write-Host "      [$Label] PE, security and privacy verification"
    python scripts/verify_release.py $setup $portable $uninstall --arch $Label | Tee-Object -FilePath $verification
    if ($LASTEXITCODE -ne 0) { throw "$Label release verification failed." }

    $script:publicFiles.Add($portable)
    $script:publicFiles.Add($setup)
    $script:verificationFiles.Add($verification)
}

Write-Host '[6/10] Windows x64 production build'
Build-ByFTPArchitecture -GoArch 'amd64' -Label 'x64'

Write-Host '[7/10] Windows x86 production build'
Build-ByFTPArchitecture -GoArch '386' -Label 'x86'

Write-Host '[8/10] SHA-256 of public binaries'
$hashLines = foreach ($file in ($publicFiles | Sort-Object)) {
    $item = Get-Item -LiteralPath $file
    $hash = (Get-FileHash -LiteralPath $item.FullName -Algorithm SHA256).Hash.ToLowerInvariant()
    "$hash  $($item.Name)"
}
$hashLines | Set-Content "$dist\SHA256.txt" -Encoding ascii

Write-Host '[9/10] Digital-signature status'
$unsigned = $false
foreach ($verification in $verificationFiles) {
    $text = Get-Content -LiteralPath $verification -Raw
    if ($text -match 'AUTHENTICODE_SIGNED=NO') {
        $unsigned = $true
    }
}
if ($unsigned) {
    Write-Warning 'Binaries are not Authenticode-signed. Verified Publisher requires a real Brendigo code-signing certificate.'
}

Write-Host '[10/10] Final output verification'
foreach ($file in $publicFiles) {
    if (-not (Test-Path -LiteralPath $file -PathType Leaf)) {
        throw "Missing production output: $file"
    }
}
if (Test-Path -LiteralPath "$payload\payload.zip") {
    throw 'Temporary installer payload was not removed.'
}
Write-Host "ByFTP $version Windows x64+x86 build completed: $dist"
