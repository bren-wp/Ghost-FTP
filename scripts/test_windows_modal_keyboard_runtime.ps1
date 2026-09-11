param(
    [Parameter(Mandatory = $true)]
    [string]$Executable
)

$ErrorActionPreference = 'Stop'
Set-StrictMode -Version Latest

Add-Type @"
using System;
using System.Text;
using System.Runtime.InteropServices;

public static class GhostFtpKeyboardNative
{
    [StructLayout(LayoutKind.Sequential)]
    public struct RECT
    {
        public int Left;
        public int Top;
        public int Right;
        public int Bottom;
    }

    [StructLayout(LayoutKind.Sequential)]
    public struct GUITHREADINFO
    {
        public int cbSize;
        public uint flags;
        public IntPtr hwndActive;
        public IntPtr hwndFocus;
        public IntPtr hwndCapture;
        public IntPtr hwndMenuOwner;
        public IntPtr hwndMoveSize;
        public IntPtr hwndCaret;
        public RECT rcCaret;
    }

    public delegate bool EnumWindowsProc(IntPtr hWnd, IntPtr lParam);

    [DllImport("user32.dll")]
    public static extern bool EnumWindows(EnumWindowsProc callback, IntPtr extraData);

    [DllImport("user32.dll", CharSet = CharSet.Unicode)]
    public static extern int GetWindowText(IntPtr hWnd, StringBuilder text, int maxCount);

    [DllImport("user32.dll")]
    public static extern uint GetWindowThreadProcessId(IntPtr hWnd, out uint processId);

    [DllImport("user32.dll")]
    public static extern bool GetGUIThreadInfo(uint idThread, ref GUITHREADINFO info);

    [DllImport("user32.dll")]
    public static extern IntPtr GetDlgItem(IntPtr hDlg, int nIDDlgItem);

    [DllImport("user32.dll")]
    public static extern bool IsWindow(IntPtr hWnd);

    [DllImport("user32.dll")]
    public static extern bool IsWindowVisible(IntPtr hWnd);

    [DllImport("user32.dll")]
    public static extern bool IsWindowEnabled(IntPtr hWnd);

    [DllImport("user32.dll")]
    public static extern bool IsIconic(IntPtr hWnd);

    [DllImport("user32.dll")]
    public static extern bool PostMessage(IntPtr hWnd, uint message, IntPtr wParam, IntPtr lParam);

    [DllImport("user32.dll")]
    public static extern bool SetForegroundWindow(IntPtr hWnd);
}
"@

$wmCommand = 0x0111
$wmKeyDown = 0x0100
$wmKeyUp = 0x0101
$wmClose = 0x0010
$vkTab = 0x09
$vkReturn = 0x0D
$vkEscape = 0x1B
$siteManagerCommand = 701
$bookmarksCommand = 97
$siteManagerCloseControlId = 8114
$bookmarksCloseControlId = 8206

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

    throw 'Timed out waiting for the Ghost FTP main window.'
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
        $script:ghostFtpKeyboardFoundWindow = [IntPtr]::Zero
        $script:ghostFtpKeyboardTargetPid = [uint32]$ProcessId
        $script:ghostFtpKeyboardTitle = $TitleContains
        $callback = [GhostFtpKeyboardNative+EnumWindowsProc]{
            param([IntPtr]$hWnd, [IntPtr]$lParam)

            [uint32]$windowProcessId = 0
            [GhostFtpKeyboardNative]::GetWindowThreadProcessId($hWnd, [ref]$windowProcessId) | Out-Null
            if ($windowProcessId -ne $script:ghostFtpKeyboardTargetPid -or -not [GhostFtpKeyboardNative]::IsWindowVisible($hWnd)) {
                return $true
            }
            $buffer = New-Object System.Text.StringBuilder 512
            [GhostFtpKeyboardNative]::GetWindowText($hWnd, $buffer, $buffer.Capacity) | Out-Null
            if ($buffer.ToString() -like "*$($script:ghostFtpKeyboardTitle)*") {
                $script:ghostFtpKeyboardFoundWindow = $hWnd
                return $false
            }
            return $true
        }
        [GhostFtpKeyboardNative]::EnumWindows($callback, [IntPtr]::Zero) | Out-Null
        if ($script:ghostFtpKeyboardFoundWindow -ne [IntPtr]::Zero) {
            return $script:ghostFtpKeyboardFoundWindow
        }
        Start-Sleep -Milliseconds 150
    } while ([DateTime]::UtcNow -lt $deadline)

    throw "Timed out waiting for Ghost FTP window containing '$TitleContains'."
}

