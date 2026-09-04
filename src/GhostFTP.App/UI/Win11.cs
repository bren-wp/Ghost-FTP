using System.Runtime.InteropServices;
using System.Windows;
using System.Windows.Interop;

namespace GhostFTP.UI;

internal static class Win11
{
    private const int DwmwaUseImmersiveDarkMode = 20;
    private const int DwmwaWindowCornerPreference = 33;
    private const int DwmwaSystemBackdropType = 38;

    public static void Apply(Window window, bool dark)
    {
        if (!OperatingSystem.IsWindows()) return;
        try
        {
            var hwnd = new WindowInteropHelper(window).Handle;
            if (hwnd == IntPtr.Zero) return;
            var darkValue = dark ? 1 : 0;
            _ = DwmSetWindowAttribute(hwnd, DwmwaUseImmersiveDarkMode, ref darkValue, sizeof(int));
            var corner = 2; // DWMWCP_ROUND
            _ = DwmSetWindowAttribute(hwnd, DwmwaWindowCornerPreference, ref corner, sizeof(int));
            if (OperatingSystem.IsWindowsVersionAtLeast(10, 0, 22000) && !SystemParameters.HighContrast)
            {
                var backdrop = 2; // DWMSBT_MAINWINDOW (Mica)
                _ = DwmSetWindowAttribute(hwnd, DwmwaSystemBackdropType, ref backdrop, sizeof(int));
            }
        }
        catch
        {
            // Visual enhancement only. Never block application startup.
        }
    }

    [DllImport("dwmapi.dll")]
    private static extern int DwmSetWindowAttribute(IntPtr hwnd, int attribute, ref int value, int size);
}
