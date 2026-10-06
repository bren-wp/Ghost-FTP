import AppKit
import SwiftUI

struct FilesWorkspace: View {
    @ObservedObject var profiles: ProfileStore
    @Binding var selectedID: UUID?
    let onOpenSites: () -> Void

    var body: some View {
        Group {
            if let selectedID,
               let profile = profiles.profiles.first(where: { $0.id == selectedID }) {
                ConnectionEditor(
                    profile: profile,
                    onSave: profiles.save,
                    onDelete: { id in
                        try? KeychainStore().removePassword(for: id)
                        profiles.delete(id)
                        self.selectedID = nil
                    }
                )
                .id(profile.id)
            } else {
                VStack(spacing: 14) {
                    Image(systemName: "externaldrive.connected.to.line.below")
                        .font(.system(size: 42))
                        .foregroundStyle(.secondary)
                    Text("Choose a site")
                        .font(.title2.weight(.semibold))
                    Text("Open Sites to select a saved connection, or use New connection in the sidebar.")
                        .multilineTextAlignment(.center)
                        .foregroundStyle(.secondary)
                    Button("Open Sites") {
                        onOpenSites()
                    }
                }
                .frame(maxWidth: .infinity, maxHeight: .infinity)
                .padding(24)
            }
        }
        .frame(minWidth: 760, minHeight: 600)
    }
}

private struct ConnectionEditor: View {
    let onSave: (ConnectionProfile) -> Void
    let onDelete: (UUID) -> Void

    @State private var draft: ConnectionProfile
    @State private var password = ""
    @State private var rememberPassword = false
    @State private var validationMessage: String?
    @State private var credentialMessage: String?
    @StateObject private var probe = EndpointProbe()
    @StateObject private var ftpSession = FTPConnectionController()
    @State private var remoteDirectory = ""

    private let keychain = KeychainStore()

    init(
        profile: ConnectionProfile,
        onSave: @escaping (ConnectionProfile) -> Void,
        onDelete: @escaping (UUID) -> Void
    ) {
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

            if draft.protocolKind == .ftp {
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
                            .disabled(remoteDirectory.isEmpty)

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
                            .disabled(ftpSession.isTransferring)

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
                            .disabled(ftpSession.isTransferring)
                        }

                        transferStatus

                        if ftpSession.entries.isEmpty && !ftpSession.isListing {
                            Text("The current directory is empty or the server returned no MLSD entries.")
                                .font(.callout)
                                .foregroundStyle(.secondary)
                        } else {
                            List(ftpSession.entries) { entry in
                                HStack(spacing: 10) {
                                    Image(systemName: entry.isDirectory ? "folder.fill" : "doc")
                                        .foregroundStyle(.secondary)
                                    VStack(alignment: .leading, spacing: 2) {
                                        Text(entry.name)
                                        HStack(spacing: 8) {
                                            if let size = entry.size, !entry.isDirectory {
                                                Text(ByteCountFormatter.string(fromByteCount: Int64(size), countStyle: .file))
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
                                        .help("Download (entry.name)")
                                        .disabled(ftpSession.isTransferring)
                                    }
                                }
                            }
                            .frame(minHeight: 180, idealHeight: 240)
                        }
                    }

                    Text("This opens a real unencrypted FTP control session, loads the current directory through EPSV + MLSD, and performs file upload/download through passive FTP data channels. FTPS/SFTP session engines are not enabled by this control.")
                        .font(.callout)
                        .foregroundStyle(.secondary)
                }
            }

            Section {
                HStack {
                    Button("Delete site", role: .destructive) {
                        onDelete(draft.id)
                    }

                    Spacer()

                    Button("Save") {
                        save()
                    }
                    .keyboardShortcut("s", modifiers: .command)
                    .buttonStyle(.borderedProminent)
                }
            }
        }
        .formStyle(.grouped)
        .navigationTitle(draft.name.isEmpty ? "New site" : draft.name)
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
        ftpSession.downloadFile(entry, to: url)
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
            credentialMessage = "The site was saved, but the password could not be updated in macOS Keychain."
            onSave(draft)
        }
    }
}
