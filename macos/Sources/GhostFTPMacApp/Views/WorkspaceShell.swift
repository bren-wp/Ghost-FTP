import AppKit
import SwiftUI
import UniformTypeIdentifiers

// Shared 2026 premium design tokens, aligned with React/Tauri and Android.
private enum GhostPremiumPalette {
    static let background = Color(red: 7/255, green: 14/255, blue: 26/255)
    static let sidebar = Color(red: 13/255, green: 25/255, blue: 43/255)
    static let accent = Color(red: 56/255, green: 171/255, blue: 255/255)
    static let text = Color(red: 234/255, green: 246/255, blue: 255/255)
}

private enum MacWorkspace: String, CaseIterable, Identifiable, Hashable {
    case files
    case sites
    case transfers
    case sync
    case settings
    case about

    var id: String { rawValue }

    var title: String {
        switch self {
        case .files: return "Files"
        case .sites: return "Sites"
        case .transfers: return "Transfers"
        case .sync: return "Sync & Backup"
        case .settings: return "Settings"
        case .about: return "Help & About"
        }
    }

    var symbol: String {
        switch self {
        case .files: return "folder"
        case .sites: return "server.rack"
        case .transfers: return "arrow.up.arrow.down"
        case .sync: return "arrow.triangle.2.circlepath"
        case .settings: return "gearshape"
        case .about: return "questionmark.circle"
        }
    }
}

struct ContentView: View {
    @StateObject private var profiles = ProfileStore()
    @StateObject private var transferHistory = TransferHistoryStore.shared
    @State private var workspace: MacWorkspace = .files
    @State private var selectedSiteID: UUID?

    @AppStorage("ghostftp.macos.workspace") private var storedWorkspace = MacWorkspace.files.rawValue
    @AppStorage("ghostftp.macos.remember-workspace") private var rememberWorkspace = true
    @AppStorage("ghostftp.macos.dark-appearance") private var darkAppearance = true

    var body: some View {
        NavigationSplitView {
            List(selection: $workspace) {
                Section {
                    Button {
                        createConnection()
                    } label: {
                        Label("New connection", systemImage: "plus.circle.fill")
                    }
                    .font(.system(size: 13, weight: .semibold))
                    .foregroundStyle(GhostPremiumPalette.text)
                    .frame(maxWidth: .infinity, minHeight: 40)
                    .background(GhostPremiumPalette.accent.opacity(0.22), in: RoundedRectangle(cornerRadius: 10))
                    .overlay(RoundedRectangle(cornerRadius: 10).stroke(GhostPremiumPalette.accent.opacity(0.45)))
                    .buttonStyle(.plain)
                }

                Section("Workspace") {
                    ForEach(MacWorkspace.allCases) { item in
                        Label(item.title, systemImage: item.symbol)
                            .tag(item)
                    }
                }

                Section {
                    HStack(spacing: 10) {
                        // Use the app's bundled GhostFTP.icns for the sidebar brand.
                        Image(nsImage: NSApplication.shared.applicationIconImage)
                            .resizable()
                            .interpolation(.high)
                            .frame(width: 34, height: 34)
                            .accessibilityHidden(true)

                        VStack(alignment: .leading, spacing: 3) {
                            Text("Ghost FTP")
                                .font(.system(size: 16, weight: .bold))
                            Text("TOTAL CONTROL")
                                .font(.system(size: 9, weight: .semibold))
                                .tracking(1.7)
                                .foregroundStyle(GhostPremiumPalette.accent)
                        }
                    }
                    .padding(.vertical, 4)
                }
            }
            .listStyle(.sidebar)
            .scrollContentBackground(.hidden)
            .background(darkAppearance ? GhostPremiumPalette.sidebar : Color(nsColor: .windowBackgroundColor))
            .navigationTitle("Ghost FTP")
            .frame(minWidth: 210)
        } detail: {
            workspaceDetail
                .background(darkAppearance ? GhostPremiumPalette.background : Color(nsColor: .windowBackgroundColor))
                .navigationTitle(workspace.title)
        }
        // Preserve the 1290px premium reference at full size while permitting
        // usable windowed layouts on smaller MacBook displays.
        .frame(minWidth: 840, minHeight: 560)
        .tint(GhostPremiumPalette.accent)
        .preferredColorScheme(darkAppearance ? .dark : .light)
        .onAppear {
            if rememberWorkspace {
                workspace = MacWorkspace(rawValue: storedWorkspace) ?? .files
            } else {
                workspace = .files
            }
        }
        .onChange(of: workspace) { newValue in
            storedWorkspace = rememberWorkspace ? newValue.rawValue : MacWorkspace.files.rawValue
        }
        .onChange(of: rememberWorkspace) { enabled in
            if !enabled {
                storedWorkspace = MacWorkspace.files.rawValue
            }
        }
    }

