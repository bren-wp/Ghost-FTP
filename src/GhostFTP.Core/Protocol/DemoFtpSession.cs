using System.Text;
using GhostFTP.Core.Models;

namespace GhostFTP.Core.Protocol;

public sealed class DemoFtpSession : IFtpSession
{
    private const long MaxDemoStoredBytes = 64L * 1024 * 1024;
    private sealed class Node
    {
        public required string Name { get; set; }
        public bool IsDirectory { get; init; }
        public byte[] Data { get; set; } = [];
        public DateTimeOffset ModifiedUtc { get; set; } = DateTimeOffset.UtcNow;
        public Dictionary<string, Node> Children { get; } = new(StringComparer.OrdinalIgnoreCase);
    }

    private readonly Node _root;
    private readonly SemaphoreSlim _gate = new(1, 1);
    private bool _disposed;

    public bool IsConnected { get; private set; }
    public bool IsEncrypted => false;
    public string Host => "demo.ghostftp.local";
    public string WorkingDirectory { get; private set; } = "/";

    public DemoFtpSession()
    {
        _root = Dir("");
        var publicHtml = Dir("public_html");
        publicHtml.Children["index.html"] = File("index.html", "<!doctype html>\n<html><head><title>GhostFTP Demo</title></head><body><h1>Hello from GhostFTP</h1></body></html>\n");
        publicHtml.Children["robots.txt"] = File("robots.txt", "User-agent: *\nDisallow:\n");
        var assets = Dir("assets");
        assets.Children["app.css"] = File("app.css", "body { font-family: system-ui; margin: 3rem; }\n");
        assets.Children["app.js"] = File("app.js", "console.log('GhostFTP demo');\n");
        publicHtml.Children["assets"] = assets;
        _root.Children["public_html"] = publicHtml;

        var backups = Dir("backups");
        backups.Children["site-2026-09-01.zip"] = Binary("site-2026-09-01.zip", 512 * 1024, 0x47);
        backups.Children["site-2026-08-31.zip"] = Binary("site-2026-08-31.zip", 384 * 1024, 0x42);
        _root.Children["backups"] = backups;

        var logs = Dir("logs");
        logs.Children["access.log"] = File("access.log", "127.0.0.1 - - [04/Sep/2026:20:00:00 +0000] \"GET / HTTP/1.1\" 200 1024\n");
        logs.Children["error.log"] = File("error.log", "# Demo log file - no real telemetry is collected by GhostFTP.\n");
        _root.Children["logs"] = logs;
        _root.Children["README.txt"] = File("README.txt", "GhostFTP demo server\n\nAll demo data is local and generated inside the application. No network request is made.\n");
    }

    public async Task ConnectAsync(CancellationToken cancellationToken = default)
    {
        await Task.Delay(140, cancellationToken).ConfigureAwait(false);
        ThrowIfDisposed();
        IsConnected = true;
    }

    public Task DisconnectAsync(CancellationToken cancellationToken = default)
    {
        cancellationToken.ThrowIfCancellationRequested();
        IsConnected = false;
        return Task.CompletedTask;
    }

    public Task<IReadOnlyList<FtpEntry>> ListAsync(string remotePath, CancellationToken cancellationToken = default) =>
        LockedAsync(ct =>
        {
            ct.ThrowIfCancellationRequested();
            var path = InputGuard.RemotePath(remotePath);
            var node = Resolve(path);
            if (!node.IsDirectory)
                throw new FtpException("Demo path is not a directory.");
            IReadOnlyList<FtpEntry> items = node.Children.Values
                .Select(child => ToEntry(path, child))
                .OrderByDescending(x => x.IsDirectory)
                .ThenBy(x => x.Name, StringComparer.OrdinalIgnoreCase)
                .ToArray();
            return Task.FromResult(items);
        }, cancellationToken);

    public Task<string> GetWorkingDirectoryAsync(CancellationToken cancellationToken = default) => Task.FromResult(WorkingDirectory);

    public Task ChangeDirectoryAsync(string remotePath, CancellationToken cancellationToken = default) =>
        LockedAsync(ct =>
        {
            ct.ThrowIfCancellationRequested();
            var path = InputGuard.RemotePath(remotePath);
            if (!Resolve(path).IsDirectory)
                throw new FtpException("Demo path is not a directory.");
            WorkingDirectory = path;
            return Task.CompletedTask;
        }, cancellationToken);

