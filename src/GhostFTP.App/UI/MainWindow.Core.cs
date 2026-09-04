using GhostFTP.Core.Models;
using GhostFTP.Core.Protocol;
using GhostFTP.Core.Services;
using GhostFTP.Services;
using System.Collections.ObjectModel;
using System.ComponentModel;
using System.Windows;
using System.Windows.Controls;
using System.Windows.Input;
using System.Windows.Media;


namespace GhostFTP.UI;

public sealed partial class MainWindow : Window
{
    private sealed class LocalItem
    {
        public required string Name { get; init; }
        public required string FullPath { get; init; }
        public required bool IsDirectory { get; init; }
        public required long Size { get; init; }
        public required DateTimeOffset Modified { get; init; }
        public string Type => IsDirectory ? "Folder" : "File";
        public string SizeText => IsDirectory ? "" : FormatBytes(Size);
        public string ModifiedText => Modified.LocalDateTime.ToString("yyyy-MM-dd HH:mm");
    }

    private sealed class RemoteItem
    {
        public required FtpEntry Entry { get; init; }
        public string Name => Entry.Name;
        public string FullPath => Entry.FullPath;
        public bool IsDirectory => Entry.IsDirectory;
        public string Type => Entry.Type;
        public string SizeText => Entry.IsDirectory ? "" : FormatBytes(Entry.Size);
        public string ModifiedText => Entry.ModifiedUtc?.LocalDateTime.ToString("yyyy-MM-dd HH:mm") ?? "";
        public string Permissions => Entry.Permissions ?? "";
    }

    private readonly AppPaths _paths = new();
    private readonly DpapiSecretProtector _secrets = new();
    private readonly ObservableCollection<ServerProfile> _profiles = [];
    private readonly ObservableCollection<LocalItem> _localItems = [];
    private readonly ObservableCollection<RemoteItem> _remoteItems = [];
    private readonly HashSet<Guid> _completedHandled = [];
    private ProfileStore? _profileStore;
    private AppSettingsStore? _settingsStore;
    private AppSettings _settings = new();
    private TransferQueueService? _queue;
    private IFtpSession? _session;
    private FtpConnectionOptions? _activeOptions;
    private CancellationTokenSource? _connectionCts;
    private string _localPath = Environment.GetFolderPath(Environment.SpecialFolder.UserProfile);
    private string _remotePath = "/";
    private bool _busy;
    private bool _allowClose;

    private readonly ListBox _profilesList = new();
    private readonly TextBox _host = Theme.TextBox();
    private readonly TextBox _port = Theme.TextBox("21");
    private readonly TextBox _username = Theme.TextBox();
    private readonly PasswordBox _password = Theme.PasswordBox();
    private readonly ComboBox _security = Theme.ComboBox();
    private readonly Button _connectButton = Theme.Button("Connect", primary: true);
    private readonly Button _disconnectButton = Theme.Button("Disconnect");
    private readonly TextBox _localPathBox = Theme.TextBox();
    private readonly TextBox _remotePathBox = Theme.TextBox("/");
    private readonly TextBox _localFilter = Theme.TextBox();
    private readonly TextBox _remoteFilter = Theme.TextBox();
    private readonly ListView _localList = new();
    private readonly ListView _remoteList = new();
    private readonly ListView _queueList = new();
    private readonly Border _statusBadge = new();
    private readonly TextBlock _statusText = Theme.Text("Offline", 12, muted: true, weight: FontWeights.SemiBold);
    private readonly TextBlock _queueSummary = Theme.Text("No transfers", 12, muted: true);

    private List<LocalItem> _localAll = [];
    private List<RemoteItem> _remoteAll = [];

    public MainWindow()
    {
        Title = "GhostFTP";
        Width = 1480;
        Height = 900;
        MinWidth = 1080;
        MinHeight = 700;
        WindowStartupLocation = WindowStartupLocation.CenterScreen;
        Background = Theme.R("Bg");
        Foreground = Theme.R("Text");
        FontFamily = Theme.UiFont;
        UseLayoutRounding = true;
        SnapsToDevicePixels = true;

        _security.ItemsSource = new[] { "FTP", "FTPS Explicit", "FTPS Implicit" };
        _security.SelectedIndex = 1;

        Content = BuildLayout();
        ConfigureLists();
        ConfigureEvents();

        SourceInitialized += (_, _) => Win11.Apply(this, ThemeState.IsDark);
        Loaded += OnLoadedAsync;
        Closing += OnClosingAsync;
    }

}
