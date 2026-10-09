import XCTest
@testable import GhostFTPMacApp

final class SavedSiteDuplicatesTests: XCTestCase {
    private func profile(
        name: String,
        protocolKind: ConnectionProtocol = .ftp,
        host: String = "FTP.Example.Org.",
        port: UInt16 = 21,
        username: String = "Deploy"
    ) -> ConnectionProfile {
        ConnectionProfile(
            name: name,
            protocolKind: protocolKind,
            host: host,
            port: port,
            username: username
        )
    }

    func testDNSCaseTrailingDotAndWhitespaceMatchWithoutMerging() {
        let first = profile(name: "Production")
        let second = profile(
            name: "Backup alias",
            host: " ftp.example.org ",
            username: " Deploy "
        )
        let sites = [first, second]
        XCTAssertEqual(SavedSiteDuplicates.duplicateIDs(in: sites), Set([first.id, second.id]))
        XCTAssertEqual(SavedSiteDuplicates.matchingProfiles(for: first, in: sites), [second])
        // The identity helper must be advisory, never modifying supplied profiles.
        XCTAssertEqual(sites[0].name, "Production")
        XCTAssertEqual(sites[1].name, "Backup alias")
    }

    func testProtocolPortAndUsernameCaseRemainDistinct() {
        let base = profile(name: "A")
        let differentProtocol = profile(name: "B", protocolKind: .ftps)
        let differentPort = profile(name: "C", port: 2121)
        let differentAccount = profile(name: "D", username: "deploy")
        XCTAssertTrue(
            SavedSiteDuplicates.duplicateIDs(
                in: [base, differentProtocol, differentPort, differentAccount]
            ).isEmpty
        )
    }

    func testIncompleteProfilesAreExcluded() {
        let valid = profile(name: "Saved")
        let emptyHost = profile(name: "Empty host", host: " . ")
        let emptyAccount = profile(name: "Empty user", username: " \n ")
        let invalidPort = profile(name: "Empty port", port: 0)
        XCTAssertTrue(
            SavedSiteDuplicates.duplicateIDs(
                in: [valid, emptyHost, emptyAccount, invalidPort]
            ).isEmpty
        )
        XCTAssertTrue(
            SavedSiteDuplicates.matchingProfiles(
                for: emptyHost,
                in: [valid, emptyAccount]
            ).isEmpty
        )
    }

    func testAllMembersOfLargerGroupAreMarkedAndCandidateExcludedById() {
        let first = profile(name: "1")
        let second = profile(name: "2")
        let third = profile(name: "3")
        let other = profile(name: "Separate", host: "elsewhere.example.org")
        let sites = [first, second, third, other]
        XCTAssertEqual(
            SavedSiteDuplicates.duplicateIDs(in: sites),
            Set([first.id, second.id, third.id])
        )
        XCTAssertEqual(
            SavedSiteDuplicates.matchingProfiles(for: second, in: sites).map(\.id),
            [first.id, third.id]
        )
    }
}
