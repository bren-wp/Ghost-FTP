using GhostFTP.Core.Models;
using GhostFTP.Services;
using System.Diagnostics;
using System.Reflection;
using System.Windows;
using System.Windows.Controls;
using System.Windows.Media;

namespace GhostFTP.UI;

internal abstract class GhostDialog : Window
{
    protected GhostDialog(Window owner, string title, double width = 520, double height = 420)
    {
        Owner = owner;
        Title = title;
        Width = width;
        Height = height;
        MinWidth = 420;
        MinHeight = 260;
        WindowStartupLocation = WindowStartupLocation.CenterOwner;
        ResizeMode = ResizeMode.NoResize;
        Background = Theme.R("Bg");
        Foreground = Theme.R("Text");
        FontFamily = Theme.UiFont;
        ShowInTaskbar = false;
        SourceInitialized += (_, _) => Win11.Apply(this, ThemeState.IsDark);
    }

    protected static StackPanel Field(string label, UIElement input)
    {
        var stack = new StackPanel { Margin = new Thickness(0, 0, 0, 12) };
        stack.Children.Add(Theme.Text(label, 12, muted: true, weight: FontWeights.SemiBold));
        if (input is FrameworkElement element) element.Margin = new Thickness(0, 6, 0, 0);
        stack.Children.Add(input);
        return stack;
    }

    protected static Grid Footer(Button primary, Button? secondary = null)
    {
        var grid = new Grid { Margin = new Thickness(0, 18, 0, 0) };
        grid.ColumnDefinitions.Add(new ColumnDefinition { Width = new GridLength(1, GridUnitType.Star) });
        grid.ColumnDefinitions.Add(new ColumnDefinition { Width = GridLength.Auto });
        if (secondary is not null)
        {
            secondary.Margin = new Thickness(0, 0, 8, 0);
            Grid.SetColumn(secondary, 1);
            grid.Children.Add(secondary);
            grid.ColumnDefinitions.Add(new ColumnDefinition { Width = GridLength.Auto });
            Grid.SetColumn(primary, 2);
        }
        else Grid.SetColumn(primary, 1);
        grid.Children.Add(primary);
        return grid;
    }
}

internal static class ThemeState
{
    public static bool IsDark { get; set; } = true;
}

internal sealed class ProfileDialog : GhostDialog
{
    private readonly TextBox _name;
    private readonly TextBox _host;
    private readonly TextBox _port;
    private readonly TextBox _username;
    private readonly PasswordBox _password;
    private readonly ComboBox _security;
    private readonly TextBox _initialPath;
    private readonly CheckBox _remember;
    private readonly ServerProfile _profile;

    public string Password => _password.Password;
    public ServerProfile Result => _profile;

