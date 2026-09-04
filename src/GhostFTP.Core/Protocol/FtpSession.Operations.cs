using System.Globalization;
using System.Net.Security;
using System.Net.Sockets;
using System.Security.Authentication;
using System.Text;
using System.Text.RegularExpressions;
using GhostFTP.Core.Models;

namespace GhostFTP.Core.Protocol;

public sealed partial class FtpSession
{
    private async Task AuthenticateAsync(CancellationToken cancellationToken)
    {
        var user = await SendCommandAsync("USER " + _options.Username, cancellationToken).ConfigureAwait(false);
        if (user.Code is >= 200 and < 300)
            return;
        if (!user.IsPositiveIntermediate)
            throw CreateReplyException(user, "FTP username was rejected.");

        var pass = await SendCommandAsync("PASS " + _options.Password, cancellationToken, redactArgument: true).ConfigureAwait(false);
        Ensure(pass, 200, 299, "FTP authentication failed.");
    }

    private async Task<HashSet<string>> ReadFeaturesAsync(CancellationToken cancellationToken)
    {
        var reply = await TryCommandAsync("FEAT", cancellationToken).ConfigureAwait(false);
        var result = new HashSet<string>(StringComparer.OrdinalIgnoreCase);
        if (reply is null || !reply.IsPositiveCompletion)
            return result;

        foreach (var line in reply.Lines.Skip(1))
        {
            var trimmed = line.Trim();
            if (trimmed.Length == 0 || Regex.IsMatch(trimmed, @"^\d{3}\s"))
                continue;
            var token = trimmed.Split(' ', StringSplitOptions.RemoveEmptyEntries)[0];
            if (token.Length > 0)
                result.Add(token);
        }
        return result;
    }

    private async Task<IReadOnlyList<FtpEntry>> ListCoreAsync(string remotePath, CancellationToken cancellationToken)
    {
        EnsureConnected();
        remotePath = InputGuard.RemotePath(remotePath);
        var preferMlsd = _features.Contains("MLST") || _features.Contains("MLSD");

        if (preferMlsd)
        {
            try
            {
                var mlsd = await ReceiveTextDataAsync("MLSD " + remotePath, cancellationToken).ConfigureAwait(false);
                return FtpListingParser.ParseMlsd(mlsd, remotePath)
                    .OrderByDescending(x => x.IsDirectory).ThenBy(x => x.Name, StringComparer.OrdinalIgnoreCase).ToArray();
            }
            catch (FtpException ex) when (ex.ReplyCode is 500 or 501 or 502 or 504)
            {
                _features.Remove("MLST");
                _features.Remove("MLSD");
            }
        }

        var listText = await ReceiveTextDataAsync("LIST " + remotePath, cancellationToken).ConfigureAwait(false);
        return FtpListingParser.ParseList(listText, remotePath, DateTimeOffset.UtcNow)
            .OrderByDescending(x => x.IsDirectory).ThenBy(x => x.Name, StringComparer.OrdinalIgnoreCase).ToArray();
    }

    private async Task<string> GetWorkingDirectoryCoreAsync(CancellationToken cancellationToken)
    {
        EnsureConnectedTransport();
        var reply = await SendCommandAsync("PWD", cancellationToken).ConfigureAwait(false);
        Ensure(reply, 200, 299, "Unable to determine the FTP working directory.");
        var match = PwdRegex().Match(reply.Message);
        var path = match.Success ? match.Groups["path"].Value.Replace("\"\"", "\"", StringComparison.Ordinal) : "/";
        WorkingDirectory = InputGuard.RemotePath(path);
        return WorkingDirectory;
    }

    private async Task DeleteDirectoryCoreAsync(string remotePath, bool recursive, int depth, TraversalBudget budget, CancellationToken cancellationToken)
    {
        EnsureConnected();
        GuardTraversal(depth, budget);
        if (remotePath == "/")
            throw new InvalidOperationException("Deleting the FTP root directory is blocked.");

        if (recursive)
        {
            foreach (var entry in await ListCoreAsync(remotePath, cancellationToken).ConfigureAwait(false))
            {
                cancellationToken.ThrowIfCancellationRequested();
                if (entry.IsDirectory)
                    await DeleteDirectoryCoreAsync(entry.FullPath, true, depth + 1, budget, cancellationToken).ConfigureAwait(false);
                else
                    Ensure(await SendCommandAsync("DELE " + entry.FullPath, cancellationToken).ConfigureAwait(false), 200, 299, "Unable to delete a remote file while deleting the directory.");
            }
        }

        Ensure(await SendCommandAsync("RMD " + remotePath, cancellationToken).ConfigureAwait(false), 200, 299, "Unable to delete the remote directory.");
    }

