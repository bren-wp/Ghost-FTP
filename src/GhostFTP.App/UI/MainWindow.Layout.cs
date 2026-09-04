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

public sealed partial class MainWindow
{
    private UIElement BuildLayout()
    {
        var root = new Grid { Background = Brushes.Transparent, Margin = new Thickness(14) };
        root.ColumnDefinitions.Add(new ColumnDefinition { Width = new GridLength(260) });
        root.ColumnDefinitions.Add(new ColumnDefinition { Width = new GridLength(14) });
        root.ColumnDefinitions.Add(new ColumnDefinition { Width = new GridLength(1, GridUnitType.Star) });
        root.RowDefinitions.Add(new RowDefinition { Height = GridLength.Auto });
        root.RowDefinitions.Add(new RowDefinition { Height = new GridLength(14) });
        root.RowDefinitions.Add(new RowDefinition { Height = new GridLength(1, GridUnitType.Star) });
        root.RowDefinitions.Add(new RowDefinition { Height = new GridLength(14) });
        root.RowDefinitions.Add(new RowDefinition { Height = new GridLength(220) });

        var sidebar = BuildSidebar();
        Grid.SetColumn(sidebar, 0);
        Grid.SetRow(sidebar, 0);
        Grid.SetRowSpan(sidebar, 5);
        root.Children.Add(sidebar);

        var top = BuildTopBar();
        Grid.SetColumn(top, 2);
        Grid.SetRow(top, 0);
        root.Children.Add(top);

        var panes = BuildFilePanes();
        Grid.SetColumn(panes, 2);
        Grid.SetRow(panes, 2);
        root.Children.Add(panes);

        var transfers = BuildTransfers();
        Grid.SetColumn(transfers, 2);
        Grid.SetRow(transfers, 4);
        root.Children.Add(transfers);

        return root;
    }

    private Border BuildSidebar()
    {
        var stack = new DockPanel();

        var brand = new StackPanel { Margin = new Thickness(4, 2, 4, 18) };
        var logoRow = new StackPanel { Orientation = Orientation.Horizontal };
        var logo = new Border
        {
            Width = 42,
            Height = 42,
            CornerRadius = new CornerRadius(13),
            Background = new LinearGradientBrush((Color)ColorConverter.ConvertFromString("#7C5CFF"), (Color)ColorConverter.ConvertFromString("#35C6F4"), 45),
            Child = new TextBlock { Text = "G", Foreground = Brushes.White, FontSize = 22, FontWeight = FontWeights.Bold, HorizontalAlignment = HorizontalAlignment.Center, VerticalAlignment = VerticalAlignment.Center, FontFamily = Theme.DisplayFont }
        };
        logoRow.Children.Add(logo);
        var brandText = new StackPanel { Margin = new Thickness(10, 0, 0, 0), VerticalAlignment = VerticalAlignment.Center };
        brandText.Children.Add(Theme.Text("GhostFTP", 19, weight: FontWeights.SemiBold));
        brandText.Children.Add(Theme.Text("by Brendigo", 11, muted: true));
        logoRow.Children.Add(brandText);
        brand.Children.Add(logoRow);
        brand.Children.Add(new Border { Height = 12 });
        brand.Children.Add(Theme.Text("Private FTP/FTPS workspace", 11, muted: true));
        DockPanel.SetDock(brand, Dock.Top);
        stack.Children.Add(brand);

        var bottom = new StackPanel { Margin = new Thickness(4, 16, 4, 4) };
        var settings = Theme.Button("Settings");
        settings.HorizontalContentAlignment = HorizontalAlignment.Left;
        settings.Click += async (_, _) => await OpenSettingsAsync();
        var about = Theme.Button("About GhostFTP");
        about.HorizontalContentAlignment = HorizontalAlignment.Left;
        about.Margin = new Thickness(0, 8, 0, 0);
        about.Click += (_, _) => new AboutDialog(this).ShowDialog();
        bottom.Children.Add(settings);
        bottom.Children.Add(about);
        bottom.Children.Add(new Border { Height = 12 });
        bottom.Children.Add(Theme.Text("No telemetry · No tracking", 10.5, muted: true));
        DockPanel.SetDock(bottom, Dock.Bottom);
        stack.Children.Add(bottom);

        var servers = new DockPanel();
        var heading = new Grid { Margin = new Thickness(4, 0, 4, 10) };
        heading.ColumnDefinitions.Add(new ColumnDefinition { Width = new GridLength(1, GridUnitType.Star) });
        heading.ColumnDefinitions.Add(new ColumnDefinition { Width = GridLength.Auto });
        heading.Children.Add(Theme.Text("Saved servers", 12, muted: true, weight: FontWeights.SemiBold));
        var add = Theme.Button("+");
        add.Width = 36;
        add.Padding = new Thickness(0);
        add.Click += async (_, _) => await AddProfileAsync();
        Grid.SetColumn(add, 1);
        heading.Children.Add(add);
        DockPanel.SetDock(heading, Dock.Top);
        servers.Children.Add(heading);

        _profilesList.Background = Brushes.Transparent;
        _profilesList.BorderThickness = new Thickness(0);
        _profilesList.Foreground = Theme.R("Text");
        _profilesList.FontFamily = Theme.UiFont;
        _profilesList.DisplayMemberPath = nameof(ServerProfile.Name);
        _profilesList.ItemsSource = _profiles;
        servers.Children.Add(_profilesList);

        var serverButtons = new Grid { Margin = new Thickness(4, 10, 4, 0) };
        serverButtons.ColumnDefinitions.Add(new ColumnDefinition { Width = new GridLength(1, GridUnitType.Star) });
        serverButtons.ColumnDefinitions.Add(new ColumnDefinition { Width = new GridLength(8) });
        serverButtons.ColumnDefinitions.Add(new ColumnDefinition { Width = new GridLength(1, GridUnitType.Star) });
        var edit = Theme.Button("Edit");
        edit.Click += async (_, _) => await EditSelectedProfileAsync();
        var remove = Theme.Button("Remove");
        remove.Click += async (_, _) => await RemoveSelectedProfileAsync();
        Grid.SetColumn(edit, 0);
        Grid.SetColumn(remove, 2);
        serverButtons.Children.Add(edit);
        serverButtons.Children.Add(remove);
        DockPanel.SetDock(serverButtons, Dock.Bottom);
        servers.Children.Add(serverButtons);

        stack.Children.Add(servers);
        return Theme.Card(stack, new Thickness(14));
    }