    public Task CreateDirectoryAsync(string remotePath, CancellationToken cancellationToken = default) =>
        LockedAsync(ct =>
        {
            ct.ThrowIfCancellationRequested();
            var (parent, name) = ResolveParent(InputGuard.RemotePath(remotePath));
            if (parent.Children.ContainsKey(name))
                throw new FtpException("An item with this name already exists.", 550);
            parent.Children[name] = Dir(name);
            return Task.CompletedTask;
        }, cancellationToken);

    public Task RenameAsync(string sourcePath, string destinationPath, CancellationToken cancellationToken = default) =>
        LockedAsync(ct =>
        {
            ct.ThrowIfCancellationRequested();
            var source = InputGuard.RemotePath(sourcePath);
            var destination = InputGuard.RemotePath(destinationPath);
            if (source == "/" || destination == "/")
                throw new FtpException("Demo root cannot be renamed.");
            var (sourceParent, sourceName) = ResolveParent(source);
            if (!sourceParent.Children.Remove(sourceName, out var node))
                throw new FtpException("Source item was not found.", 550);
            var (destinationParent, destinationName) = ResolveParent(destination);
            if (destinationParent.Children.ContainsKey(destinationName))
            {
                sourceParent.Children[sourceName] = node;
                throw new FtpException("Destination item already exists.", 550);
            }
            node.Name = destinationName;
            node.ModifiedUtc = DateTimeOffset.UtcNow;
            destinationParent.Children[destinationName] = node;
            return Task.CompletedTask;
        }, cancellationToken);

    public Task DeleteFileAsync(string remotePath, CancellationToken cancellationToken = default) =>
        LockedAsync(ct =>
        {
            ct.ThrowIfCancellationRequested();
            var (parent, name) = ResolveParent(InputGuard.RemotePath(remotePath));
            if (!parent.Children.TryGetValue(name, out var node) || node.IsDirectory)
                throw new FtpException("Demo file was not found.", 550);
            parent.Children.Remove(name);
            return Task.CompletedTask;
        }, cancellationToken);

    public Task DeleteDirectoryAsync(string remotePath, bool recursive, CancellationToken cancellationToken = default) =>
        LockedAsync(ct =>
        {
            ct.ThrowIfCancellationRequested();
            var path = InputGuard.RemotePath(remotePath);
            if (path == "/")
                throw new FtpException("Demo root cannot be deleted.");
            var (parent, name) = ResolveParent(path);
            if (!parent.Children.TryGetValue(name, out var node) || !node.IsDirectory)
                throw new FtpException("Demo directory was not found.", 550);
            if (!recursive && node.Children.Count > 0)
                throw new FtpException("Directory is not empty.", 550);
            parent.Children.Remove(name);
            return Task.CompletedTask;
        }, cancellationToken);

    public Task DownloadFileAsync(string remotePath, string localPath, IProgress<(long transferred, long? total)>? progress = null, CancellationToken cancellationToken = default) =>
        LockedAsync(async ct =>
        {
            var node = Resolve(InputGuard.RemotePath(remotePath));
            if (node.IsDirectory)
                throw new FtpException("Demo item is a directory.");
            var full = Path.GetFullPath(localPath);
            Directory.CreateDirectory(Path.GetDirectoryName(full) ?? Directory.GetCurrentDirectory());
            var part = full + ".ghostftp.part";
            await WriteBytesAsync(node.Data, part, progress, ct).ConfigureAwait(false);
            File.Move(part, full, true);
        }, cancellationToken);

    public Task UploadFileAsync(string localPath, string remotePath, IProgress<(long transferred, long? total)>? progress = null, CancellationToken cancellationToken = default) =>
        LockedAsync(async ct =>
        {
            var full = Path.GetFullPath(localPath);
            if (!System.IO.File.Exists(full))
                throw new FileNotFoundException("Local file does not exist.", full);
            var length = new FileInfo(full).Length;
            if (length > MaxDemoStoredBytes)
                throw new IOException("Demo mode stores files in memory and accepts files up to 64 MB.");
            var (parent, name) = ResolveParent(InputGuard.RemotePath(remotePath));
            var data = await ReadBytesAsync(full, progress, ct).ConfigureAwait(false);
            parent.Children[name] = new Node { Name = name, IsDirectory = false, Data = data, ModifiedUtc = DateTimeOffset.UtcNow };
        }, cancellationToken);

