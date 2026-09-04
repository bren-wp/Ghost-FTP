using System.Windows;

namespace GhostFTP.Setup;

public static class Program
{
    [STAThread]
    public static int Main(string[] args)
    {
        var executableName = Path.GetFileNameWithoutExtension(Environment.ProcessPath ?? string.Empty);
        var uninstall = args.Any(x => string.Equals(x, "--uninstall", StringComparison.OrdinalIgnoreCase))
            || executableName.Contains("Uninstall", StringComparison.OrdinalIgnoreCase);
        var app = new Application { ShutdownMode = ShutdownMode.OnMainWindowClose };
        var window = new SetupWindow(uninstall);
        app.MainWindow = window;
        return app.Run(window);
    }
}