    public ProfileDialog(Window owner, ServerProfile profile, string existingPassword, bool isNew = false) : base(owner, isNew ? "Add server" : "Edit server", 540, 650)
    {
        _profile = profile.Clone();
        _name = Theme.TextBox(_profile.Name);
        _host = Theme.TextBox(_profile.Host);
        _port = Theme.TextBox(_profile.Port.ToString());
        _username = Theme.TextBox(_profile.Username);
        _password = Theme.PasswordBox();
        _password.Password = existingPassword;
        _security = Theme.ComboBox();
        _security.ItemsSource = new[] { "FTP (plain)", "FTPS explicit TLS", "FTPS implicit TLS" };
        _security.SelectedIndex = (int)_profile.Security;
        _initialPath = Theme.TextBox(_profile.InitialPath);
        _remember = new CheckBox
        {
            Content = "Remember password on this Windows account (DPAPI encrypted)",
            IsChecked = _profile.RememberPassword,
            Foreground = Theme.R("Text"),
            FontFamily = Theme.UiFont,
            Margin = new Thickness(0, 2, 0, 4)
        };

        var body = new StackPanel();
        body.Children.Add(Theme.Text("Server profile", 24, weight: FontWeights.SemiBold));
        body.Children.Add(Theme.Text("Credentials stay on this device. GhostFTP never transmits them anywhere except to the FTP server you choose.", 12, muted: true));
        body.Children.Add(new Border { Height = 16 });
        body.Children.Add(Field("Profile name", _name));
        body.Children.Add(Field("Host", _host));

        var row = new Grid();
        row.ColumnDefinitions.Add(new ColumnDefinition { Width = new GridLength(130) });
        row.ColumnDefinitions.Add(new ColumnDefinition { Width = new GridLength(12) });
        row.ColumnDefinitions.Add(new ColumnDefinition { Width = new GridLength(1, GridUnitType.Star) });
        var portField = Field("Port", _port);
        var securityField = Field("Security", _security);
        Grid.SetColumn(portField, 0);
        Grid.SetColumn(securityField, 2);
        row.Children.Add(portField);
        row.Children.Add(securityField);
        body.Children.Add(row);
        body.Children.Add(Field("Username", _username));
        body.Children.Add(Field("Password", _password));
        body.Children.Add(_remember);
        body.Children.Add(Field("Initial remote path", _initialPath));

        var save = Theme.Button("Save server", primary: true);
        save.Click += (_, _) => Save();
        var cancel = Theme.Button("Cancel");
        cancel.Click += (_, _) => Close();
        body.Children.Add(Footer(save, cancel));

        Content = Theme.Card(body, new Thickness(24));
        Padding = new Thickness(16);
    }

    private void Save()
    {
        try
        {
            if (string.IsNullOrWhiteSpace(_name.Text)) throw new InvalidOperationException("Profile name is required.");
            if (string.IsNullOrWhiteSpace(_host.Text)) throw new InvalidOperationException("Host is required.");
            if (!int.TryParse(_port.Text, out var port) || port is < 1 or > 65535) throw new InvalidOperationException("Port must be between 1 and 65535.");
            _profile.Name = _name.Text.Trim();
            _profile.Host = _host.Text.Trim();
            _profile.Port = port;
            _profile.Username = _username.Text.Trim();
            _profile.Security = (FtpSecurityMode)Math.Max(0, _security.SelectedIndex);
            _profile.InitialPath = string.IsNullOrWhiteSpace(_initialPath.Text) ? "/" : _initialPath.Text.Trim();
            _profile.RememberPassword = _remember.IsChecked == true;
            _profile.IsDemo = false;
            DialogResult = true;
        }
        catch (Exception ex)
        {
            MessageBox.Show(this, ex.Message, "GhostFTP", MessageBoxButton.OK, MessageBoxImage.Warning);
        }
    }
}

internal sealed class TextPromptDialog : GhostDialog
{
    private readonly TextBox _input;
    public string Value => _input.Text.Trim();

    public TextPromptDialog(Window owner, string title, string label, string value = "") : base(owner, title, 470, 260)
    {
        _input = Theme.TextBox(value);
        var body = new StackPanel();
        body.Children.Add(Theme.Text(title, 22, weight: FontWeights.SemiBold));
        body.Children.Add(new Border { Height = 14 });
        body.Children.Add(Field(label, _input));
        var ok = Theme.Button("Continue", primary: true);
        ok.Click += (_, _) => { if (!string.IsNullOrWhiteSpace(_input.Text)) DialogResult = true; };
        var cancel = Theme.Button("Cancel");
        cancel.Click += (_, _) => Close();
        body.Children.Add(Footer(ok, cancel));
        Content = Theme.Card(body, new Thickness(24));
        Padding = new Thickness(16);
        Loaded += (_, _) => { _input.Focus(); _input.SelectAll(); };
    }
}

internal sealed class SettingsDialog : GhostDialog
{
    private readonly ComboBox _theme;
    private readonly CheckBox _confirmDeletes;
    public AppTheme SelectedTheme => (AppTheme)Math.Max(0, _theme.SelectedIndex);
    public bool ConfirmDeletes => _confirmDeletes.IsChecked == true;

