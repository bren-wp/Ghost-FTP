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
    private static void GuardTraversal(int depth, TraversalBudget budget)
    {
        if (depth > MaxTraversalDepth)
            throw new IOException($"Remote directory depth exceeds the safety limit of {MaxTraversalDepth} levels.");
        if (++budget.Entries > MaxTraversalEntries)
            throw new IOException($"Remote traversal exceeds the safety limit of {MaxTraversalEntries:N0} entries.");
    }

    [GeneratedRegex("\\\"(?<path>(?:[^\\\"]|\\\"\\\")*)\\\"")]
    private static partial Regex PwdRegex();

    [GeneratedRegex(@"\(.*\|(?<port>\d+)\|\)")]
    private static partial Regex EpsvRegex();

    [GeneratedRegex(@"\d+")]
    private static partial Regex PasvNumberRegex();
}
