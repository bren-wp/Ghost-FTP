$ErrorActionPreference = 'Stop'
Set-Location $PSScriptRoot

$versionFile = Join-Path $PSScriptRoot 'VERSION'
if (-not (Test-Path -LiteralPath $versionFile -PathType Leaf)) {
    throw 'Nedostaje VERSION datoteka.'
}
$version = (Get-Content -LiteralPath $versionFile -Raw).Trim()
if ($version -notmatch '^\d+\.\d+\.\d+$') {
    throw "Neispravna ByFTP verzija u VERSION datoteci: $version"
}

$minimumGo = [Version]'1.26.5'
$dist = Join-Path $PWD 'dist'
$internalDist = Join-Path $dist 'internal'
$payload = Join-Path $PWD 'cmd\installer\payload'
$icon = Join-Path $PWD 'build\icon.ico'

# Produkcijski build je offline: ByFTP nema vanjske Go module i ne smije
# automatski preuzimati toolchain ili module tijekom produkcijske izgradnje.
$env:GOTOOLCHAIN = 'local'
$env:GOPROXY = 'off'
$env:GOSUMDB = 'off'
$env:CGO_ENABLED = '0'

if (-not (Get-Command go -ErrorAction SilentlyContinue)) { throw 'Go nije instaliran.' }
if (-not (Get-Command python -ErrorAction SilentlyContinue)) { throw 'Python 3 nije instaliran.' }

$rawGoVersion = (go env GOVERSION).Trim()
if ($rawGoVersion -notmatch '^go(\d+)\.(\d+)(?:\.(\d+))?$') {
    throw "Nije moguće provjeriti Go verziju: $rawGoVersion"
}
$patch = if ($Matches[3]) { [int]$Matches[3] } else { 0 }
$goVersion = [Version]::new([int]$Matches[1], [int]$Matches[2], $patch)
if ($goVersion -lt $minimumGo) {
    throw "Za produkcijski ByFTP build potreban je Go $minimumGo ili noviji sigurnosni patch. Trenutačno: $rawGoVersion"
}

$telemetryMode = (go telemetry).Trim()
if ($LASTEXITCODE -ne 0 -or $telemetryMode -ne 'off') {
    throw "Go telemetrija mora biti isključena prije produkcijskog builda. Pokrenite: go telemetry off (trenutačno: $telemetryMode)"
}

Write-Host "ByFTP $version"
Write-Host '[1/10] Slikovni resursi, hrvatski sadržaj i verzija'
python scripts/generate_brand_assets.py --check
if ($LASTEXITCODE -ne 0) { throw 'Slikovni resursi nisu prošli provjeru.' }
python scripts/audit_croatian.py
if ($LASTEXITCODE -ne 0) { throw 'Provjera hrvatskog sadržaja nije prošla.' }
python scripts/audit_version.py
if ($LASTEXITCODE -ne 0) { throw 'Provjera verzijske konzistentnosti nije prošla.' }

Write-Host '[2/10] Dokumentacija, sigurnost, privatnost i release ugovor'
python scripts/audit_docs.py
if ($LASTEXITCODE -ne 0) { throw 'Provjera dokumentacije nije prošla.' }
python scripts/audit_security.py
if ($LASTEXITCODE -ne 0) { throw 'Sigurnosni audit nije prošao.' }
python scripts/audit_privacy.py
if ($LASTEXITCODE -ne 0) { throw 'Provjera privatnosti nije prošla.' }
python scripts/audit_release.py
if ($LASTEXITCODE -ne 0) { throw 'Release audit nije prošao.' }

Write-Host '[3/10] Python regresije release alata'
python -m unittest discover -s scripts -p 'test_*.py'
if ($LASTEXITCODE -ne 0) { throw 'Python regresije release alata nisu prošle.' }

Write-Host "[4/10] Go testovi i statička provjera ($rawGoVersion, telemetry=$telemetryMode)"
go test ./...
if ($LASTEXITCODE -ne 0) { throw 'Go testovi nisu prošli.' }
go vet ./...
if ($LASTEXITCODE -ne 0) { throw 'Go vet nije prošao.' }

Write-Host '[5/10] Čišćenje izlaznih datoteka'
Remove-Item -Recurse -Force $dist -ErrorAction SilentlyContinue
New-Item -ItemType Directory -Force -Path $dist, $internalDist, $payload | Out-Null
Remove-Item "$payload\payload.zip" -Force -ErrorAction SilentlyContinue

$guiLdflags = "-s -w -H=windowsgui -X main.version=$version"
$consoleLdflags = "-s -w -X main.version=$version"
$publicFiles = New-Object System.Collections.Generic.List[string]
$verificationFiles = New-Object System.Collections.Generic.List[string]

$clientSpecs = @(
    @{ Name = 'FTP';  Path = './cmd/ftpclient';  Gui = $true  },
    @{ Name = 'SFTP'; Path = './cmd/sftpclient'; Gui = $true  },
    @{ Name = 'SSH';  Path = './cmd/sshclient';  Gui = $false },
    @{ Name = 'S3';   Path = './cmd/s3client';   Gui = $false }
)

