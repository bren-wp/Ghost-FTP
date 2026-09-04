namespace GhostFTP.Core.Protocol;

public sealed class FtpException : IOException
{
    public int? ReplyCode { get; }

    public FtpException(string message, int? replyCode = null, Exception? innerException = null)
        : base(message, innerException)
    {
        ReplyCode = replyCode;
    }
}
