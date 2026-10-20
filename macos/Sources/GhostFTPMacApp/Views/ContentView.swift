import AppKit
import SwiftUI

struct FilesWorkspace: View {
    @ObservedObject var profiles: ProfileStore
    @Binding var selectedID: UUID?
    @Binding var newSiteDraft: ConnectionProfile?
    let onOpenSites: () -> Void

    var body: some View {
        Group {
            if let newSiteDraft {
                ConnectionEditor(
                    profile: newSiteDraft,
                    savedProfiles: profiles.profiles,
                    isNewSite: true,
                    onSave: { profile in
                        profiles.save(profile)
                        self.newSiteDraft = nil
                        self.selectedID = profile.id
                    },
                    onDelete: { _ in self.newSiteDraft = nil }
                )
                .id(newSiteDraft.id)
            } else if let selectedID,
               let profile = profiles.profiles.first(where: { $0.id == selectedID }) {
                ConnectionEditor(
                    profile: profile,
                    savedProfiles: profiles.profiles,
                    onSave: profiles.save,
                    onDelete: { id in
                        // Never orphan a still-saved Keychain credential.
                        try KeychainStore().removePassword(for: id)
                        profiles.delete(id)
                        self.selectedID = nil
                    }
                )
                .id(profile.id)
            } else {
                MacQuickConnect(onOpenSites: onOpenSites)
            }
        }
        .frame(minWidth: 760, minHeight: 600)
    }
}

/// Real temporary FTP connection — no ProfileStore or Keychain writes.
private struct MacQuickConnect: View {
    let onOpenSites: () -> Void
    @StateObject private var session = FTPConnectionController()
    @State private var host = ""
    @State private var username = ""
    @State private var password = ""
    @State private var port = "21"
    @State private var validationMessage: String?

    var body: some View {
        Form {
            Section("Quick Connect · temporary FTP session") {
                TextField("Host", text: $host)
                TextField("Username", text: $username)
                SecureField("Password", text: $password)
                    .privacySensitive()
                TextField("Port", text: $port)
                Text("Connect uses the entered credentials directly and never creates a saved site. FTP traffic is unencrypted; macOS FTPS/SFTP session engines are not yet available.")
                    .font(.callout)
                    .foregroundStyle(.secondary)

                HStack {
                    if session.isConnected {
                        Button("Disconnect") { session.disconnect() }
                    } else {
                        Button {
                            connectNow()
                        } label: {
                            if session.state == .connecting {
                                ProgressView().controlSize(.small)
                                Text("Connecting…")
                            } else {
                                Label("Connect", systemImage: "bolt.horizontal.circle")
                            }
                        }
                        .disabled(session.state == .connecting)
                        .keyboardShortcut(.return, modifiers: [])
                    }
                    Button("Saved sites") { onOpenSites() }
                }
            }
            if let validationMessage {
                Text(validationMessage).foregroundStyle(.red)
            }
            switch session.state {
            case .idle:
                EmptyView()
            case .connecting:
                Text("Authenticating with the FTP server…")
            case .connected(let path):
                Label("Connected · \(path)", systemImage: "checkmark.circle.fill")
                    .foregroundStyle(.green)
            case .failed(let message):
                Text(message).foregroundStyle(.red)
            }
            if session.isConnected {
                Section("Remote files") {
                    Button("Refresh listing") { session.refreshDirectory() }
                    if session.entries.isEmpty && !session.isListing {
                        Text("The server returned no entries.")
                    }
                    ForEach(session.entries) { item in
                        Label(item.name, systemImage: item.isDirectory ? "folder" : "doc")
                    }
                }
            }
        }
        .formStyle(.grouped)
        .onChange(of: session.state) { state in
            if case .connected = state { password = "" }
        }
        .onDisappear {
            session.disconnect()
            password = ""
        }
    }

    private func connectNow() {
        guard let parsedPort = UInt16(port), !host.trimmingCharacters(in: .whitespaces).isEmpty,
              !username.trimmingCharacters(in: .whitespaces).isEmpty else {
            validationMessage = "Enter a host, username and port (1–65535)."
            return
        }
        let profile = ConnectionProfile(
            name: host.trimmingCharacters(in: .whitespaces),
            protocolKind: .ftp,
            host: host.trimmingCharacters(in: .whitespaces),
            port: parsedPort,
            username: username.trimmingCharacters(in: .whitespaces)
        )
        do {
            try ConnectionValidator.validate(profile)
            validationMessage = nil
            session.connect(profile: profile, password: password)
        } catch {
            validationMessage = error.localizedDescription
        }
    }
}