    @ViewBuilder
    private var workspaceDetail: some View {
        switch workspace {
        case .files:
            FilesWorkspace(
                profiles: profiles,
                selectedID: $selectedSiteID,
                onOpenSites: { workspace = .sites }
            )
        case .sites:
            SitesWorkspace(
                profiles: profiles,
                selectedID: $selectedSiteID,
                onOpen: { id in
                    selectedSiteID = id
                    workspace = .files
                }
            )
        case .transfers:
            TransfersWorkspace(history: transferHistory)
        case .sync:
            SyncBackupWorkspace(profiles: profiles)
        case .settings:
            SettingsWorkspace(profiles: profiles, history: transferHistory)
        case .about:
            AboutWorkspace()
        }
    }

    private func createConnection() {
        let profile = profiles.addProfile()
        selectedSiteID = profile.id
        workspace = .files
    }
}

private struct SitesWorkspace: View {
    @ObservedObject var profiles: ProfileStore
    @Binding var selectedID: UUID?
    let onOpen: (UUID) -> Void
    @State private var duplicatesOnly = false

    private var duplicateIDs: Set<UUID> {
        SavedSiteDuplicates.duplicateIDs(in: profiles.profiles)
    }

    private var displayedProfiles: [ConnectionProfile] {
        duplicatesOnly
            ? profiles.profiles.filter { duplicateIDs.contains($0.id) }
            : profiles.profiles
    }

    var body: some View {
        VStack(alignment: .leading, spacing: 18) {
            workspaceHeader(
                title: "Sites",
                subtitle: "Saved connections use the same site model as Ghost FTP on Windows and Linux."
            )

            if profiles.profiles.isEmpty {
                emptyWorkspace(
                    symbol: "server.rack",
                    title: "No saved sites",
                    detail: "Create a connection from the sidebar to add your first site."
                )
            } else {
                Toggle(isOn: $duplicatesOnly) {
                    Text("Possible duplicates (\\(duplicateIDs.count))")
                }
                .toggleStyle(.checkbox)
                .accessibilityHint("Show only saved sites with matching protocol, server, port and username.")

                if displayedProfiles.isEmpty {
                    Text("No possible duplicate saved sites.")
                        .foregroundStyle(.secondary)
                        .frame(maxWidth: .infinity, maxHeight: .infinity)
                } else {
                    List {
                    ForEach(displayedProfiles) { profile in
                        HStack(spacing: 14) {
                            Image(systemName: profile.protocolKind == .sftp ? "lock.shield" : "server.rack")
                                .font(.title3)
                                .frame(width: 28)

                            VStack(alignment: .leading, spacing: 3) {
                                Text(profile.name.isEmpty ? "New site" : profile.name)
                                    .font(.headline)
                                Text("\(profile.protocolKind.title) · \(profile.host):\(profile.port)")
                                    .font(.caption)
                                    .foregroundStyle(.secondary)
                                if duplicateIDs.contains(profile.id) {
                                    Label("Possible duplicate", systemImage: "rectangle.on.rectangle")
                                        .font(.caption2)
                                        .foregroundStyle(.orange)
                                }
                            }

                            Spacer()

                            Button("Open") {
                                onOpen(profile.id)
                            }

                            Button(role: .destructive) {
                                try? KeychainStore().removePassword(for: profile.id)
                                profiles.delete(profile.id)
                                if selectedID == profile.id {
                                    selectedID = nil
                                }
                            } label: {
                                Image(systemName: "trash")
                            }
                            .buttonStyle(.borderless)
                            .help("Delete site")
                        }
                        .padding(.vertical, 5)
                    }
                }
                    .listStyle(.inset)
                }
            }
        }
        .padding(24)
    }
}

private struct TransfersWorkspace: View {
    @ObservedObject var history: TransferHistoryStore