    public Task DownloadDirectoryAsync(string remotePath, string localDirectory, IProgress<(long transferred, long? total)>? progress = null, CancellationToken cancellationToken = default) =>
        LockedAsync(async ct =>
        {
            var node = Resolve(InputGuard.RemotePath(remotePath));
            if (!node.IsDirectory)
                throw new FtpException("Demo item is not a directory.");
            var total = SumBytes(node);
            long transferred = 0;
            await DownloadNodeAsync(node, Path.GetFullPath(localDirectory), total, () => transferred, v => transferred = v, progress, ct).ConfigureAwait(false);
        }, cancellationToken);

    public Task UploadDirectoryAsync(string localDirectory, string remotePath, IProgress<(long transferred, long? total)>? progress = null, CancellationToken cancellationToken = default) =>
        LockedAsync(async ct =>
        {
            var local = Path.GetFullPath(localDirectory);
            if (!Directory.Exists(local))
                throw new DirectoryNotFoundException(local);
            var remote = InputGuard.RemotePath(remotePath);
            var files = Directory.EnumerateFiles(local, "*", SearchOption.AllDirectories).ToArray();
            var total = files.Sum(x => new FileInfo(x).Length);
            if (total > MaxDemoStoredBytes)
                throw new IOException("Demo mode stores files in memory and accepts folders up to 64 MB in total.");
            var node = EnsureDirectory(remote);
            long transferred = 0;
            foreach (var directory in Directory.EnumerateDirectories(local, "*", SearchOption.AllDirectories))
            {
                var relative = Path.GetRelativePath(local, directory).Replace('\\', '/');
                EnsureDirectory(FtpListingParser.CombineRemote(remote, relative));
            }
            foreach (var file in files)
            {
                ct.ThrowIfCancellationRequested();
                var relative = Path.GetRelativePath(local, file).Replace('\\', '/');
                var destination = FtpListingParser.CombineRemote(remote, relative);
                var (parent, name) = ResolveParent(destination);
                await using var input = new FileStream(file, FileMode.Open, FileAccess.Read, FileShare.Read, 128 * 1024, FileOptions.Asynchronous | FileOptions.SequentialScan);
                var memory = new MemoryStream();
                var buffer = new byte[128 * 1024];
                while (true)
                {
                    var read = await input.ReadAsync(buffer, ct).ConfigureAwait(false);
                    if (read == 0) break;
                    await memory.WriteAsync(buffer.AsMemory(0, read), ct).ConfigureAwait(false);
                    transferred += read;
                    progress?.Report((transferred, total));
                }
                parent.Children[name] = new Node { Name = name, IsDirectory = false, Data = memory.ToArray(), ModifiedUtc = DateTimeOffset.UtcNow };
            }
            _ = node;
        }, cancellationToken);

    private Node EnsureDirectory(string path)
    {
        path = InputGuard.RemotePath(path);
        if (path == "/") return _root;
        var current = _root;
        foreach (var segment in Segments(path))
        {
            if (!current.Children.TryGetValue(segment, out var next))
            {
                next = Dir(segment);
                current.Children[segment] = next;
            }
            if (!next.IsDirectory)
                throw new FtpException("A file blocks creation of the requested demo folder.", 550);
            current = next;
        }
        return current;
    }

    private Node Resolve(string path)
    {
        path = InputGuard.RemotePath(path);
        if (path == "/") return _root;
        var current = _root;
        foreach (var segment in Segments(path))
        {
            if (!current.IsDirectory || !current.Children.TryGetValue(segment, out var next))
                throw new FtpException("Demo path was not found.", 550);
            current = next;
        }
        return current;
    }

    private (Node parent, string name) ResolveParent(string path)
    {
        path = InputGuard.RemotePath(path);
        if (path == "/")
            throw new FtpException("Operation is not allowed on demo root.");
        var parentPath = FtpListingParser.ParentRemote(path);
        var name = path.Split('/', StringSplitOptions.RemoveEmptyEntries).Last();
        return (Resolve(parentPath), name);
    }

    private static IEnumerable<string> Segments(string path) => path.Split('/', StringSplitOptions.RemoveEmptyEntries);

    private static FtpEntry ToEntry(string parentPath, Node node) => new(
        node.Name,
        FtpListingParser.CombineRemote(parentPath, node.Name),
        node.IsDirectory,
        node.IsDirectory ? 0 : node.Data.LongLength,
        node.ModifiedUtc,
        node.IsDirectory ? "rwxr-xr-x" : "rw-r--r--",
        null);