function Wait-ForControlById {
    param(
        [Parameter(Mandatory = $true)][IntPtr]$Parent,
        [Parameter(Mandatory = $true)][int]$ControlId,
        [Parameter(Mandatory = $true)][string]$Name,
        [int]$TimeoutSeconds = 10
    )

    $deadline = [DateTime]::UtcNow.AddSeconds($TimeoutSeconds)
    do {
        $control = [GhostFtpKeyboardNative]::GetDlgItem($Parent, $ControlId)
        if ($control -ne [IntPtr]::Zero -and [GhostFtpKeyboardNative]::IsWindow($control)) {
            return $control
        }
        Start-Sleep -Milliseconds 100
    } while ([DateTime]::UtcNow -lt $deadline)

    throw "Timed out waiting for $Name control ID $ControlId."
}

function Get-FocusedWindow {
    param([Parameter(Mandatory = $true)][IntPtr]$Window)

    [uint32]$processId = 0
    $threadId = [GhostFtpKeyboardNative]::GetWindowThreadProcessId($Window, [ref]$processId)
    if ($threadId -eq 0) {
        throw 'Unable to determine the Ghost FTP UI thread.'
    }
    $info = New-Object GhostFtpKeyboardNative+GUITHREADINFO
    $info.cbSize = [Runtime.InteropServices.Marshal]::SizeOf($info)
    if (-not [GhostFtpKeyboardNative]::GetGUIThreadInfo($threadId, [ref]$info)) {
        throw 'GetGUIThreadInfo failed for the Ghost FTP UI thread.'
    }
    return $info.hwndFocus
}

function Wait-ForWindowClosed {
    param(
        [Parameter(Mandatory = $true)][IntPtr]$Window,
        [string]$Name,
        [int]$TimeoutSeconds = 5
    )

    $deadline = [DateTime]::UtcNow.AddSeconds($TimeoutSeconds)
    do {
        if (-not [GhostFtpKeyboardNative]::IsWindow($Window)) {
            return
        }
        Start-Sleep -Milliseconds 100
    } while ([DateTime]::UtcNow -lt $deadline)
    throw "$Name did not close after the keyboard action."
}

function Assert-MainRestored {
    param(
        [Parameter(Mandatory = $true)][IntPtr]$Main,
        [Parameter(Mandatory = $true)][System.Diagnostics.Process]$Process,
        [string]$Context,
        [int]$TimeoutSeconds = 5
    )

    $deadline = [DateTime]::UtcNow.AddSeconds($TimeoutSeconds)
    do {
        if ($Process.HasExited) {
            throw "Ghost FTP exited unexpectedly after $Context."
        }
        if ([GhostFtpKeyboardNative]::IsWindowEnabled($Main) -and -not [GhostFtpKeyboardNative]::IsIconic($Main)) {
            return
        }
        Start-Sleep -Milliseconds 100
    } while ([DateTime]::UtcNow -lt $deadline)
    throw "Main window was not restored and enabled after $Context."
}

function Open-ModalWindow {
    param(
        [Parameter(Mandatory = $true)][IntPtr]$Main,
        [Parameter(Mandatory = $true)][int]$Command,
        [Parameter(Mandatory = $true)][int]$ProcessId,
        [Parameter(Mandatory = $true)][string]$Title
    )

    if (-not [GhostFtpKeyboardNative]::PostMessage($Main, $wmCommand, [IntPtr]$Command, [IntPtr]::Zero)) {
        throw "Could not request $Title."
    }
    $window = Find-ProcessWindow -ProcessId $ProcessId -TitleContains $Title
    [GhostFtpKeyboardNative]::SetForegroundWindow($window) | Out-Null
    Start-Sleep -Milliseconds 250
    return $window
}

