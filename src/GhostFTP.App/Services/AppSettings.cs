using System.Text.Json;

namespace GhostFTP.Services;

public enum AppTheme
{
    System,
    Dark,
    Light
}

public sealed class AppSettings
{
    public AppTheme Theme { get; set; } = AppTheme.System;
    public string LastLocalDirectory { get; set; } = Environment.GetFolderPath(Environment.SpecialFolder.UserProfile);
    public bool ConfirmDeletes { get; set; } = true;
}

public sealed class AppSettingsStore
{
    private readonly string _path;
    private readonly SemaphoreSlim _gate = new(1, 1);
    private static readonly JsonSerializerOptions JsonOptions = new() { WriteIndented = true, PropertyNamingPolicy = JsonNamingPolicy.CamelCase };

    public AppSettingsStore(string path) => _path = Path.GetFullPath(path);

    public async Task<AppSettings> LoadAsync(CancellationToken cancellationToken = default)
    {
        await _gate.WaitAsync(cancellationToken).ConfigureAwait(false);
        try
        {
            if (!File.Exists(_path)) return new AppSettings();
            try
            {
                await using var stream = File.OpenRead(_path);
                var settings = await JsonSerializer.DeserializeAsync<AppSettings>(stream, JsonOptions, cancellationToken).ConfigureAwait(false) ?? new AppSettings();
                if (!Directory.Exists(settings.LastLocalDirectory))
                    settings.LastLocalDirectory = Environment.GetFolderPath(Environment.SpecialFolder.UserProfile);
                return settings;
            }
            catch (JsonException)
            {
                return new AppSettings();
            }
        }
        finally { _gate.Release(); }
    }

    public async Task SaveAsync(AppSettings settings, CancellationToken cancellationToken = default)
    {
        ArgumentNullException.ThrowIfNull(settings);
        await _gate.WaitAsync(cancellationToken).ConfigureAwait(false);
        try
        {
            Directory.CreateDirectory(Path.GetDirectoryName(_path)!);
            var temp = _path + ".tmp-" + Guid.NewGuid().ToString("N");
            try
            {
                await using (var stream = new FileStream(temp, FileMode.CreateNew, FileAccess.Write, FileShare.None, 16 * 1024, FileOptions.Asynchronous | FileOptions.WriteThrough))
                {
                    await JsonSerializer.SerializeAsync(stream, settings, JsonOptions, cancellationToken).ConfigureAwait(false);
                    await stream.FlushAsync(cancellationToken).ConfigureAwait(false);
                }
                File.Move(temp, _path, true);
            }
            finally
            {
                try { if (File.Exists(temp)) File.Delete(temp); } catch { }
            }
        }
        finally { _gate.Release(); }
    }
}