    private static Node Dir(string name) => new() { Name = name, IsDirectory = true };
    private static Node File(string name, string text) => new() { Name = name, IsDirectory = false, Data = Encoding.UTF8.GetBytes(text) };
    private static Node Binary(string name, int size, byte seed)
    {
        var data = new byte[size];
        for (var i = 0; i < data.Length; i++) data[i] = (byte)(seed + i % 17);
        return new Node { Name = name, IsDirectory = false, Data = data };
    }

    private static long SumBytes(Node node) => node.IsDirectory ? node.Children.Values.Sum(SumBytes) : node.Data.LongLength;

    private static async Task DownloadNodeAsync(Node node, string localDirectory, long total, Func<long> getTransferred, Action<long> setTransferred, IProgress<(long transferred, long? total)>? progress, CancellationToken ct)
    {
        Directory.CreateDirectory(localDirectory);
        foreach (var child in node.Children.Values)
        {
            ct.ThrowIfCancellationRequested();
            var path = LocalPathSafety.CombineUnderRoot(localDirectory, child.Name);
            if (child.IsDirectory)
            {
                await DownloadNodeAsync(child, path, total, getTransferred, setTransferred, progress, ct).ConfigureAwait(false);
                continue;
            }
            await using var output = new FileStream(path + ".ghostftp.part", FileMode.Create, FileAccess.Write, FileShare.None, 128 * 1024, FileOptions.Asynchronous);
            const int chunk = 64 * 1024;
            var offset = 0;
            while (offset < child.Data.Length)
            {
                var count = Math.Min(chunk, child.Data.Length - offset);
                await output.WriteAsync(child.Data.AsMemory(offset, count), ct).ConfigureAwait(false);
                offset += count;
                var transferred = getTransferred() + count;
                setTransferred(transferred);
                progress?.Report((transferred, total));
            }
            output.Close();
            File.Move(path + ".ghostftp.part", path, true);
        }
    }

    private static async Task WriteBytesAsync(byte[] data, string path, IProgress<(long transferred, long? total)>? progress, CancellationToken ct)
    {
        await using var output = new FileStream(path, FileMode.Create, FileAccess.Write, FileShare.None, 128 * 1024, FileOptions.Asynchronous);
        var offset = 0;
        while (offset < data.Length)
        {
            var count = Math.Min(64 * 1024, data.Length - offset);
            await output.WriteAsync(data.AsMemory(offset, count), ct).ConfigureAwait(false);
            offset += count;
            progress?.Report((offset, data.LongLength));
        }
        await output.FlushAsync(ct).ConfigureAwait(false);
    }

    private static async Task<byte[]> ReadBytesAsync(string path, IProgress<(long transferred, long? total)>? progress, CancellationToken ct)
    {
        var total = new FileInfo(path).Length;
        await using var input = new FileStream(path, FileMode.Open, FileAccess.Read, FileShare.Read, 128 * 1024, FileOptions.Asynchronous | FileOptions.SequentialScan);
        using var memory = new MemoryStream(total > int.MaxValue ? 0 : (int)total);
        var buffer = new byte[128 * 1024];
        long transferred = 0;
        while (true)
        {
            var read = await input.ReadAsync(buffer, ct).ConfigureAwait(false);
            if (read == 0) break;
            await memory.WriteAsync(buffer.AsMemory(0, read), ct).ConfigureAwait(false);
            transferred += read;
            progress?.Report((transferred, total));
        }
        return memory.ToArray();
    }

    private async Task LockedAsync(Func<CancellationToken, Task> action, CancellationToken cancellationToken)
    {
        await _gate.WaitAsync(cancellationToken).ConfigureAwait(false);
        try { ThrowIfDisposed(); EnsureConnected(); await action(cancellationToken).ConfigureAwait(false); }
        finally { _gate.Release(); }
    }

    private async Task<T> LockedAsync<T>(Func<CancellationToken, Task<T>> action, CancellationToken cancellationToken)
    {
        await _gate.WaitAsync(cancellationToken).ConfigureAwait(false);
        try { ThrowIfDisposed(); EnsureConnected(); return await action(cancellationToken).ConfigureAwait(false); }
        finally { _gate.Release(); }
    }

    private void EnsureConnected()
    {
        if (!IsConnected) throw new InvalidOperationException("Demo session is not connected.");
    }

    private void ThrowIfDisposed() => ObjectDisposedException.ThrowIf(_disposed, this);

    public ValueTask DisposeAsync()
    {
        _disposed = true;
        IsConnected = false;
        _gate.Dispose();
        return ValueTask.CompletedTask;
    }
}
