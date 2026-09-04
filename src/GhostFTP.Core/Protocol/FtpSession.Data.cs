using System.Globalization;
using System.Net.Security;
using System.Net.Sockets;
using System.Security.Authentication;
using System.Text;
using System.Text.RegularExpressions;
using GhostFTP.Core.Models;

namespace GhostFTP.Core.Protocol;

public sealed partial class FtpSession
{
    private async Task<string> ReceiveTextDataAsync(string command, CancellationToken cancellationToken)
    {
        await using var memory = new MemoryStream();
        await ReceiveDataToStreamAsync(command, memory, 0, null, null, cancellationToken).ConfigureAwait(false);
        return ControlEncoding.GetString(memory.ToArray());
    }

    private async Task ReceiveDataToStreamAsync(string command, Stream destination, long initialBytes, long? total, IProgress<(long transferred, long? total)>? progress, CancellationToken cancellationToken)
    {
        EnsureConnected();
        _ = await TryCommandAsync("TYPE I", cancellationToken).ConfigureAwait(false);
        using var data = await OpenPassiveTcpAsync(cancellationToken).ConfigureAwait(false);
        var preliminary = await SendCommandAsync(command, cancellationToken).ConfigureAwait(false);
        if (!preliminary.IsPositivePreliminary && !preliminary.IsPositiveCompletion)
            throw CreateReplyException(preliminary, "FTP server refused the data transfer.");

        if (preliminary.IsPositiveCompletion)
            return;

        await using var dataStream = await CreateDataStreamAsync(data, cancellationToken).ConfigureAwait(false);
        var buffer = new byte[1024 * 128];
        long transferred = initialBytes;
        while (true)
        {
            var read = await dataStream.ReadAsync(buffer, cancellationToken).AsTask().WaitAsync(_options.TransferTimeout, cancellationToken).ConfigureAwait(false);
            if (read == 0)
                break;
            await destination.WriteAsync(buffer.AsMemory(0, read), cancellationToken).ConfigureAwait(false);
            transferred += read;
            progress?.Report((transferred, total));
        }
        await dataStream.DisposeAsync().ConfigureAwait(false);
        var final = await ReadReplyAsync(cancellationToken).ConfigureAwait(false);
        Ensure(final, 200, 299, "FTP transfer did not complete successfully.");
    }

    private async Task SendStreamAsDataAsync(string command, Stream source, long total, IProgress<(long transferred, long? total)>? progress, CancellationToken cancellationToken)
    {
        EnsureConnected();
        _ = await TryCommandAsync("TYPE I", cancellationToken).ConfigureAwait(false);
        using var data = await OpenPassiveTcpAsync(cancellationToken).ConfigureAwait(false);
        var preliminary = await SendCommandAsync(command, cancellationToken).ConfigureAwait(false);
        if (!preliminary.IsPositivePreliminary && !preliminary.IsPositiveCompletion)
            throw CreateReplyException(preliminary, "FTP server refused the upload.");
        if (preliminary.IsPositiveCompletion)
            return;

        await using var dataStream = await CreateDataStreamAsync(data, cancellationToken).ConfigureAwait(false);
        var buffer = new byte[1024 * 128];
        long transferred = 0;
        while (true)
        {
            var read = await source.ReadAsync(buffer, cancellationToken).ConfigureAwait(false);
            if (read == 0)
                break;
            await dataStream.WriteAsync(buffer.AsMemory(0, read), cancellationToken).AsTask().WaitAsync(_options.TransferTimeout, cancellationToken).ConfigureAwait(false);
            transferred += read;
            progress?.Report((transferred, total));
        }
        await dataStream.FlushAsync(cancellationToken).ConfigureAwait(false);
        await dataStream.DisposeAsync().ConfigureAwait(false);
        var final = await ReadReplyAsync(cancellationToken).ConfigureAwait(false);
        Ensure(final, 200, 299, "FTP upload did not complete successfully.");
    }

