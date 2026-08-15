$ErrorActionPreference = 'Stop'
Set-Location $PSScriptRoot
$version = '2.12.0'
$minimumGo = [Version]'1.26.5'
$dist = Join-Path $PWD 'dist'
$payload = Join-Path $PWD 'cmd\installer\payload'
$icon = Join-Path $PWD 'build\icon.ico'

# Release builds are intentionally offline: ByFTP has no external Go modules and
# must never auto-download a toolchain/module during a production build.
$env:GOTOOLCHAIN='local'
$env:GOPROXY='off'
$env:GOSUMDB='off'

if (-not (Get-Command go -ErrorAction SilentlyContinue)) { throw 'Go nije instaliran.' }
if (-not (Get-Command python -ErrorAction SilentlyContinue)) { throw 'Python 3 nije instaliran.' }

$rawGoVersion = (go env GOVERSION).Trim()
if ($rawGoVersion -notmatch '^go(\d+)\.(\d+)(?:\.(\d+))?$') {
    throw "Nije moguće provjeriti Go verziju: $rawGoVersion"
}
$patch = if ($Matches[3]) { [int]$Matches[3] } else { 0 }
$goVersion = [Version]::new([int]$Matches[1], [int]$Matches[2], $patch)
if ($goVersion -lt $minimumGo) {
    throw "Za produkcijski ByFTP build potreban je Go $minimumGo ili noviji security-patch release. Trenutno: $rawGoVersion"
}

Write-Host "[1/10] Privacy/network audit"
python scripts/audit_privacy.py
if ($LASTEXITCODE -ne 0) { throw 'Privacy audit nije prošao.' }

Write-Host "[2/10] Testovi i statička provjera ($rawGoVersion)"
go test ./...
if ($LASTEXITCODE -ne 0) { throw 'Go testovi nisu prošli.' }
go vet ./...
if ($LASTEXITCODE -ne 0) { throw 'Go vet nije prošao.' }

Write-Host '[3/10] Čišćenje outputa'
Remove-Item -Recurse -Force $dist -ErrorAction SilentlyContinue
New-Item -ItemType Directory -Force -Path $dist,$payload | Out-Null
Remove-Item "$payload\payload.zip" -Force -ErrorAction SilentlyContinue
$env:GOOS='windows'; $env:GOARCH='amd64'; $env:CGO_ENABLED='0'

$portable = Join-Path $dist "ByFTP-$version-Portable-x64.exe"
$uninstall = Join-Path $dist 'ByFTP-Uninstall.exe'
$setup = Join-Path $dist "ByFTP-$version-Setup-x64.exe"
$ldflags = "-s -w -H=windowsgui -X main.version=$version"

Write-Host '[4/10] Portable'
go build -trimpath -buildvcs=false -ldflags=$ldflags -o $portable ./cmd/byftp
if ($LASTEXITCODE -ne 0) { throw 'Portable build nije uspio.' }
python scripts/pe_resources.py $portable --ico $icon --version $version --role portable --original-filename "ByFTP-$version-Portable-x64.exe"
if ($LASTEXITCODE -ne 0) { throw 'PE resource obrada portable builda nije uspjela.' }

Write-Host '[5/10] Uninstaller'
go build -trimpath -buildvcs=false -ldflags=$ldflags -o $uninstall ./cmd/uninstaller
if ($LASTEXITCODE -ne 0) { throw 'Uninstaller build nije uspio.' }
python scripts/pe_resources.py $uninstall --ico $icon --version $version --role uninstaller --original-filename 'ByFTP-Uninstall.exe'
if ($LASTEXITCODE -ne 0) { throw 'PE resource obrada uninstallera nije uspjela.' }

Write-Host '[6/10] Komprimirani setup payload'
python scripts/make_payload.py --app $portable --uninstaller $uninstall --output "$payload\payload.zip"
if ($LASTEXITCODE -ne 0) { throw 'Kompresija setup payloada nije uspjela.' }

Write-Host '[7/10] Setup'
try {
    go build -trimpath -buildvcs=false -ldflags=$ldflags -o $setup ./cmd/installer
    if ($LASTEXITCODE -ne 0) { throw 'Setup build nije uspio.' }
} finally {
    Remove-Item "$payload\payload.zip" -Force -ErrorAction SilentlyContinue
}
python scripts/pe_resources.py $setup --ico $icon --version $version --role setup --original-filename "ByFTP-$version-Setup-x64.exe"
if ($LASTEXITCODE -ne 0) { throw 'PE resource obrada setupa nije uspjela.' }

Write-Host '[8/10] PE/security verifikacija'
python scripts/verify_release.py $setup $portable $uninstall | Tee-Object -FilePath "$dist\verification.txt"
if ($LASTEXITCODE -ne 0) { throw 'Release verifikacija nije prošla.' }

Write-Host '[9/10] SHA-256'
Get-FileHash $setup,$portable,$uninstall -Algorithm SHA256 | Format-Table -AutoSize | Out-String | Set-Content "$dist\SHA256.txt"

Write-Host '[10/10] Potpis'
$verification = Get-Content "$dist\verification.txt" -Raw
if ($verification -match 'AUTHENTICODE_SIGNED=NO') {
    Write-Warning 'Binariji nisu Authenticode potpisani. Za javnu distribuciju potpiši portable, uninstaller i setup nakon PE resource obrade pa ponovno pokreni verifikaciju.'
}
Write-Host "ByFTP build završen: $dist"
