import XCTest
@testable import GhostFTPMacApp

final class FTPControlSessionTests: XCTestCase {

    func testFTPRefusesRemoteUploadOverwriteAndAmbiguousUnicodeCollisions() throws {
        let remoteEntries = [
            FTPDirectoryEntry(name: "REPORT.txt", isDirectory: false, size: 10, modified: nil),
            FTPDirectoryEntry(name: "Račun.pdf", isDirectory: false, size: 10, modified: nil),
            FTPDirectoryEntry(name: "Documents", isDirectory: true, size: nil, modified: nil),
        ]
        for name in ["report.txt", "Račun.pdf", "Račun.pdf", "documents"] {
            XCTAssertThrowsError(
                try FTPControlCodec.validateUploadDestination(name, within: remoteEntries),
                "Refusing to overwrite remote item: \(name)"
            )
        }
        XCTAssertNoThrow(
            try FTPControlCodec.validateUploadDestination("completely-new.txt", within: remoteEntries)
        )
    }

    func testFTPSAndSFTPCannotDowngradeIntoPlaintextFTPWithoutAuthenticatedTransports() async throws {
        let session = FTPControlSession()
        for protocolKind in [ConnectionProtocol.ftps, .sftp] {
            let profile = ConnectionProfile(
                name: "Secure transport cannot silently downgrade",
                protocolKind: protocolKind,
                host: "127.0.0.1",
                port: 21,
                username: "test"
            )
            do {
                _ = try await session.connect(profile: profile, password: "never-send-to-FTP")
                XCTFail("No unsupported secure profile may fall through to plaintext FTP")
            } catch FTPControlError.unsupportedProtocol {
                // Expected until real TLS/SSH identity verification is wired.
            }
        }
    }

    func testMacUploadRejectsSymlinksAndDoesNotReadTheirTarget() throws {
        let directory = FileManager.default.temporaryDirectory.appendingPathComponent(UUID().uuidString)
        try FileManager.default.createDirectory(at: directory, withIntermediateDirectories: true)
        defer { try? FileManager.default.removeItem(at: directory) }
        let source = directory.appendingPathComponent("confidential.bin")
        let link = directory.appendingPathComponent("link.bin")
        try Data("secret".utf8).write(to: source)
        try FileManager.default.createSymbolicLink(at: link, withDestinationURL: source)
        XCTAssertThrowsError(try PrivateUploadSource.openRegularFile(at: link))
        XCTAssertEqual(try Data(contentsOf: source), Data("secret".utf8))
    }

    func testMacUploadRefusesChangedSourceAndIncompleteTransfer() throws {
        let directory = FileManager.default.temporaryDirectory.appendingPathComponent(UUID().uuidString)
        try FileManager.default.createDirectory(at: directory, withIntermediateDirectories: true)
        defer { try? FileManager.default.removeItem(at: directory) }
        let source = directory.appendingPathComponent("large-file.bin")
        try Data(repeating: 0x61, count: 8192).write(to: source)
        let (handle, original) = try PrivateUploadSource.openRegularFile(at: source)
        defer { try? handle.close() }
        XCTAssertThrowsError(try PrivateUploadSource.verifyUnchanged(handle, original: original, transmitted: 4096))
        XCTAssertNoThrow(try PrivateUploadSource.verifyUnchanged(handle, original: original, transmitted: 8192))
        try Data(repeating: 0x62, count: 16384).write(to: source)
        XCTAssertThrowsError(try PrivateUploadSource.verifyUnchanged(handle, original: original, transmitted: 8192))
    }

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

