using System.Text.Json;
using GhostFTP.Core.Models;

namespace GhostFTP.Core.Services;

public sealed class ProfileStore
{
    private readonly string _filePath;
    private readonly ISecretProtector _secretProtector;
    private readonly SemaphoreSlim _gate = new(1, 1);
    private static readonly JsonSerializerOptions JsonOptions = new()
    {
        WriteIndented = true,
        PropertyNamingPolicy = JsonNamingPolicy.CamelCase
    };

    public ProfileStore(string filePath, ISecretProtector secretProtector)
    {
        _filePath = Path.GetFullPath(filePath);
        _secretProtector = secretProtector ?? throw new ArgumentNullException(nameof(secretProtector));
    }

    public async Task<IReadOnlyList<ServerProfile>> LoadAsync(CancellationToken cancellationToken = default)
    {
        await _gate.WaitAsync(cancellationToken).ConfigureAwait(false);
        try
        {
            if (!File.Exists(_filePath))
                return [CreateDemoProfile()];

            try
            {
                await using var stream = new FileStream(_filePath, FileMode.Open, FileAccess.Read, FileShare.Read, 32 * 1024, FileOptions.Asynchronous | FileOptions.SequentialScan);
                var profiles = await JsonSerializer.DeserializeAsync<List<ServerProfile>>(stream, JsonOptions, cancellationToken).ConfigureAwait(false) ?? [];
                Normalize(profiles);
                EnsureDemo(profiles);
                return profiles;
            }
            catch (JsonException)
            {
                var backup = _filePath + ".bak";
                if (!File.Exists(backup))
                    throw;
                await using var stream = new FileStream(backup, FileMode.Open, FileAccess.Read, FileShare.Read, 32 * 1024, FileOptions.Asynchronous | FileOptions.SequentialScan);
                var profiles = await JsonSerializer.DeserializeAsync<List<ServerProfile>>(stream, JsonOptions, cancellationToken).ConfigureAwait(false) ?? [];
                Normalize(profiles);
                EnsureDemo(profiles);
                return profiles;
            }
        }
        finally
        {
            _gate.Release();
        }
    }

    public async Task SaveAsync(IEnumerable<ServerProfile> profiles, CancellationToken cancellationToken = default)
    {
        ArgumentNullException.ThrowIfNull(profiles);
        await _gate.WaitAsync(cancellationToken).ConfigureAwait(false);
        try
        {
            var list = profiles.Select(x => x.Clone()).ToList();
            Normalize(list);
            EnsureDemo(list);
            Directory.CreateDirectory(Path.GetDirectoryName(_filePath)!);
            var temp = _filePath + ".tmp-" + Guid.NewGuid().ToString("N");
            try
            {
                await using (var stream = new FileStream(temp, FileMode.CreateNew, FileAccess.Write, FileShare.None, 32 * 1024, FileOptions.Asynchronous | FileOptions.WriteThrough))
                {
                    await JsonSerializer.SerializeAsync(stream, list, JsonOptions, cancellationToken).ConfigureAwait(false);
                    await stream.FlushAsync(cancellationToken).ConfigureAwait(false);
                }

                if (File.Exists(_filePath))
                {
                    var backup = _filePath + ".bak";
                    File.Replace(temp, _filePath, backup, ignoreMetadataErrors: true);
                }
                else
                {
                    File.Move(temp, _filePath);
                }
            }
            finally
            {
                try { if (File.Exists(temp)) File.Delete(temp); } catch { }
            }
        }
        finally
        {
            _gate.Release();
        }
    }

    public void SetPassword(ServerProfile profile, string? password)
    {
        ArgumentNullException.ThrowIfNull(profile);
        if (!profile.RememberPassword || string.IsNullOrEmpty(password))
        {
            profile.ProtectedPassword = null;
            return;
        }
        profile.ProtectedPassword = _secretProtector.Protect(password);
    }

    public string GetPassword(ServerProfile profile)
    {
        ArgumentNullException.ThrowIfNull(profile);
        if (!profile.RememberPassword || string.IsNullOrWhiteSpace(profile.ProtectedPassword))
            return string.Empty;
        try { return _secretProtector.Unprotect(profile.ProtectedPassword); }
        catch { return string.Empty; }
    }

    private static void Normalize(List<ServerProfile> profiles)
    {
        var seen = new HashSet<Guid>();
        foreach (var profile in profiles)
        {
            if (profile.Id == Guid.Empty || !seen.Add(profile.Id))
                profile.Id = Guid.NewGuid();
            profile.Name = string.IsNullOrWhiteSpace(profile.Name) ? "Unnamed server" : profile.Name.Trim();
            profile.InitialPath = string.IsNullOrWhiteSpace(profile.InitialPath) ? "/" : profile.InitialPath.Trim();
            if (!profile.IsDemo)
            {
                profile.Host = profile.Host.Trim();
                profile.Port = profile.Port is >= 1 and <= 65535 ? profile.Port : (profile.Security == FtpSecurityMode.ImplicitTls ? 990 : 21);
                profile.Username = profile.Username.Trim();
                if (!profile.RememberPassword)
                    profile.ProtectedPassword = null;
            }
        }
    }

    private static void EnsureDemo(List<ServerProfile> profiles)
    {
        if (!profiles.Any(x => x.IsDemo))
            profiles.Insert(0, CreateDemoProfile());
    }

    private static ServerProfile CreateDemoProfile() => new()
    {
        Name = "GhostFTP Demo",
        Host = "demo.ghostftp.local",
        Port = 21,
        Username = "demo",
        Security = FtpSecurityMode.Plain,
        InitialPath = "/",
        IsDemo = true,
        RememberPassword = false
    };
}
