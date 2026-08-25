import SwiftUI

struct ConnectionView: View {
    @ObservedObject var store: SessionStore

    var body: some View {
        Form {
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
                        Spacer()
                    }
                }
                .disabled(store.busy)
            }

            Section("iOS transport") {
                Text("This native iOS release supports FTP and implicit FTPS. Explicit FTPS and SFTP remain available on the other ByFTP platforms and are not falsely emulated on iOS.")
                    .font(.footnote)
                    .foregroundStyle(.secondary)
            }
        }
        .navigationTitle("ByFTP")
    }
}
