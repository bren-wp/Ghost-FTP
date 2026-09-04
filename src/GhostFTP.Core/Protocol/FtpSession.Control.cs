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
    private async Task<FtpReply> SendCommandAsync(string command, CancellationToken cancellationToken, bool redactArgument = false)
    {
        EnsureConnectedTransport();
        InputGuard.RejectControl(command, nameof(command));
        if (command.Length > 8192)
            throw new ArgumentException("FTP command is too long.", nameof(command));
        _ = redactArgument; // Intentionally no command logging exists; parameter documents secret handling.
        await _writer!.WriteLineAsync(command.AsMemory(), cancellationToken).WaitAsync(_options.CommandTimeout, cancellationToken).ConfigureAwait(false);
        return await ReadReplyAsync(cancellationToken).ConfigureAwait(false);
    }

    private async Task<FtpReply?> TryCommandAsync(string command, CancellationToken cancellationToken)
    {
        try { return await SendCommandAsync(command, cancellationToken).ConfigureAwait(false); }
        catch (FtpException) { return null; }
    }

    private async Task<FtpReply> ReadReplyAsync(CancellationToken cancellationToken)
    {
        EnsureConnectedTransport();
        var first = await _reader!.ReadLineAsync(cancellationToken).AsTask().WaitAsync(_options.CommandTimeout, cancellationToken).ConfigureAwait(false);
        if (first is null)
            throw new FtpException("FTP server closed the control connection unexpectedly.");
        if (first.Length > MaxReplyLineChars)
            throw new FtpException("FTP response line exceeded safe parsing limits.");
        if (first.Length < 3 || !int.TryParse(first.AsSpan(0, 3), NumberStyles.None, CultureInfo.InvariantCulture, out var code))
            throw new FtpException("FTP server returned a malformed response.");

        var lines = new List<string> { first };
        var charCount = first.Length;
        if (first.Length >= 4 && first[3] == '-')
        {
            var terminator = code.ToString("000", CultureInfo.InvariantCulture) + " ";
            while (true)
            {
                if (lines.Count >= MaxReplyLines || charCount >= MaxReplyChars)
                    throw new FtpException("FTP response exceeded safe parsing limits.", code);
                var line = await _reader.ReadLineAsync(cancellationToken).AsTask().WaitAsync(_options.CommandTimeout, cancellationToken).ConfigureAwait(false);
                if (line is null)
                    throw new FtpException("FTP server closed a multiline response unexpectedly.", code);
                if (line.Length > MaxReplyLineChars)
                    throw new FtpException("FTP response line exceeded safe parsing limits.", code);
                lines.Add(line);
                charCount += line.Length;
                if (line.StartsWith(terminator, StringComparison.Ordinal))
                    break;
            }
        }

        var message = lines[^1].Length > 4 ? lines[^1][4..] : lines[^1];
        return new FtpReply(code, message, lines);
    }

    private async Task ResetTransportAsync()
    {
        IsConnected = false;
        IsEncrypted = false;
        _dataProtection = false;
        _features.Clear();
        WorkingDirectory = "/";
        try { _writer?.Dispose(); } catch { }
        try { _reader?.Dispose(); } catch { }
        if (_controlStream is not null)
        {
            try { await _controlStream.DisposeAsync().ConfigureAwait(false); } catch { }
        }
        try { _controlClient?.Dispose(); } catch { }
        _writer = null;
        _reader = null;
        _controlStream = null;
        _controlClient = null;
    }

    private async Task LockedAsync(Func<CancellationToken, Task> action, CancellationToken cancellationToken)
    {
        await _gate.WaitAsync(cancellationToken).ConfigureAwait(false);
        try { ThrowIfDisposed(); EnsureConnected(); await action(cancellationToken).ConfigureAwait(false); }
        finally { _gate.Release(); }
    }

    private async Task<T> LockedAsync<T>(Func<CancellationToken, Task<T>> action, CancellationToken cancellationToken)
    {
        await _gate.WaitAsync(cancellationToken).ConfigureAwait(false);
        try { ThrowIfDisposed(); EnsureConnected(); return await action(cancellationToken).ConfigureAwait(false); }
        finally { _gate.Release(); }
    }

    private void EnsureConnected()
    {
        if (!IsConnected)
            throw new InvalidOperationException("FTP session is not connected.");
        EnsureConnectedTransport();
    }

    private void EnsureConnectedTransport()
    {
        ThrowIfDisposed();
        if (_controlClient is null || _controlStream is null || _reader is null || _writer is null || !_controlClient.Connected)
            throw new InvalidOperationException("FTP control connection is unavailable.");
    }

    private static void Ensure(FtpReply reply, int minCode, int maxCode, string message)
    {
        if (reply.Code < minCode || reply.Code > maxCode)
            throw CreateReplyException(reply, message);
    }

    private static FtpException CreateReplyException(FtpReply reply, string message) =>
        new($"{message} Server response: {reply.Code} {reply.Message}", reply.Code);

    private static TimeSpan Clamp(TimeSpan value, TimeSpan min, TimeSpan max) => value < min ? min : value > max ? max : value;
    private static void TryDeleteLocal(string path) { try { if (File.Exists(path)) File.Delete(path); } catch { } }

    private void ThrowIfDisposed()
    {
        ObjectDisposedException.ThrowIf(_disposed, this);
    }

    public async ValueTask DisposeAsync()
    {
        if (_disposed)
            return;
        await _gate.WaitAsync().ConfigureAwait(false);
        try
        {
            if (_disposed)
                return;
            _disposed = true;
            await ResetTransportAsync().ConfigureAwait(false);
        }
        finally
        {
            _gate.Release();
            _gate.Dispose();
        }
    }

    private sealed class TraversalBudget
    {
        public int Entries;
    }

}