    private async Task DownloadFileCoreAsync(string remotePath, string localPath, IProgress<(long transferred, long? total)>? progress, CancellationToken cancellationToken)
    {
        EnsureConnected();
        Directory.CreateDirectory(Path.GetDirectoryName(localPath) ?? Directory.GetCurrentDirectory());
        var partPath = localPath + ".ghostftp.part";
        var total = await TryGetFileSizeAsync(remotePath, cancellationToken).ConfigureAwait(false);
        long offset = File.Exists(partPath) ? new FileInfo(partPath).Length : 0;
        if (total is not null && offset > total.Value)
            offset = 0;

        if (offset > 0)
        {
            var rest = await SendCommandAsync("REST " + offset.ToString(CultureInfo.InvariantCulture), cancellationToken).ConfigureAwait(false);
            if (!rest.IsPositiveIntermediate)
            {
                offset = 0;
                TryDeleteLocal(partPath);
            }
        }

        await using var output = new FileStream(partPath, offset > 0 ? FileMode.Append : FileMode.Create, FileAccess.Write, FileShare.None, 1024 * 128, FileOptions.Asynchronous | FileOptions.SequentialScan);
        await ReceiveDataToStreamAsync("RETR " + remotePath, output, offset, total, progress, cancellationToken).ConfigureAwait(false);
        await output.FlushAsync(cancellationToken).ConfigureAwait(false);
        output.Close();
        File.Move(partPath, localPath, true);
    }

    private async Task UploadFileCoreAsync(string localPath, string remotePath, IProgress<(long transferred, long? total)>? progress, CancellationToken cancellationToken)
    {
        EnsureConnected();
        if (!File.Exists(localPath))
            throw new FileNotFoundException("Local file does not exist.", localPath);

        var total = new FileInfo(localPath).Length;
        var tempRemote = remotePath + ".ghostftp-upload-" + Guid.NewGuid().ToString("N") + ".part";
        try
        {
            await using var input = new FileStream(localPath, FileMode.Open, FileAccess.Read, FileShare.Read, 1024 * 128, FileOptions.Asynchronous | FileOptions.SequentialScan);
            await SendStreamAsDataAsync("STOR " + tempRemote, input, total, progress, cancellationToken).ConfigureAwait(false);

            var existingDelete = await TryCommandAsync("DELE " + remotePath, cancellationToken).ConfigureAwait(false);
            if (existingDelete is not null && !existingDelete.IsPositiveCompletion && existingDelete.Code != 550)
                throw CreateReplyException(existingDelete, "The uploaded temporary file is complete, but the previous destination could not be replaced safely.");

            Ensure(await SendCommandAsync("RNFR " + tempRemote, cancellationToken).ConfigureAwait(false), 300, 399, "Unable to finalize uploaded file.");
            Ensure(await SendCommandAsync("RNTO " + remotePath, cancellationToken).ConfigureAwait(false), 200, 299, "Unable to finalize uploaded file.");
        }
        catch
        {
            try { _ = await TryCommandAsync("DELE " + tempRemote, CancellationToken.None).ConfigureAwait(false); } catch { }
            throw;
        }
    }

    private async Task DownloadDirectoryCoreAsync(string remotePath, string localDirectory, IProgress<(long transferred, long? total)>? progress, CancellationToken cancellationToken)
    {
        var plan = new List<(FtpEntry entry, string localPath)>();
        await BuildDownloadPlanAsync(remotePath, localDirectory, plan, 0, new TraversalBudget(), cancellationToken).ConfigureAwait(false);
        long total = plan.Where(x => !x.entry.IsDirectory).Sum(x => Math.Max(0, x.entry.Size));
        long aggregate = 0;

        foreach (var item in plan)
        {
            cancellationToken.ThrowIfCancellationRequested();
            if (item.entry.IsDirectory)
            {
                Directory.CreateDirectory(item.localPath);
                continue;
            }

            Directory.CreateDirectory(Path.GetDirectoryName(item.localPath)!);
            var baseAggregate = aggregate;
            var fileProgress = new Progress<(long transferred, long? total)>(p => progress?.Report((baseAggregate + p.transferred, total)));
            await DownloadFileCoreAsync(item.entry.FullPath, item.localPath, fileProgress, cancellationToken).ConfigureAwait(false);
            aggregate += Math.Max(0, item.entry.Size);
            progress?.Report((aggregate, total));
        }
    }

