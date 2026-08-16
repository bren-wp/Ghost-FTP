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
    throw 'GH_TOKEN nije postavljen za objavu izdanja.'
}
if ($Version -notmatch '^\d+\.\d+\.\d+$') {
    throw "Neispravna verzija izdanja: $Version"
}
if ($Commit -notmatch '^[0-9a-fA-F]{40}$') {
    throw "Neispravan commit izdanja: $Commit"
}
if (-not (Test-Path -LiteralPath $NotesFile -PathType Leaf)) {
    throw "Nedostaju bilješke izdanja: $NotesFile"
}
if (-not $Assets -or $Assets.Count -eq 0) {
    throw 'Nije naveden nijedan release asset.'
}

function Invoke-GhJson {
    param([Parameter(Mandatory = $true)][string[]]$Arguments)
    $raw = & gh @Arguments
    if ($LASTEXITCODE -ne 0) {
        throw "GitHub CLI naredba nije uspjela: gh $($Arguments -join ' ')"
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
            throw "Tag $Tag pokazuje na neočekivani Git objekt tipa '$($obj.type)'."
        }
        $tagObject = Invoke-GhJson -Arguments @('api', "repos/$Repository/git/tags/$($obj.sha)")
        $obj = $tagObject.object
    }
    throw "Tag $Tag ima predubok lanac anotiranih tagova."
}

function Get-Release {
    param([Parameter(Mandatory = $true)][string]$Tag)
    return Try-GhJson -Arguments @('api', "repos/$Repository/releases/tags/$Tag")
}

function Assert-TagCommit {
    param([Parameter(Mandatory = $true)][string]$Tag)
    $tagCommit = Resolve-TagCommit -Tag $Tag
    if ($null -eq $tagCommit) {
        throw "Izdanje postoji, ali tag $Tag nije moguće pronaći."
    }
    if (-not [string]::Equals($tagCommit, $Commit, [StringComparison]::OrdinalIgnoreCase)) {
        throw "Tag $Tag pokazuje na $tagCommit umjesto očekivanog commita $Commit."
    }
}

function Local-AssetInfo {
    param([Parameter(Mandatory = $true)][string]$Path)
    if (-not (Test-Path -LiteralPath $Path -PathType Leaf)) {
        throw "Nedostaje release asset: $Path"
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
        throw "Postojeći asset $($Local.Name) ima drugu veličinu ($($Remote.size) != $($Local.Size))."
    }
    $remoteDigest = [string]$Remote.digest
    if ([string]::IsNullOrWhiteSpace($remoteDigest)) {
        throw "GitHub nije vratio digest postojećeg asseta $($Local.Name); ne mogu sigurno potvrditi identitet."
    }
    if (-not [string]::Equals($remoteDigest, $Local.Digest, [StringComparison]::OrdinalIgnoreCase)) {
        throw "Postojeći asset $($Local.Name) ima drugi digest ($remoteDigest != $($Local.Digest))."
    }
}

$tag = "v$Version"
$localAssets = @{}
foreach ($assetPath in $Assets) {
    $info = Local-AssetInfo -Path $assetPath
    if ($localAssets.ContainsKey($info.Name)) {
        throw "Dupliciran naziv lokalnog release asseta: $($info.Name)"
    }
    $localAssets[$info.Name] = $info
}

$release = Get-Release -Tag $tag
if ($null -ne $release) {
    Assert-TagCommit -Tag $tag
    & gh release edit $tag --repo $Repository --title "ByFTP $Version" --notes-file $NotesFile | Out-Null
    if ($LASTEXITCODE -ne 0) { throw "Ažuriranje metapodataka izdanja $tag nije uspjelo." }
} else {
    $existingTagCommit = Resolve-TagCommit -Tag $tag
    if ($null -ne $existingTagCommit) {
        if (-not [string]::Equals($existingTagCommit, $Commit, [StringComparison]::OrdinalIgnoreCase)) {
            throw "Postojeći tag $tag pokazuje na $existingTagCommit umjesto $Commit."
        }
        & gh release create $tag --repo $Repository --title "ByFTP $Version" --notes-file $NotesFile --verify-tag | Out-Null
    } else {
        & gh release create $tag --repo $Repository --title "ByFTP $Version" --notes-file $NotesFile --target $Commit | Out-Null
    }
    if ($LASTEXITCODE -ne 0) { throw "Stvaranje izdanja $tag nije uspjelo." }
    $release = Get-Release -Tag $tag
    if ($null -eq $release) { throw "Izdanje $tag nije dostupno nakon stvaranja." }
    Assert-TagCommit -Tag $tag
}

# Postojeći release smije sadržavati samo ugovoreni skup datoteka. Time rerun
# može nadopuniti djelomično izdanje, ali ne može tiho prihvatiti strani/stari asset.
foreach ($remote in @($release.assets)) {
    if (-not $localAssets.ContainsKey([string]$remote.name)) {
        throw "Izdanje $tag sadrži neočekivani asset '$($remote.name)'."
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
        Write-Host "Asset je već potvrđen: $name"
        continue
    }
    Write-Host "Nadopunjujem nedostajući asset: $name"
    & gh release upload $tag $local.Path --repo $Repository | Out-Null
    if ($LASTEXITCODE -ne 0) { throw "Upload asseta $name nije uspio." }
}

# Završna provjera ponovno čita GitHub stanje i zahtijeva točno isti skup i digest.
$release = Get-Release -Tag $tag
if ($null -eq $release) { throw "Izdanje $tag nije dostupno za završnu provjeru." }
Assert-TagCommit -Tag $tag
$finalByName = @{}
foreach ($remote in @($release.assets)) {
    $name = [string]$remote.name
    if ($finalByName.ContainsKey($name)) { throw "GitHub izdanje ima dupliciran asset: $name" }
    $finalByName[$name] = $remote
}
if ($finalByName.Count -ne $localAssets.Count) {
    throw "Izdanje $tag nema očekivani broj asseta ($($finalByName.Count) != $($localAssets.Count))."
}
foreach ($name in $localAssets.Keys) {
    if (-not $finalByName.ContainsKey($name)) { throw "Nedostaje završni asset: $name" }
    Assert-RemoteAsset -Remote $finalByName[$name] -Local $localAssets[$name]
}

Write-Host "RELEASE_PUBLISH_VERIFICATION=PASS ($tag -> $Commit, $($localAssets.Count) asseta)"
