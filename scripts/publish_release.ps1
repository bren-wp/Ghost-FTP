param(
    [Parameter(Mandatory = $true)][string]$Repository,
    [Parameter(Mandatory = $true)][string]$Version,
    [Parameter(Mandatory = $true)][string]$Commit,
    [Parameter(Mandatory = $true)][string]$NotesFile,
    [Parameter(Mandatory = $true)][string[]]$Assets
)

$ErrorActionPreference = 'Stop'
Set-StrictMode -Version Latest

if ([string]::IsNullOrWhiteSpace($env:GH_TOKEN)) {
    throw 'GH_TOKEN is not configured for release publishing.'
}
if ($Version -notmatch '^\d+\.\d+\.\d+$') {
    throw "Invalid release version: $Version"
}
if ($Commit -notmatch '^[0-9a-fA-F]{40}$') {
    throw "Invalid release commit: $Commit"
}
if (-not (Test-Path -LiteralPath $NotesFile -PathType Leaf)) {
    throw "Release notes are missing: $NotesFile"
}
if (-not $Assets -or $Assets.Count -eq 0) {
    throw 'No release assets were provided.'
}

function Invoke-GhJson {
    param([Parameter(Mandatory = $true)][string[]]$Arguments)
    $raw = & gh @Arguments
    if ($LASTEXITCODE -ne 0) {
        throw "GitHub CLI command failed: gh $($Arguments -join ' ')"
    }
    $text = ($raw -join "`n").Trim()
    if (-not $text) { return $null }
    return $text | ConvertFrom-Json
}

function Try-GhJson {
    param([Parameter(Mandatory = $true)][string[]]$Arguments)
    $raw = & gh @Arguments 2>$null
    if ($LASTEXITCODE -ne 0) { return $null }
    $text = ($raw -join "`n").Trim()
    if (-not $text) { return $null }
    return $text | ConvertFrom-Json
}

function Resolve-TagCommit {
    param([Parameter(Mandatory = $true)][string]$Tag)
    $ref = Try-GhJson -Arguments @('api', "repos/$Repository/git/ref/tags/$Tag")
    if ($null -eq $ref) { return $null }
    $obj = $ref.object
    for ($depth = 0; $depth -lt 8; $depth++) {
        if ($obj.type -eq 'commit') {
            return [string]$obj.sha
        }
        if ($obj.type -ne 'tag') {
            throw "Tag $Tag points to unexpected Git object type '$($obj.type)'."
        }
        $tagObject = Invoke-GhJson -Arguments @('api', "repos/$Repository/git/tags/$($obj.sha)")
        $obj = $tagObject.object
    }
    throw "Tag $Tag has an excessively deep annotated-tag chain."
}

function Get-Release {
    param([Parameter(Mandatory = $true)][string]$Tag)
    return Try-GhJson -Arguments @('api', "repos/$Repository/releases/tags/$Tag")
}

function Assert-TagCommit {
    param([Parameter(Mandatory = $true)][string]$Tag)
    $tagCommit = Resolve-TagCommit -Tag $Tag
    if ($null -eq $tagCommit) {
        throw "Release exists, but tag $Tag cannot be resolved."
    }
    if (-not [string]::Equals($tagCommit, $Commit, [StringComparison]::OrdinalIgnoreCase)) {
        throw "Tag $Tag points to $tagCommit instead of expected commit $Commit."
    }
}

function Local-AssetInfo {
    param([Parameter(Mandatory = $true)][string]$Path)
    if (-not (Test-Path -LiteralPath $Path -PathType Leaf)) {
        throw "Release asset is missing: $Path"
    }
    $item = Get-Item -LiteralPath $Path
    $hash = (Get-FileHash -LiteralPath $item.FullName -Algorithm SHA256).Hash.ToLowerInvariant()
    return [pscustomobject]@{
        Path = $item.FullName
        Name = $item.Name
        Size = [int64]$item.Length
        Digest = "sha256:$hash"
    }
}

