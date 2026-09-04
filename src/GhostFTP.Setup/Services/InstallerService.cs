using Microsoft.Win32;
using System.Diagnostics;
using System.Reflection;
using System.Runtime.InteropServices;

namespace GhostFTP.Setup.Services;

internal sealed class InstallerService
{
    public string InstallDirectory { get; } = Path.Combine(Environment.GetFolderPath(Environment.SpecialFolder.LocalApplicationData), "Programs", "GhostFTP");
    public string AppPath => Path.Combine(InstallDirectory, "GhostFTP.exe");
    public string UninstallerPath => Path.Combine(InstallDirectory, "GhostFTP-Uninstall.exe");
    public bool IsInstalled => File.Exists(AppPath);

    public async Task InstallAsync(bool desktopShortcut, CancellationToken cancellationToken)
    {
        Directory.CreateDirectory(InstallDirectory);
        var tempApp = Path.Combine(InstallDirectory, "GhostFTP.exe.new");
        await ExtractPayloadAsync(tempApp, cancellationToken).ConfigureAwait(false);

        if (File.Exists(AppPath))
        {
            try { File.Move(tempApp, AppPath, true); }
            catch (IOException ex) { throw new IOException("GhostFTP appears to be running. Close the app and run setup again.", ex); }
        }
        else File.Move(tempApp, AppPath);

        var currentSetup = Environment.ProcessPath ?? throw new InvalidOperationException("Setup executable path is unavailable.");
        if (!string.Equals(currentSetup, UninstallerPath, StringComparison.OrdinalIgnoreCase))
            File.Copy(currentSetup, UninstallerPath, true);

        var startMenu = Path.Combine(Environment.GetFolderPath(Environment.SpecialFolder.Programs), "GhostFTP.lnk");
        ShellLink.Create(startMenu, AppPath, null, InstallDirectory, "GhostFTP FTP/FTPS client", AppPath);
        var desktop = Path.Combine(Environment.GetFolderPath(Environment.SpecialFolder.DesktopDirectory), "GhostFTP.lnk");
        if (desktopShortcut)
            ShellLink.Create(desktop, AppPath, null, InstallDirectory, "GhostFTP FTP/FTPS client", AppPath);
        else TryDelete(desktop);

        WriteUninstallRegistry();
    }

    public Task UninstallAsync(bool removeUserData, CancellationToken cancellationToken)
    {
        cancellationToken.ThrowIfCancellationRequested();
        TryDelete(Path.Combine(Environment.GetFolderPath(Environment.SpecialFolder.Programs), "GhostFTP.lnk"));
        TryDelete(Path.Combine(Environment.GetFolderPath(Environment.SpecialFolder.DesktopDirectory), "GhostFTP.lnk"));
        TryDelete(AppPath);
        TryDelete(Path.Combine(InstallDirectory, "GhostFTP.exe.new"));
        using (var root = Registry.CurrentUser.OpenSubKey(@"Software\Microsoft\Windows\CurrentVersion\Uninstall", writable: true))
            root?.DeleteSubKeyTree("GhostFTP", throwOnMissingSubKey: false);

        if (removeUserData)
        {
            var data = Path.Combine(Environment.GetFolderPath(Environment.SpecialFolder.LocalApplicationData), "GhostFTP");
            TryDeleteDirectory(data);
        }

        var current = Environment.ProcessPath;
        if (!string.IsNullOrWhiteSpace(current) && File.Exists(current))
            _ = MoveFileEx(current, null, 0x4); // delete at reboot because the running uninstaller cannot delete itself
        return Task.CompletedTask;
    }

    public void LaunchApp()
    {
        if (File.Exists(AppPath))
            Process.Start(new ProcessStartInfo(AppPath) { UseShellExecute = true, WorkingDirectory = InstallDirectory });
    }

    private async Task ExtractPayloadAsync(string targetPath, CancellationToken cancellationToken)
    {
        var assembly = Assembly.GetExecutingAssembly();
        await using var input = assembly.GetManifestResourceStream("GhostFTP.PortablePayload.exe")
            ?? throw new InvalidOperationException("GhostFTP application payload is missing from this setup build.");
        await using var output = new FileStream(targetPath, FileMode.Create, FileAccess.Write, FileShare.None, 128 * 1024, FileOptions.Asynchronous | FileOptions.WriteThrough);
        await input.CopyToAsync(output, 128 * 1024, cancellationToken).ConfigureAwait(false);
        await output.FlushAsync(cancellationToken).ConfigureAwait(false);
    }

    private void WriteUninstallRegistry()
    {
        using var root = Registry.CurrentUser.CreateSubKey(@"Software\Microsoft\Windows\CurrentVersion\Uninstall\GhostFTP", writable: true);
        root.SetValue("DisplayName", "GhostFTP");
        root.SetValue("DisplayVersion", typeof(InstallerService).Assembly.GetName().Version?.ToString(3) ?? "1.1.0");
        root.SetValue("Publisher", "Brendigo");
        root.SetValue("URLInfoAbout", "https://ghostftp.com");
        root.SetValue("HelpLink", "https://ghostftp.com");
        root.SetValue("InstallLocation", InstallDirectory);
        root.SetValue("DisplayIcon", AppPath);
        root.SetValue("UninstallString", $"\"{UninstallerPath}\" --uninstall");
        root.SetValue("NoModify", 1, RegistryValueKind.DWord);
        root.SetValue("NoRepair", 1, RegistryValueKind.DWord);
    }

    private static void TryDelete(string path) { try { if (File.Exists(path)) File.Delete(path); } catch { } }
    private static void TryDeleteDirectory(string path) { try { if (Directory.Exists(path)) Directory.Delete(path, true); } catch { } }

    [DllImport("kernel32.dll", SetLastError = true, CharSet = CharSet.Unicode)]
    [return: MarshalAs(UnmanagedType.Bool)]
    private static extern bool MoveFileEx(string lpExistingFileName, string? lpNewFileName, int dwFlags);
}
