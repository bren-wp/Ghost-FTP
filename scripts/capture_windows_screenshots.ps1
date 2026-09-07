param(
    [Parameter(Mandatory = $true)]
    [string]$Executable,

    [string]$OutputDirectory = "ui-screenshots"
)

$ErrorActionPreference = "Stop"
Set-StrictMode -Version Latest

Add-Type -AssemblyName System.Drawing
Add-Type @"
using System;
using System.Text;
using System.Runtime.InteropServices;

public static class GhostFtpCaptureNative
{
    [StructLayout(LayoutKind.Sequential)]
    public struct RECT
    {
        public int Left;
        public int Top;
        public int Right;
        public int Bottom;
    }

    public delegate bool EnumWindowsProc(IntPtr hWnd, IntPtr lParam);

    [DllImport("user32.dll")]
    public static extern bool GetWindowRect(IntPtr hWnd, out RECT rect);

    [DllImport("user32.dll")]
    public static extern bool PrintWindow(IntPtr hWnd, IntPtr hdcBlt, uint nFlags);

    [DllImport("user32.dll", CharSet = CharSet.Unicode)]
    public static extern int GetWindowText(IntPtr hWnd, StringBuilder text, int maxCount);

    [DllImport("user32.dll")]
    public static extern bool EnumWindows(EnumWindowsProc callback, IntPtr extraData);

    [DllImport("user32.dll")]
    public static extern bool EnumChildWindows(IntPtr hWndParent, EnumWindowsProc callback, IntPtr extraData);

    [DllImport("user32.dll")]
    public static extern uint GetWindowThreadProcessId(IntPtr hWnd, out uint processId);

    [DllImport("user32.dll")]
    public static extern bool IsWindowVisible(IntPtr hWnd);

    [DllImport("user32.dll")]
    public static extern bool PostMessage(IntPtr hWnd, uint message, IntPtr wParam, IntPtr lParam);

    [DllImport("user32.dll")]
    public static extern bool SetForegroundWindow(IntPtr hWnd);
}
"@

function Wait-ForMainWindow {
    param(
        [Parameter(Mandatory = $true)]
        [System.Diagnostics.Process]$Process,
        [int]$TimeoutSeconds = 20
    )

    $deadline = [DateTime]::UtcNow.AddSeconds($TimeoutSeconds)
    do {
        if ($Process.HasExited) {
            throw "Ghost FTP exited before its main window became available. Exit code: $($Process.ExitCode)"
        }
        $Process.Refresh()
        if ($Process.MainWindowHandle -ne [IntPtr]::Zero) {
            return $Process.MainWindowHandle
        }
        Start-Sleep -Milliseconds 200
    } while ([DateTime]::UtcNow -lt $deadline)

    throw "Timed out waiting for the Ghost FTP main window."
}

function Find-ProcessWindow {
    param(
        [Parameter(Mandatory = $true)]
        [int]$ProcessId,
        [Parameter(Mandatory = $true)]
        [string]$TitleContains,
        [int]$TimeoutSeconds = 15
    )

    $deadline = [DateTime]::UtcNow.AddSeconds($TimeoutSeconds)
    do {
        $script:ghostFtpFoundWindow = [IntPtr]::Zero
        $script:ghostFtpTargetPid = [uint32]$ProcessId
        $script:ghostFtpTitle = $TitleContains

        $callback = [GhostFtpCaptureNative+EnumWindowsProc]{
            param([IntPtr]$hWnd, [IntPtr]$lParam)

            [uint32]$windowProcessId = 0
            [GhostFtpCaptureNative]::GetWindowThreadProcessId($hWnd, [ref]$windowProcessId) | Out-Null
            if ($windowProcessId -ne $script:ghostFtpTargetPid -or -not [GhostFtpCaptureNative]::IsWindowVisible($hWnd)) {
                return $true
            }

            $buffer = New-Object System.Text.StringBuilder 512
            [GhostFtpCaptureNative]::GetWindowText($hWnd, $buffer, $buffer.Capacity) | Out-Null
            $title = $buffer.ToString()
            if ($title -like "*$($script:ghostFtpTitle)*") {
                $script:ghostFtpFoundWindow = $hWnd
                return $false
            }
            return $true
        }

        [GhostFtpCaptureNative]::EnumWindows($callback, [IntPtr]::Zero) | Out-Null
        if ($script:ghostFtpFoundWindow -ne [IntPtr]::Zero) {
            return $script:ghostFtpFoundWindow
        }
        Start-Sleep -Milliseconds 200
    } while ([DateTime]::UtcNow -lt $deadline)

    throw "Timed out waiting for a Ghost FTP window containing title '$TitleContains'."
}

