param(
    [string]$OutputDirectory = "dev-signing",
    [string]$Subject = "CN=Ghost FTP Development",
    [int]$ValidDays = 30,
    [SecureString]$Password
)

$ErrorActionPreference = 'Stop'
Set-StrictMode -Version Latest

if ($ValidDays -lt 1 -or $ValidDays -gt 365) {
    throw 'ValidDays must be between 1 and 365.'
}

if (-not $IsWindows) {
    throw 'Development Authenticode certificate creation is supported only on Windows.'
}

if (-not $Password) {
    $Password = Read-Host 'Password for the temporary development PFX' -AsSecureString
}

$root = [IO.Path]::GetFullPath((Join-Path $PSScriptRoot '..'))
$output = if ([IO.Path]::IsPathRooted($OutputDirectory)) {
    [IO.Path]::GetFullPath($OutputDirectory)
} else {
    [IO.Path]::GetFullPath((Join-Path $root $OutputDirectory))
}

New-Item -ItemType Directory -Force -Path $output | Out-Null
$pfxPath = Join-Path $output 'Ghost-FTP-Development-Code-Signing.pfx'
$cerPath = Join-Path $output 'Ghost-FTP-Development-Code-Signing.cer'

foreach ($path in @($pfxPath, $cerPath)) {
    if (Test-Path -LiteralPath $path) {
        Remove-Item -LiteralPath $path -Force
    }
}

$certificate = $null
try {
    $certificate = New-SelfSignedCertificate `
        -Type CodeSigningCert `
        -Subject $Subject `
        -KeyAlgorithm RSA `
        -KeyLength 3072 `
        -HashAlgorithm SHA256 `
        -KeyExportPolicy Exportable `
        -NotAfter (Get-Date).AddDays($ValidDays) `
        -CertStoreLocation 'Cert:\CurrentUser\My'

    if (-not $certificate -or -not $certificate.HasPrivateKey) {
        throw 'Development code-signing certificate creation failed.'
    }

    Export-PfxCertificate -Cert $certificate -FilePath $pfxPath -Password $Password | Out-Null
    Export-Certificate -Cert $certificate -FilePath $cerPath -Type CERT | Out-Null

    if (-not (Test-Path -LiteralPath $pfxPath -PathType Leaf) -or (Get-Item $pfxPath).Length -le 0) {
        throw 'Development PFX export failed.'
    }
    if (-not (Test-Path -LiteralPath $cerPath -PathType Leaf) -or (Get-Item $cerPath).Length -le 0) {
        throw 'Development CER export failed.'
    }

    Write-Host 'DEVELOPMENT_CODE_SIGNING_CERTIFICATE=CREATED'
    Write-Host "THUMBPRINT=$($certificate.Thumbprint)"
    Write-Host "PFX=$pfxPath"
    Write-Host "CER=$cerPath"
    Write-Warning 'This self-signed certificate is for development/testing only. It does not create a trusted Windows publisher identity and must never be committed to Git.'
}
finally {
    if ($certificate) {
        $storePath = "Cert:\CurrentUser\My\$($certificate.Thumbprint)"
        if (Test-Path -LiteralPath $storePath) {
            Remove-Item -LiteralPath $storePath -Force
        }
    }
}