function Build-StandaloneClient {
    param(
        [Parameter(Mandatory = $true)][string]$GoArch,
        [Parameter(Mandatory = $true)][ValidateSet('x64','x86')][string]$Label,
        [Parameter(Mandatory = $true)][hashtable]$Spec
    )

    $name = $Spec.Name
    $output = Join-Path $dist "ByFTP-$name-Client-$version-Portable-$Label.exe"
    $verification = Join-Path $internalDist "verification-$($name.ToLowerInvariant())-$Label.txt"
    $ldflags = if ($Spec.Gui) { $guiLdflags } else { $consoleLdflags }
    $subsystem = if ($Spec.Gui) { 'gui' } else { 'console' }

    Write-Host "      [$Label] $name Client"
    go build -trimpath -buildvcs=false -ldflags $ldflags -o $output $Spec.Path
    if ($LASTEXITCODE -ne 0) { throw "$name Client $Label build nije uspio." }
    python scripts/pe_resources.py $output --ico $icon --version $version --role portable --original-filename (Split-Path -Leaf $output)
    if ($LASTEXITCODE -ne 0) { throw "$name Client $Label PE resource obrada nije uspjela." }
    python scripts/verify_client_pe.py $output --arch $Label --subsystem $subsystem | Tee-Object -FilePath $verification
    if ($LASTEXITCODE -ne 0) { throw "$name Client $Label PE provjera nije prošla." }

    $script:publicFiles.Add($output)
    $script:verificationFiles.Add($verification)
}

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
    $verification = Join-Path $internalDist "verification-suite-$Label.txt"

    Write-Host "      [$Label] ByFTP Portable"
    go build -trimpath -buildvcs=false -ldflags $guiLdflags -o $portable ./cmd/byftp
    if ($LASTEXITCODE -ne 0) { throw "Portable $Label build nije uspio." }
    python scripts/pe_resources.py $portable --ico $icon --version $version --role portable --original-filename (Split-Path -Leaf $portable)
    if ($LASTEXITCODE -ne 0) { throw "PE resource obrada Portable $Label builda nije uspjela." }

    Write-Host "      [$Label] Interna komponenta uklanjanja"
    go build -trimpath -buildvcs=false -ldflags $guiLdflags -o $uninstall ./cmd/uninstaller
    if ($LASTEXITCODE -ne 0) { throw "Interna komponenta uklanjanja $Label nije izgrađena." }
    python scripts/pe_resources.py $uninstall --ico $icon --version $version --role uninstaller --original-filename 'Uninstall.exe'
    if ($LASTEXITCODE -ne 0) { throw "PE resource obrada interne komponente uklanjanja $Label nije uspjela." }

    Write-Host "      [$Label] Instalacijski payload"
    python scripts/make_payload.py --app $portable --uninstaller $uninstall --output "$payload\payload.zip"
    if ($LASTEXITCODE -ne 0) { throw "Kompresija $Label instalacijskog payloada nije uspjela." }

    Write-Host "      [$Label] ByFTP Setup"
    try {
        go build -trimpath -buildvcs=false -ldflags $guiLdflags -o $setup ./cmd/installer
        if ($LASTEXITCODE -ne 0) { throw "Setup $Label build nije uspio." }
    } finally {
        Remove-Item "$payload\payload.zip" -Force -ErrorAction SilentlyContinue
    }
    python scripts/pe_resources.py $setup --ico $icon --version $version --role setup --original-filename (Split-Path -Leaf $setup)
    if ($LASTEXITCODE -ne 0) { throw "PE resource obrada Setup $Label builda nije uspjela." }

    Write-Host "      [$Label] Suite PE, sigurnost i privatnost"
    python scripts/verify_release.py $setup $portable $uninstall --arch $Label | Tee-Object -FilePath $verification
    if ($LASTEXITCODE -ne 0) { throw "Provjera Suite $Label izdanja nije prošla." }

    $script:publicFiles.Add($portable)
    $script:publicFiles.Add($setup)
    $script:verificationFiles.Add($verification)

    foreach ($spec in $clientSpecs) {
        Build-StandaloneClient -GoArch $GoArch -Label $Label -Spec $spec
    }
}

Write-Host '[6/10] Windows x64 — Suite + FTP/SFTP/SSH/S3'
Build-ByFTPArchitecture -GoArch 'amd64' -Label 'x64'

Write-Host '[7/10] Windows x86 — Suite + FTP/SFTP/SSH/S3'
Build-ByFTPArchitecture -GoArch '386' -Label 'x86'

Write-Host '[8/10] SHA-256 interni dokaz svih javnih EXE datoteka'
$hashLines = foreach ($file in ($publicFiles | Sort-Object)) {
    $item = Get-Item -LiteralPath $file
    $hash = (Get-FileHash -LiteralPath $item.FullName -Algorithm SHA256).Hash.ToLowerInvariant()
    "$hash  $($item.Name)"
}
$hashLines | Set-Content "$internalDist\SHA256.txt" -Encoding ascii

Write-Host '[9/10] Status digitalnog potpisa'
$unsigned = $false
foreach ($verification in $verificationFiles) {
    $text = Get-Content -LiteralPath $verification -Raw
    if ($text -match 'AUTHENTICODE_SIGNED=NO') {
        $unsigned = $true
    }
}
if ($unsigned) {
    Write-Warning 'Binariji nisu Authenticode potpisani. Za Verified Publisher potreban je pravi Brendigo code-signing certifikat.'
}

Write-Host '[10/10] Završna kontrola 12 javnih EXE izlaza'
if ($publicFiles.Count -ne 12) {
    throw "Očekuje se točno 12 javnih Windows EXE izlaza, pronađeno: $($publicFiles.Count)"
}
foreach ($file in $publicFiles) {
    if (-not (Test-Path -LiteralPath $file -PathType Leaf)) {
        throw "Nedostaje produkcijski izlaz: $file"
    }
}
if (Test-Path -LiteralPath "$payload\payload.zip") {
    throw 'Privremeni instalacijski payload nije uklonjen.'
}
Write-Host "ByFTP $version Windows x64+x86 build dovršen: 12 javnih EXE datoteka u $dist"