    func testControlReplyBufferCapsUntrustedServerOutput() throws {
        var buffer = Data(repeating: 0x41, count: FTPControlCodec.maximumControlReplyBytes - 1)
        XCTAssertNoThrow(try FTPControlCodec.appendControlChunk(Data([0x42]), to: &buffer))
        XCTAssertEqual(buffer.count, FTPControlCodec.maximumControlReplyBytes)
        XCTAssertThrowsError(try FTPControlCodec.appendControlChunk(Data([0x43]), to: &buffer))
        XCTAssertEqual(buffer.count, FTPControlCodec.maximumControlReplyBytes)
        buffer.append(0x43)
        XCTAssertThrowsError(try FTPControlCodec.takeReply(from: &buffer))
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
        for value in [
            "user\r\nDELE /",
            "bad\nname",
            "bad\0name",
            "tab\tname",
            "escape\u{1B}[31mname",
            "delete\u{7F}name",
            "backspace\u{8}name",
            "vertical\u{B}tab"
        ] {
            XCTAssertThrowsError(try FTPControlCodec.validateCommandArgument(value), value)
        }

        XCTAssertNoThrow(try FTPControlCodec.validateCommandArgument("/safe path"))
        XCTAssertNoThrow(try FTPControlCodec.validateCommandArgument("/računi/čćž 文件 2026.pdf"))
    }


    func testRemoteTransferNameRejectsTraversalAndCommandInjection() {
        for value in [
            "", ".", "..", "nested/file.txt", "bad\r\nDELE target",
            "file\tname", "file\u{1B}name", "file\u{7F}name"
        ] {
            XCTAssertThrowsError(try FTPControlCodec.validateRemoteFileName(value))
        }

        XCTAssertNoThrow(try FTPControlCodec.validateRemoteFileName("archive 2026.zip"))
        XCTAssertNoThrow(try FTPControlCodec.validateRemoteFileName("name\\with-backslash.txt"))
    }


    func testFTPTransferCommandsIncludeRealRemoteName() throws {
        XCTAssertEqual(
            try FTPControlCodec.transferCommand("RETR", remoteName: "Račun 2026.pdf"),
            "RETR Račun 2026.pdf"
        )
        XCTAssertEqual(
            try FTPControlCodec.transferCommand("STOR", remoteName: "backup-01.zip"),
            "STOR backup-01.zip"
        )
    }

    func testFTPTransferCommandsRejectUnsafeArguments() {
        XCTAssertThrowsError(
            try FTPControlCodec.transferCommand("DELE", remoteName: "file.txt")
        )
        XCTAssertThrowsError(
            try FTPControlCodec.transferCommand("RETR", remoteName: "../secret.txt")
        )
        XCTAssertThrowsError(
            try FTPControlCodec.transferCommand("STOR", remoteName: "file\r\nDELE /")
        )
    }

    func testDownloadStagingNameIsUniqueAndDerivedFromDestination() throws {
        let first = UUID(uuidString: "11111111-2222-3333-4444-555555555555")!
        let second = UUID(uuidString: "AAAAAAAA-BBBB-CCCC-DDDD-EEEEEEEEEEEE")!
        let a = try FTPControlCodec.temporaryDownloadFilename(for: "invoice.pdf", identifier: first)
        let b = try FTPControlCodec.temporaryDownloadFilename(for: "invoice.pdf", identifier: second)

        XCTAssertEqual(a, ".invoice.pdf.ghostftp-11111111-2222-3333-4444-555555555555.part")
        XCTAssertNotEqual(a, b)
        XCTAssertThrowsError(try FTPControlCodec.temporaryDownloadFilename(for: "../escape"))
    }

    func testMLSDRefusesIncompleteDirectoryWhenServerReturnsHostileFileNames() throws {
        for bad in [
            "type=file;size=3; ..",
            "type=file;size=3; nested/escape.txt",
            "type=file;size=3; attack\u{0000}file",
        ] {
            let listing = Data(
                ["type=file;size=3; good.txt", bad]
                    .joined(separator: "\r\n").appending("\r\n").utf8
            )
            XCTAssertThrowsError(try FTPControlCodec.parseMLSD(listing))
        }
    }

    func testMLSDRefusesUnknownFormatsAndMalformedFactsRatherThanHidingFiles() {
        let malformed = [
            "unexpected listing record",
            "type=file;size=3; good.txt\r\nunknown entry",
            "size=3; missing-type.txt",
            "type=file;size=not-a-number; broken.txt",
            "type=file;type=dir; duplicate.txt",
            "type=OS.unix=slink; symlink.txt",
            "type=cdir; ",
            "type=pdir; bad\u{0000}name",
            "type=file;invalidfact; broken.txt",
        ]
        for line in malformed {
            XCTAssertThrowsError(
                try FTPControlCodec.parseMLSD(Data("\(line)\r\n".utf8)),
                "Unexpectedly accepted invalid MLSD row: \(line)"
            )
        }
    }