    var body: some View {
        VStack(alignment: .leading, spacing: 18) {
            HStack(alignment: .firstTextBaseline) {
                workspaceHeader(
                    title: "Transfers",
                    subtitle: "Credential-free FTP upload and download history from this Mac."
                )
                Spacer()
                Button("Clear history", role: .destructive) {
                    history.clear()
                }
                .disabled(history.records.isEmpty)
            }

            if history.records.isEmpty {
                emptyWorkspace(
                    symbol: "arrow.up.arrow.down",
                    title: "No transfers yet",
                    detail: "Uploads and downloads started from Files will appear here."
                )
            } else {
                List(history.records) { record in
                    HStack(spacing: 14) {
                        Image(systemName: record.direction == .upload ? "arrow.up.circle" : "arrow.down.circle")
                            .font(.title3)
                            .frame(width: 28)

                        VStack(alignment: .leading, spacing: 3) {
                            Text(record.fileName)
                                .font(.headline)
                            Text("\(record.direction.title) · \(record.status.title)")
                                .font(.caption)
                                .foregroundStyle(.secondary)
                        }

                        Spacer()

                        Text(record.startedAt, style: .time)
                            .font(.caption.monospacedDigit())
                            .foregroundStyle(.secondary)
                    }
                    .padding(.vertical, 5)
                }
                .listStyle(.inset)
            }
        }
        .padding(24)
    }
}

private struct SyncBackupWorkspace: View {
    @ObservedObject var profiles: ProfileStore
    @State private var statusMessage: String?

    var body: some View {
        VStack(alignment: .leading, spacing: 18) {
            workspaceHeader(
                title: "Sync & Backup",
                subtitle: "Synchronize saved site definitions between Ghost FTP installations without exporting passwords."
            )

            GroupBox("Profile backup") {
                VStack(alignment: .leading, spacing: 12) {
                    Text("The backup contains site names, hosts, ports, usernames and connection options. Passwords stay in macOS Keychain and are never written to the backup.")
                        .font(.callout)
                        .foregroundStyle(.secondary)

                    HStack {
                        Button {
                            exportProfiles()
                        } label: {
                            Label("Back up sites", systemImage: "square.and.arrow.up")
                        }
                        .disabled(profiles.profiles.isEmpty)

                        Button {
                            importProfiles()
                        } label: {
                            Label("Restore sites", systemImage: "square.and.arrow.down")
                        }
                    }

                    if let statusMessage {
                        Text(statusMessage)
                            .font(.callout)
                            .foregroundStyle(.secondary)
                    }
                }
                .frame(maxWidth: .infinity, alignment: .leading)
                .padding(8)
            }

            Spacer()
        }
        .padding(24)
    }

    private func exportProfiles() {
        let panel = NSSavePanel()
        panel.title = "Back up Ghost FTP sites"
        panel.nameFieldStringValue = "GhostFTP-Sites.json"

        guard panel.runModal() == .OK, let url = panel.url else { return }

        do {
            let data = try profiles.exportProfiles()
            try data.write(to: url, options: .atomic)
            try FileManager.default.setAttributes(
                [.posixPermissions: 0o600],
                ofItemAtPath: url.path
            )
            statusMessage = "Backed up \(profiles.profiles.count) site(s). Passwords were not exported."
        } catch {
            statusMessage = "The site backup could not be written."
        }
    }

    private func importProfiles() {
        let panel = NSOpenPanel()
        panel.title = "Restore Ghost FTP sites"
        panel.allowedContentTypes = [.json]
        panel.canChooseFiles = true
        panel.canChooseDirectories = false
        panel.allowsMultipleSelection = false

        guard panel.runModal() == .OK, let url = panel.url else { return }

        let scoped = url.startAccessingSecurityScopedResource()
        defer {
            if scoped {
                url.stopAccessingSecurityScopedResource()
            }
        }

        do {
            let data = try readBoundedBackup(from: url)
            let count = try profiles.importProfiles(from: data)
            statusMessage = "Restored \(count) site definition(s). Passwords remain unchanged in Keychain."
        } catch {
            statusMessage = "The selected backup is invalid or could not be read."
        }
    }

    private func readBoundedBackup(
        from url: URL,
        maximumBytes: Int = 256 * 1024
    ) throws -> Data {
        let handle = try FileHandle(forReadingFrom: url)
        defer { try? handle.close() }

        let data = try handle.read(upToCount: maximumBytes + 1) ?? Data()
        guard data.count <= maximumBytes else {
            throw CocoaError(.fileReadTooLarge)
        }
        return data
    }
}

