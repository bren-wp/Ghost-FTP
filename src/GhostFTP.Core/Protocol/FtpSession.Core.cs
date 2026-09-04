using System.Globalization;
using System.Net.Security;
using System.Net.Sockets;
using System.Security.Authentication;
using System.Text;
using System.Text.RegularExpressions;
using GhostFTP.Core.Models;

namespace GhostFTP.Core.Protocol;


public sealed partial class FtpSession : IFtpSession
{
    private const int MaxReplyLines = 256;
    private const int MaxReplyChars = 1_048_576;
    private const int MaxReplyLineChars = 65_536;
    private const int MaxTraversalDepth = 64;
    private const int MaxTraversalEntries = 100_000;
    private static readonly Encoding ControlEncoding = new UTF8Encoding(false, false);

    private readonly FtpConnectionOptions _options;
    private readonly SemaphoreSlim _gate = new(1, 1);
    private TcpClient? _controlClient;
    private Stream? _controlStream;
    private StreamReader? _reader;
    private StreamWriter? _writer;
    private bool _disposed;
    private bool _dataProtection;
    private HashSet<string> _features = new(StringComparer.OrdinalIgnoreCase);

    public bool IsConnected { get; private set; }
    public bool IsEncrypted { get; private set; }
    public string Host => _options.Host;
    public string WorkingDirectory { get; private set; } = "/";

    public FtpSession(FtpConnectionOptions options)
    {
        ArgumentNullException.ThrowIfNull(options);
        _options = new FtpConnectionOptions
        {
            Host = InputGuard.Host(options.Host),
            Port = InputGuard.Port(options.Port),
            Username = InputGuard.CommandArgument(options.Username, nameof(options.Username)),
            Password = InputGuard.CommandArgument(options.Password, nameof(options.Password)),
            Security = options.Security,
            ConnectTimeout = Clamp(options.ConnectTimeout, TimeSpan.FromSeconds(3), TimeSpan.FromMinutes(2)),
            CommandTimeout = Clamp(options.CommandTimeout, TimeSpan.FromSeconds(5), TimeSpan.FromMinutes(5)),
            TransferTimeout = Clamp(options.TransferTimeout, TimeSpan.FromSeconds(15), TimeSpan.FromHours(1))
        };
    }

    public async Task ConnectAsync(CancellationToken cancellationToken = default)
    {
        await _gate.WaitAsync(cancellationToken).ConfigureAwait(false);
        try
        {
            ThrowIfDisposed();
            if (IsConnected)
                return;

            await ResetTransportAsync().ConfigureAwait(false);
            _controlClient = new TcpClient(AddressFamily.InterNetworkV6) { NoDelay = true };
            _controlClient.Client.DualMode = true;
            await _controlClient.ConnectAsync(_options.Host, _options.Port, cancellationToken)
                .AsTask().WaitAsync(_options.ConnectTimeout, cancellationToken).ConfigureAwait(false);

            _controlStream = _controlClient.GetStream();

            if (_options.Security == FtpSecurityMode.ImplicitTls)
                await UpgradeControlToTlsAsync(cancellationToken).ConfigureAwait(false);

            BuildControlTextStreams();
            var greeting = await ReadReplyAsync(cancellationToken).ConfigureAwait(false);
            Ensure(greeting, 200, 299, "Server did not accept the connection.");

            if (_options.Security == FtpSecurityMode.ExplicitTls)
            {
                var auth = await SendCommandAsync("AUTH TLS", cancellationToken).ConfigureAwait(false);
                Ensure(auth, 200, 399, "Server refused explicit TLS.");
                await UpgradeControlToTlsAsync(cancellationToken).ConfigureAwait(false);
                BuildControlTextStreams();
            }

            if (IsEncrypted)
            {
                var pbsz = await SendCommandAsync("PBSZ 0", cancellationToken).ConfigureAwait(false);
                Ensure(pbsz, 200, 299, "Server refused PBSZ for FTPS.");
                var prot = await SendCommandAsync("PROT P", cancellationToken).ConfigureAwait(false);
                Ensure(prot, 200, 299, "Server refused encrypted FTP data channels.");
                _dataProtection = true;
            }

            await AuthenticateAsync(cancellationToken).ConfigureAwait(false);
            _ = await TryCommandAsync("OPTS UTF8 ON", cancellationToken).ConfigureAwait(false);
            _features = await ReadFeaturesAsync(cancellationToken).ConfigureAwait(false);
            WorkingDirectory = await GetWorkingDirectoryCoreAsync(cancellationToken).ConfigureAwait(false);
            IsConnected = true;
        }
        catch
        {
            await ResetTransportAsync().ConfigureAwait(false);
            throw;
        }
        finally
        {
            _gate.Release();
        }
    }

