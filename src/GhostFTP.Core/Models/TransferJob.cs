using System.ComponentModel;
using System.Runtime.CompilerServices;

namespace GhostFTP.Core.Models;

public enum TransferDirection
{
    Upload,
    Download
}

public enum TransferState
{
    Queued,
    Running,
    Completed,
    Cancelled,
    Failed
}

public sealed class TransferJob : INotifyPropertyChanged
{
    private TransferState _state = TransferState.Queued;
    private double _progress;
    private long _bytesTransferred;
    private string? _error;
    private double _speedBytesPerSecond;

    public Guid Id { get; } = Guid.NewGuid();
    public TransferDirection Direction { get; init; }
    public string Source { get; init; } = string.Empty;
    public string Destination { get; init; } = string.Empty;
    public bool IsDirectory { get; init; }
    public long? TotalBytes { get; init; }
    public DateTimeOffset CreatedUtc { get; } = DateTimeOffset.UtcNow;

    public TransferState State
    {
        get => _state;
        set => Set(ref _state, value);
    }

    public double Progress
    {
        get => _progress;
        set
        {
            if (Math.Abs(_progress - value) < 0.001) return;
            _progress = Math.Clamp(value, 0, 100);
            OnPropertyChanged();
            OnPropertyChanged(nameof(ProgressText));
        }
    }

    public long BytesTransferred
    {
        get => _bytesTransferred;
        set => Set(ref _bytesTransferred, Math.Max(0, value));
    }

    public double SpeedBytesPerSecond
    {
        get => _speedBytesPerSecond;
        set
        {
            if (Math.Abs(_speedBytesPerSecond - value) < 1) return;
            _speedBytesPerSecond = Math.Max(0, value);
            OnPropertyChanged();
            OnPropertyChanged(nameof(SpeedText));
        }
    }

    public string? Error
    {
        get => _error;
        set => Set(ref _error, value);
    }

    public string DisplayName
    {
        get
        {
            var value = Source.TrimEnd(Path.DirectorySeparatorChar, Path.AltDirectorySeparatorChar, '/', '\\');
            var name = Path.GetFileName(value);
            return string.IsNullOrWhiteSpace(name) ? Source : name;
        }
    }

    public string ProgressText => $"{Progress:0}%";
    public string SpeedText => FormatBytes(SpeedBytesPerSecond) + "/s";

    public event PropertyChangedEventHandler? PropertyChanged;

    private void Set<T>(ref T field, T value, [CallerMemberName] string? propertyName = null)
    {
        if (EqualityComparer<T>.Default.Equals(field, value))
            return;
        field = value;
        PropertyChanged?.Invoke(this, new PropertyChangedEventArgs(propertyName));
    }

    private void OnPropertyChanged([CallerMemberName] string? propertyName = null) =>
        PropertyChanged?.Invoke(this, new PropertyChangedEventArgs(propertyName));

    private static string FormatBytes(double value)
    {
        string[] units = ["B", "KB", "MB", "GB", "TB"];
        var index = 0;
        while (value >= 1024 && index < units.Length - 1) { value /= 1024; index++; }
        return $"{value:0.#} {units[index]}";
    }
}
