<#
.SYNOPSIS
    Launches a GoForms application, parks its window on the secondary monitor
    and screenshots it.

.DESCRIPTION
    The documentation screenshots have to be reproducible, and they must never
    take over the primary display - all of this runs on the secondary monitor
    by deliberate choice.

    Fyne has no API for placing a window on a particular display, so the
    window is moved with SetWindowPos after it appears. The secondary
    monitor's origin is read from the system rather than hardcoded: on this
    machine it sits at x = -1920 (to the *left* of the primary), and a
    hardcoded positive offset would have put every window off-screen.

.PARAMETER Exe
    The built application to run.

.PARAMETER Out
    Where to write the PNG.

.PARAMETER Width, Height
    Window size to force before capturing, so screenshots are consistent.

.PARAMETER SettleMs
    How long to let the app draw its first frame before capturing.

.PARAMETER Keys
    Optional SendKeys sequence delivered to the window before capture, for
    reaching a second form (e.g. "%c" for Alt+C).

.PARAMETER KeepOpen
    Leave the process running, for capturing several views of one app.

.PARAMETER Pid
    Attach to an already-running process instead of starting a new one.

.EXAMPLE
    .\capture.ps1 -Exe ..\..\..\GoFormsShowcase\showcase.exe -Out ..\images\showcase-main.png
#>
[CmdletBinding()]
param(
    [string]$Exe,
    [Parameter(Mandatory = $true)][string]$Out,
    [int]$Width = 1280,
    [int]$Height = 860,
    [int]$SettleMs = 3500,
    [string]$Keys = '',
    # Clicks to make before capturing, "x,y" relative to the window's visible
    # top-left corner; repeatable.
    [string[]]$Click = @(),
    # Pixels to shave off each edge. The frame bounds can round a pixel wide,
    # which shows up as a sliver of whatever is behind the window.
    [int]$Inset = 2,
    [switch]$KeepOpen,
    [int]$AttachPid = 0,
    # Arguments for the application, e.g. the form name for cmd/shot.
    [string[]]$AppArgs = @()
)

$ErrorActionPreference = 'Stop'
Add-Type -AssemblyName System.Windows.Forms, System.Drawing

if (-not ([System.Management.Automation.PSTypeName]'Win32Win').Type) {
    Add-Type @'
using System;
using System.Runtime.InteropServices;
public static class Win32Win {
    [DllImport("user32.dll")] public static extern bool SetWindowPos(IntPtr hWnd, IntPtr after, int x, int y, int cx, int cy, uint flags);
    [DllImport("user32.dll")] public static extern bool SetForegroundWindow(IntPtr hWnd);
    [DllImport("user32.dll")] public static extern bool ShowWindow(IntPtr hWnd, int cmd);
    [DllImport("user32.dll")] public static extern bool GetWindowRect(IntPtr hWnd, out RECT r);
    [DllImport("user32.dll")] public static extern bool SetCursorPos(int x, int y);
    [DllImport("user32.dll")] public static extern void mouse_event(uint flags, uint dx, uint dy, uint data, IntPtr extra);
    [DllImport("dwmapi.dll")] public static extern int DwmGetWindowAttribute(IntPtr hWnd, int attr, out RECT value, int size);
    [StructLayout(LayoutKind.Sequential)] public struct RECT { public int Left, Top, Right, Bottom; }

    // What the window looks like on screen. GetWindowRect reports the resize
    // border too, which is invisible on Windows 10 and later - about eight
    // pixels of whatever is behind the window, down each side and along the
    // bottom of every screenshot. DWMWA_EXTENDED_FRAME_BOUNDS is the rectangle
    // actually painted; it is unavailable on older systems, hence the fallback.
    public static RECT FrameBounds(IntPtr hWnd) {
        RECT r;
        if (DwmGetWindowAttribute(hWnd, 9, out r, Marshal.SizeOf(typeof(RECT))) == 0 &&
            r.Right > r.Left && r.Bottom > r.Top) {
            return r;
        }
        GetWindowRect(hWnd, out r);
        return r;
    }

    public static void ClickAt(int x, int y) {
        SetCursorPos(x, y);
        mouse_event(0x0002, 0, 0, 0, IntPtr.Zero);   // left down
        mouse_event(0x0004, 0, 0, 0, IntPtr.Zero);   // left up
    }
}
'@
}

# The screen that is not the primary one. Reading it from the system is the
# whole point: its origin can be negative.
$secondary = [System.Windows.Forms.Screen]::AllScreens | Where-Object { -not $_.Primary } | Select-Object -First 1
if (-not $secondary) {
    throw 'No secondary monitor found - everything here is meant to run on it.'
}
Write-Host "Secondary monitor: $($secondary.DeviceName) at $($secondary.Bounds)"

$proc = $null
if ($AttachPid -gt 0) {
    $proc = Get-Process -Id $AttachPid
} else {
    if (-not $Exe) { throw 'Give either -Exe or -AttachPid.' }
    $Exe = (Resolve-Path $Exe).Path
    if ($AppArgs.Count -gt 0) {
        $proc = Start-Process -FilePath $Exe -ArgumentList $AppArgs -PassThru
    } else {
        $proc = Start-Process -FilePath $Exe -PassThru
    }
    Start-Sleep -Milliseconds $SettleMs
    if ($proc.HasExited) {
        throw "$([System.IO.Path]::GetFileName($Exe)) exited immediately with code $($proc.ExitCode) - check its arguments."
    }
}

