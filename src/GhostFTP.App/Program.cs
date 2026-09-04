using GhostFTP.Services;
using GhostFTP.UI;
using System.Windows;
using System.Windows.Threading;

namespace GhostFTP;

public static class Program
{
    [STAThread]
    public static int Main()
    {
        var app = new Application
        {
            ShutdownMode = ShutdownMode.OnMainWindowClose
        };

        AppTheme configuredTheme = AppTheme.System;
        try
        {
            var paths = new AppPaths();
            var settings = new AppSettingsStore(paths.SettingsFile).LoadAsync().GetAwaiter().GetResult();
            configuredTheme = settings.Theme;
        }
        catch
        {
            // Safe defaults. No crash report or telemetry is emitted.
        }

        var dark = configuredTheme switch
        {
            AppTheme.Dark => true,
            AppTheme.Light => false,
            _ => Theme.IsSystemDark()
        };
        ThemeState.IsDark = dark;
        Theme.Apply(dark);

        app.DispatcherUnhandledException += OnDispatcherUnhandledException;
        var window = new MainWindow();
        app.MainWindow = window;
        return app.Run(window);
    }

    private static void OnDispatcherUnhandledException(object sender, DispatcherUnhandledExceptionEventArgs e)
    {
        MessageBox.Show(
            "GhostFTP encountered an unexpected local error and will close to protect session integrity.\n\n" + e.Exception.Message +
            "\n\nNo crash report, telemetry or diagnostic data was transmitted.",
            "GhostFTP",
            MessageBoxButton.OK,
            MessageBoxImage.Error);
        e.Handled = true;
        Application.Current.Shutdown(1);
    }
}
