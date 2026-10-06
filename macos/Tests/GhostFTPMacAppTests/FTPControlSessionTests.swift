import XCTest
@testable import GhostFTPMacApp

final class FTPControlSessionTests: XCTestCase {
    func testSingleReplyIsParsedAndConsumed() throws {
        var buffer = Data("220 Service ready\r\n".utf8)

        let reply = try XCTUnwrap(FTPControlCodec.takeReply(from: &buffer))

        XCTAssertEqual(reply.code, 220)
        XCTAssertEqual(reply.lines, ["220 Service ready"])
        XCTAssertTrue(buffer.isEmpty)
    }

    func testMultilineReplyWaitsForMatchingTerminator() throws {
        var buffer = Data(
            "211-Features\r\n UTF8\r\n MLST type*;size*;modify*;\r\n211 End\r\n".utf8
        )

        let reply = try XCTUnwrap(FTPControlCodec.takeReply(from: &buffer))

        XCTAssertEqual(reply.code, 211)
        XCTAssertEqual(reply.lines.count, 4)
        XCTAssertEqual(reply.lines.last, "211 End")
        XCTAssertTrue(buffer.isEmpty)
    }

    func testPartialReplyIsBufferedUntilCRLFArrives() throws {
        var buffer = Data("220 Service".utf8)

        XCTAssertNil(try FTPControlCodec.takeReply(from: &buffer))
        XCTAssertEqual(String(decoding: buffer, as: UTF8.self), "220 Service")

        buffer.append(Data(" ready\r\n".utf8))
        let reply = try XCTUnwrap(FTPControlCodec.takeReply(from: &buffer))

        XCTAssertEqual(reply.code, 220)
        XCTAssertTrue(buffer.isEmpty)
    }

    func testCoalescedRepliesAreConsumedOneAtATime() throws {
        var buffer = Data("220 Ready\r\n230 Logged in\r\n".utf8)

        let first = try XCTUnwrap(FTPControlCodec.takeReply(from: &buffer))
        let second = try XCTUnwrap(FTPControlCodec.takeReply(from: &buffer))

        XCTAssertEqual(first.code, 220)
        XCTAssertEqual(second.code, 230)
        XCTAssertTrue(buffer.isEmpty)
    }

    func testWorkingDirectoryParsesEscapedQuotes() throws {
        let reply = FTPReply(
            code: 257,
            lines: ["257 \"/deploy/\"\"current\"\"\" is the current directory"]
        )

        let path = try FTPControlCodec.workingDirectory(from: reply)

        XCTAssertEqual(path, "/deploy/\"current\"")
    }

    func testWorkingDirectoryRejectsMalformedPWDReply() {
        let reply = FTPReply(code: 257, lines: ["257 no quoted path"])

        XCTAssertThrowsError(try FTPControlCodec.workingDirectory(from: reply))
    }

    func testCommandArgumentsRejectLineInjection() {
        for value in ["user\r\nDELE /", "bad\nname", "bad\0name"] {
            XCTAssertThrowsError(try FTPControlCodec.validateCommandArgument(value))
        }

        XCTAssertNoThrow(try FTPControlCodec.validateCommandArgument("/safe path"))
    }


    func testExtendedPassivePortParsesEPSVReply() throws {
        let reply = FTPReply(
            code: 229,
            lines: ["229 Entering Extended Passive Mode (|||6446|)"]
        )

        XCTAssertEqual(try FTPControlCodec.extendedPassivePort(from: reply), 6446)
    }

    func testExtendedPassivePortRejectsInvalidReply() {
        let reply = FTPReply(code: 229, lines: ["229 Entering Extended Passive Mode (|||0|)"])

        XCTAssertThrowsError(try FTPControlCodec.extendedPassivePort(from: reply))
    }

    func testMLSDParserBuildsTypedSortedEntries() throws {
        let data = Data(
            [
                "type=file;size=25;modify=20261005120000; zebra.txt",
                "type=dir;modify=20261005115900; assets",
                "type=file;size=3;modify=20261005115800; Alpha.txt",
                "type=cdir;modify=20261005115700; .",
                "type=pdir;modify=20261005115600; ..",
            ]
            .joined(separator: "\r\n")
            .appending("\r\n")
            .utf8
        )

        let entries = try FTPControlCodec.parseMLSD(data)

        XCTAssertEqual(entries.map(\.name), ["assets", "Alpha.txt", "zebra.txt"])
        XCTAssertTrue(entries[0].isDirectory)
        XCTAssertEqual(entries[1].size, 3)
        XCTAssertEqual(entries[2].modified, "20261005120000")
    }

    func testMLSDParserRejectsNonUTF8Payload() {
        let data = Data([0xFF, 0xFE, 0xFD])

        XCTAssertThrowsError(try FTPControlCodec.parseMLSD(data))
    }


    func testDirectoryListingBufferRejectsOversizedPayload() throws {
        var output = Data(repeating: 0x41, count: 7)

        XCTAssertNoThrow(
            try FTPControlCodec.appendListingChunk(
                Data([0x42]),
                to: &output,
                maximumBytes: 8
            )
        )
        XCTAssertEqual(output.count, 8)

        XCTAssertThrowsError(
            try FTPControlCodec.appendListingChunk(
                Data([0x43]),
                to: &output,
                maximumBytes: 8
            )
        )
        XCTAssertEqual(output.count, 8)
    }
}