private struct ConnectionEditor: View {
    let savedProfiles: [ConnectionProfile]
    let isNewSite: Bool
    let onSave: (ConnectionProfile) -> Void
    let onDelete: (UUID) throws -> Void

    @State private var draft: ConnectionProfile
    @State private var password = ""
    @State private var rememberPassword = false
    @State private var validationMessage: String?
    @State private var credentialMessage: String?
    @StateObject private var probe = EndpointProbe()
    @StateObject private var ftpSession = FTPConnectionController()
    @State private var remoteDirectory = ""
    @State private var newRemoteFolderName = ""
    @State private var pendingRemoteDelete: FTPDirectoryEntry?
    @State private var showRemoteDeleteConfirmation = false

    private let keychain = KeychainStore()

    init(
        profile: ConnectionProfile,
        savedProfiles: [ConnectionProfile],
        isNewSite: Bool = false,
        onSave: @escaping (ConnectionProfile) -> Void,
        onDelete: @escaping (UUID) throws -> Void
    ) {
        self.savedProfiles = savedProfiles
        self.isNewSite = isNewSite
        self.onSave = onSave
        self.onDelete = onDelete
        _draft = State(initialValue: profile)
    }

    var body: some View {
        Form {
            Section("Connection") {
                TextField("Site name", text: $draft.name)

                Picker("Protocol", selection: $draft.protocolKind) {
                    ForEach(ConnectionProtocol.allCases) { item in
                        Text(item.title).tag(item)
                    }
                }
                .onChange(of: draft.protocolKind) { newValue in
                    draft.port = newValue.defaultPort
                    probe.cancel()
                    ftpSession.cancel()
                    remoteDirectory = ""
                }

                TextField("Server", text: $draft.host)

                HStack {
                    Text("Port")
                    Spacer()
                    TextField(
                        "Port",
                        value: $draft.port,
                        format: .number.grouping(.never)
                    )
                    .frame(width: 90)
                    .multilineTextAlignment(.trailing)
                }

                TextField("Username", text: $draft.username)

                SecureField("Password", text: $password)

                Toggle("Remember password in macOS Keychain", isOn: $rememberPassword)
            }

            // Advisory only: never merge or remove another saved profile.
            // Recomputed from the live editor draft as host/account values change.
            let matchingSites = SavedSiteDuplicates.matchingProfiles(
                for: draft,
                in: savedProfiles
            )
            if !matchingSites.isEmpty {
                Section("Saved site inventory") {
                    Label(
                        "Possible duplicate saved site (\(matchingSites.count) match\(matchingSites.count == 1 ? "" : "es"))",
                        systemImage: "rectangle.on.rectangle"
                    )
                    .foregroundStyle(.orange)

                    Text("Another saved site uses this protocol, server, port and username. Review the entries before saving. Ghost FTP will not merge, overwrite or delete either site.")
                        .font(.callout)
                        .foregroundStyle(.secondary)
                }
            }

            Section("Connection reliability") {
                if draft.protocolKind == .sftp {
                    Stepper(
                        "Keep-alive every \(draft.keepAliveSeconds) seconds",
                        value: $draft.keepAliveSeconds,
                        in: 5...300,
                        step: 5
                    )
                } else {
                    Text("Per-profile keep-alive is currently available for SFTP. FTP/FTPS will expose this only when protocol-level NOOP scheduling is implemented.")
                        .font(.callout)
                        .foregroundStyle(.secondary)
                }

                Stepper(
                    "Automatic reconnect attempts: \(draft.reconnectAttempts)",
                    value: $draft.reconnectAttempts,
                    in: 0...3
                )

                Text("Reconnect attempts apply only after a transport failure. Authentication, certificate and host-key failures are never retried silently.")
                    .font(.callout)
                    .foregroundStyle(.secondary)
            }

            Section("Transport security") {
                Label {
                    Text(draft.protocolKind.securitySummary)
                } icon: {
                    Image(systemName: draft.protocolKind == .ftp ? "exclamationmark.triangle.fill" : "lock.shield.fill")
                }
                .foregroundStyle(draft.protocolKind == .ftp ? .orange : .secondary)

                if draft.protocolKind == .ftps {
                    Text("Endpoint check verifies TCP reachability only. A full FTPS session must complete AUTH TLS and certificate/hostname validation before file operations.")
                        .font(.callout)
                        .foregroundStyle(.secondary)
                } else if draft.protocolKind == .sftp {
                    Text("Endpoint check verifies TCP reachability only. A full SFTP session must verify the SSH host key before authentication.")
                        .font(.callout)
                        .foregroundStyle(.secondary)
                }
            }

            if let validationMessage {
                Section {
                    Label(validationMessage, systemImage: "exclamationmark.circle")
                        .foregroundStyle(.red)
                }
            }

            if let credentialMessage {
                Section {
                    Text(credentialMessage)
                        .font(.callout)
                        .foregroundStyle(.secondary)
                }
            }

            Section("Endpoint") {
                HStack {
                    Button {
                        checkEndpoint()
                    } label: {
                        if probe.state == .checking {
                            ProgressView()
                                .controlSize(.small)
                            Text("Checking…")
                        } else {
                            Label("Check endpoint", systemImage: "network")
                        }
                    }
                    .disabled(probe.state == .checking)

                    endpointStatus
                }
            }

            if !isNewSite && draft.protocolKind == .ftp {
                Section("FTP session") {
                    HStack {
                        if ftpSession.isConnected {
                            Button("Disconnect") {
                                ftpSession.disconnect()
                            }
                        } else {
                            Button {
                                openFTPSession()
                            } label: {
                                if ftpSession.state == .connecting {
                                    ProgressView()
                                        .controlSize(.small)
                                    Text("Connecting…")
                                } else {
                                    Label("Open FTP session", systemImage: "bolt.horizontal.circle")
                                }
                            }
                            .disabled(ftpSession.state == .connecting)
                        }

                        ftpSessionStatus
                    }

                    if ftpSession.isConnected {
                        HStack {
                            TextField("Remote directory", text: $remoteDirectory)
                                .textFieldStyle(.roundedBorder)

                            Button("Change directory") {
                                ftpSession.changeDirectory(to: remoteDirectory)
                            }
                            .disabled(remoteDirectory.isEmpty || ftpSession.isTransferring || ftpSession.isListing)

                            Button {
                                ftpSession.refreshWorkingDirectory()
                            } label: {
                                Label("Refresh PWD", systemImage: "arrow.clockwise")
                            }
                        }

                        HStack {
                            Button {
                                ftpSession.verifyConnection()
                            } label: {
                                Label("Check session with NOOP", systemImage: "checkmark.shield")
                            }
                            .disabled(ftpSession.isTransferring || ftpSession.isListing)

                            Button {
                                ftpSession.refreshDirectory()
                            } label: {
                                if ftpSession.isListing {
                                    ProgressView()
                                        .controlSize(.small)
                                    Text("Refreshing…")
                                } else {
                                    Label("Refresh listing", systemImage: "arrow.clockwise")
                                }
                            }
                            .disabled(ftpSession.isListing || ftpSession.isTransferring)

                            Button {
                                chooseUpload()
                            } label: {
                                Label("Upload file", systemImage: "arrow.up.doc")
                            }
                            .disabled(ftpSession.isTransferring || ftpSession.isListing)
                        }

                        HStack {
                            TextField("New folder name", text: $newRemoteFolderName)
                                .textFieldStyle(.roundedBorder)
                            Button {
                                let name = newRemoteFolderName.trimmingCharacters(in: .whitespacesAndNewlines)
                                guard !name.isEmpty else { return }
                                ftpSession.createDirectory(named: name)
                                newRemoteFolderName = ""
                            } label: {
                                Label("Create folder", systemImage: "folder.badge.plus")
                            }
                            .disabled(
                                newRemoteFolderName.trimmingCharacters(in: .whitespacesAndNewlines).isEmpty ||
                                ftpSession.isTransferring || ftpSession.isListing
                            )
                        }

                        transferStatus
                        if ftpSession.isTransferring || ftpSession.isListing {
                            Text("File actions are temporarily unavailable while a transfer or directory listing is in progress.")
                                .font(.caption)
                                .foregroundStyle(.secondary)
                                .accessibilityAddTraits(.updatesFrequently)
                        }

                        if ftpSession.entries.isEmpty && !ftpSession.isListing {
                            Text("The current directory is empty or the FTP server returned no listable entries.")
                                .font(.callout)
                                .foregroundStyle(.secondary)
                        } else {
                            List(ftpSession.entries) { entry in
                                HStack(spacing: 10) {
                                    Image(systemName: entry.isDirectory ? "folder.fill" : "doc")
                                        .foregroundStyle(.secondary)
                                    VStack(alignment: .leading, spacing: 2) {
                                        if entry.isDirectory {
                                            Button(entry.name) {
                                                ftpSession.changeDirectory(to: entry.name)
                                            }
                                            .buttonStyle(.plain)
                                            .disabled(ftpSession.isTransferring || ftpSession.isListing)
                                        } else {
                                            Text(entry.name)
                                        }
                                        HStack(spacing: 8) {
                                            if let size = entry.size, !entry.isDirectory {
                                                Text(FTPControlCodec.displayFileSize(size))
                                            }
                                            if let modified = entry.modified {
                                                Text(modified)
                                            }
                                        }
                                        .font(.caption)
                                        .foregroundStyle(.secondary)
                                    }
                                    Spacer()
                                    if !entry.isDirectory {
                                        Button {
                                            chooseDownload(entry)
                                        } label: {
                                            Image(systemName: "arrow.down.circle")
                                        }
                                        .buttonStyle(.borderless)
                                        .help("Download \(entry.name)")
                                        .disabled(ftpSession.isTransferring || ftpSession.isListing)
                                    }
                                    Button {
                                        pendingRemoteDelete = entry
                                        showRemoteDeleteConfirmation = true
                                    } label: {
                                        Image(systemName: "trash")
                                    }
                                    .buttonStyle(.borderless)
                                    .help(entry.isDirectory ? "Remove empty folder \(entry.name)" : "Delete \(entry.name)")
                                    .disabled(ftpSession.isTransferring || ftpSession.isListing)
                                }
                            }
                            .frame(minHeight: 180, idealHeight: 240)
                        }
                    }

                    Text("FTP session: passive EPSV + MLSD (or validated Unix LIST fallback), directory navigation, folder creation, confirmed file/empty-folder deletion, upload and download. FTPS/SFTP engines are not yet available on macOS.")
                        .font(.callout)
                        .foregroundStyle(.secondary)
                }
            }

            Section {
                HStack {
                    Button(isNewSite ? "Cancel new site" : "Delete site", role: .destructive) {
                        do {
                            try onDelete(draft.id)
                        } catch {
                            credentialMessage = "The site was not deleted because its Keychain credential could not be removed."
                        }
                    }

                    Spacer()

                    Button(isNewSite ? "Save profile" : "Save") {
                        save()
                    }
                    .keyboardShortcut("s", modifiers: .command)
                    .buttonStyle(.borderedProminent)
                }
            }
        }
        .formStyle(.grouped)
        .confirmationDialog(
            "Delete remote file?",
            isPresented: $showRemoteDeleteConfirmation,
            titleVisibility: .visible
        ) {
            Button("Delete file", role: .destructive) {
                if let entry = pendingRemoteDelete {
                    ftpSession.deleteEntry(entry)
                }
                pendingRemoteDelete = nil
            }
            Button("Cancel", role: .cancel) {
                pendingRemoteDelete = nil
            }
        } message: {
            Text(pendingRemoteDelete?.isDirectory == true ? "Only empty folders can be removed. Nothing inside will be deleted." : "This permanently deletes the selected file from the connected FTP server.")
        }
        .navigationTitle(isNewSite ? "New saved site" : (draft.name.isEmpty ? "Site settings" : draft.name))
        .onAppear {
            do {
                if let saved = try keychain.password(for: draft.id) {
                    password = saved
                    rememberPassword = true
                }
            } catch {
                credentialMessage = "The saved credential could not be read from macOS Keychain."
            }
        }
        .onChange(of: ftpSession.state) { newValue in
            if case .connected(let path) = newValue {
                remoteDirectory = path
            }
        }
        .onDisappear {
            probe.cancel()
            ftpSession.cancel()
        }
    }

