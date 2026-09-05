param(
    [Parameter(Mandatory = $true)]
    [string]$PfxPath,

    [Parameter(Mandatory = $true)]
    [SecureString]$Password,

    [Parameter(Mandatory = $true)]
    [string[]]$Paths,

    [string]$TimestampUrl = '',

    [switch]$AllowUntrustedSigner
)

$ErrorActionPreference = 'Stop'
Set-StrictMode -Version Latest

if (-not $IsWindows) {
    throw 'Authenticode signing is supported only on Windows.'
}

$pfx = (Resolve-Path -LiteralPath $PfxPath).Path
if (-not (Test-Path -LiteralPath $pfx -PathType Leaf)) {
    throw "Signing PFX is missing: $pfx"
}

$resolvedPaths = @()
foreach ($path in $Paths) {
    $resolved = (Resolve-Path -LiteralPath $path).Path
    if (-not (Test-Path -LiteralPath $resolved -PathType Leaf)) {
        throw "Signing target is missing: $resolved"
    }
    if ([IO.Path]::GetExtension($resolved) -ine '.exe') {
        throw "Refusing to Authenticode-sign a non-EXE target: $resolved"
    }
    $resolvedPaths += $resolved
}
if ($resolvedPaths.Count -eq 0) {
    throw 'No signing targets were supplied.'
}

$store = 'Cert:\CurrentUser\My'
$imported = @()
$signingCertificate = $null
try {
    $before = @(Get-ChildItem -LiteralPath $store | Select-Object -ExpandProperty Thumbprint)
    Import-PfxCertificate -FilePath $pfx -CertStoreLocation $store -Password $Password | Out-Null
    $after = @(Get-ChildItem -LiteralPath $store)
    $imported = @($after | Where-Object { $_.Thumbprint -notin $before })

    $codeSigningOid = '1.3.6.1.5.5.7.3.3'
    $now = Get-Date
    $candidates = @($imported | Where-Object {
        $_.HasPrivateKey -and
        $_.NotBefore -le $now -and
        $_.NotAfter -gt $now -and
        @($_.EnhancedKeyUsageList | ForEach-Object { $_.ObjectId.Value }) -contains $codeSigningOid
    })
    if ($candidates.Count -ne 1) {
        throw "PFX must import exactly one currently valid code-signing certificate with a private key; found $($candidates.Count)."
    }
    $signingCertificate = $candidates[0]

    foreach ($target in $resolvedPaths) {
        $parameters = @{
            FilePath = $target
            Certificate = $signingCertificate
            HashAlgorithm = 'SHA256'
        }
        if (-not [string]::IsNullOrWhiteSpace($TimestampUrl)) {
            if ($TimestampUrl -notmatch '^https?://') {
                throw 'TimestampUrl must use http:// or https://.'
            }
            $parameters.TimestampServer = $TimestampUrl
        }

        $result = Set-AuthenticodeSignature @parameters
        if (-not $result.SignerCertificate) {
            throw "Authenticode signing returned no signer certificate for $target."
        }
        if ($result.SignerCertificate.Thumbprint -ne $signingCertificate.Thumbprint) {
            throw "Unexpected Authenticode signer for $target."
        }

        $verification = Get-AuthenticodeSignature -FilePath $target
        if (-not $verification.SignerCertificate) {
            throw "Signed file has no readable Authenticode signer: $target"
        }
        if ($verification.SignerCertificate.Thumbprint -ne $signingCertificate.Thumbprint) {
            throw "Signed file thumbprint mismatch: $target"
        }
        if ($verification.Status -eq 'HashMismatch' -or $verification.Status -eq 'NotSigned') {
            throw "Authenticode verification failed for ${target}: $($verification.Status)"
        }
        if (-not $AllowUntrustedSigner -and $verification.Status -ne 'Valid') {
            throw "Authenticode signer is not trusted/valid for production: ${target} ($($verification.Status): $($verification.StatusMessage))"
        }

        Write-Host "AUTHENTICODE_SIGNED=$([IO.Path]::GetFileName($target))"
        Write-Host "AUTHENTICODE_THUMBPRINT=$($signingCertificate.Thumbprint)"
        Write-Host "AUTHENTICODE_STATUS=$($verification.Status)"
    }
}
finally {
    foreach ($certificate in $imported) {
        $path = "Cert:\CurrentUser\My\$($certificate.Thumbprint)"
        if (Test-Path -LiteralPath $path) {
            Remove-Item -LiteralPath $path -Force
        }
    }
}