private struct SettingsWorkspace: View {
    @ObservedObject var profiles: ProfileStore
    @ObservedObject var history: TransferHistoryStore

    @AppStorage("ghostftp.macos.remember-workspace") private var rememberWorkspace = true
    @AppStorage("ghostftp.macos.dark-appearance") private var darkAppearance = true
    @State private var credentialStatus: String?

    var body: some View {
        Form {
            Section("Appearance") {
                Toggle("Use dark appearance", isOn: $darkAppearance)
            }

            Section("Startup") {
                Toggle("Reopen the last workspace", isOn: $rememberWorkspace)
            }

            Section("Privacy") {
                LabeledContent("Telemetry") {
                    Text("Disabled")
                }
                LabeledContent("Saved passwords") {
                    Text("macOS Keychain only")
                }

                Button("Clear transfer history", role: .destructive) {
                    history.clear()
                }
                .disabled(history.records.isEmpty)

                Button("Remove all saved passwords", role: .destructive) {
                    removeSavedPasswords()
                }
                .disabled(profiles.profiles.isEmpty)

                if let credentialStatus {
                    Text(credentialStatus)
                        .font(.callout)
                        .foregroundStyle(.secondary)
                }
            }

            Section("Security") {
                Text("FTP is unencrypted. FTPS and SFTP remain blocked from file operations until their certificate/hostname or SSH host-key verification engines are complete.")
                    .font(.callout)
                    .foregroundStyle(.secondary)
            }
        }
        .formStyle(.grouped)
        .padding(10)
    }

    private func removeSavedPasswords() {
        let keychain = KeychainStore()
        var removed = 0
        for profile in profiles.profiles {
            do {
                try keychain.removePassword(for: profile.id)
                removed += 1
            } catch {
                continue
            }
        }
        credentialStatus = "Removed saved credentials for \(removed) site(s)."
    }
}

private struct AboutWorkspace: View {
    private let repositoryURL = "https://github.com/bren-wp/Ghost-FTP"

    var body: some View {
        let version = Bundle.main.object(forInfoDictionaryKey: "CFBundleShortVersionString") as? String
            ?? "Preview"

        ScrollView {
            VStack(alignment: .leading, spacing: 16) {
                workspaceHeader(
                    title: "Help & About",
                    subtitle: "Ghost FTP · More Than Transfer. Total Control."
                )

                // The supplied ZIP contains no macOS screenshots; use shared
                // premium hierarchy with real SwiftUI controls and AppKit icon.
                ViewThatFits(in: .horizontal) {
                    HStack(alignment: .top, spacing: 14) {
                        identityCard(version: version)
                            .frame(minWidth: 370, maxWidth: .infinity)
                        resourcesCard
                            .frame(width: 282)
                    }

                    VStack(spacing: 14) {
                        identityCard(version: version)
                        resourcesCard
                    }
                }
            }
            .frame(maxWidth: .infinity, alignment: .leading)
            .padding(20)
        }
    }

    private func identityCard(version: String) -> some View {
        VStack(alignment: .leading, spacing: 17) {
            HStack(spacing: 18) {
                Image(nsImage: NSApplication.shared.applicationIconImage)
                    .resizable()
                    .interpolation(.high)
                    .frame(width: 76, height: 76)
                    .accessibilityHidden(true)

                VStack(alignment: .leading, spacing: 7) {
                    Text("Ghost FTP")
                        .font(.system(size: 30, weight: .bold))
                        .foregroundStyle(GhostPremiumPalette.text)
                    Text("MORE THAN TRANSFER. TOTAL CONTROL.")
                        .font(.system(size: 10, weight: .medium))
                        .tracking(1.5)
                        .foregroundStyle(GhostPremiumPalette.accent)
                }
            }

            Text("Version \(version)")
                .font(.system(size: 11, weight: .semibold))
                .foregroundStyle(GhostPremiumPalette.accent)
                .padding(.horizontal, 11)
                .padding(.vertical, 6)
                .background(GhostPremiumPalette.accent.opacity(0.12), in: Capsule())

            Divider()

            Text("Designed for complete control")
                .font(.system(size: 16, weight: .semibold))
                .foregroundStyle(GhostPremiumPalette.text)
            Text("Manage your FTP file operations and saved connections without mandatory analytics or a Ghost FTP account. FTPS and SFTP remain disabled for file operations until trusted certificate and SSH host-key verification is complete.")
                .font(.system(size: 12))
                .foregroundStyle(.secondary)
                .fixedSize(horizontal: false, vertical: true)

            aboutMetadataRow("Publisher", value: "Brendigo")
            aboutMetadataRow("Platform", value: "macOS")
            aboutMetadataRow("Version", value: version)
            aboutMetadataRow("Security", value: "FTP available · FTPS/SFTP pending")
            aboutMetadataRow("Privacy", value: "No required telemetry")
        }
        .frame(maxWidth: .infinity, alignment: .leading)
        .padding(20)
        .background(GhostPremiumPalette.sidebar, in: RoundedRectangle(cornerRadius: 14))
        .overlay(RoundedRectangle(cornerRadius: 14)
            .stroke(GhostPremiumPalette.accent.opacity(0.15), lineWidth: 1))
    }

