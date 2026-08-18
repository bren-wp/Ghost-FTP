param(
    [Parameter(Mandatory = $true)][string]$Version,
    [Parameter(Mandatory = $true)][string]$Dist,
    [Parameter(Mandatory = $true)][string]$Output
)

$ErrorActionPreference = 'Stop'
Set-StrictMode -Version Latest

if ($Version -notmatch '^\d+\.\d+\.\d+$') { throw "Neispravna package verzija: $Version" }
$distPath = (Resolve-Path -LiteralPath $Dist).Path
New-Item -ItemType Directory -Force -Path $Output | Out-Null
$outputPath = (Resolve-Path -LiteralPath $Output).Path

$packages = @(
    @{ Id = 'ByFTP.Suite'; Files = @(
        "ByFTP-$Version-Setup-x64.exe", "ByFTP-$Version-Setup-x86.exe",
        "ByFTP-$Version-Portable-x64.exe", "ByFTP-$Version-Portable-x86.exe"
    )},
    @{ Id = 'ByFTP.FTP.Client'; Files = @("ByFTP-FTP-Client-$Version-Portable-x64.exe", "ByFTP-FTP-Client-$Version-Portable-x86.exe") },
    @{ Id = 'ByFTP.SFTP.Client'; Files = @("ByFTP-SFTP-Client-$Version-Portable-x64.exe", "ByFTP-SFTP-Client-$Version-Portable-x86.exe") },
    @{ Id = 'ByFTP.SSH.Client'; Files = @("ByFTP-SSH-Client-$Version-Portable-x64.exe", "ByFTP-SSH-Client-$Version-Portable-x86.exe") },
    @{ Id = 'ByFTP.S3.Client'; Files = @("ByFTP-S3-Client-$Version-Portable-x64.exe", "ByFTP-S3-Client-$Version-Portable-x86.exe") }
)

foreach ($package in $packages) {
    foreach ($name in $package.Files) {
        if (-not (Test-Path -LiteralPath (Join-Path $distPath $name) -PathType Leaf)) {
            throw "Nedostaje package ulaz: $name"
        }
    }
}

$tempRoot = Join-Path ([System.IO.Path]::GetTempPath()) ("byftp-packages-" + [Guid]::NewGuid().ToString('N'))
New-Item -ItemType Directory -Force -Path $tempRoot | Out-Null
try {
    foreach ($package in $packages) {
        $projectDir = Join-Path $tempRoot ($package.Id -replace '[^A-Za-z0-9._-]', '_')
        New-Item -ItemType Directory -Force -Path $projectDir | Out-Null
        $project = Join-Path $projectDir 'package.csproj'

        $items = New-Object System.Text.StringBuilder
        foreach ($name in $package.Files) {
            $full = Join-Path $distPath $name
            [void]$items.AppendLine("    <None Include=\"$full\" Pack=\"true\" PackagePath=\"tools\\$name\" />")
        }
        $readme = (Resolve-Path -LiteralPath (Join-Path $PSScriptRoot '..\README.md')).Path
        $license = (Resolve-Path -LiteralPath (Join-Path $PSScriptRoot '..\LICENSE')).Path
        [void]$items.AppendLine("    <None Include=\"$readme\" Pack=\"true\" PackagePath=\"README.md\" />")
        [void]$items.AppendLine("    <None Include=\"$license\" Pack=\"true\" PackagePath=\"LICENSE\" />")

        $xml = @"
<Project Sdk="Microsoft.NET.Sdk">
  <PropertyGroup>
    <TargetFramework>net8.0</TargetFramework>
    <PackageId>$($package.Id)</PackageId>
    <Version>$Version</Version>
    <Authors>Brendigo</Authors>
    <Company>Brendigo</Company>
    <Description>ByFTP $Version produkcijski Windows paket.</Description>
    <PackageProjectUrl>https://github.com/bren-wp/by-ftp</PackageProjectUrl>
    <PackageReadmeFile>README.md</PackageReadmeFile>
    <PackageLicenseFile>LICENSE</PackageLicenseFile>
    <IncludeBuildOutput>false</IncludeBuildOutput>
    <NoBuild>true</NoBuild>
    <NoWarn>NU5128</NoWarn>
  </PropertyGroup>
  <ItemGroup>
$($items.ToString())  </ItemGroup>
</Project>
"@
        Set-Content -LiteralPath $project -Value $xml -Encoding utf8NoBOM
        & dotnet pack $project -c Release -o $outputPath -p:PackageVersion=$Version
        if ($LASTEXITCODE -ne 0) { throw "GitHub Package build nije uspio: $($package.Id)" }
    }
} finally {
    Remove-Item -Recurse -Force $tempRoot -ErrorAction SilentlyContinue
}

$nupkgs = @(Get-ChildItem -LiteralPath $outputPath -Filter '*.nupkg' -File)
if ($nupkgs.Count -ne 5) { throw "Očekuje se točno 5 nupkg datoteka, pronađeno: $($nupkgs.Count)" }
foreach ($package in $packages) {
    $expected = "$($package.Id).$Version.nupkg"
    if (-not (Test-Path -LiteralPath (Join-Path $outputPath $expected) -PathType Leaf)) {
        throw "Nedostaje očekivani GitHub Package: $expected"
    }
}

Write-Host "GITHUB_PACKAGES_BUILD=PASS ($Version, 5 paketa)"
