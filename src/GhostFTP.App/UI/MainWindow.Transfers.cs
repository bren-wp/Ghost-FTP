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
    private void QueueUploadSelected()
    {
        if (!IsConnected || _queue is null || _localList.SelectedItems.Count == 0) return;
        foreach (var item in _localList.SelectedItems.OfType<LocalItem>())
            _queue.EnqueueUpload(item.FullPath, FtpListingParser.CombineRemote(_remotePath, item.Name), item.IsDirectory);
        UpdateQueueSummary();
    }

    private void QueueDownloadSelected()
    {
        if (!IsConnected || _queue is null || _remoteList.SelectedItems.Count == 0) return;
        foreach (var item in _remoteList.SelectedItems.OfType<RemoteItem>())
        {
            var destination = LocalPathSafety.CombineUnderRoot(_localPath, item.Name);
            _queue.EnqueueDownload(item.FullPath, destination, item.IsDirectory, item.IsDirectory ? null : item.Entry.Size);
        }
        UpdateQueueSummary();
    }

    private void CancelSelectedTransfer()
    {
        if (_queueList.SelectedItem is TransferJob job) _queue?.Cancel(job.Id);
    }

    private void CancelAllTransfers()
    {
        if (_queue is null) return;
        foreach (var job in _queue.Jobs.Where(x => x.State is TransferState.Queued or TransferState.Running).ToArray()) _queue.Cancel(job.Id);
    }

    private async void QueueJobUpdated(object? sender, TransferJob job)
    {
        UpdateQueueSummary();
        if (job.State == TransferState.Completed && _completedHandled.Add(job.Id))
        {
            try
            {
                RefreshLocal();
                await RefreshRemoteAsync();
            }
            catch { }
        }
    }

    private void UpdateQueueSummary()
    {
        if (_queue is null || _queue.Jobs.Count == 0)
        {
            _queueSummary.Text = "No transfers";
            return;
        }
        var running = _queue.Jobs.Count(x => x.State == TransferState.Running);
        var queued = _queue.Jobs.Count(x => x.State == TransferState.Queued);
        var failed = _queue.Jobs.Count(x => x.State == TransferState.Failed);
        _queueSummary.Text = $"{running} running · {queued} queued" + (failed > 0 ? $" · {failed} failed" : "");
    }

    private async Task AddProfileAsync()
    {
        if (_profileStore is null) return;
        var profile = new ServerProfile { Id = Guid.NewGuid(), Name = "New server", Port = 21, Security = FtpSecurityMode.ExplicitTls, InitialPath = "/" };
        var dialog = new ProfileDialog(this, profile, "", isNew: true);
        if (dialog.ShowDialog() != true) return;
        var result = dialog.Result;
        _profileStore.SetPassword(result, dialog.Password);
        _profiles.Add(result);
        await SaveProfilesSafeAsync();
        _profilesList.SelectedItem = result;
    }

    private async Task EditSelectedProfileAsync()
    {
        if (_profileStore is null || _profilesList.SelectedItem is not ServerProfile selected || selected.IsDemo) return;
        var dialog = new ProfileDialog(this, selected, _profileStore.GetPassword(selected));
        if (dialog.ShowDialog() != true) return;
        var updated = dialog.Result;
        _profileStore.SetPassword(updated, dialog.Password);
        var index = _profiles.IndexOf(selected);
        _profiles[index] = updated;
        _profilesList.SelectedItem = updated;
        await SaveProfilesSafeAsync();
    }

    private async Task RemoveSelectedProfileAsync()
    {
        if (_profilesList.SelectedItem is not ServerProfile selected || selected.IsDemo) return;
        if (MessageBox.Show(this, $"Remove saved server '{selected.Name}'?", "GhostFTP", MessageBoxButton.YesNo, MessageBoxImage.Question) != MessageBoxResult.Yes) return;
        _profiles.Remove(selected);
        await SaveProfilesSafeAsync();
    }

    private async Task SaveProfilesSafeAsync()
    {
        try { if (_profileStore is not null) await _profileStore.SaveAsync(_profiles); }
        catch (Exception ex) { ShowOperationError("Could not save server profiles.", ex); }
    }

    private async Task OpenSettingsAsync()
    {
        var dialog = new SettingsDialog(this, _settings);
        if (dialog.ShowDialog() != true) return;
        _settings.Theme = dialog.SelectedTheme;
        _settings.ConfirmDeletes = dialog.ConfirmDeletes;
        if (_settingsStore is not null) await _settingsStore.SaveAsync(_settings);
        MessageBox.Show(this, "Settings saved. Appearance changes apply the next time GhostFTP starts.", "GhostFTP", MessageBoxButton.OK, MessageBoxImage.Information);
    }

    private void SetStatus(string text, string brushKey)
    {
        _statusText.Text = text;
        _statusText.Foreground = brushKey is "Success" or "Danger" or "Warning" ? Brushes.White : Theme.R("Text");
        _statusBadge.Background = Theme.R(brushKey);
    }

    private void UpdateConnectionUi()
    {
        var connected = IsConnected;
        _connectButton.IsEnabled = !connected && !_busy;
        _disconnectButton.IsEnabled = connected && !_busy;
        _host.IsEnabled = !connected && !_busy;
        _port.IsEnabled = !connected && !_busy;
        _username.IsEnabled = !connected && !_busy;
        _password.IsEnabled = !connected && !_busy;
        _security.IsEnabled = !connected && !_busy;
        _remotePathBox.IsEnabled = connected;
    }

    private bool IsConnected => _session?.IsConnected == true;

}