    private async Task<TcpClient> OpenPassiveTcpAsync(CancellationToken cancellationToken)
    {
        var epsv = await TryCommandAsync("EPSV", cancellationToken).ConfigureAwait(false);
        int port;
        if (epsv is not null && epsv.IsPositiveCompletion)
        {
            var match = EpsvRegex().Match(epsv.Message);
            if (!match.Success || !int.TryParse(match.Groups["port"].Value, NumberStyles.None, CultureInfo.InvariantCulture, out port))
                throw new FtpException("Server returned an invalid EPSV response.", epsv.Code);
        }
        else
        {
            var pasv = await SendCommandAsync("PASV", cancellationToken).ConfigureAwait(false);
            Ensure(pasv, 200, 299, "Server does not support passive FTP data connections.");
            var numbers = PasvNumberRegex().Matches(pasv.Message).Select(m => int.Parse(m.Value, CultureInfo.InvariantCulture)).ToArray();
            if (numbers.Length < 6)
                throw new FtpException("Server returned an invalid PASV response.", pasv.Code);
            var p1 = numbers[^2];
            var p2 = numbers[^1];
            if (p1 is < 0 or > 255 || p2 is < 0 or > 255)
                throw new FtpException("Server returned an invalid PASV port.", pasv.Code);
            port = p1 * 256 + p2;
        }

        InputGuard.Port(port);
        var client = new TcpClient(AddressFamily.InterNetworkV6) { NoDelay = true };
        client.Client.DualMode = true;
        try
        {
            // Deliberately connect data channels to the authenticated control host rather than trusting PASV host data.
            await client.ConnectAsync(_options.Host, port, cancellationToken).AsTask().WaitAsync(_options.ConnectTimeout, cancellationToken).ConfigureAwait(false);
            return client;
        }
        catch
        {
            client.Dispose();
            throw;
        }
    }

    private async Task<Stream> CreateDataStreamAsync(TcpClient client, CancellationToken cancellationToken)
    {
        var stream = client.GetStream();
        if (!_dataProtection)
            return stream;
        var ssl = new SslStream(stream, leaveInnerStreamOpen: false);
        await ssl.AuthenticateAsClientAsync(new SslClientAuthenticationOptions
        {
            TargetHost = _options.Host,
            EnabledSslProtocols = SslProtocols.Tls12 | SslProtocols.Tls13,
            CertificateRevocationCheckMode = System.Security.Cryptography.X509Certificates.X509RevocationMode.Offline
        }, cancellationToken).WaitAsync(_options.ConnectTimeout, cancellationToken).ConfigureAwait(false);
        return ssl;
    }

    private async Task UpgradeControlToTlsAsync(CancellationToken cancellationToken)
    {
        if (_controlStream is null)
            throw new InvalidOperationException("Control stream is unavailable.");
        _reader?.Dispose();
        _writer?.Dispose();
        var ssl = new SslStream(_controlStream, leaveInnerStreamOpen: false);
        await ssl.AuthenticateAsClientAsync(new SslClientAuthenticationOptions
        {
            TargetHost = _options.Host,
            EnabledSslProtocols = SslProtocols.Tls12 | SslProtocols.Tls13,
            CertificateRevocationCheckMode = System.Security.Cryptography.X509Certificates.X509RevocationMode.Offline
        }, cancellationToken).WaitAsync(_options.ConnectTimeout, cancellationToken).ConfigureAwait(false);
        _controlStream = ssl;
        IsEncrypted = true;
    }

    private void BuildControlTextStreams()
    {
        if (_controlStream is null)
            throw new InvalidOperationException("Control stream is unavailable.");
        _reader = new StreamReader(_controlStream, ControlEncoding, detectEncodingFromByteOrderMarks: false, bufferSize: 8192, leaveOpen: true);
        _writer = new StreamWriter(_controlStream, ControlEncoding, bufferSize: 8192, leaveOpen: true)
        {
            NewLine = "\r\n",
            AutoFlush = true
        };
    }

}
