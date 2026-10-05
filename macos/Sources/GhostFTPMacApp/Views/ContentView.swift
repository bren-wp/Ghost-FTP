import SwiftUI

struct ContentView: View {
    @StateObject private var profiles = ProfileStore()
    @State private var selectedID: UUID?

    var body: some View {
        NavigationSplitView {
            List(selection: $selectedID) {
                Section("Sites") {
                    ForEach(profiles.profiles) { profile in
                        Label(
                            profile.name.isEmpty ? "New site" : profile.name,
                            systemImage: profile.protocolKind == .sftp ? "lock.shield" : "server.rack"
                        )
                        .tag(profile.id)
                    }
                }
            }
            .navigationTitle("Ghost FTP")
            .toolbar {
                ToolbarItem {
                    Button {
                        let profile = profiles.addProfile()
                        selectedID = profile.id
                    } label: {
                        Label("New site", systemImage: "plus")
                    }
                }
            }
        } detail: {
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
                VStack(spacing: 12) {
                    Image(systemName: "externaldrive.connected.to.line.below")
                        .font(.system(size: 42))
                        .foregroundStyle(.secondary)
                    Text("Choose a site")
                        .font(.title2.weight(.semibold))
                    Text("Select a saved site or create a new one.")
                        .foregroundStyle(.secondary)
                }
                .frame(maxWidth: .infinity, maxHeight: .infinity)
            }
        }
        .frame(minWidth: 900, minHeight: 600)
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
        .onDisappear {
            probe.cancel()
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
