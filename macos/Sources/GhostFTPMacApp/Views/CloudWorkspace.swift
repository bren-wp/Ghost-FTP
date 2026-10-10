import AppKit
import SwiftUI

/// One-object transfers use provider-generated signed links, not stored API keys.
struct CloudWorkspace: View {
    @State private var uploadLink = ""
    @State private var downloadLink = ""
    @State private var activeTask: Task<Void, Never>?
    @State private var status = "No cloud transfer in progress."
    @State private var busy = false

    var body: some View {
        ScrollView {
            VStack(alignment: .leading, spacing: 20) {
                VStack(alignment: .leading, spacing: 6) {
                    Text("Cloud storage").font(.largeTitle.bold())
                    Text("Secure, direct object transfers for S3 / R2 / B2, Google Cloud Storage and Azure SAS. No account credentials or cloud links are saved.")
                        .foregroundStyle(.secondary)
                }
                GroupBox("Upload to a signed cloud link") {
                    VStack(alignment: .leading, spacing: 12) {
                        SecureField("Paste signed HTTPS PUT URL", text: $uploadLink)
                            .textFieldStyle(.roundedBorder)
                            .privacySensitive()
                        Button("Choose file and upload") { chooseUpload() }
                            .disabled(busy || uploadLink.isEmpty)
                        Text("Generate a short-lived PUT URL for the exact object in your provider console. This sends the selected file directly to the signed destination.")
                            .font(.caption).foregroundStyle(.secondary)
                    }.padding(10)
                }
                GroupBox("Download from a signed cloud link") {
                    VStack(alignment: .leading, spacing: 12) {
                        SecureField("Paste signed HTTPS GET URL", text: $downloadLink)
                            .textFieldStyle(.roundedBorder)
                            .privacySensitive()
                        Button("Choose destination and download") { chooseDownload() }
                            .disabled(busy || downloadLink.isEmpty)
                        Text("The download is staged and only replaces the chosen file after the provider returns success.")
                            .font(.caption).foregroundStyle(.secondary)
                    }.padding(10)
                }
                Text(status).font(.callout).accessibilityLabel("Cloud transfer status: \(status)")
                Text("Google Drive account browsing requires a separate OAuth authorization flow. Sharing a Drive page link is not a signed object URL.")
                    .font(.caption).foregroundStyle(.secondary)
            }
            .frame(maxWidth: 760, alignment: .leading)
            .padding(28)
        }
        .onDisappear {
            activeTask?.cancel()
            activeTask = nil
            uploadLink = ""
            downloadLink = ""
        }
    }

    private func chooseUpload() {
        guard !busy else { return }
        let link: SignedCloudLink
        do { link = try SignedCloudLink(uploadLink) }
        catch {
            status = (error as? LocalizedError)?.errorDescription ?? "Invalid cloud link."
            uploadLink = ""
            return
        }
        let panel = NSOpenPanel()
        panel.canChooseDirectories = false
        panel.canChooseFiles = true
        guard panel.runModal() == .OK, let file = panel.url else { return }
        busy = true
        status = "Uploading to \(link.host)…"
        activeTask = Task {
            defer { busy = false; uploadLink = ""; activeTask = nil }
            do {
                try await SignedCloudTransfer.upload(file: file, to: link)
                status = "Upload completed successfully."
            } catch is CancellationError {
                status = "Upload canceled."
            } catch {
                status = (error as? LocalizedError)?.errorDescription ?? "Secure upload failed. No URL or credentials were recorded."
            }
        }
    }

    private func chooseDownload() {
        guard !busy else { return }
        let link: SignedCloudLink
        do { link = try SignedCloudLink(downloadLink) }
        catch {
            status = (error as? LocalizedError)?.errorDescription ?? "Invalid cloud link."
            downloadLink = ""
            return
        }
        let panel = NSSavePanel()
        panel.nameFieldStringValue = "cloud-download.bin"
        guard panel.runModal() == .OK, let destination = panel.url else { return }
        busy = true
        status = "Downloading from \(link.host)…"
        activeTask = Task {
            defer { busy = false; downloadLink = ""; activeTask = nil }
            do {
                try await SignedCloudTransfer.download(from: link, to: destination)
                status = "Download completed successfully."
            } catch is CancellationError {
                status = "Download canceled."
            } catch {
                status = (error as? LocalizedError)?.errorDescription ?? "Secure download failed. The signed URL was not recorded."
            }
        }
    }
}