    private Border BuildTopBar()
    {
        var outer = new Grid();
        outer.ColumnDefinitions.Add(new ColumnDefinition { Width = GridLength.Auto });
        outer.ColumnDefinitions.Add(new ColumnDefinition { Width = new GridLength(18) });
        outer.ColumnDefinitions.Add(new ColumnDefinition { Width = new GridLength(1, GridUnitType.Star) });

        var title = new StackPanel { VerticalAlignment = VerticalAlignment.Center };
        title.Children.Add(Theme.Text("File workspace", 25, weight: FontWeights.SemiBold));
        title.Children.Add(Theme.Text("Connect only to servers you trust. FTPS validates the server certificate by default.", 11.5, muted: true));
        Grid.SetColumn(title, 0);
        outer.Children.Add(title);

        var quick = new Grid { VerticalAlignment = VerticalAlignment.Center };
        quick.ColumnDefinitions.Add(new ColumnDefinition { Width = new GridLength(210) });
        quick.ColumnDefinitions.Add(new ColumnDefinition { Width = new GridLength(8) });
        quick.ColumnDefinitions.Add(new ColumnDefinition { Width = new GridLength(76) });
        quick.ColumnDefinitions.Add(new ColumnDefinition { Width = new GridLength(8) });
        quick.ColumnDefinitions.Add(new ColumnDefinition { Width = new GridLength(140) });
        quick.ColumnDefinitions.Add(new ColumnDefinition { Width = new GridLength(8) });
        quick.ColumnDefinitions.Add(new ColumnDefinition { Width = new GridLength(150) });
        quick.ColumnDefinitions.Add(new ColumnDefinition { Width = new GridLength(8) });
        quick.ColumnDefinitions.Add(new ColumnDefinition { Width = new GridLength(128) });
        quick.ColumnDefinitions.Add(new ColumnDefinition { Width = new GridLength(8) });
        quick.ColumnDefinitions.Add(new ColumnDefinition { Width = GridLength.Auto });
        quick.ColumnDefinitions.Add(new ColumnDefinition { Width = new GridLength(8) });
        quick.ColumnDefinitions.Add(new ColumnDefinition { Width = GridLength.Auto });

        AddAt(quick, _host, 0);
        AddAt(quick, _port, 2);
        AddAt(quick, _username, 4);
        AddAt(quick, _password, 6);
        AddAt(quick, _security, 8);
        AddAt(quick, _connectButton, 10);
        AddAt(quick, _disconnectButton, 12);
        _disconnectButton.IsEnabled = false;

        Grid.SetColumn(quick, 2);
        outer.Children.Add(quick);
        return Theme.Card(outer, new Thickness(16, 14, 16, 14));
    }

