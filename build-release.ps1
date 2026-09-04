$ErrorActionPreference = 'Stop'
$root = Split-Path -Parent $MyInvocation.MyCommand.Path
Set-Location $root

$release = Join-Path $root 'release'
$artifacts = Join-Path $root 'artifacts'
Remove-Item $release -Recurse -Force -ErrorAction SilentlyContinue
Remove-Item $artifacts -Recurse -Force -ErrorAction SilentlyContinue
New-Item $release -ItemType Directory | Out-Null
New-Item $artifacts -ItemType Directory | Out-Null

$architectures = @(
    @{ Rid = 'win-x64'; Suffix = 'win-x64' },
    @{ Rid = 'win-arm64'; Suffix = 'win-arm64' }
)

foreach ($arch in $architectures) {
    $portableDir = Join-Path $artifacts ("portable-" + $arch.Suffix)
    $setupDir = Join-Path $artifacts ("setup-" + $arch.Suffix)

    dotnet publish src/GhostFTP.App/GhostFTP.App.csproj -c Release -r $arch.Rid --self-contained true -o $portableDir
    if ($LASTEXITCODE -ne 0) { throw "Portable publish failed for $($arch.Rid)." }

    $payload = Join-Path $portableDir 'GhostFTP.exe'
    if (!(Test-Path $payload)) { throw "Portable payload is missing: $payload" }

    dotnet publish src/GhostFTP.Setup/GhostFTP.Setup.csproj -c Release -r $arch.Rid --self-contained true -o $setupDir -p:GhostFtpPayloadPath="$payload"
    if ($LASTEXITCODE -ne 0) { throw "Setup publish failed for $($arch.Rid)." }

    Copy-Item $payload (Join-Path $release ("GhostFTP-Portable-" + $arch.Suffix + '.exe'))
    Copy-Item (Join-Path $setupDir 'GhostFTP-Setup.exe') (Join-Path $release ("GhostFTP-Setup-" + $arch.Suffix + '.exe'))
}

$checksumLines = Get-ChildItem $release -Filter *.exe | Sort-Object Name | ForEach-Object {
    $hash = (Get-FileHash $_.FullName -Algorithm SHA256).Hash.ToLowerInvariant()
    "$hash  $($_.Name)"
}
$checksumLines | Set-Content (Join-Path $release 'SHA256SUMS.txt') -Encoding ascii

Write-Host "Release ready: $release"
Get-ChildItem $release | Format-Table Name, Length
