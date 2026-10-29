$ErrorActionPreference = 'Stop'
$Repo = (Resolve-Path (Join-Path $PSScriptRoot '..')).Path
Push-Location (Join-Path $Repo 'ghostftp-desktop/src-tauri')
try {
    & cargo build --release --locked -p ghostftp-cli
    if ($LASTEXITCODE -ne 0) { throw 'Ghost CLI native build failed.' }
} finally { Pop-Location }
& (Join-Path $Repo 'ghostftp-desktop/src-tauri/target/release/ghostftp-cli.exe') --version
if ($LASTEXITCODE -ne 0) { throw 'Ghost CLI native version smoke failed.' }
