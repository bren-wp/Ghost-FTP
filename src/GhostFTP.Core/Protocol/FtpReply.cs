namespace GhostFTP.Core.Protocol;

public sealed record FtpReply(int Code, string Message, IReadOnlyList<string> Lines)
{
    public bool IsPositivePreliminary => Code is >= 100 and < 200;
    public bool IsPositiveCompletion => Code is >= 200 and < 300;
    public bool IsPositiveIntermediate => Code is >= 300 and < 400;
    public bool IsTransientError => Code is >= 400 and < 500;
    public bool IsPermanentError => Code >= 500;

    public override string ToString() => $"{Code} {Message}";
}
