using GhostFTP.Core.Protocol;

namespace GhostFTP.SelfTest;

public static class Program
{
    public static int Main()
    {
        var tests = new (string Name, Action Run)[]
        {
            ("InputGuard blocks CRLF command injection", TestCommandInjection),
            ("Remote paths normalize safely", TestRemotePath),
            ("Remote path traversal canonicalizes safely", TestRemoteTraversal),
            ("Remote names block traversal", TestRemoteName),
            ("Malicious listing names are ignored", TestMaliciousListingName),
            ("MLSD listing parser", TestMlsd),
            ("Unix LIST parser", TestUnixList),
            ("Windows LIST parser", TestWindowsList),
            ("Remote parent path", TestParent),
            ("Windows local filename safety", TestLocalName)
        };

        var failures = new List<string>();
        foreach (var test in tests)
        {
            try
            {
                test.Run();
                Console.WriteLine("PASS  " + test.Name);
            }
            catch (Exception ex)
            {
                failures.Add(test.Name + ": " + ex.Message);
                Console.WriteLine("FAIL  " + test.Name + " — " + ex.Message);
            }
        }

        Console.WriteLine();
        Console.WriteLine($"{tests.Length - failures.Count}/{tests.Length} self-tests passed.");
        if (failures.Count == 0) return 0;
        foreach (var failure in failures) Console.Error.WriteLine(failure);
        return 1;
    }

    private static void TestCommandInjection()
    {
        var blocked = false;
        try { _ = InputGuard.CommandArgument("safe\r\nDELE /important", "value"); }
        catch (ArgumentException) { blocked = true; }
        Assert(blocked, "CRLF was accepted.");
    }

    private static void TestRemotePath()
    {
        Assert(InputGuard.RemotePath("public_html\\assets") == "/public_html/assets", "Remote path normalization failed.");
    }

    private static void TestRemoteTraversal()
    {
        Assert(InputGuard.RemotePath("/public_html/assets/../index") == "/public_html/index", "Parent traversal did not canonicalize safely.");
        Assert(InputGuard.RemotePath("../../../../") == "/", "Traversal above root was not clamped to root.");
    }

    private static void TestRemoteName()
    {
        var blocked = false;
        try { _ = InputGuard.RemoteName("../escape"); }
        catch (ArgumentException) { blocked = true; }
        Assert(blocked, "Remote traversal-style name was accepted.");
    }

    private static void TestMaliciousListingName()
    {
        var text = "type=file;size=10; ../../escape.txt\r\ntype=file;size=11; safe.txt\r\n";
        var items = FtpListingParser.ParseMlsd(text, "/public_html");
        Assert(items.Count == 1 && items[0].Name == "safe.txt", "Traversal-style listing entry was not ignored.");
    }

    private static void TestMlsd()
    {
        var text = "type=dir;modify=20260904190000; assets\r\ntype=file;size=1234;modify=20260904190100; index.html\r\n";
        var items = FtpListingParser.ParseMlsd(text, "/public_html");
        Assert(items.Count == 2, "Unexpected MLSD item count.");
        Assert(items[0].IsDirectory && items[0].FullPath == "/public_html/assets", "MLSD directory parse failed.");
        Assert(!items[1].IsDirectory && items[1].Size == 1234, "MLSD file parse failed.");
    }

    private static void TestUnixList()
    {
        var text = "drwxr-xr-x 2 owner group 4096 Sep  4 20:00 assets\n-rw-r--r-- 1 owner group 512 Sep  4 2026 index.html\n";
        var items = FtpListingParser.ParseList(text, "/public_html", new DateTimeOffset(2026, 9, 4, 22, 0, 0, TimeSpan.Zero));
        Assert(items.Count == 2, "Unexpected Unix LIST item count.");
        Assert(items[0].IsDirectory, "Unix directory not recognized.");
        Assert(items[1].Size == 512, "Unix file size not parsed.");
    }

    private static void TestWindowsList()
    {
        var text = "09-04-26  08:00PM       <DIR>          assets\r\n09-04-26  08:01PM                  512 index.html\r\n";
        var items = FtpListingParser.ParseList(text, "/public_html", DateTimeOffset.UtcNow);
        Assert(items.Count == 2, "Unexpected Windows LIST item count.");
        Assert(items[0].IsDirectory, "Windows directory not recognized.");
        Assert(items[1].Size == 512, "Windows file size not parsed.");
    }

    private static void TestParent()
    {
        Assert(FtpListingParser.ParentRemote("/a/b/c") == "/a/b", "Parent path is wrong.");
        Assert(FtpListingParser.ParentRemote("/a") == "/", "Root parent path is wrong.");
    }

    private static void TestLocalName()
    {
        Assert(LocalPathSafety.SafeFileName("CON.txt").StartsWith('_'), "Reserved Windows filename was not escaped.");
        Assert(!LocalPathSafety.SafeFileName("a:b.txt").Contains(':'), "Invalid Windows character was not escaped.");
    }

    private static void Assert(bool condition, string message)
    {
        if (!condition) throw new InvalidOperationException(message);
    }
}