    private Grid BuildFilePanes()
    {
        var grid = new Grid();
        grid.ColumnDefinitions.Add(new ColumnDefinition { Width = new GridLength(1, GridUnitType.Star) });
        grid.ColumnDefinitions.Add(new ColumnDefinition { Width = new GridLength(14) });
        grid.ColumnDefinitions.Add(new ColumnDefinition { Width = new GridLength(1, GridUnitType.Star) });

        var local = BuildPane("Local", _localPathBox, _localFilter, _localList, isRemote: false);
        var remote = BuildPane("Remote", _remotePathBox, _remoteFilter, _remoteList, isRemote: true);
        Grid.SetColumn(local, 0);
        Grid.SetColumn(remote, 2);
        grid.Children.Add(local);
        grid.Children.Add(remote);
        return grid;
    }

    private Border BuildPane(string title, TextBox pathBox, TextBox filter, ListView list, bool isRemote)
    {
        var dock = new DockPanel();
        var header = new StackPanel();
        var titleRow = new Grid { Margin = new Thickness(0, 0, 0, 10) };
        titleRow.ColumnDefinitions.Add(new ColumnDefinition { Width = new GridLength(1, GridUnitType.Star) });
        titleRow.ColumnDefinitions.Add(new ColumnDefinition { Width = GridLength.Auto });
        titleRow.Children.Add(Theme.Text(title, 18, weight: FontWeights.SemiBold));
        if (isRemote)
        {
            _statusBadge.CornerRadius = new CornerRadius(10);
            _statusBadge.Padding = new Thickness(9, 4, 9, 4);
            _statusBadge.Background = Theme.R("Surface2");
            _statusBadge.Child = _statusText;
            Grid.SetColumn(_statusBadge, 1);
            titleRow.Children.Add(_statusBadge);
        }
        header.Children.Add(titleRow);

        pathBox.Margin = new Thickness(0, 0, 0, 10);
        header.Children.Add(pathBox);

        var toolbar = new StackPanel { Orientation = Orientation.Horizontal, Margin = new Thickness(0, 0, 0, 10) };
        if (isRemote)
        {
            toolbar.Children.Add(ToolButton("Up", async () => await RemoteUpAsync()));
            toolbar.Children.Add(ToolButton("Refresh", async () => await RefreshRemoteAsync()));
            toolbar.Children.Add(ToolButton("New folder", async () => await NewRemoteFolderAsync()));
            toolbar.Children.Add(ToolButton("Download", QueueDownloadSelected));
            toolbar.Children.Add(ToolButton("Rename", async () => await RenameRemoteSelectedAsync()));
            toolbar.Children.Add(ToolButton("Delete", async () => await DeleteRemoteSelectedAsync()));
        }
        else
        {
            toolbar.Children.Add(ToolButton("Up", LocalUp));
            toolbar.Children.Add(ToolButton("Refresh", RefreshLocal));
            toolbar.Children.Add(ToolButton("New folder", NewLocalFolder));
            toolbar.Children.Add(ToolButton("Upload", QueueUploadSelected));
            toolbar.Children.Add(ToolButton("Rename", RenameLocalSelected));
            toolbar.Children.Add(ToolButton("Delete", DeleteLocalSelected));
        }
        foreach (var button in toolbar.Children.OfType<Button>().Skip(1)) button.Margin = new Thickness(7, 0, 0, 0);
        header.Children.Add(toolbar);

        filter.Margin = new Thickness(0, 0, 0, 10);
        filter.ToolTip = "Filter current folder";
        header.Children.Add(filter);
        DockPanel.SetDock(header, Dock.Top);
        dock.Children.Add(header);

        list.Background = Brushes.Transparent;
        list.BorderThickness = new Thickness(0);
        list.Foreground = Theme.R("Text");
        list.FontFamily = Theme.UiFont;
        list.FontSize = 12.5;
        dock.Children.Add(list);
        return Theme.Card(dock, new Thickness(14));
    }