    @ViewBuilder
    private var endpointStatus: some View {
        switch probe.state {
        case .idle:
            EmptyView()
        case .checking:
            Text("Opening TCP connection…")
                .foregroundStyle(.secondary)
        case .reachable:
            Label("Endpoint reachable", systemImage: "checkmark.circle.fill")
                .foregroundStyle(.green)
        case .failed(let message):
            Label(message, systemImage: "xmark.circle.fill")
                .foregroundStyle(.red)
        }
    }

    @ViewBuilder
    private var transferStatus: some View {
        switch ftpSession.transferState {
        case .idle:
            EmptyView()
        case .uploading(let name):
            Label("Uploading \(name)…", systemImage: "arrow.up.circle")
                .foregroundStyle(.secondary)
        case .downloading(let name):
            Label("Downloading \(name)…", systemImage: "arrow.down.circle")
                .foregroundStyle(.secondary)
        case .completed(let message):
            Label(message, systemImage: "checkmark.circle.fill")
                .foregroundStyle(.green)
        case .failed(let message):
            Label(message, systemImage: "xmark.circle.fill")
                .foregroundStyle(.red)
        }
    }

    @ViewBuilder
    private var ftpSessionStatus: some View {
        switch ftpSession.state {
        case .idle:
            Text("Not connected")
                .foregroundStyle(.secondary)
        case .connecting:
            Text("Authenticating…")
                .foregroundStyle(.secondary)
        case .connected(let path):
            Label("Connected · \(path)", systemImage: "checkmark.circle.fill")
                .foregroundStyle(.green)
        case .failed(let message):
            Label(message, systemImage: "xmark.circle.fill")
                .foregroundStyle(.red)
        }
    }

