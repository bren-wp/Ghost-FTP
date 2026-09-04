import SwiftUI

struct ConnectionView: View {
    @ObservedObject var store: SessionStore

    var body: some View {
        Form {
            Section {
                HStack(spacing: 12) {
                    Image(systemName: "externaldrive.connected.to.line.below")
                        .font(.system(size: 30, weight: .semibold))
                        .foregroundStyle(.indigo)
                        .frame(width: 44, height: 44)
                        .background(.indigo.opacity(0.1), in: RoundedRectangle(cornerRadius: 12))
                    VStack(alignment: .leading, spacing: 3) {
                        Text("GhostFTP")
                            .font(.title2.bold())
                        Text("Private, direct file transfer")
                            .font(.subheadline)
                            .foregroundStyle(.secondary)
                    }
                }
                .padding(.vertical, 4)
            }

            if store.hasSavedConnection {
                Section("Saved connection") {
                    Label("Connection details restored from this device", systemImage: "checkmark.shield")
                    Text("Your password is never stored in the saved connection preset.")
                        .font(.footnote)
                        .foregroundStyle(.secondary)
                    Button("Forget saved connection", role: .destructive) {
                        store.forgetSavedConnection()
                    }
                }
            }

            Section("Connection") {
                Picker("Protocol", selection: $store.protocolKind) {
                    ForEach(TransferProtocol.allCases) { value in
                        Text(value.rawValue).tag(value)
                    }
                }

                TextField("Host", text: $store.host)
                    .textInputAutocapitalization(.never)
                    .autocorrectionDisabled()
                    .keyboardType(.URL)

                TextField("Port (optional)", text: $store.port)
                    .keyboardType(.numberPad)

                TextField("Username", text: $store.username)
                    .textInputAutocapitalization(.never)
                    .autocorrectionDisabled()

                SecureField("Password", text: $store.password)
                    .textContentType(.password)
                    .privacySensitive()
            }

            if store.protocolKind == .ftp {
                Section {
                    Label("FTP is unencrypted. Prefer FTPS whenever the server supports it.", systemImage: "exclamationmark.triangle")
                        .foregroundStyle(.orange)
                }
            }

            Section {
                Button {
                    store.connect()
                } label: {
                    HStack {
                        Spacer()
                        if store.busy { ProgressView().padding(.trailing, 8) }
                        Text(store.busy ? "Connecting…" : "Connect")
                            .fontWeight(.semibold)
                        Spacer()
                    }
                    .frame(minHeight: 34)
                }
                .disabled(store.busy)
            }

            Section("iOS transport") {
                Text("This native iOS release supports FTP and implicit FTPS. Explicit FTPS and SFTP remain available on the other GhostFTP platforms and are not falsely emulated on iOS.")
                    .font(.footnote)
                    .foregroundStyle(.secondary)
            }
        }
        .navigationTitle("Connect")
    }
}
