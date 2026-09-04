namespace GhostFTP.Services;

public sealed class AppPaths
{
    public bool IsPortable { get; }
    public string ExecutableDirectory { get; }
    public string DataDirectory { get; }
    public string ProfilesFile => Path.Combine(DataDirectory, "profiles.json");
    public string SettingsFile => Path.Combine(DataDirectory, "settings.json");

    public AppPaths()
    {
        var executable = Environment.ProcessPath ?? AppContext.BaseDirectory;
        ExecutableDirectory = Directory.Exists(executable) ? executable : Path.GetDirectoryName(executable) ?? AppContext.BaseDirectory;
        var name = Path.GetFileNameWithoutExtension(executable);
        IsPortable = name.Contains("Portable", StringComparison.OrdinalIgnoreCase) || File.Exists(Path.Combine(ExecutableDirectory, "portable.flag"));
        DataDirectory = IsPortable
            ? Path.Combine(ExecutableDirectory, "Data")
            : Path.Combine(Environment.GetFolderPath(Environment.SpecialFolder.LocalApplicationData), "GhostFTP");
        Directory.CreateDirectory(DataDirectory);
    }
}