    private func validate() -> Bool {
        do {
            try ConnectionValidator.validate(draft)
            validationMessage = nil
            return true
        } catch {
            validationMessage = error.localizedDescription
            return false
        }
    }

    private func checkEndpoint() {
        guard validate() else { return }
        probe.check(draft)
    }

    private func openFTPSession() {
        guard validate() else { return }
        guard draft.protocolKind == .ftp else { return }
        ftpSession.connect(profile: draft, password: password)
    }

    private func chooseUpload() {
        guard ftpSession.isConnected, !ftpSession.isTransferring else { return }

        let panel = NSOpenPanel()
        panel.title = "Choose a file to upload"
        panel.canChooseFiles = true
        panel.canChooseDirectories = false
        panel.allowsMultipleSelection = false

        guard panel.runModal() == .OK, let url = panel.url else { return }
        ftpSession.uploadFile(from: url)
    }

    private func chooseDownload(_ entry: FTPDirectoryEntry) {
        guard ftpSession.isConnected, !ftpSession.isTransferring, !entry.isDirectory else { return }

        let panel = NSSavePanel()
        panel.title = "Save \(entry.name)"
        panel.nameFieldStringValue = entry.name

        guard panel.runModal() == .OK, let url = panel.url else { return }
        // NSSavePanel asks for confirmation when replacing an existing file.
        // A new destination must remain no-clobber if another process creates
        // the filename while the download is in progress.
        let confirmedReplacement = FileManager.default.fileExists(atPath: url.path)
        ftpSession.downloadFile(
            entry,
            to: url,
            allowReplacingExisting: confirmedReplacement
        )
    }

    private func save() {
        guard validate() else { return }

        do {
            if rememberPassword {
                try keychain.setPassword(password, for: draft.id)
                credentialMessage = "Password saved in macOS Keychain."
            } else {
                try keychain.removePassword(for: draft.id)
                credentialMessage = "Password is not stored."
            }
            onSave(draft)
        } catch {
            // A password write/remove failure must not persist a changed
            // destination while the old secret is still linked to its UUID.
            credentialMessage = "The site was not saved because its macOS Keychain credential could not be updated."
        }
    }
}