# MainWindowHandle is not set the instant the process starts.
$deadline = (Get-Date).AddSeconds(20)
while ($proc.MainWindowHandle -eq 0 -and (Get-Date) -lt $deadline) {
    Start-Sleep -Milliseconds 250
    $proc.Refresh()
}
if ($proc.MainWindowHandle -eq 0) {
    if (-not $KeepOpen) { $proc | Stop-Process -Force }
    throw "The application never showed a window (pid $($proc.Id))."
}
$h = $proc.MainWindowHandle

# Centre it on the secondary monitor, sized for a consistent screenshot.
#
# Moving it once is not enough. Fyne positions and sizes a window itself as it
# is shown, and some forms call CenterOnScreen() in their own setup, so a
# window moved too early is put straight back on the primary display. Move,
# check, and move again until it is really there - and fail loudly rather than
# quietly screenshotting the primary monitor, which is the one thing this is
# supposed never to touch.
$x = $secondary.Bounds.X + [int](($secondary.Bounds.Width - $Width) / 2)
$y = $secondary.Bounds.Y + [int](($secondary.Bounds.Height - $Height) / 2)

$onSecondary = $false
$r = New-Object Win32Win+RECT
for ($attempt = 1; $attempt -le 6; $attempt++) {
    [void][Win32Win]::ShowWindow($h, 9)      # SW_RESTORE
    [void][Win32Win]::SetWindowPos($h, [IntPtr]::Zero, $x, $y, $Width, $Height, 0x0040)  # SWP_SHOWWINDOW
    [void][Win32Win]::SetForegroundWindow($h)
    Start-Sleep -Milliseconds 700

    $proc.Refresh()
    if ($proc.MainWindowHandle -ne 0) { $h = $proc.MainWindowHandle }
    [void][Win32Win]::GetWindowRect($h, [ref]$r)

    # Judge by the window's centre: a shadow border can put an edge a pixel or
    # two outside the monitor without the window having moved displays.
    $cx = [int](($r.Left + $r.Right) / 2)
    $cy = [int](($r.Top + $r.Bottom) / 2)
    if ($cx -ge $secondary.Bounds.Left -and $cx -lt $secondary.Bounds.Right -and
        $cy -ge $secondary.Bounds.Top -and $cy -lt $secondary.Bounds.Bottom) {
        $onSecondary = $true
        break
    }
    Write-Host "  attempt ${attempt}: window centre ($cx,$cy) is not on the secondary monitor yet"
}
if (-not $onSecondary) {
    if (-not $KeepOpen) { $proc | Stop-Process -Force }
    throw "Could not keep the window on $($secondary.DeviceName); refusing to capture the primary display."
}
Start-Sleep -Milliseconds 500                 # let it repaint at the new size

if ($Keys) {
    [System.Windows.Forms.SendKeys]::SendWait($Keys)
    Start-Sleep -Milliseconds 1500
    $proc.Refresh()
    if ($proc.MainWindowHandle -ne 0) { $h = $proc.MainWindowHandle }
}

# Clicks that prepare what the screenshot is supposed to show: clearing the
# event log of whatever arrived while the window was taking focus, selecting
# the row a dialog is about. Coordinates are relative to the window's visible
# top-left corner, which is what you measure off a previous screenshot.
if ($Click.Count -gt 0) {
    $fr = [Win32Win]::FrameBounds($h)
    foreach ($spot in $Click) {
        $parts = $spot -split '\s*,\s*'
        if ($parts.Count -ne 2) { throw "Click wants 'x,y', got '$spot'." }
        [Win32Win]::ClickAt($fr.Left + [int]$parts[0], $fr.Top + [int]$parts[1])
        Start-Sleep -Milliseconds 400
    }
    Start-Sleep -Milliseconds 600
}

# Capture exactly the window, not the whole monitor: the surrounding desktop
# is nobody's business and changes between runs.
$r = [Win32Win]::FrameBounds($h)
$r.Left += $Inset; $r.Top += $Inset; $r.Right -= $Inset; $r.Bottom -= $Inset
$w = $r.Right - $r.Left
$ht = $r.Bottom - $r.Top
if ($w -le 0 -or $ht -le 0) { throw "Window rect is empty: $($r.Left),$($r.Top),$($r.Right),$($r.Bottom)" }

$bmp = New-Object System.Drawing.Bitmap $w, $ht
$g = [System.Drawing.Graphics]::FromImage($bmp)
$g.CopyFromScreen($r.Left, $r.Top, 0, 0, (New-Object System.Drawing.Size $w, $ht))
$g.Dispose()

$outDir = Split-Path -Parent $Out
if ($outDir -and -not (Test-Path $outDir)) { New-Item -ItemType Directory -Force $outDir | Out-Null }
# Bitmap.Save resolves relative paths against the process working directory,
# not PowerShell's, so make it absolute here - and only when it is relative,
# since joining an absolute path onto the current directory produces a path
# that cannot exist.
$outPath = if ([System.IO.Path]::IsPathRooted($Out)) {
    [System.IO.Path]::GetFullPath($Out)
} else {
    [System.IO.Path]::GetFullPath((Join-Path (Get-Location).Path $Out))
}
$bmp.Save($outPath, [System.Drawing.Imaging.ImageFormat]::Png)
$bmp.Dispose()
$Out = $outPath

Write-Host "Captured $w x $ht from ($($r.Left),$($r.Top)) -> $Out"

if (-not $KeepOpen) {
    $proc | Stop-Process -Force
    Write-Host "Closed pid $($proc.Id)"
} else {
    Write-Host "Left pid $($proc.Id) running"
}
