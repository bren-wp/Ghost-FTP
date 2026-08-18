param(
    [Parameter(Mandatory = $true)][string]$Repository,
    [Parameter(Mandatory = $true)][string]$Version
)

$ErrorActionPreference = 'Stop'
Set-StrictMode -Version Latest

if ($Version -ne '1.0.0') {
    throw 'Reset stare release linije dopušten je isključivo za prijelaz na 1.0.0.'
}
if ([string]::IsNullOrWhiteSpace($env:GH_TOKEN)) {
    throw 'GH_TOKEN nije postavljen za reset release linije.'
}

$targetTag = 'v1.0.0'

Write-Host 'Brišem stare GitHub Release zapise...'
$releaseJson = & gh api --paginate "repos/$Repository/releases?per_page=100" --slurp
if ($LASTEXITCODE -ne 0) { throw 'Nije moguće dohvatiti postojeća GitHub izdanja.' }
$releasePages = $releaseJson | ConvertFrom-Json
$releases = @($releasePages | ForEach-Object { @($_) } | ForEach-Object { $_ })
# --slurp može vratiti array stranica; flatten eksplicitno i ne oslanjaj se na
# PowerShell implicitnu enumeraciju različitih JSON oblika.
$flatReleases = New-Object System.Collections.Generic.List[object]
foreach ($page in @($releasePages)) {
    foreach ($release in @($page)) { $flatReleases.Add($release) }
}
foreach ($release in $flatReleases) {
    $tagName = [string]$release.tag_name
    if ($tagName -eq $targetTag) { continue }
    Write-Host "  brišem release $tagName (#$($release.id))"
    & gh api --method DELETE "repos/$Repository/releases/$($release.id)"
    if ($LASTEXITCODE -ne 0) { throw "Nije moguće obrisati release $tagName." }
}

Write-Host 'Brišem stare v* Git tagove...'
$refsJson = & gh api --paginate "repos/$Repository/git/matching-refs/tags/v?per_page=100" --slurp
if ($LASTEXITCODE -ne 0) { throw 'Nije moguće dohvatiti postojeće v* tagove.' }
$refPages = $refsJson | ConvertFrom-Json
$flatRefs = New-Object System.Collections.Generic.List[object]
foreach ($page in @($refPages)) {
    foreach ($ref in @($page)) { $flatRefs.Add($ref) }
}
foreach ($ref in $flatRefs) {
    $fullRef = [string]$ref.ref
    $tagName = $fullRef -replace '^refs/tags/', ''
    if ($tagName -eq $targetTag) { continue }
    if ($tagName -notmatch '^v\d+\.\d+\.\d+(?:[-+].*)?$') { continue }
    Write-Host "  brišem tag $tagName"
    $encodedRef = [uri]::EscapeDataString("tags/$tagName")
    & gh api --method DELETE "repos/$Repository/git/refs/$encodedRef"
    if ($LASTEXITCODE -ne 0) { throw "Nije moguće obrisati tag $tagName." }
}

# Završna fail-closed provjera: prije stvaranja novog 1.0.0 releasea ne smije
# postojati nijedan drugi release ili v* verzijski tag.
$remainingReleasesRaw = & gh api "repos/$Repository/releases?per_page=100"
if ($LASTEXITCODE -ne 0) { throw 'Završna provjera izdanja nije dostupna.' }
$remainingReleases = @($remainingReleasesRaw | ConvertFrom-Json)
foreach ($release in $remainingReleases) {
    if ([string]$release.tag_name -ne $targetTag) {
        throw "Nakon reseta je ostao neočekivani release: $($release.tag_name)"
    }
}

Write-Host 'RELEASE_LINE_RESET=PASS (nova linija počinje s 1.0.0)'
