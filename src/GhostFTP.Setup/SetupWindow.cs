using GhostFTP.Setup.Services;
using System.Reflection;
using System.Windows;
using System.Windows.Controls;
using System.Windows.Media;

namespace GhostFTP.Setup;

public sealed class SetupWindow : Window
{
    private readonly InstallerService _installer = new();
    private readonly bool _uninstallMode;
    private readonly CheckBox _desktopShortcut;
    private readonly CheckBox _removeData;
    private readonly Button _primary;
    private readonly TextBlock _status;

    public SetupWindow(bool uninstallMode)
    {
        _uninstallMode = uninstallMode;
        Title = uninstallMode ? "Uninstall GhostFTP" : "GhostFTP Setup";
        Width = 610;
        Height = 520;
        MinWidth = 610;
        MinHeight = 520;
        MaxWidth = 610;
        MaxHeight = 520;
        ResizeMode = ResizeMode.NoResize;
        WindowStartupLocation = WindowStartupLocation.CenterScreen;
        Background = Brush("#0B0D12");
        Foreground = Brush("#F5F7FB");
        FontFamily = new FontFamily("Segoe UI Variable Text, Segoe UI");
        SourceInitialized += (_, _) => Win11Backdrop.Apply(this);

        var root = new Border
        {
            Margin = new Thickness(18),
            Padding = new Thickness(28),
            Background = Brush("#11151D"),
            BorderBrush = Brush("#2A3242"),
            BorderThickness = new Thickness(1),
            CornerRadius = new CornerRadius(18)
        };
        var stack = new StackPanel();
        root.Child = stack;

        var logo = new Border
        {
            Width = 62,
            Height = 62,
            HorizontalAlignment = HorizontalAlignment.Left,
            CornerRadius = new CornerRadius(18),
            Background = new LinearGradientBrush((Color)ColorConverter.ConvertFromString("#7C5CFF"), (Color)ColorConverter.ConvertFromString("#35C6F4"), 45),
            Child = new TextBlock { Text = "G", FontSize = 31, FontWeight = FontWeights.Bold, Foreground = Brushes.White, HorizontalAlignment = HorizontalAlignment.Center, VerticalAlignment = VerticalAlignment.Center }
        };
        stack.Children.Add(logo);
        stack.Children.Add(Spacer(14));
        var version = typeof(SetupWindow).Assembly.GetName().Version?.ToString(3) ?? "1.1.0";
        stack.Children.Add(Text(uninstallMode ? "Uninstall GhostFTP" : $"GhostFTP {version}", 28, FontWeights.SemiBold));
        stack.Children.Add(Text(uninstallMode ? "Remove the application from this Windows account." : "Premium private FTP/FTPS client for Windows 11", 13, FontWeights.Normal, muted: true));
        stack.Children.Add(Spacer(22));

        if (!uninstallMode)
        {
            stack.Children.Add(Text("Install location", 12, FontWeights.SemiBold, muted: true));
            stack.Children.Add(Spacer(6));
            var path = new Border
            {
                Padding = new Thickness(12, 10, 12, 10),
                Background = Brush("#171C26"),
                BorderBrush = Brush("#2A3242"),
                BorderThickness = new Thickness(1),
                CornerRadius = new CornerRadius(10),
                Child = Text(_installer.InstallDirectory, 12, FontWeights.Normal)
            };
            stack.Children.Add(path);
            stack.Children.Add(Spacer(16));
            _desktopShortcut = Check("Create desktop shortcut", true);
            _removeData = Check("", false);
            stack.Children.Add(_desktopShortcut);
            stack.Children.Add(Spacer(18));
            stack.Children.Add(Text("Privacy", 12, FontWeights.SemiBold));
            stack.Children.Add(Text("GhostFTP contains no telemetry, analytics, ads, tracking SDK or automatic update checker. The app makes network connections only when you explicitly connect to an FTP/FTPS server or open a website link.", 12, FontWeights.Normal, muted: true));
        }
        else
        {
            _desktopShortcut = Check("", false);
            _removeData = Check("Also remove local GhostFTP settings and saved server profiles", false);
            stack.Children.Add(_removeData);
            stack.Children.Add(Spacer(16));
            stack.Children.Add(Text("Saved passwords, when enabled, are protected by Windows DPAPI for the current user. Leaving local data unchecked preserves your profiles for a later reinstall.", 12, FontWeights.Normal, muted: true));
        }

        stack.Children.Add(Spacer(18));
        _status = Text(uninstallMode ? "Ready to uninstall." : _installer.IsInstalled ? "An existing installation will be updated safely." : "Ready to install.", 12, FontWeights.Normal, muted: true);
        stack.Children.Add(_status);
        stack.Children.Add(Spacer(16));

        var footer = new Grid();
        footer.ColumnDefinitions.Add(new ColumnDefinition { Width = new GridLength(1, GridUnitType.Star) });
        footer.ColumnDefinitions.Add(new ColumnDefinition { Width = GridLength.Auto });
        footer.ColumnDefinitions.Add(new ColumnDefinition { Width = new GridLength(8) });
        footer.ColumnDefinitions.Add(new ColumnDefinition { Width = GridLength.Auto });
        var cancel = Button("Cancel", false);
        cancel.Click += (_, _) => Close();
        Grid.SetColumn(cancel, 1);
        footer.Children.Add(cancel);
        _primary = Button(uninstallMode ? "Uninstall" : _installer.IsInstalled ? "Update GhostFTP" : "Install GhostFTP", true);
        _primary.Click += async (_, _) => await ExecuteAsync();
        Grid.SetColumn(_primary, 3);
        footer.Children.Add(_primary);
        stack.Children.Add(footer);

        Content = root;
    }

