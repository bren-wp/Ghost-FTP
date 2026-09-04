namespace GhostFTP.Core.Models;

public sealed record FtpEntry(
    string Name,
    string FullPath,
    bool IsDirectory,
    long Size,
    DateTimeOffset? ModifiedUtc,
    string? Permissions = null,
    string? Raw = null)
{
    public string Type => IsDirectory ? "Folder" : "File";
}
