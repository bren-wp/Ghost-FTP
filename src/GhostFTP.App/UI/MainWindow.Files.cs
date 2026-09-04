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
    private void RefreshLocal()
    {
        try
        {
            var directory = new DirectoryInfo(_localPath);
            if (!directory.Exists) throw new DirectoryNotFoundException(_localPath);
            var options = new EnumerationOptions { IgnoreInaccessible = true, RecurseSubdirectories = false, ReturnSpecialDirectories = false };
            _localAll = directory.EnumerateFileSystemInfos("*", options)
                .Select(info => new LocalItem
                {
                    Name = info.Name,
                    FullPath = info.FullName,
                    IsDirectory = info is DirectoryInfo,
                    Size = info is FileInfo file ? file.Length : 0,
                    Modified = new DateTimeOffset(info.LastWriteTimeUtc, TimeSpan.Zero)
                })
                .OrderByDescending(x => x.IsDirectory)
                .ThenBy(x => x.Name, StringComparer.OrdinalIgnoreCase)
                .ToList();
            ApplyLocalFilter();
            _localPathBox.Text = _localPath;
        }
        catch (Exception ex)
        {
            ShowOperationError("Could not open the local folder.", ex);
        }
    }

    private void ApplyLocalFilter()
    {
        var filter = _localFilter.Text.Trim();
        _localItems.Clear();
        foreach (var item in _localAll.Where(x => filter.Length == 0 || x.Name.Contains(filter, StringComparison.OrdinalIgnoreCase))) _localItems.Add(item);
    }

    private void ApplyRemoteFilter()
    {
        var filter = _remoteFilter.Text.Trim();
        _remoteItems.Clear();
        foreach (var item in _remoteAll.Where(x => filter.Length == 0 || x.Name.Contains(filter, StringComparison.OrdinalIgnoreCase))) _remoteItems.Add(item);
    }

    private void OpenLocalSelected()
    {
        if (_localList.SelectedItem is not LocalItem item) return;
        if (item.IsDirectory)
        {
            _localPath = item.FullPath;
            RefreshLocal();
        }
    }

    private async Task OpenRemoteSelectedAsync()
    {
        if (_remoteList.SelectedItem is not RemoteItem item || !IsConnected) return;
        if (item.IsDirectory)
        {
            _remotePath = item.FullPath;
            await RefreshRemoteAsync();
        }
        else
        {
            QueueDownloadSelected();
        }
    }

    private void LocalUp()
    {
        var parent = Directory.GetParent(_localPath);
        if (parent is null) return;
        _localPath = parent.FullName;
        RefreshLocal();
    }

    private async Task RemoteUpAsync()
    {
        if (!IsConnected) return;
        _remotePath = FtpListingParser.ParentRemote(_remotePath);
        await RefreshRemoteAsync();
    }

    private void NavigateLocalPathBox()
    {
        try
        {
            var path = Path.GetFullPath(Environment.ExpandEnvironmentVariables(_localPathBox.Text.Trim()));
            if (!Directory.Exists(path)) throw new DirectoryNotFoundException(path);
            _localPath = path;
            RefreshLocal();
        }
        catch (Exception ex) { ShowOperationError("Invalid local path.", ex); }
    }

    private async Task NavigateRemotePathBoxAsync()
    {
        if (!IsConnected) return;
        try
        {
            var path = InputGuard.RemotePath(_remotePathBox.Text);
            await _session!.ChangeDirectoryAsync(path);
            _remotePath = await _session.GetWorkingDirectoryAsync();
            await RefreshRemoteAsync();
        }
        catch (Exception ex) { ShowOperationError("Invalid remote path.", ex); }
    }

    private void NewLocalFolder()
    {
        var dialog = new TextPromptDialog(this, "New local folder", "Folder name");
        if (dialog.ShowDialog() != true) return;
        try
        {
            var name = LocalPathSafety.SafeFileName(dialog.Value);
            Directory.CreateDirectory(Path.Combine(_localPath, name));
            RefreshLocal();
        }
        catch (Exception ex) { ShowOperationError("Could not create the local folder.", ex); }
    }

    private void RenameLocalSelected()
    {
        if (_localList.SelectedItems.Count != 1 || _localList.SelectedItem is not LocalItem item) return;
        var dialog = new TextPromptDialog(this, "Rename local item", "New name", item.Name);
        if (dialog.ShowDialog() != true) return;
        try
        {
            var safeName = LocalPathSafety.SafeFileName(dialog.Value);
            var destination = LocalPathSafety.CombineUnderRoot(_localPath, safeName);
            if (string.Equals(destination, item.FullPath, StringComparison.OrdinalIgnoreCase)) return;
            if (File.Exists(destination) || Directory.Exists(destination))
                throw new IOException("An item with that name already exists.");
            if (item.IsDirectory) Directory.Move(item.FullPath, destination);
            else File.Move(item.FullPath, destination);
            RefreshLocal();
        }
        catch (Exception ex) { ShowOperationError("Could not rename the local item.", ex); }
    }

    private void DeleteLocalSelected()
    {
        if (_localList.SelectedItems.Count == 0) return;
        var items = _localList.SelectedItems.OfType<LocalItem>().ToArray();
        if (_settings.ConfirmDeletes)
        {
            var result = MessageBox.Show(this,
                $"Permanently delete {items.Length} selected local item(s)?\n\nLocal deletion does not use the Recycle Bin.",
                "GhostFTP", MessageBoxButton.YesNo, MessageBoxImage.Warning, MessageBoxResult.No);
            if (result != MessageBoxResult.Yes) return;
        }

        try
        {
            foreach (var item in items)
            {
                if (item.IsDirectory)
                {
                    var attributes = File.GetAttributes(item.FullPath);
                    if ((attributes & FileAttributes.ReparsePoint) != 0) Directory.Delete(item.FullPath);
                    else Directory.Delete(item.FullPath, recursive: true);
                }
                else File.Delete(item.FullPath);
            }
            RefreshLocal();
        }
        catch (Exception ex) { ShowOperationError("Could not delete the selected local item.", ex); }
    }

    private async Task NewRemoteFolderAsync()
    {
        if (!IsConnected) return;
        var dialog = new TextPromptDialog(this, "New remote folder", "Folder name");
        if (dialog.ShowDialog() != true) return;
        try
        {
            var destination = FtpListingParser.CombineRemote(_remotePath, InputGuard.RemoteName(dialog.Value));
            await _session!.CreateDirectoryAsync(destination);
            await RefreshRemoteAsync();
        }
        catch (Exception ex) { ShowOperationError("Could not create the remote folder.", ex); }
    }

    private async Task RenameRemoteSelectedAsync()
    {
        if (!IsConnected || _remoteList.SelectedItem is not RemoteItem item) return;
        var dialog = new TextPromptDialog(this, "Rename remote item", "New name", item.Name);
        if (dialog.ShowDialog() != true) return;
        try
        {
            var destination = FtpListingParser.CombineRemote(FtpListingParser.ParentRemote(item.FullPath), InputGuard.RemoteName(dialog.Value));
            await _session!.RenameAsync(item.FullPath, destination);
            await RefreshRemoteAsync();
        }
        catch (Exception ex) { ShowOperationError("Could not rename the remote item.", ex); }
    }

    private async Task DeleteRemoteSelectedAsync()
    {
        if (!IsConnected || _remoteList.SelectedItems.Count == 0) return;
        if (_settings.ConfirmDeletes)
        {
            var result = MessageBox.Show(this, $"Delete {_remoteList.SelectedItems.Count} selected remote item(s)?\n\nFolders are deleted recursively.", "GhostFTP", MessageBoxButton.YesNo, MessageBoxImage.Warning, MessageBoxResult.No);
            if (result != MessageBoxResult.Yes) return;
        }
        try
        {
            foreach (var item in _remoteList.SelectedItems.OfType<RemoteItem>().ToArray())
            {
                if (item.IsDirectory) await _session!.DeleteDirectoryAsync(item.FullPath, recursive: true);
                else await _session!.DeleteFileAsync(item.FullPath);
            }
            await RefreshRemoteAsync();
        }
        catch (Exception ex) { ShowOperationError("Could not delete the selected remote item.", ex); }
    }

}
