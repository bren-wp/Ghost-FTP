// swift-tools-version: 5.10
import PackageDescription

let package = Package(
    name: "GhostFTPMac",
    platforms: [
        .macOS(.v13),
    ],
    products: [
        .executable(name: "GhostFTPMacApp", targets: ["GhostFTPMacApp"]),
    ],
    targets: [
        .executableTarget(
            name: "GhostFTPMacApp",
            path: "Sources/GhostFTPMacApp"
        ),
        .testTarget(
            name: "GhostFTPMacAppTests",
            dependencies: ["GhostFTPMacApp"],
            path: "Tests/GhostFTPMacAppTests"
        ),
    ],
    swiftLanguageVersions: [.v5]
)