    func testMLSDPreservesValidRFC3659NamedPseudoEntriesAndEmptyFactValues() throws {
        let listing = Data(
            [
                "type=cdir;perm=; /srv/data",
                "type=pdir;perm=; /srv",
                "type=file;size=0;perm=; ordinary.txt",
            ].joined(separator: "\r\n").appending("\r\n").utf8
        )
        let entries = try FTPControlCodec.parseMLSD(listing)
        XCTAssertEqual(entries.map(\.name), ["ordinary.txt"])
        XCTAssertEqual(entries.first?.size, 0)
    }

    func testMLSDRejectsAmbiguousDuplicateNames() throws {
        let payload = Data(
            [
                "type=file;size=1; repeated.txt",
                "type=dir;size=0; repeated.txt",
            ].joined(separator: "\r\n").appending("\r\n").utf8
        )
        XCTAssertThrowsError(try FTPControlCodec.parseMLSD(payload))
    }

    func testMLSDAllowsOrdinaryUnicodeAndSpacesWithStrictFacts() throws {
        let listing = Data(
            "type=file;size=4;modify=20261014203000; Račun 2026.txt\r\n".utf8
        )
        let entries = try FTPControlCodec.parseMLSD(listing)
        XCTAssertEqual(entries.count, 1)
        XCTAssertEqual(entries.first?.name, "Račun 2026.txt")
        XCTAssertEqual(entries.first?.size, 4)
    }

    func testUnixLISTFallbackParsesCompleteValidInventory() throws {
        let listing = Data(
            [
                "total 16",
                "drwxr-xr-x 2 user group 4096 Oct 14 10:30 Assets Folder",
                "-rw-r--r-- 1 user group 4 Oct 15 2026 Račun 2026.txt",
                "-rw------- 1 user group 0 Jan 1 00:00 .private",
            ].joined(separator: "\r\n").appending("\r\n").utf8
        )
        let entries = try FTPControlCodec.parseUnixLIST(listing)
        XCTAssertEqual(entries.map(\.name), ["Assets Folder", ".private", "Račun 2026.txt"])
        XCTAssertTrue(entries[0].isDirectory)
        XCTAssertEqual(entries[2].size, 4)
        XCTAssertNil(entries[2].modified) // ls timestamps are not reliable UTC instants
    }

    func testUnixLISTFallbackRejectsUnknownRowsInsteadOfHidingEntries() {
        for row in [
            "something malicious.txt",
            "lrwxrwxrwx 1 user group 5 Oct 14 10:30 link -> secret",
            "-rw-r--r-- 1 user group 4 Oct 14 10:30 nested/escape.txt",
            "-rw-r--r-- 1 user group 4 Oct 14 10:30 ../traversal",
            "-rw-r--r-- 1 user group 18446744073709551616 Oct 14 10:30 huge",
            "-rw-r--r-- 1 user group 4 Oct 32 10:30 invalid-date",
            "-rw-r--r-- 1 user group 4 Oct 14 25:99 invalid-time",
        ] {
            let listing = Data(
                "drwxr-xr-x 2 user group 0 Oct 14 10:30 good\r\n\(row)\r\n".utf8
            )
            XCTAssertThrowsError(try FTPControlCodec.parseUnixLIST(listing), row)
        }
    }

    func testUnixLISTFallbackRejectsAmbiguousDuplicateNames() {
        let listing = Data(
            [
                "-rw-r--r-- 1 user group 3 Oct 14 10:30 same.txt",
                "drwxr-xr-x 2 user group 0 Oct 14 10:30 same.txt",
            ].joined(separator: "\r\n").appending("\r\n").utf8
        )
        XCTAssertThrowsError(try FTPControlCodec.parseUnixLIST(listing))
        XCTAssertThrowsError(try FTPControlCodec.parseUnixLIST(Data([0xff, 0xfe])))
    }

