using System.Collections.ObjectModel;
using System.Diagnostics;
using System.Threading.Channels;
using GhostFTP.Core.Models;
using GhostFTP.Core.Protocol;

namespace GhostFTP.Core.Services;

public sealed class TransferQueueService : IAsyncDisposable
{
    private sealed record Queued(TransferJob Job, CancellationTokenSource Cancellation);

    private readonly Func<CancellationToken, Task<(IFtpSession Session, bool DisposeAfter)>> _sessionFactory;
    private const int MaxQueuedTransfers = 4096;
    private readonly Channel<Queued> _channel = Channel.CreateBounded<Queued>(new BoundedChannelOptions(MaxQueuedTransfers)
    {
        SingleReader = true,
        SingleWriter = false,
        FullMode = BoundedChannelFullMode.Wait
    });
    private readonly CancellationTokenSource _shutdown = new();
    private readonly Dictionary<Guid, CancellationTokenSource> _cancellations = new();
    private readonly object _sync = new();
    private readonly Task _worker;
    private readonly SynchronizationContext? _uiContext;

    public ObservableCollection<TransferJob> Jobs { get; } = [];
    public event EventHandler<TransferJob>? JobUpdated;

    public TransferQueueService(Func<CancellationToken, Task<(IFtpSession Session, bool DisposeAfter)>> sessionFactory, SynchronizationContext? uiContext = null)
    {
        _sessionFactory = sessionFactory ?? throw new ArgumentNullException(nameof(sessionFactory));
        _uiContext = uiContext;
        _worker = Task.Run(WorkerAsync);
    }

    public TransferJob EnqueueUpload(string source, string destination, bool isDirectory)
    {
        var job = new TransferJob { Direction = TransferDirection.Upload, Source = source, Destination = destination, IsDirectory = isDirectory };
        Enqueue(job);
        return job;
    }

    public TransferJob EnqueueDownload(string source, string destination, bool isDirectory, long? totalBytes = null)
    {
        var job = new TransferJob { Direction = TransferDirection.Download, Source = source, Destination = destination, IsDirectory = isDirectory, TotalBytes = totalBytes };
        Enqueue(job);
        return job;
    }

    public void Cancel(Guid jobId)
    {
        lock (_sync)
        {
            if (_cancellations.TryGetValue(jobId, out var cts))
                cts.Cancel();
        }
    }

    public void ClearFinished()
    {
        var finished = Jobs.Where(x => x.State is TransferState.Completed or TransferState.Cancelled or TransferState.Failed).ToArray();
        foreach (var job in finished)
            Jobs.Remove(job);
    }

    private void Enqueue(TransferJob job)
    {
        var cts = CancellationTokenSource.CreateLinkedTokenSource(_shutdown.Token);
        lock (_sync) _cancellations[job.Id] = cts;
        Jobs.Add(job);
        if (!_channel.Writer.TryWrite(new Queued(job, cts)))
        {
            Jobs.Remove(job);
            lock (_sync) _cancellations.Remove(job.Id);
            cts.Dispose();
            throw new InvalidOperationException($"Transfer queue is full. Maximum queued transfers: {MaxQueuedTransfers:N0}.");
        }
        JobUpdated?.Invoke(this, job);
    }

    private async Task WorkerAsync()
    {
        try
        {
            await foreach (var queued in _channel.Reader.ReadAllAsync(_shutdown.Token).ConfigureAwait(false))
            {
                var job = queued.Job;
                var ct = queued.Cancellation.Token;
                IFtpSession? transferSession = null;
                var disposeAfter = false;
                try
                {
                    ct.ThrowIfCancellationRequested();
                    var lease = await _sessionFactory(ct).ConfigureAwait(false);
                    transferSession = lease.Session;
                    disposeAfter = lease.DisposeAfter;
                    if (!transferSession.IsConnected)
                        throw new InvalidOperationException("Transfer session is not connected.");

                    Ui(() => job.State = TransferState.Running, job);
                    var stopwatch = Stopwatch.StartNew();
                    long lastBytes = 0;
                    var lastTime = TimeSpan.Zero;
                    var progress = new Progress<(long transferred, long? total)>(p =>
                    {
                        var total = p.total ?? job.TotalBytes;
                        var elapsed = stopwatch.Elapsed;
                        var deltaSeconds = (elapsed - lastTime).TotalSeconds;
                        var speed = deltaSeconds >= 0.5 ? (p.transferred - lastBytes) / deltaSeconds : -1;
                        if (deltaSeconds >= 0.5)
                        {
                            lastBytes = p.transferred;
                            lastTime = elapsed;
                        }
                        Ui(() =>
                        {
                            job.BytesTransferred = p.transferred;
                            if (total is > 0) job.Progress = p.transferred * 100d / total.Value;
                            if (speed >= 0) job.SpeedBytesPerSecond = speed;
                        }, job);
                    });

                    if (job.Direction == TransferDirection.Upload)
                    {
                        if (job.IsDirectory) await transferSession.UploadDirectoryAsync(job.Source, job.Destination, progress, ct).ConfigureAwait(false);
                        else await transferSession.UploadFileAsync(job.Source, job.Destination, progress, ct).ConfigureAwait(false);
                    }
                    else
                    {
                        if (job.IsDirectory) await transferSession.DownloadDirectoryAsync(job.Source, job.Destination, progress, ct).ConfigureAwait(false);
                        else await transferSession.DownloadFileAsync(job.Source, job.Destination, progress, ct).ConfigureAwait(false);
                    }

                    Ui(() => { job.Progress = 100; job.State = TransferState.Completed; }, job);
                }
                catch (OperationCanceledException)
                {
                    Ui(() => job.State = TransferState.Cancelled, job);
                }
                catch (Exception ex)
                {
                    Ui(() => { job.Error = ex.Message; job.State = TransferState.Failed; }, job);
                }
                finally
                {
                    if (disposeAfter && transferSession is not null)
                    {
                        try { await transferSession.DisposeAsync().ConfigureAwait(false); } catch { }
                    }
                    Ui(() => { }, job);
                    lock (_sync) _cancellations.Remove(job.Id);
                    queued.Cancellation.Dispose();
                }
            }
        }
        catch (OperationCanceledException) when (_shutdown.IsCancellationRequested)
        {
        }
    }

    private void Ui(Action update, TransferJob job)
    {
        if (_uiContext is null)
        {
            update();
            JobUpdated?.Invoke(this, job);
            return;
        }
        _uiContext.Post(_ =>
        {
            update();
            JobUpdated?.Invoke(this, job);
        }, null);
    }

    public async ValueTask DisposeAsync()
    {
        _channel.Writer.TryComplete();
        _shutdown.Cancel();
        lock (_sync)
        {
            foreach (var cts in _cancellations.Values) cts.Cancel();
        }
        try { await _worker.ConfigureAwait(false); } catch (OperationCanceledException) { }
        lock (_sync)
        {
            foreach (var cts in _cancellations.Values) cts.Dispose();
            _cancellations.Clear();
        }
        _shutdown.Dispose();
    }
}
