namespace GhostFTP.Core.Models;

public sealed class ServerProfile
{
    public Guid Id { get; set; } = Guid.NewGuid();
    public string Name { get; set; } = "New server";
    public string Host { get; set; } = string.Empty;
    public int Port { get; set; } = 21;
    public string Username { get; set; } = string.Empty;
    public FtpSecurityMode Security { get; set; } = FtpSecurityMode.ExplicitTls;
    public string InitialPath { get; set; } = "/";
    public bool RememberPassword { get; set; }
    public string? ProtectedPassword { get; set; }
    public bool IsDemo { get; set; }

    public ServerProfile Clone() => new()
    {
        Id = Id,
        Name = Name,
        Host = Host,
        Port = Port,
        Username = Username,
        Security = Security,
        InitialPath = InitialPath,
        RememberPassword = RememberPassword,
        ProtectedPassword = ProtectedPassword,
        IsDemo = IsDemo
    };
}
