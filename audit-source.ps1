$ErrorActionPreference = 'Stop'
$root = Split-Path -Parent $MyInvocation.MyCommand.Path

$packageRefs = Get-ChildItem $root -Recurse -Filter *.csproj | Select-String -Pattern '<PackageReference'
if ($packageRefs) {
    $packageRefs | ForEach-Object { Write-Error "Third-party/package dependency found: $($_.Path):$($_.LineNumber)" }
    exit 1
}

$forbidden = @(
    'ApplicationInsights', 'Sentry', 'TelemetryClient', 'GoogleAnalytics',
    'Segment.Analytics', 'Mixpanel', 'PostHog', 'AppCenter', 'Crashlytics'
)
foreach ($token in $forbidden) {
    $matches = Get-ChildItem (Join-Path $root 'src') -Recurse -Filter *.cs | Select-String -SimpleMatch $token
    if ($matches) {
        $matches | ForEach-Object { Write-Error "Forbidden telemetry/tracking reference '$token': $($_.Path):$($_.LineNumber)" }
        exit 1
    }
}

$version = (Get-Content (Join-Path $root 'VERSION') -Raw).Trim()
$props = [xml](Get-Content (Join-Path $root 'Directory.Build.props') -Raw)
$propsVersion = [string]$props.Project.PropertyGroup.Version
$assemblyVersion = [string]$props.Project.PropertyGroup.AssemblyVersion
$fileVersion = [string]$props.Project.PropertyGroup.FileVersion
if ($version -ne $propsVersion) { throw "VERSION ($version) does not match Directory.Build.props Version ($propsVersion)." }
$expectedAssembly = "$version.0"
if ($assemblyVersion -ne $expectedAssembly -or $fileVersion -ne $expectedAssembly) {
    throw "AssemblyVersion/FileVersion must both be $expectedAssembly."
}

foreach ($manifest in @('src/GhostFTP.App/app.manifest','src/GhostFTP.Setup/app.manifest')) {
    $text = Get-Content (Join-Path $root $manifest) -Raw
    if ($text -notmatch [regex]::Escape("version=`"$expectedAssembly`"")) {
        throw "$manifest does not use assembly identity $expectedAssembly."
    }
}

$forbiddenSourceExtensions = @('*.go','*.rs','*.cpp','*.c','*.cc','*.java','*.kt','*.swift')
foreach ($pattern in $forbiddenSourceExtensions) {
    $matches = Get-ChildItem (Join-Path $root 'src') -Recurse -File -Filter $pattern
    if ($matches) { throw "Non-C# source exists under src/: $($matches.FullName -join ', ')" }
}

Write-Host "Source audit passed for GhostFTP $version: C# only, zero PackageReference entries, no known telemetry/tracking SDK references, version metadata synchronized."