    public SettingsDialog(Window owner, AppSettings settings) : base(owner, "Settings", 500, 360)
    {
        _theme = Theme.ComboBox();
        _theme.ItemsSource = new[] { "Use Windows setting", "Dark", "Light" };
        _theme.SelectedIndex = (int)settings.Theme;
        _confirmDeletes = new CheckBox
        {
            Content = "Ask for confirmation before deleting local or remote files and folders",
            IsChecked = settings.ConfirmDeletes,
            Foreground = Theme.R("Text"),
            FontFamily = Theme.UiFont,
            Margin = new Thickness(0, 6, 0, 8)
        };

        var body = new StackPanel();
        body.Children.Add(Theme.Text("Settings", 24, weight: FontWeights.SemiBold));
        body.Children.Add(Theme.Text("Privacy-first defaults. No telemetry, analytics, automatic update checks or background network traffic.", 12, muted: true));
        body.Children.Add(new Border { Height = 18 });
        body.Children.Add(Field("Appearance", _theme));
        body.Children.Add(_confirmDeletes);
        var save = Theme.Button("Save", primary: true);
        save.Click += (_, _) => DialogResult = true;
        var cancel = Theme.Button("Cancel");
        cancel.Click += (_, _) => Close();
        body.Children.Add(Footer(save, cancel));
        Content = Theme.Card(body, new Thickness(24));
        Padding = new Thickness(16);
    }
}

internal sealed class AboutDialog : GhostDialog
{
    public AboutDialog(Window owner) : base(owner, "About GhostFTP", 540, 430)
    {
        var body = new StackPanel();
        var logo = new Border
        {
            Width = 56,
            Height = 56,
            CornerRadius = new CornerRadius(16),
            Background = new LinearGradientBrush((Color)ColorConverter.ConvertFromString("#7C5CFF"), (Color)ColorConverter.ConvertFromString("#35C6F4"), 45),
            Child = new TextBlock { Text = "G", FontSize = 28, FontWeight = FontWeights.Bold, Foreground = Brushes.White, HorizontalAlignment = HorizontalAlignment.Center, VerticalAlignment = VerticalAlignment.Center, FontFamily = Theme.DisplayFont }
        };
        body.Children.Add(logo);
        body.Children.Add(new Border { Height = 14 });
        body.Children.Add(Theme.Text("GhostFTP", 28, weight: FontWeights.SemiBold));
        var version = typeof(AboutDialog).Assembly.GetName().Version?.ToString(3) ?? "1.1.0";
        body.Children.Add(Theme.Text($"Version {version} · Premium Windows FTP/FTPS client", 12, muted: true));
        body.Children.Add(new Border { Height = 18 });
        body.Children.Add(Theme.Text("Author: Brendigo", 13, weight: FontWeights.SemiBold));
        body.Children.Add(Theme.Text("ghostftp.com", 13));
        body.Children.Add(Theme.Text("brendigo.com", 13));
        body.Children.Add(new Border { Height = 16 });
        body.Children.Add(Theme.Text("Privacy", 13, weight: FontWeights.SemiBold));
        body.Children.Add(Theme.Text("GhostFTP contains no telemetry, analytics, tracking SDK, ads or automatic update checker. Network traffic is created only by FTP/FTPS actions you initiate or when you manually open a website link.", 12, muted: true));

        var buttons = new StackPanel { Orientation = Orientation.Horizontal, Margin = new Thickness(0, 20, 0, 0) };
        var web = Theme.Button("ghostftp.com", primary: true);
        web.Click += (_, _) => OpenUrl("https://ghostftp.com");
        var author = Theme.Button("Brendigo");
        author.Margin = new Thickness(8, 0, 0, 0);
        author.Click += (_, _) => OpenUrl("https://brendigo.com");
        buttons.Children.Add(web);
        buttons.Children.Add(author);
        body.Children.Add(buttons);

        Content = Theme.Card(body, new Thickness(24));
        Padding = new Thickness(16);
    }

    private static void OpenUrl(string url)
    {
        try { Process.Start(new ProcessStartInfo(url) { UseShellExecute = true }); } catch { }
    }
}