    private async Task BuildDownloadPlanAsync(string remotePath, string localDirectory, List<(FtpEntry entry, string localPath)> plan, int depth, TraversalBudget budget, CancellationToken cancellationToken)
    {
        GuardTraversal(depth, budget);
        Directory.CreateDirectory(localDirectory);
        foreach (var entry in await ListCoreAsync(remotePath, cancellationToken).ConfigureAwait(false))
        {
            var local = LocalPathSafety.CombineUnderRoot(localDirectory, entry.Name);
            plan.Add((entry, local));
            if (entry.IsDirectory)
                await BuildDownloadPlanAsync(entry.FullPath, local, plan, depth + 1, budget, cancellationToken).ConfigureAwait(false);
        }
    }

    private async Task UploadDirectoryCoreAsync(string localDirectory, string remotePath, IProgress<(long transferred, long? total)>? progress, CancellationToken cancellationToken)
    {
        if (!Directory.Exists(localDirectory))
            throw new DirectoryNotFoundException(localDirectory);

        var enumeration = new EnumerationOptions
        {
            RecurseSubdirectories = true,
            IgnoreInaccessible = false,
            ReturnSpecialDirectories = false,
            AttributesToSkip = FileAttributes.ReparsePoint
        };
        var files = Directory.EnumerateFiles(localDirectory, "*", enumeration).Take(MaxTraversalEntries + 1).ToArray();
        if (files.Length > MaxTraversalEntries)
            throw new IOException($"Upload contains more than {MaxTraversalEntries:N0} files, which exceeds the safety limit.");
        var total = files.Sum(path => new FileInfo(path).Length);
        long aggregate = 0;

        await EnsureRemoteTreeAsync(remotePath, cancellationToken).ConfigureAwait(false);
        var directories = Directory.EnumerateDirectories(localDirectory, "*", enumeration).Take(MaxTraversalEntries + 1).ToArray();
        if (directories.Length > MaxTraversalEntries || (long)directories.Length + files.Length > MaxTraversalEntries)
            throw new IOException($"Upload contains more than {MaxTraversalEntries:N0} items, which exceeds the safety limit.");
        foreach (var directory in directories)
        {
            cancellationToken.ThrowIfCancellationRequested();
            var relative = Path.GetRelativePath(localDirectory, directory).Replace('\\', '/');
            await EnsureRemoteTreeAsync(FtpListingParser.CombineRemote(remotePath, relative), cancellationToken).ConfigureAwait(false);
        }

        foreach (var file in files)
        {
            cancellationToken.ThrowIfCancellationRequested();
            var relative = Path.GetRelativePath(localDirectory, file).Replace('\\', '/');
            var destination = FtpListingParser.CombineRemote(remotePath, relative);
            var size = new FileInfo(file).Length;
            var baseAggregate = aggregate;
            var fileProgress = new Progress<(long transferred, long? total)>(p => progress?.Report((baseAggregate + p.transferred, total)));
            await UploadFileCoreAsync(file, destination, fileProgress, cancellationToken).ConfigureAwait(false);
            aggregate += size;
            progress?.Report((aggregate, total));
        }
    }

    private async Task EnsureRemoteTreeAsync(string remotePath, CancellationToken cancellationToken)
    {
        remotePath = InputGuard.RemotePath(remotePath);
        if (remotePath == "/")
            return;
        var current = string.Empty;
        foreach (var segment in remotePath.Split('/', StringSplitOptions.RemoveEmptyEntries))
        {
            InputGuard.CommandArgument(segment, nameof(remotePath));
            current += "/" + segment;
            var reply = await TryCommandAsync("MKD " + current, cancellationToken).ConfigureAwait(false);
            if (reply is not null && !reply.IsPositiveCompletion && reply.Code != 550)
                throw CreateReplyException(reply, "Unable to create a remote folder required for upload.");
        }
    }

    private async Task<long?> TryGetFileSizeAsync(string remotePath, CancellationToken cancellationToken)
    {
        _ = await TryCommandAsync("TYPE I", cancellationToken).ConfigureAwait(false);
        var reply = await TryCommandAsync("SIZE " + remotePath, cancellationToken).ConfigureAwait(false);
        if (reply is null || !reply.IsPositiveCompletion)
            return null;
        var token = reply.Message.Split(' ', StringSplitOptions.RemoveEmptyEntries).LastOrDefault();
        return long.TryParse(token, NumberStyles.None, CultureInfo.InvariantCulture, out var value) ? value : null;
    }

}
