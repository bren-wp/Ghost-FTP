$ErrorActionPreference = 'Stop'
Set-Location $PSScriptRoot

$versionFile = Join-Path $PSScriptRoot 'VERSION'
if (-not (Test-Path -LiteralPath $versionFile -PathType Leaf)) { throw 'Nedostaje VERSION datoteka.' }
$version = (Get-Content -LiteralPath $versionFile -Raw).Trim()
if ($version -notmatch '^\d+\.\d+\.\d+$') { throw "Neispravna ByFTP verzija u VERSION datoteci: $version" }

$minimumGo = [Version]'1.26.5'
$dist = Join-Path $PWD 'dist'
$payload = Join-Path $PWD 'cmd\installer\payload'
$icon = Join-Path $PWD 'build\icon.ico'
$env:GOTOOLCHAIN='local'; $env:GOPROXY='off'; $env:GOSUMDB='off'; $env:GOTELEMETRY='off'
if (-not (Get-Command go -ErrorAction SilentlyContinue)) { throw 'Go nije instaliran.' }
if (-not (Get-Command python -ErrorAction SilentlyContinue)) { throw 'Python 3 nije instaliran.' }
$rawGoVersion = (go env GOVERSION).Trim()
if ($rawGoVersion -notmatch '^go(\d+)\.(\d+)(?:\.(\d+))?$') { throw "Nije moguće provjeriti Go verziju: $rawGoVersion" }
$patch = if ($Matches[3]) { [int]$Matches[3] } else { 0 }
$goVersion = [Version]::new([int]$Matches[1], [int]$Matches[2], $patch)
if ($goVersion -lt $minimumGo) { throw "Za produkcijski ByFTP build potreban je Go $minimumGo ili noviji sigurnosni patch. Trenutačno: $rawGoVersion" }

Write-Host "ByFTP $version"
Write-Host '[1/12] Provjera slikovnih resursa'
python scripts/generate_brand_assets.py --check
if ($LASTEXITCODE -ne 0) { throw 'Slikovni resursi nisu prošli provjeru.' }
Write-Host '[2/12] Provjera hrvatskog korisničkog sadržaja'
python scripts/audit_croatian.py
if ($LASTEXITCODE -ne 0) { throw 'Provjera hrvatskog sadržaja nije prošla.' }
Write-Host '[3/12] Provjera privatnosti i mrežne politike'
python scripts/audit_privacy.py
if ($LASTEXITCODE -ne 0) { throw 'Provjera privatnosti nije prošla.' }
Write-Host "[4/12] Testovi i statička provjera ($rawGoVersion)"
go test ./...
if ($LASTEXITCODE -ne 0) { throw 'Go testovi nisu prošli.' }
go vet ./...
if ($LASTEXITCODE -ne 0) { throw 'Go vet nije prošao.' }
Write-Host '[5/12] Čišćenje izlaznih datoteka'
Remove-Item -Recurse -Force $dist -ErrorAction SilentlyContinue
New-Item -ItemType Directory -Force -Path $dist,$payload | Out-Null
Remove-Item "$payload\payload.zip" -Force -ErrorAction SilentlyContinue
$env:GOOS='windows'; $env:GOARCH='amd64'; $env:CGO_ENABLED='0'
$portable = Join-Path $dist "ByFTP-$version-Portable-x64.exe"
$uninstall = Join-Path $dist 'ByFTP-Uninstall.exe'
$setup = Join-Path $dist "ByFTP-$version-Setup-x64.exe"
$ldflags = "-s -w -H=windowsgui -X main.version=$version"
Write-Host '[6/12] Portable'
go build -trimpath -buildvcs=false -ldflags $ldflags -o $portable ./cmd/byftp
if ($LASTEXITCODE -ne 0) { throw 'Portable build nije uspio.' }
python scripts/pe_resources.py $portable --ico $icon --version $version --role portable --original-filename "ByFTP-$version-Portable-x64.exe"
if ($LASTEXITCODE -ne 0) { throw 'PE resource obrada Portable builda nije uspjela.' }
Write-Host '[7/12] Program za uklanjanje'
go build -trimpath -buildvcs=false -ldflags $ldflags -o $uninstall ./cmd/uninstaller
if ($LASTEXITCODE -ne 0) { throw 'Build programa za uklanjanje nije uspio.' }
python scripts/pe_resources.py $uninstall --ico $icon --version $version --role uninstaller --original-filename 'ByFTP-Uninstall.exe'
if ($LASTEXITCODE -ne 0) { throw 'PE resource obrada programa za uklanjanje nije uspjela.' }
Write-Host '[8/12] Komprimirani instalacijski paket'
python scripts/make_payload.py --app $portable --uninstaller $uninstall --output "$payload\payload.zip"
if ($LASTEXITCODE -ne 0) { throw 'Kompresija instalacijskog payloada nije uspjela.' }
Write-Host '[9/12] Instalacijski program'
try {
    go build -trimpath -buildvcs=false -ldflags $ldflags -o $setup ./cmd/installer
    if ($LASTEXITCODE -ne 0) { throw 'Setup build nije uspio.' }
} finally { Remove-Item "$payload\payload.zip" -Force -ErrorAction SilentlyContinue }
python scripts/pe_resources.py $setup --ico $icon --version $version --role setup --original-filename "ByFTP-$version-Setup-x64.exe"
if ($LASTEXITCODE -ne 0) { throw 'PE resource obrada Setup builda nije uspjela.' }
Write-Host '[10/12] PE i sigurnosna provjera'
python scripts/verify_release.py $setup $portable $uninstall | Tee-Object -FilePath "$dist\verification.txt"
if ($LASTEXITCODE -ne 0) { throw 'Provjera izdanja nije prošla.' }
Write-Host '[11/12] SHA-256'
$hashLines = foreach ($file in @($setup,$portable,$uninstall)) {
    $item = Get-Item -LiteralPath $file
    $hash = (Get-FileHash -LiteralPath $item.FullName -Algorithm SHA256).Hash.ToLowerInvariant()
    "$hash  $($item.Name)"
}
$hashLines | Set-Content "$dist\SHA256.txt" -Encoding ascii
Write-Host '[12/12] Status digitalnog potpisa'
$verification = Get-Content "$dist\verification.txt" -Raw
if ($verification -match 'AUTHENTICODE_SIGNED=NO') { Write-Warning 'Binariji nisu Authenticode potpisani. Za širu javnu distribuciju potpišite Portable, Uninstaller i Setup nakon PE resource obrade pa ponovno pokrenite provjeru.' }
Write-Host "ByFTP $version build dovršen: $dist"
