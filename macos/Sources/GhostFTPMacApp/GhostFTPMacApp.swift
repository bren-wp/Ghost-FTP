import SwiftUI

@main
struct GhostFTPMacApp: App {
    var body: some Scene {
        WindowGroup {
            ContentView()
        }

        Settings {
            VStack(alignment: .leading, spacing: 10) {
                Text("Ghost FTP")
                    .font(.title2.bold())
                Text("macOS client")
                    .foregroundStyle(.secondary)
                Text("Passwords are stored only in macOS Keychain when credential storage is enabled.")
                    .font(.callout)
                    .foregroundStyle(.secondary)
            }
            .padding(24)
            .frame(width: 430)
        }
    }
}
