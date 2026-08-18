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
$releaseRows = & gh api --paginate "repos/$Repository/releases?per_page=100" --jq '.[] | [.id, .tag_name] | @tsv'
if ($LASTEXITCODE -ne 0) { throw 'Nije moguće dohvatiti postojeća GitHub izdanja.' }
foreach ($row in @($releaseRows)) {
    if ([string]::IsNullOrWhiteSpace($row)) { continue }
    $parts = $row -split "`t", 2
    if ($parts.Count -ne 2) { throw "Neočekivani GitHub release zapis: $row" }
    $id, $tagName = $parts[0], $parts[1]
    if ($tagName -eq $targetTag) { continue }
    if ($id -notmatch '^\d+$') { throw "Neispravan release ID za $tagName: $id" }
    Write-Host "  brišem release $tagName (#$id)"
    & gh api --method DELETE "repos/$Repository/releases/$id"
    if ($LASTEXITCODE -ne 0) { throw "Nije moguće obrisati release $tagName." }
}

Write-Host 'Brišem stare v* Git tagove...'
$tagRows = & gh api --paginate "repos/$Repository/git/matching-refs/tags/v?per_page=100" --jq '.[].ref'
if ($LASTEXITCODE -ne 0) { throw 'Nije moguće dohvatiti postojeće v* tagove.' }
foreach ($fullRef in @($tagRows)) {
    if ([string]::IsNullOrWhiteSpace($fullRef)) { continue }
    $tagName = [string]$fullRef -replace '^refs/tags/', ''
    if ($tagName -eq $targetTag) { continue }
    if ($tagName -notmatch '^v\d+\.\d+\.\d+(?:[-+].*)?$') { continue }
    Write-Host "  brišem tag $tagName"
    & gh api --method DELETE "repos/$Repository/git/refs/tags/$tagName"
    if ($LASTEXITCODE -ne 0) { throw "Nije moguće obrisati tag $tagName." }
}

# Završna fail-closed provjera: prije stvaranja novog 1.0.0 releasea ne smije
# postojati nijedan drugi release ili v* verzijski tag.
$remainingReleaseTags = & gh api --paginate "repos/$Repository/releases?per_page=100" --jq '.[].tag_name'
if ($LASTEXITCODE -ne 0) { throw 'Završna provjera GitHub izdanja nije dostupna.' }
foreach ($tagName in @($remainingReleaseTags)) {
    if ($tagName -and $tagName -ne $targetTag) {
        throw "Nakon reseta je ostao neočekivani release: $tagName"
    }
}
$remainingRefs = & gh api --paginate "repos/$Repository/git/matching-refs/tags/v?per_page=100" --jq '.[].ref'
if ($LASTEXITCODE -ne 0) { throw 'Završna provjera Git tagova nije dostupna.' }
foreach ($fullRef in @($remainingRefs)) {
    $tagName = [string]$fullRef -replace '^refs/tags/', ''
    if ($tagName -match '^v\d+\.\d+\.\d+(?:[-+].*)?$' -and $tagName -ne $targetTag) {
        throw "Nakon reseta je ostao neočekivani verzijski tag: $tagName"
    }
}

Write-Host 'RELEASE_LINE_RESET=PASS (nova linija počinje s 1.0.0)'
