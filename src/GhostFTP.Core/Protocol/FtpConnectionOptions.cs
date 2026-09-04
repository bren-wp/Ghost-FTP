using GhostFTP.Core.Models;

namespace GhostFTP.Core.Protocol;

public sealed class FtpConnectionOptions
{
    public required string Host { get; init; }
    public int Port { get; init; } = 21;
    public required string Username { get; init; }
    public required string Password { get; init; }
    public FtpSecurityMode Security { get; init; } = FtpSecurityMode.ExplicitTls;
    public TimeSpan ConnectTimeout { get; init; } = TimeSpan.FromSeconds(15);
    public TimeSpan CommandTimeout { get; init; } = TimeSpan.FromSeconds(30);
    public TimeSpan TransferTimeout { get; init; } = TimeSpan.FromMinutes(2);
}