    private async Task ExecuteAsync()
    {
        _primary.IsEnabled = false;
        try
        {
            if (_uninstallMode)
            {
                _status.Text = "Removing GhostFTP…";
                await _installer.UninstallAsync(_removeData.IsChecked == true, CancellationToken.None);
                _status.Text = "GhostFTP has been removed.";
                MessageBox.Show(this, "GhostFTP was uninstalled successfully.", "GhostFTP", MessageBoxButton.OK, MessageBoxImage.Information);
                Close();
            }
            else
            {
                _status.Text = "Installing GhostFTP…";
                await _installer.InstallAsync(_desktopShortcut.IsChecked == true, CancellationToken.None);
                _status.Text = "Installation complete.";
                var launch = MessageBox.Show(this, "GhostFTP was installed successfully. Launch it now?", "GhostFTP", MessageBoxButton.YesNo, MessageBoxImage.Information);
                if (launch == MessageBoxResult.Yes) _installer.LaunchApp();
                Close();
            }
        }
        catch (Exception ex)
        {
            _status.Text = "The operation could not be completed.";
            MessageBox.Show(this, ex.Message, "GhostFTP Setup", MessageBoxButton.OK, MessageBoxImage.Error);
            _primary.IsEnabled = true;
        }
    }

    private static Button Button(string text, bool primary) => new()
    {
        Content = text,
        Padding = new Thickness(16, 9, 16, 9),
        MinHeight = 38,
        FontWeight = FontWeights.SemiBold,
        Foreground = primary ? Brushes.White : Brush("#F5F7FB"),
        Background = Brush(primary ? "#7C5CFF" : "#171C26"),
        BorderBrush = Brush(primary ? "#7C5CFF" : "#2A3242"),
        BorderThickness = new Thickness(1),
        Cursor = System.Windows.Input.Cursors.Hand
    };

    private static CheckBox Check(string text, bool selected) => new()
    {
        Content = text,
        IsChecked = selected,
        Foreground = Brush("#F5F7FB"),
        FontSize = 12.5,
        VerticalContentAlignment = VerticalAlignment.Center
    };

    private static TextBlock Text(string text, double size, FontWeight weight, bool muted = false) => new()
    {
        Text = text,
        FontSize = size,
        FontWeight = weight,
        Foreground = Brush(muted ? "#9BA6B7" : "#F5F7FB"),
        TextWrapping = TextWrapping.Wrap
    };

    private static Border Spacer(double height) => new() { Height = height };

    private static SolidColorBrush Brush(string value)
    {
        var brush = new SolidColorBrush((Color)ColorConverter.ConvertFromString(value));
        brush.Freeze();
        return brush;
    }
}