    private Border BuildTransfers()
    {
        var dock = new DockPanel();
        var header = new Grid { Margin = new Thickness(0, 0, 0, 10) };
        header.ColumnDefinitions.Add(new ColumnDefinition { Width = new GridLength(1, GridUnitType.Star) });
        header.ColumnDefinitions.Add(new ColumnDefinition { Width = GridLength.Auto });
        var left = new StackPanel { Orientation = Orientation.Horizontal, VerticalAlignment = VerticalAlignment.Center };
        left.Children.Add(Theme.Text("Transfers", 17, weight: FontWeights.SemiBold));
        _queueSummary.Margin = new Thickness(12, 0, 0, 0);
        left.Children.Add(_queueSummary);
        header.Children.Add(left);

        var actions = new StackPanel { Orientation = Orientation.Horizontal };
        var cancel = Theme.Button("Cancel selected");
        cancel.Click += (_, _) => CancelSelectedTransfer();
        var clear = Theme.Button("Clear finished");
        clear.Margin = new Thickness(8, 0, 0, 0);
        clear.Click += (_, _) => { _queue?.ClearFinished(); UpdateQueueSummary(); };
        actions.Children.Add(cancel);
        actions.Children.Add(clear);
        Grid.SetColumn(actions, 1);
        header.Children.Add(actions);
        DockPanel.SetDock(header, Dock.Top);
        dock.Children.Add(header);

        _queueList.Background = Brushes.Transparent;
        _queueList.BorderThickness = new Thickness(0);
        _queueList.Foreground = Theme.R("Text");
        _queueList.FontFamily = Theme.UiFont;
        dock.Children.Add(_queueList);
        return Theme.Card(dock, new Thickness(14));
    }

    private void ConfigureLists()
    {
        _localList.ItemsSource = _localItems;
        _remoteList.ItemsSource = _remoteItems;
        _localList.View = CreateFileGrid(local: true);
        _remoteList.View = CreateFileGrid(local: false);
        _queueList.View = CreateQueueGrid();
        _localList.SelectionMode = SelectionMode.Extended;
        _remoteList.SelectionMode = SelectionMode.Extended;

        _localList.ContextMenu = CreateContextMenu(
            ("Upload", (_, _) => QueueUploadSelected()),
            ("Rename", (_, _) => RenameLocalSelected()),
            ("Delete", (_, _) => DeleteLocalSelected()),
            ("Refresh", (_, _) => RefreshLocal()));
        _remoteList.ContextMenu = CreateContextMenu(
            ("Download", (_, _) => QueueDownloadSelected()),
            ("Rename", async (_, _) => await RenameRemoteSelectedAsync()),
            ("Delete", async (_, _) => await DeleteRemoteSelectedAsync()),
            ("Refresh", async (_, _) => await RefreshRemoteAsync()));
    }

    private void ConfigureEvents()
    {
        _connectButton.Click += async (_, _) => await ConnectAsync();
        _disconnectButton.Click += async (_, _) => await DisconnectAsync();
        _profilesList.SelectionChanged += (_, _) => ProfileSelected();
        _profilesList.MouseDoubleClick += async (_, _) => { if (_profilesList.SelectedItem is ServerProfile) await ConnectAsync(); };

        _localList.MouseDoubleClick += (_, _) => OpenLocalSelected();
        _remoteList.MouseDoubleClick += async (_, _) => await OpenRemoteSelectedAsync();
        _localPathBox.KeyDown += (_, e) => { if (e.Key == Key.Enter) NavigateLocalPathBox(); };
        _remotePathBox.KeyDown += async (_, e) => { if (e.Key == Key.Enter) await NavigateRemotePathBoxAsync(); };
        _localFilter.TextChanged += (_, _) => ApplyLocalFilter();
        _remoteFilter.TextChanged += (_, _) => ApplyRemoteFilter();

        _remoteList.AllowDrop = true;
        _remoteList.DragOver += (_, e) =>
        {
            e.Effects = e.Data.GetDataPresent(DataFormats.FileDrop) && IsConnected ? DragDropEffects.Copy : DragDropEffects.None;
            e.Handled = true;
        };
        _remoteList.Drop += (_, e) =>
        {
            if (!IsConnected || !e.Data.GetDataPresent(DataFormats.FileDrop)) return;
            if (e.Data.GetData(DataFormats.FileDrop) is not string[] paths) return;
            foreach (var path in paths)
            {
                if (File.Exists(path)) _queue?.EnqueueUpload(path, FtpListingParser.CombineRemote(_remotePath, Path.GetFileName(path)), false);
                else if (Directory.Exists(path)) _queue?.EnqueueUpload(path, FtpListingParser.CombineRemote(_remotePath, Path.GetFileName(path.TrimEnd(Path.DirectorySeparatorChar))), true);
            }
            UpdateQueueSummary();
        };
    }

}
