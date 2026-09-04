using System.Runtime.InteropServices;
using System.Windows;
using System.Windows.Interop;

namespace GhostFTP.Setup.Services;

internal static class Win11Backdrop
{
    public static void Apply(Window window)
    {
        try
        {
            var hwnd = new WindowInteropHelper(window).Handle;
            if (hwnd == IntPtr.Zero) return;
            var dark = 1;
            var corner = 2;
            var backdrop = 2;
            _ = DwmSetWindowAttribute(hwnd, 20, ref dark, sizeof(int));
            _ = DwmSetWindowAttribute(hwnd, 33, ref corner, sizeof(int));
            if (OperatingSystem.IsWindowsVersionAtLeast(10, 0, 22000))
                _ = DwmSetWindowAttribute(hwnd, 38, ref backdrop, sizeof(int));
        }
        catch { }
    }

    [DllImport("dwmapi.dll")]
    private static extern int DwmSetWindowAttribute(IntPtr hwnd, int attribute, ref int value, int size);
}