    private func aboutMetadataRow(_ label: String, value: String) -> some View {
        VStack(spacing: 10) {
            Divider()
            HStack(alignment: .firstTextBaseline, spacing: 12) {
                Text(label)
                    .font(.system(size: 12, weight: .semibold))
                    .foregroundStyle(GhostPremiumPalette.text)
                Spacer(minLength: 8)
                Text(value)
                    .font(.system(size: 12))
                    .foregroundStyle(.secondary)
                    .multilineTextAlignment(.trailing)
            }
        }
    }

    private var resourcesCard: some View {
        VStack(alignment: .leading, spacing: 12) {
            Text("SUPPORT & RESOURCES")
                .font(.system(size: 12, weight: .bold))
                .foregroundStyle(GhostPremiumPalette.text)

            resourceLink("Documentation", symbol: "book.closed",
                         url: "\(repositoryURL)/tree/main/docs")
            resourceLink("Support", symbol: "questionmark.circle",
                         url: "\(repositoryURL)/issues")
            resourceLink("Privacy Policy", symbol: "lock.shield",
                         url: "\(repositoryURL)/blob/main/docs/legal/PRIVACY.md")
            resourceLink("EULA", symbol: "doc.text",
                         url: "\(repositoryURL)/blob/main/EULA.txt")
            resourceLink("Project and releases", symbol: "externaldrive",
                         url: repositoryURL)
        }
        .frame(maxWidth: .infinity, alignment: .leading)
        .padding(16)
        .background(GhostPremiumPalette.sidebar, in: RoundedRectangle(cornerRadius: 14))
        .overlay(RoundedRectangle(cornerRadius: 14)
            .stroke(GhostPremiumPalette.accent.opacity(0.15), lineWidth: 1))
    }

    private func resourceLink(_ title: String, symbol: String, url: String) -> some View {
        Link(destination: URL(string: url)!) {
            HStack(spacing: 12) {
                Image(systemName: symbol)
                    .frame(width: 25)
                    .foregroundStyle(GhostPremiumPalette.accent)
                Text(title)
                    .font(.system(size: 13, weight: .medium))
                    .foregroundStyle(GhostPremiumPalette.text)
                Spacer(minLength: 0)
                Image(systemName: "arrow.up.right")
                    .font(.system(size: 10))
                    .foregroundStyle(.secondary)
            }
            .padding(.horizontal, 12)
            .frame(minHeight: 49)
            .background(GhostPremiumPalette.background, in: RoundedRectangle(cornerRadius: 10))
            .overlay(RoundedRectangle(cornerRadius: 10)
                .stroke(GhostPremiumPalette.accent.opacity(0.2), lineWidth: 1))
        }
        .buttonStyle(.plain)
    }
}

@ViewBuilder
private func workspaceHeader(title: String, subtitle: String) -> some View {
    VStack(alignment: .leading, spacing: 5) {
        Text(title)
            .font(.system(size: 26, weight: .bold))
        Text(subtitle)
            .foregroundStyle(.secondary)
    }
}

@ViewBuilder
private func emptyWorkspace(symbol: String, title: String, detail: String) -> some View {
    VStack(spacing: 12) {
        Image(systemName: symbol)
            .font(.system(size: 40))
            .foregroundStyle(.secondary)
        Text(title)
            .font(.title2.weight(.semibold))
        Text(detail)
            .multilineTextAlignment(.center)
            .foregroundStyle(.secondary)
    }
    .frame(maxWidth: .infinity, maxHeight: .infinity)
}