function Find-ChildWindowByText {
    param(
        [Parameter(Mandatory = $true)]
        [IntPtr]$Parent,
        [Parameter(Mandatory = $true)]
        [string]$Text,
        [int]$TimeoutSeconds = 10
    )

    $deadline = [DateTime]::UtcNow.AddSeconds($TimeoutSeconds)
    do {
        $script:ghostFtpFoundChild = [IntPtr]::Zero
        $script:ghostFtpChildText = $Text
        $callback = [GhostFtpCaptureNative+EnumWindowsProc]{
            param([IntPtr]$hWnd, [IntPtr]$lParam)

            if (-not [GhostFtpCaptureNative]::IsWindowVisible($hWnd)) {
                return $true
            }
            $buffer = New-Object System.Text.StringBuilder 512
            [GhostFtpCaptureNative]::GetWindowText($hWnd, $buffer, $buffer.Capacity) | Out-Null
            if ($buffer.ToString() -eq $script:ghostFtpChildText) {
                $script:ghostFtpFoundChild = $hWnd
                return $false
            }
            return $true
        }
        [GhostFtpCaptureNative]::EnumChildWindows($Parent, $callback, [IntPtr]::Zero) | Out-Null
        if ($script:ghostFtpFoundChild -ne [IntPtr]::Zero) {
            return $script:ghostFtpFoundChild
        }
        Start-Sleep -Milliseconds 150
    } while ([DateTime]::UtcNow -lt $deadline)

    throw "Timed out waiting for a Ghost FTP child control named '$Text'."
}

function Save-WindowScreenshot {
    param(
        [Parameter(Mandatory = $true)]
        [IntPtr]$Window,
        [Parameter(Mandatory = $true)]
        [string]$Path
    )

    $rect = New-Object GhostFtpCaptureNative+RECT
    if (-not [GhostFtpCaptureNative]::GetWindowRect($Window, [ref]$rect)) {
        throw "GetWindowRect failed for screenshot target."
    }

    $width = $rect.Right - $rect.Left
    $height = $rect.Bottom - $rect.Top
    if ($width -lt 320 -or $height -lt 200 -or $width -gt 8000 -or $height -gt 8000) {
        throw "Refusing implausible screenshot dimensions ${width}x${height}."
    }

    $bitmap = New-Object System.Drawing.Bitmap $width, $height, ([System.Drawing.Imaging.PixelFormat]::Format32bppArgb)
    $graphics = [System.Drawing.Graphics]::FromImage($bitmap)
    try {
        $hdc = $graphics.GetHdc()
        try {
            # PW_RENDERFULLCONTENT requests the composed native window rather
            # than fabricating a mockup. It also works when the hosted runner's
            # desktop is not physically visible.
            $rendered = [GhostFtpCaptureNative]::PrintWindow($Window, $hdc, 2)
            if (-not $rendered) {
                throw "PrintWindow failed for screenshot target."
            }
        }
        finally {
            $graphics.ReleaseHdc($hdc)
        }

        $fullPath = [System.IO.Path]::GetFullPath($Path)
        $parent = [System.IO.Path]::GetDirectoryName($fullPath)
        [System.IO.Directory]::CreateDirectory($parent) | Out-Null
        $bitmap.Save($fullPath, [System.Drawing.Imaging.ImageFormat]::Png)
    }
    finally {
        $graphics.Dispose()
        $bitmap.Dispose()
    }

    $file = Get-Item -LiteralPath $Path
    if ($file.Length -lt 10000) {
        throw "Screenshot output is unexpectedly small: $($file.FullName) ($($file.Length) bytes)."
    }
}

$exe = (Resolve-Path -LiteralPath $Executable).Path
New-Item -ItemType Directory -Force -Path $OutputDirectory | Out-Null

