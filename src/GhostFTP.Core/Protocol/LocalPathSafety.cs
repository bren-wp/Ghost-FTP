namespace GhostFTP.Core.Protocol;

public static class LocalPathSafety
{
    private static readonly HashSet<string> ReservedNames = new(StringComparer.OrdinalIgnoreCase)
    {
        "CON", "PRN", "AUX", "NUL",
        "COM1", "COM2", "COM3", "COM4", "COM5", "COM6", "COM7", "COM8", "COM9",
        "LPT1", "LPT2", "LPT3", "LPT4", "LPT5", "LPT6", "LPT7", "LPT8", "LPT9"
    };

    public static string SafeFileName(string remoteName)
    {
        InputGuard.CommandArgument(remoteName, nameof(remoteName));
        var invalid = Path.GetInvalidFileNameChars();
        var chars = remoteName.Select(ch => invalid.Contains(ch) || char.IsControl(ch) ? '_' : ch).ToArray();
        var result = new string(chars).Trim().TrimEnd('.');
        if (string.IsNullOrWhiteSpace(result) || result is "." or "..")
            result = "unnamed";

        var stem = Path.GetFileNameWithoutExtension(result);
        if (ReservedNames.Contains(stem))
            result = "_" + result;
        return result;
    }

    public static string CombineUnderRoot(string root, string childName)
    {
        var rootFull = Path.GetFullPath(root);
        var candidate = Path.GetFullPath(Path.Combine(rootFull, SafeFileName(childName)));
        var prefix = rootFull.EndsWith(Path.DirectorySeparatorChar) ? rootFull : rootFull + Path.DirectorySeparatorChar;
        if (!candidate.StartsWith(prefix, StringComparison.OrdinalIgnoreCase) && !string.Equals(candidate, rootFull, StringComparison.OrdinalIgnoreCase))
            throw new IOException("Resolved path escapes the selected local directory.");
        return candidate;
    }
}