function Assert-RemoteAsset {
    param(
        [Parameter(Mandatory = $true)]$Remote,
        [Parameter(Mandatory = $true)]$Local
    )
    if ([int64]$Remote.size -ne $Local.Size) {
        throw "Existing asset $($Local.Name) has a different size ($($Remote.size) != $($Local.Size))."
    }
    $remoteDigest = [string]$Remote.digest
    if ([string]::IsNullOrWhiteSpace($remoteDigest)) {
        throw "GitHub did not return a digest for existing asset $($Local.Name); identity cannot be verified safely."
    }
    if (-not [string]::Equals($remoteDigest, $Local.Digest, [StringComparison]::OrdinalIgnoreCase)) {
        throw "Existing asset $($Local.Name) has a different digest ($remoteDigest != $($Local.Digest))."
    }
}

$tag = "v$Version"
$localAssets = @{}
foreach ($assetPath in $Assets) {
    $info = Local-AssetInfo -Path $assetPath
    if ($localAssets.ContainsKey($info.Name)) {
        throw "Duplicate local release asset name: $($info.Name)"
    }
    $localAssets[$info.Name] = $info
}

$release = Get-Release -Tag $tag
if ($null -ne $release) {
    Assert-TagCommit -Tag $tag
    & gh release edit $tag --repo $Repository --title "ByFTP $Version" --notes-file $NotesFile | Out-Null
    if ($LASTEXITCODE -ne 0) { throw "Updating release metadata for $tag failed." }
} else {
    $existingTagCommit = Resolve-TagCommit -Tag $tag
    if ($null -ne $existingTagCommit) {
        if (-not [string]::Equals($existingTagCommit, $Commit, [StringComparison]::OrdinalIgnoreCase)) {
            throw "Existing tag $tag points to $existingTagCommit instead of $Commit."
        }
        & gh release create $tag --repo $Repository --title "ByFTP $Version" --notes-file $NotesFile --verify-tag | Out-Null
    } else {
        & gh release create $tag --repo $Repository --title "ByFTP $Version" --notes-file $NotesFile --target $Commit | Out-Null
    }
    if ($LASTEXITCODE -ne 0) { throw "Creating release $tag failed." }
    $release = Get-Release -Tag $tag
    if ($null -eq $release) { throw "Release $tag is unavailable after creation." }
    Assert-TagCommit -Tag $tag
}

# An existing release may contain only the explicit asset allowlist. A rerun can
# safely repair a partial release, but cannot silently accept foreign/old assets.
foreach ($remote in @($release.assets)) {
    if (-not $localAssets.ContainsKey([string]$remote.name)) {
        throw "Release $tag contains unexpected asset '$($remote.name)'."
    }
}

$remoteByName = @{}
foreach ($remote in @($release.assets)) {
    $remoteByName[[string]$remote.name] = $remote
}

foreach ($name in ($localAssets.Keys | Sort-Object)) {
    $local = $localAssets[$name]
    if ($remoteByName.ContainsKey($name)) {
        Assert-RemoteAsset -Remote $remoteByName[$name] -Local $local
        Write-Host "Asset already verified: $name"
        continue
    }
    Write-Host "Uploading missing asset: $name"
    & gh release upload $tag $local.Path --repo $Repository | Out-Null
    if ($LASTEXITCODE -ne 0) { throw "Uploading asset $name failed." }
}

# Final verification reads GitHub state again and requires the exact same asset
# set, tag/commit binding, size and digest.
$release = Get-Release -Tag $tag
if ($null -eq $release) { throw "Release $tag is unavailable for final verification." }
Assert-TagCommit -Tag $tag
$finalByName = @{}
foreach ($remote in @($release.assets)) {
    $name = [string]$remote.name
    if ($finalByName.ContainsKey($name)) { throw "GitHub release contains duplicate asset: $name" }
    $finalByName[$name] = $remote
}
if ($finalByName.Count -ne $localAssets.Count) {
    throw "Release $tag has an unexpected asset count ($($finalByName.Count) != $($localAssets.Count))."
}
foreach ($name in $localAssets.Keys) {
    if (-not $finalByName.ContainsKey($name)) { throw "Final release asset is missing: $name" }
    Assert-RemoteAsset -Remote $finalByName[$name] -Local $localAssets[$name]
}

Write-Host "RELEASE_PUBLISH_VERIFICATION=PASS ($tag -> $Commit, $($localAssets.Count) assets)"