    public async Task DisconnectAsync(CancellationToken cancellationToken = default)
    {
        await _gate.WaitAsync(cancellationToken).ConfigureAwait(false);
        try
        {
            if (_disposed)
                return;
            if (_writer is not null)
            {
                try { _ = await SendCommandAsync("QUIT", cancellationToken).ConfigureAwait(false); }
                catch { }
            }
            await ResetTransportAsync().ConfigureAwait(false);
        }
        finally
        {
            _gate.Release();
        }
    }

    public Task<IReadOnlyList<FtpEntry>> ListAsync(string remotePath, CancellationToken cancellationToken = default) =>
        LockedAsync(ct => ListCoreAsync(remotePath, ct), cancellationToken);

    public Task<string> GetWorkingDirectoryAsync(CancellationToken cancellationToken = default) =>
        LockedAsync(GetWorkingDirectoryCoreAsync, cancellationToken);

    public Task ChangeDirectoryAsync(string remotePath, CancellationToken cancellationToken = default) =>
        LockedAsync(async ct =>
        {
            remotePath = InputGuard.RemotePath(remotePath);
            Ensure(await SendCommandAsync("CWD " + remotePath, ct).ConfigureAwait(false), 200, 299, "Unable to change the remote directory.");
            WorkingDirectory = await GetWorkingDirectoryCoreAsync(ct).ConfigureAwait(false);
        }, cancellationToken);

    public Task CreateDirectoryAsync(string remotePath, CancellationToken cancellationToken = default) =>
        LockedAsync(async ct =>
        {
            remotePath = InputGuard.RemotePath(remotePath);
            var reply = await SendCommandAsync("MKD " + remotePath, ct).ConfigureAwait(false);
            if (!reply.IsPositiveCompletion && reply.Code != 550)
                throw CreateReplyException(reply, "Unable to create the remote directory.");
        }, cancellationToken);

    public Task RenameAsync(string sourcePath, string destinationPath, CancellationToken cancellationToken = default) =>
        LockedAsync(async ct =>
        {
            sourcePath = InputGuard.RemotePath(sourcePath);
            destinationPath = InputGuard.RemotePath(destinationPath);
            Ensure(await SendCommandAsync("RNFR " + sourcePath, ct).ConfigureAwait(false), 300, 399, "Server refused the rename source.");
            Ensure(await SendCommandAsync("RNTO " + destinationPath, ct).ConfigureAwait(false), 200, 299, "Server refused the rename destination.");
        }, cancellationToken);

    public Task DeleteFileAsync(string remotePath, CancellationToken cancellationToken = default) =>
        LockedAsync(async ct =>
        {
            remotePath = InputGuard.RemotePath(remotePath);
            Ensure(await SendCommandAsync("DELE " + remotePath, ct).ConfigureAwait(false), 200, 299, "Unable to delete the remote file.");
        }, cancellationToken);

    public Task DeleteDirectoryAsync(string remotePath, bool recursive, CancellationToken cancellationToken = default) =>
        LockedAsync(ct => DeleteDirectoryCoreAsync(InputGuard.RemotePath(remotePath), recursive, 0, new TraversalBudget(), ct), cancellationToken);

    public Task DownloadFileAsync(string remotePath, string localPath, IProgress<(long transferred, long? total)>? progress = null, CancellationToken cancellationToken = default) =>
        LockedAsync(ct => DownloadFileCoreAsync(InputGuard.RemotePath(remotePath), Path.GetFullPath(localPath), progress, ct), cancellationToken);

    public Task UploadFileAsync(string localPath, string remotePath, IProgress<(long transferred, long? total)>? progress = null, CancellationToken cancellationToken = default) =>
        LockedAsync(ct => UploadFileCoreAsync(Path.GetFullPath(localPath), InputGuard.RemotePath(remotePath), progress, ct), cancellationToken);

    public Task DownloadDirectoryAsync(string remotePath, string localDirectory, IProgress<(long transferred, long? total)>? progress = null, CancellationToken cancellationToken = default) =>
        LockedAsync(ct => DownloadDirectoryCoreAsync(InputGuard.RemotePath(remotePath), Path.GetFullPath(localDirectory), progress, ct), cancellationToken);

    public Task UploadDirectoryAsync(string localDirectory, string remotePath, IProgress<(long transferred, long? total)>? progress = null, CancellationToken cancellationToken = default) =>
        LockedAsync(ct => UploadDirectoryCoreAsync(Path.GetFullPath(localDirectory), InputGuard.RemotePath(remotePath), progress, ct), cancellationToken);

}
