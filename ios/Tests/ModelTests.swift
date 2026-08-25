import Foundation

@main
struct ModelTests {
    static func main() throws {
        let normalizedHost = try ConnectionConfig.normalizeHost(" [2001:db8::1] ")
        try expect(normalizedHost == "2001:db8::1", "IPv6 brackets were not normalized")

        let config = try ConnectionConfig.make(
            protocolKind: .ftpsImplicit,
            host: "ftp.example.com",
            port: "",
            username: "account@example.com",
            password: "secret"
        )
        try expect(config.port == 990, "implicit FTPS default port changed")

        let rootChild = try RemotePath.child("/", "public_html")
        let nestedChild = try RemotePath.child("/public_html", "index.html")
        let parent = try RemotePath.parent("/public_html/site")
        try expect(rootChild == "/public_html", "root child path is wrong")
        try expect(nestedChild == "/public_html/index.html", "nested child path is wrong")
        try expect(parent == "/public_html", "parent path is wrong")

        let mappedRoot = try FTPPathMapper.map(loginRoot: "/home/account", uiPath: "/")
        let mappedWebRoot = try FTPPathMapper.map(loginRoot: "/home/account", uiPath: "/public_html")
        let relativeFallback = try FTPPathMapper.map(loginRoot: nil, uiPath: "/public_html")
        try expect(mappedRoot == "/home/account", "login root mapping changed")
        try expect(mappedWebRoot == "/home/account/public_html", "shared-hosting mapping changed")
        try expect(relativeFallback == "public_html", "login-relative fallback changed")

        try expectThrows("path traversal was accepted") {
            _ = try RemotePath.normalizeDirectory("/public_html/../etc")
        }
        try expectThrows("unsafe server login root was accepted") {
            _ = try FTPPathMapper.normalizeLoginRoot("/home/../root")
        }

        let cr = String(UnicodeScalar(13)!)
        let lf = String(UnicodeScalar(10)!)
        let crlf = cr + lf
        try expectThrows("CRLF username injection was accepted") {
            _ = try ConnectionConfig.make(protocolKind: .ftp, host: "example.com", port: "21", username: "user" + crlf + "NEXT", password: "x")
        }
        try expectThrows("CRLF password injection was accepted") {
            _ = try ConnectionConfig.make(protocolKind: .ftp, host: "example.com", port: "21", username: "user", password: "x" + crlf + "NEXT")
        }
        let nul = String(UnicodeScalar(0)!)
        try expectThrows("NUL username injection was accepted") {
            _ = try ConnectionConfig.make(protocolKind: .ftp, host: "example.com", port: "21", username: "user" + nul + "NEXT", password: "x")
        }

        print("IOS_MODEL_TESTS=PASS")
    }

    private static func expect(_ condition: @autoclosure () -> Bool, _ message: String) throws {
        if !condition() { throw TestFailure(message) }
    }

    private static func expectThrows(_ message: String, _ operation: () throws -> Void) throws {
        do {
            try operation()
            throw TestFailure(message)
        } catch is ValidationError {
            return
        }
    }
}

private struct TestFailure: Error, CustomStringConvertible {
    let message: String
    init(_ message: String) { self.message = message }
    var description: String { message }
}