    func testExtremeRemoteSizeDoesNotOverflowSwiftInt64() throws {
        let entry = try FTPControlCodec.parseMLSD(
            Data("type=file;size=18446744073709551615; huge.bin\r\n".utf8)
        )
        XCTAssertEqual(entry.first?.size, UInt64.max)
        XCTAssertEqual(
            FTPControlCodec.displayFileSize(UInt64.max),
            "18446744073709551615 bytes"
        )
        XCTAssertFalse(FTPControlCodec.displayFileSize(4).isEmpty)
    }

    func testUnixLISTFallbackOnlyForExplicitUnsupportedCommand() {
        for code in [500, 501, 502, 504] {
            XCTAssertTrue(FTPControlCodec.mlsdIsUnsupported(FTPReply(code: code, lines: [])))
        }
        for code in [125, 150, 226, 425, 426, 450, 550] {
            XCTAssertFalse(FTPControlCodec.mlsdIsUnsupported(FTPReply(code: code, lines: [])))
        }
    }

    func testFTPDownloadSizeRequiresExactRFC3659ASCIIReply() throws {
        XCTAssertEqual(try FTPControlCodec.expectedDownloadSize(
            from: FTPReply(code: 213, lines: ["213 0"])
        ), 0)
        XCTAssertEqual(try FTPControlCodec.expectedDownloadSize(
            from: FTPReply(code: 213, lines: ["213 18446744073709551615"])
        ), UInt64.max)
        for invalid in [
            FTPReply(code: 550, lines: ["550 Permission denied"]),
            FTPReply(code: 213, lines: ["213 -1"]),
            FTPReply(code: 213, lines: ["213 +2"]),
            FTPReply(code: 213, lines: ["213 123 trailing"]),
            FTPReply(code: 213, lines: ["213 "]),
            FTPReply(code: 213, lines: ["213 18446744073709551616"]),
            FTPReply(code: 213, lines: ["213 12", "213 34"]),
            FTPReply(code: 213, lines: ["213 １２３"]),
        ] {
            XCTAssertThrowsError(try FTPControlCodec.expectedDownloadSize(from: invalid))
        }
    }

    func testExtendedPassivePortParsesEPSVReply() throws {
        let reply = FTPReply(
            code: 229,
            lines: ["229 Entering Extended Passive Mode (|||6446|)"]
        )

        XCTAssertEqual(try FTPControlCodec.extendedPassivePort(from: reply), 6446)
    }

    func testExtendedPassivePortSupportsAlternateValidDelimiter() throws {
        let reply = FTPReply(
            code: 229,
            lines: ["229 Entering Extended Passive Mode (!!!21000!)"]
        )
        XCTAssertEqual(try FTPControlCodec.extendedPassivePort(from: reply), 21000)
    }

    func testExtendedPassivePortRejectsNonemptyProtocolAndAddressFields() {
        for line in [
            "229 Entering Extended Passive Mode (|1||6446|)",
            "229 Entering Extended Passive Mode (||192.0.2.44|6446|)",
        ] {
            XCTAssertThrowsError(
                try FTPControlCodec.extendedPassivePort(from: FTPReply(code: 229, lines: [line]))
            )
        }
    }

    func testExtendedPassivePortRejectsInvalidReply() {
        let reply = FTPReply(code: 229, lines: ["229 Entering Extended Passive Mode (|||0|)"])

        XCTAssertThrowsError(try FTPControlCodec.extendedPassivePort(from: reply))
    }

