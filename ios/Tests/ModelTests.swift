import Foundation

@main
struct ModelTests {
    static func main() throws {
        try expect(ConnectionConfig.normalizeHost(" [2001:db8::1] ") == "2001:db8::1", "IPv6 brackets were not normalized")

        let config = try ConnectionConfig.make(
            protocolKind: .ftpsImplicit,
            host: "ftp.example.com",
            port: "",
            username: "account@example.com",
            password: "secret"
        )
        try expect(config.port == 990, "implicit FTPS default port changed")

        try expect(RemotePath.child("/", "public_html") == "/public_html", "root child path is wrong")
        try expect(RemotePath.child("/public_html", "index.html") == "/public_html/index.html", "nested child path is wrong")
        try expect(RemotePath.parent("/public_html/site") == "/public_html", "parent path is wrong")

        try expect(FTPPathMapper.map(loginRoot: "/home/account", uiPath: "/") == "/home/account", "login root mapping changed")
        try expect(FTPPathMapper.map(loginRoot: "/home/account", uiPath: "/public_html") == "/home/account/public_html", "shared-hosting mapping changed")
        try expect(FTPPathMapper.map(loginRoot: nil, uiPath: "/public_html") == "public_html", "login-relative fallback changed")

        try expectThrows("path traversal was accepted") {
            _ = try RemotePath.normalizeDirectory("/public_html/../etc")
        }
        try expectThrows("unsafe server login root was accepted") {
            _ = try FTPPathMapper.normalizeLoginRoot("/home/../root")
        }
        try expectThrows("CRLF username injection was accepted") {
            _ = try ConnectionConfig.make(protocolKind: .ftp, host: "example.com", port: "21", username: "user\r\nDELE /", password: "x")
        }
        try expectThrows("CRLF password injection was accepted") {
            _ = try ConnectionConfig.make(protocolKind: .ftp, host: "example.com", port: "21", username: "user", password: "x\r\nQUIT")
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