$process = Start-Process -FilePath $exe -PassThru
try {
    $main = Wait-ForMainWindow -Process $process
    [GhostFtpCaptureNative]::SetForegroundWindow($main) | Out-Null
    Start-Sleep -Milliseconds 900

    Save-WindowScreenshot -Window $main -Path (Join-Path $OutputDirectory "Ghost-FTP-main-workspace.png")

    # Site Manager is modal, so use the stable WM_COMMAND route and PostMessage
    # rather than blocking this capture process with a synchronous message.
    $wmCommand = 0x0111
    $siteManagerCommand = 701
    if (-not [GhostFtpCaptureNative]::PostMessage($main, $wmCommand, [IntPtr]$siteManagerCommand, [IntPtr]::Zero)) {
        throw "Could not request the Ghost FTP Site Manager."
    }

    $siteManager = Find-ProcessWindow -ProcessId $process.Id -TitleContains "Site Manager"
    [GhostFtpCaptureNative]::SetForegroundWindow($siteManager) | Out-Null
    Start-Sleep -Milliseconds 700
    Save-WindowScreenshot -Window $siteManager -Path (Join-Path $OutputDirectory "Ghost-FTP-site-manager.png")
    [GhostFtpCaptureNative]::PostMessage($siteManager, 0x0010, [IntPtr]::Zero, [IntPtr]::Zero) | Out-Null
    Start-Sleep -Milliseconds 350

    # Open the actual Settings card through its owner-drawn button. The first
    # appearance selector exercises the shared theme-aware option-dialog shell.
    $bmClick = 0x00F5
    $settingsButton = Find-ChildWindowByText -Parent $main -Text "Settings"
    if (-not [GhostFtpCaptureNative]::PostMessage($settingsButton, $bmClick, [IntPtr]::Zero, [IntPtr]::Zero)) {
        throw "Could not click the Ghost FTP Settings card."
    }
    $settingsWindow = Find-ProcessWindow -ProcessId $process.Id -TitleContains "Settings"
    [GhostFtpCaptureNative]::SetForegroundWindow($settingsWindow) | Out-Null
    Start-Sleep -Milliseconds 650
    Save-WindowScreenshot -Window $settingsWindow -Path (Join-Path $OutputDirectory "Ghost-FTP-settings.png")
    [GhostFtpCaptureNative]::PostMessage($settingsWindow, 0x0010, [IntPtr]::Zero, [IntPtr]::Zero) | Out-Null
    Start-Sleep -Milliseconds 350

    # About now uses its own native theme-aware information card rather than a
    # stock TaskDialog. Capture it from the real runtime surface as release proof.
    $aboutButton = Find-ChildWindowByText -Parent $main -Text "About"
    if (-not [GhostFtpCaptureNative]::PostMessage($aboutButton, $bmClick, [IntPtr]::Zero, [IntPtr]::Zero)) {
        throw "Could not click the Ghost FTP About card."
    }
    $aboutWindow = Find-ProcessWindow -ProcessId $process.Id -TitleContains "About"
    [GhostFtpCaptureNative]::SetForegroundWindow($aboutWindow) | Out-Null
    Start-Sleep -Milliseconds 650
    Save-WindowScreenshot -Window $aboutWindow -Path (Join-Path $OutputDirectory "Ghost-FTP-about.png")
    [GhostFtpCaptureNative]::PostMessage($aboutWindow, 0x0010, [IntPtr]::Zero, [IntPtr]::Zero) | Out-Null
    Start-Sleep -Milliseconds 300

    [GhostFtpCaptureNative]::PostMessage($main, 0x0010, [IntPtr]::Zero, [IntPtr]::Zero) | Out-Null

    if (-not $process.WaitForExit(5000)) {
        $process.Kill()
        $process.WaitForExit()
    }
}
finally {
    if (-not $process.HasExited) {
        $process.Kill()
        $process.WaitForExit()
    }
}

Write-Host "AUTHENTIC_UI_SCREENSHOTS=PASS"
Get-ChildItem -LiteralPath $OutputDirectory -Filter "*.png" | ForEach-Object {
    Write-Host ("SCREENSHOT={0} BYTES={1}" -f $_.Name, $_.Length)
}