    func testExtendedPassivePortRejectsMalformedTrailingFields() {
        for line in [
            "229 Entering Extended Passive Mode (|||6446|junk)",
            "229 Entering Extended Passive Mode (|||6446||)",
            "229 Entering Extended Passive Mode (|||6446|unexpected|)",
        ] {
            XCTAssertThrowsError(
                try FTPControlCodec.extendedPassivePort(from: FTPReply(code: 229, lines: [line])),
                "Unexpectedly accepted malformed EPSV reply: \(line)"
            )
        }
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


    func testPrivateDownloadStagingIsOwnerOnlyAndExclusive() throws {
        let folder = FileManager.default.temporaryDirectory.appendingPathComponent(
            "ghostftp-staging-tests-\(UUID().uuidString)", isDirectory: true
        )
        try FileManager.default.createDirectory(at: folder, withIntermediateDirectories: true)
        defer { try? FileManager.default.removeItem(at: folder) }

        let temporaryURL = folder.appendingPathComponent(".invoice.pdf.ghostftp-test.part")
        let handle = try PrivateDownloadStaging.openNewFile(at: temporaryURL)
        defer { try? handle.close() }
        try handle.write(contentsOf: Data("confidential transfer".utf8))

        let attributes = try FileManager.default.attributesOfItem(atPath: temporaryURL.path)
        XCTAssertEqual((attributes[.posixPermissions] as? NSNumber)?.intValue, 0o600)
        XCTAssertThrowsError(try PrivateDownloadStaging.openNewFile(at: temporaryURL))
        try handle.synchronize()
        XCTAssertEqual(try Data(contentsOf: temporaryURL), Data("confidential transfer".utf8))
    }

    func testPrivateDownloadStagingRejectsPreexistingSymlink() throws {
        let folder = FileManager.default.temporaryDirectory.appendingPathComponent(
            "ghostftp-symlink-tests-\(UUID().uuidString)", isDirectory: true
        )
        try FileManager.default.createDirectory(at: folder, withIntermediateDirectories: true)
        defer { try? FileManager.default.removeItem(at: folder) }

        let target = folder.appendingPathComponent("protected.txt")
        let link = folder.appendingPathComponent("staging.part")
        try Data("untouched".utf8).write(to: target)
        try FileManager.default.createSymbolicLink(at: link, withDestinationURL: target)

        XCTAssertThrowsError(try PrivateDownloadStaging.openNewFile(at: link))
        XCTAssertEqual(try Data(contentsOf: target), Data("untouched".utf8))
    }

    func testFTPRejectsMalformedAndOutOfRangeReplyCodes() {
        for reply in [
            "220Unexpected separator\r\n",
            "099 Too small\r\n",
            "600 Out of range\r\n",
            "+20 Invalid leading byte\r\n",
            "2x0 Non-numeric status\r\n",
            "220: Wrong separator\r\n",
        ] {
            var buffer = Data(reply.utf8)
            XCTAssertThrowsError(try FTPControlCodec.takeReply(from: &buffer), reply)
        }

        var terseValidReply = Data("220\r\n".utf8)
        XCTAssertEqual(try? FTPControlCodec.takeReply(from: &terseValidReply)?.code, 220)
    }

    func testPrivateDownloadPublishCreatesDestinationExclusively() throws {
        let folder = FileManager.default.temporaryDirectory.appendingPathComponent(
            "ghostftp-publish-new-\(UUID().uuidString)", isDirectory: true
        )
        try FileManager.default.createDirectory(at: folder, withIntermediateDirectories: true)
        defer { try? FileManager.default.removeItem(at: folder) }
        let staging = folder.appendingPathComponent("stage.part")
        let destination = folder.appendingPathComponent("final.txt")
        let handle = try PrivateDownloadStaging.openNewFile(at: staging)
        try handle.write(contentsOf: Data("finished".utf8))
        try handle.synchronize()
        try handle.close()

        try PrivateDownloadStaging.publish(from: staging, to: destination)
        XCTAssertEqual(try Data(contentsOf: destination), Data("finished".utf8))
        XCTAssertFalse(FileManager.default.fileExists(atPath: staging.path))
        let attributes = try FileManager.default.attributesOfItem(atPath: destination.path)
        XCTAssertEqual((attributes[.posixPermissions] as? NSNumber)?.intValue, 0o600)
    }

    func testPrivateDownloadPublishNeverOverwritesWithoutConfirmation() throws {
        let folder = FileManager.default.temporaryDirectory.appendingPathComponent(
            "ghostftp-publish-existing-\(UUID().uuidString)", isDirectory: true
        )
        try FileManager.default.createDirectory(at: folder, withIntermediateDirectories: true)
        defer { try? FileManager.default.removeItem(at: folder) }
        let staging = folder.appendingPathComponent("stage.part")
        let destination = folder.appendingPathComponent("important.txt")
        try Data("user changes".utf8).write(to: destination)
        let handle = try PrivateDownloadStaging.openNewFile(at: staging)
        try handle.write(contentsOf: Data("remote content".utf8))
        try handle.close()

        XCTAssertThrowsError(try PrivateDownloadStaging.publish(from: staging, to: destination)) {
            guard case FTPControlError.destinationAlreadyExists = $0 else {
                return XCTFail("Expected a destination conflict, got \($0)")
            }
        }
        XCTAssertEqual(try Data(contentsOf: destination), Data("user changes".utf8))
        XCTAssertEqual(try Data(contentsOf: staging), Data("remote content".utf8))
    }

    func testPrivateDownloadPublishRejectsSymlinkEvenIfOverwriteAuthorized() throws {
        let folder = FileManager.default.temporaryDirectory.appendingPathComponent(
            "ghostftp-publish-link-\(UUID().uuidString)", isDirectory: true
        )
        try FileManager.default.createDirectory(at: folder, withIntermediateDirectories: true)
        defer { try? FileManager.default.removeItem(at: folder) }
        let protected = folder.appendingPathComponent("protected.txt")
        let destination = folder.appendingPathComponent("download.txt")
        let staging = folder.appendingPathComponent("stage.part")
        try Data("protected".utf8).write(to: protected)
        try FileManager.default.createSymbolicLink(at: destination, withDestinationURL: protected)
        let handle = try PrivateDownloadStaging.openNewFile(at: staging)
        try handle.write(contentsOf: Data("malicious replacement".utf8))
        try handle.close()

        XCTAssertThrowsError(try PrivateDownloadStaging.publish(
            from: staging, to: destination, allowReplacingExisting: true
        )) {
            guard case FTPControlError.unsafeLocalDestination = $0 else {
                return XCTFail("Expected link rejection, got \($0)")
            }
        }
        XCTAssertEqual(try Data(contentsOf: protected), Data("protected".utf8))
        XCTAssertEqual(try Data(contentsOf: staging), Data("malicious replacement".utf8))
    }

    func testPrivateDownloadPublishReplacesOnlyWhenExplicitlyAuthorized() throws {
        let folder = FileManager.default.temporaryDirectory.appendingPathComponent(
            "ghostftp-publish-confirmed-\(UUID().uuidString)", isDirectory: true
        )
        try FileManager.default.createDirectory(at: folder, withIntermediateDirectories: true)
        defer { try? FileManager.default.removeItem(at: folder) }
        let destination = folder.appendingPathComponent("confirmed.txt")
        let staging = folder.appendingPathComponent("stage.part")
        try Data("old content".utf8).write(to: destination)
        let handle = try PrivateDownloadStaging.openNewFile(at: staging)
        try handle.write(contentsOf: Data("new content".utf8))
        try handle.close()

        try PrivateDownloadStaging.publish(
            from: staging, to: destination, allowReplacingExisting: true
        )
        XCTAssertEqual(try Data(contentsOf: destination), Data("new content".utf8))
        XCTAssertFalse(FileManager.default.fileExists(atPath: staging.path))
    }

    func testRealLocalFTPLoginListUploadAndDownload() async throws {
        guard let rawPort = ProcessInfo.processInfo.environment["GHOSTFTP_MACOS_E2E_PORT"],
              let port = UInt16(rawPort) else {
            throw XCTSkip("Real FTP fixture is not configured on this macOS runner.")
        }

        let session = FTPControlSession()
        let profile = ConnectionProfile(
            name: "Local protocol acceptance",
            protocolKind: .ftp,
            host: "127.0.0.1",
            port: port,
            username: "ghostftp-e2e"
        )
        let directory = FileManager.default.temporaryDirectory.appendingPathComponent(
            "ghostftp-macos-ftp-e2e-\(UUID().uuidString)",
            isDirectory: true
        )
        try FileManager.default.createDirectory(at: directory, withIntermediateDirectories: true)
        defer { try? FileManager.default.removeItem(at: directory) }

        let unique = UUID().uuidString
        let remoteName = "uploaded-\(unique)-Racun.txt"
        let source = directory.appendingPathComponent("source.txt")
        let destination = directory.appendingPathComponent("downloaded.txt")
        let content = Data("Ghost FTP macOS real FTP roundtrip – \(unique)".utf8)
        try content.write(to: source)

        let cwd = try await session.connect(profile: profile, password: "ghostftp-e2e-secret")
        XCTAssertFalse(cwd.isEmpty)
        try await session.noop()
        let initial = try await session.listDirectory()
        XCTAssertTrue(initial.contains(where: { $0.name == "fixture.txt" }))

        try await session.uploadFile(from: source, remoteName: remoteName)
        let afterUpload = try await session.listDirectory()
        XCTAssertTrue(afterUpload.contains(where: { $0.name == remoteName }))
        try await session.downloadFile(remoteName: remoteName, to: destination)
        XCTAssertEqual(try Data(contentsOf: destination), content)

        // Exercise the real server path: a repeated upload must not silently
        // overwrite an existing remote file even if its local contents change.
        try Data("ATTEMPTED UNCONFIRMED OVERWRITE".utf8).write(to: source)
        do {
            try await session.uploadFile(from: source, remoteName: remoteName)
            XCTFail("Expected existing remote file to reject unconfirmed STOR")
        } catch FTPControlError.remoteFileAlreadyExists {
            try await session.noop()
        }
        let preserved = directory.appendingPathComponent("remote-preserved.txt")
        try await session.downloadFile(remoteName: remoteName, to: preserved)
        XCTAssertEqual(try Data(contentsOf: preserved), content)

        // A second download to the same destination must not silently
        // destroy edits made locally after the original transfer.
        try Data("local edits must survive".utf8).write(to: destination)
        do {
            try await session.downloadFile(remoteName: remoteName, to: destination)
            XCTFail("Expected a destination conflict without overwrite authorization")
        } catch FTPControlError.destinationAlreadyExists {
            XCTAssertEqual(
                try Data(contentsOf: destination),
                Data("local edits must survive".utf8)
            )
            try await session.noop() // final 226 reply left the control channel in sync
        }

        let folder = "ghostftp-qa-\(UUID().uuidString)"
        try await session.createDirectory(named: folder)
        let afterMKD = try await session.listDirectory()
        XCTAssertTrue(afterMKD.contains(where: {
            $0.name == folder && $0.isDirectory
        }))
        try await session.deleteFile(named: remoteName)
        try await session.removeEmptyDirectory(named: folder)
        let afterDelete = try await session.listDirectory()
        XCTAssertFalse(afterDelete.contains(where: { $0.name == remoteName || $0.name == folder }))
        await session.disconnect()
    }

    func testLiveLegacyServerRejectsMLSDAndListsViaUnixLIST() async throws {
        guard let rawPort = ProcessInfo.processInfo.environment["GHOSTFTP_MACOS_LIST_E2E_PORT"],
              let port = UInt16(rawPort) else {
            throw XCTSkip("Legacy Unix LIST FTP fixture not configured.")
        }
        let session = FTPControlSession()
        let profile = ConnectionProfile(
            name: "Legacy Unix LIST FTP acceptance",
            protocolKind: .ftp,
            host: "127.0.0.1",
            port: port,
            username: "ghostftp-e2e"
        )
        let cwd = try await session.connect(profile: profile, password: "ghostftp-e2e-secret")
        XCTAssertFalse(cwd.isEmpty)
        let entries = try await session.listDirectory()
        XCTAssertTrue(entries.contains(where: { $0.name == "fixture.txt" }))
        try await session.noop() // rejected MLSD reply and LIST completion were consumed.
        await session.disconnect()
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