function Focus-CloseWithTab {
    param(
        [Parameter(Mandatory = $true)][IntPtr]$Window,
        [Parameter(Mandatory = $true)][int]$CloseControlId,
        [Parameter(Mandatory = $true)][string]$Name
    )

    $closeButton = Wait-ForControlById -Parent $Window -ControlId $CloseControlId -Name "$Name dismiss"
    for ($index = 0; $index -lt 40; $index++) {
        $focus = Get-FocusedWindow -Window $Window
        if ($focus -eq $closeButton) {
            return $closeButton
        }
        $target = if ($focus -ne [IntPtr]::Zero) { $focus } else { $Window }
        if (-not [GhostFtpKeyboardNative]::PostMessage($target, $wmKeyDown, [IntPtr]$vkTab, [IntPtr]::Zero)) {
            throw "Could not post Tab to $Name."
        }
        [GhostFtpKeyboardNative]::PostMessage($target, $wmKeyUp, [IntPtr]$vkTab, [IntPtr]::Zero) | Out-Null
        Start-Sleep -Milliseconds 100
    }
    throw "Tab traversal did not reach the dismiss button in $Name."
}

function Verify-ModalKeyboardContract {
    param(
        [Parameter(Mandatory = $true)][IntPtr]$Main,
        [Parameter(Mandatory = $true)][System.Diagnostics.Process]$Process,
        [Parameter(Mandatory = $true)][int]$Command,
        [Parameter(Mandatory = $true)][int]$CloseControlId,
        [Parameter(Mandatory = $true)][string]$Title
    )

    $modal = Open-ModalWindow -Main $Main -Command $Command -ProcessId $Process.Id -Title $Title
    if ([GhostFtpKeyboardNative]::IsWindowEnabled($Main)) {
        throw "Main window remained enabled while $Title was open."
    }
    $closeButton = Focus-CloseWithTab -Window $modal -CloseControlId $CloseControlId -Name $Title
    $focus = Get-FocusedWindow -Window $modal
    if ($focus -ne $closeButton) {
        throw "Dismiss button did not retain keyboard focus in $Title."
    }
    if (-not [GhostFtpKeyboardNative]::PostMessage($closeButton, $wmKeyDown, [IntPtr]$vkReturn, [IntPtr]::Zero)) {
        throw "Could not post Enter to the focused dismiss button in $Title."
    }
    Wait-ForWindowClosed -Window $modal -Name $Title
    Assert-MainRestored -Main $Main -Process $Process -Context "$Title Enter close"

    $modal = Open-ModalWindow -Main $Main -Command $Command -ProcessId $Process.Id -Title $Title
    if (-not [GhostFtpKeyboardNative]::PostMessage($modal, $wmKeyDown, [IntPtr]$vkEscape, [IntPtr]::Zero)) {
        throw "Could not post Escape to $Title."
    }
    Wait-ForWindowClosed -Window $modal -Name $Title
    Assert-MainRestored -Main $Main -Process $Process -Context "$Title Escape close"

    Write-Host "MODAL_KEYBOARD_OK=$Title"
}

$exe = (Resolve-Path -LiteralPath $Executable).Path
$process = Start-Process -FilePath $exe -PassThru
try {
    $main = Wait-ForMainWindow -Process $process
    [GhostFtpKeyboardNative]::SetForegroundWindow($main) | Out-Null
    Start-Sleep -Milliseconds 500

    Verify-ModalKeyboardContract -Main $main -Process $process -Command $siteManagerCommand -CloseControlId $siteManagerCloseControlId -Title 'Site Manager'
    Verify-ModalKeyboardContract -Main $main -Process $process -Command $bookmarksCommand -CloseControlId $bookmarksCloseControlId -Title 'Bookmarks'

    if (-not [GhostFtpKeyboardNative]::PostMessage($main, $wmClose, [IntPtr]::Zero, [IntPtr]::Zero)) {
        throw 'Could not close the Ghost FTP main window.'
    }
    if (-not $process.WaitForExit(5000)) {
        throw 'Ghost FTP did not exit after the runtime keyboard test.'
    }
}
finally {
    if (-not $process.HasExited) {
        $process.Kill()
        $process.WaitForExit()
    }
}

Write-Host 'WINDOWS_MODAL_KEYBOARD_RUNTIME=PASS'
