using GhostFTP.Core.Models;

namespace GhostFTP.Core.Protocol;

public interface IFtpSession : IAsyncDisposable
{
    bool IsConnected { get; }
    bool IsEncrypted { get; }
    string Host { get; }
    string WorkingDirectory { get; }

    Task ConnectAsync(CancellationToken cancellationToken = default);
    Task DisconnectAsync(CancellationToken cancellationToken = default);
    Task<IReadOnlyList<FtpEntry>> ListAsync(string remotePath, CancellationToken cancellationToken = default);
    Task<string> GetWorkingDirectoryAsync(CancellationToken cancellationToken = default);
    Task ChangeDirectoryAsync(string remotePath, CancellationToken cancellationToken = default);
    Task CreateDirectoryAsync(string remotePath, CancellationToken cancellationToken = default);
    Task RenameAsync(string sourcePath, string destinationPath, CancellationToken cancellationToken = default);
    Task DeleteFileAsync(string remotePath, CancellationToken cancellationToken = default);
    Task DeleteDirectoryAsync(string remotePath, bool recursive, CancellationToken cancellationToken = default);
    Task DownloadFileAsync(string remotePath, string localPath, IProgress<(long transferred, long? total)>? progress = null, CancellationToken cancellationToken = default);
    Task UploadFileAsync(string localPath, string remotePath, IProgress<(long transferred, long? total)>? progress = null, CancellationToken cancellationToken = default);
    Task DownloadDirectoryAsync(string remotePath, string localDirectory, IProgress<(long transferred, long? total)>? progress = null, CancellationToken cancellationToken = default);
    Task UploadDirectoryAsync(string localDirectory, string remotePath, IProgress<(long transferred, long? total)>? progress = null, CancellationToken cancellationToken = default);
}
