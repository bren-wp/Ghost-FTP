using System.Net;

namespace GhostFTP.Core.Protocol;

public static class InputGuard
{
    public static string Host(string value)
    {
        value = (value ?? string.Empty).Trim();
        RejectControl(value, nameof(value));
        if (value.Length is < 1 or > 253)
            throw new ArgumentException("Host must contain between 1 and 253 characters.");
        if (Uri.CheckHostName(value) == UriHostNameType.Unknown && !IPAddress.TryParse(value, out _))
            throw new ArgumentException("Host name or IP address is invalid.");
        return value;
    }

    public static int Port(int value)
    {
        if (value is < 1 or > 65535)
            throw new ArgumentOutOfRangeException(nameof(value), "Port must be between 1 and 65535.");
        return value;
    }

    public static string CommandArgument(string value, string parameterName)
    {
        value ??= string.Empty;
        RejectControl(value, parameterName);
        if (value.Length > 4096)
            throw new ArgumentException("FTP command argument is too long.", parameterName);
        return value;
    }


    public static string RemoteName(string value)
    {
        value = (value ?? string.Empty).Trim();
        RejectControl(value, nameof(value));
        if (value.Length is < 1 or > 255)
            throw new ArgumentException("Remote item name must contain between 1 and 255 characters.", nameof(value));
        if (value is "." or ".." || value.Contains('/') || value.Contains('\\'))
            throw new ArgumentException("Remote item name must be a single file or folder name.", nameof(value));
        return value;
    }

    public static string RemotePath(string value)
    {
        value = string.IsNullOrWhiteSpace(value) ? "/" : value.Trim();
        RejectControl(value, nameof(value));
        if (value.Length > 4096)
            throw new ArgumentException("Remote path is too long.", nameof(value));

        value = value.Replace('\\', '/');
        var segments = new List<string>();
        foreach (var segment in value.Split('/', StringSplitOptions.RemoveEmptyEntries))
        {
            if (segment == ".")
                continue;
            if (segment == "..")
            {
                if (segments.Count > 0) segments.RemoveAt(segments.Count - 1);
                continue;
            }
            CommandArgument(segment, nameof(value));
            segments.Add(segment);
        }

        return segments.Count == 0 ? "/" : "/" + string.Join('/', segments);
    }

    public static void RejectControl(string value, string parameterName)
    {
        if (value.IndexOfAny(['\r', '\n', '\0']) >= 0)
            throw new ArgumentException("Control characters are not permitted.", parameterName);
    }
}
