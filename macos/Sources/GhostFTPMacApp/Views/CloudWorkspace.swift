import AppKit
import SwiftUI

/// One-object transfers use provider-generated signed links, not stored API keys.
struct CloudWorkspace: View {
    @State private var uploadLink = ""
    @State private var downloadLink = ""
    @State private var activeTask: Task<Void, Never>?
    @State private var status = "No cloud transfer in progress."
    @State private var busy = false

    private let premiumPanel = Color(red: 17/255, green: 31/255, blue: 52/255)
    private let premiumLine = Color(red: 56/255, green: 171/255, blue: 255/255)

    var body: some View {
        ScrollView {
            VStack(alignment: .leading, spacing: 20) {
                VStack(alignment: .leading, spacing: 6) {
                    Text(L10n.text("Cloud storage")).font(.largeTitle.bold())
                    Text(L10n.text("Secure, direct object transfers for S3 / R2 / B2, Google Cloud Storage and Azure SAS. No account credentials or cloud links are saved."))
                        .foregroundStyle(.secondary)
                }
                GroupBox("Upload to a signed cloud link") {
                    VStack(alignment: .leading, spacing: 12) {
                        SecureField(L10n.text("Paste signed HTTPS PUT URL"), text: $uploadLink)
                            .textFieldStyle(.roundedBorder)
                            .privacySensitive()
                        Button(L10n.text("Choose file and upload")) { chooseUpload() }
                            .buttonStyle(.borderedProminent)
                            .disabled(busy || uploadLink.isEmpty)
                        Text(L10n.text("Generate a short-lived PUT URL for the exact object in your provider console. This sends the selected file directly to the signed destination."))
                            .font(.caption).foregroundStyle(.secondary)
                    }.padding(10)
                }
                .background(premiumPanel, in: RoundedRectangle(cornerRadius: 12))
                .overlay(RoundedRectangle(cornerRadius: 12).stroke(premiumLine.opacity(0.23)))
                GroupBox("Download from a signed cloud link") {
                    VStack(alignment: .leading, spacing: 12) {
                        SecureField(L10n.text("Paste signed HTTPS GET URL"), text: $downloadLink)
                            .textFieldStyle(.roundedBorder)
                            .privacySensitive()
                        Button(L10n.text("Choose destination and download")) { chooseDownload() }
                            .buttonStyle(.borderedProminent)
                            .disabled(busy || downloadLink.isEmpty)
                        Text(L10n.text("The download is staged and only replaces the chosen file after the provider returns success."))
                            .font(.caption).foregroundStyle(.secondary)
                    }.padding(10)
                }
                .background(premiumPanel, in: RoundedRectangle(cornerRadius: 12))
                .overlay(RoundedRectangle(cornerRadius: 12).stroke(premiumLine.opacity(0.23)))
                HStack(spacing: 12) {
                    Text(status)
                        .font(.callout)
                        .accessibilityLabel("Cloud transfer status: \(status)")
                    Spacer(minLength: 8)
                    Button(L10n.text("Cancel transfer")) {
                        activeTask?.cancel()
                        status = "Canceling cloud transfer…"
                    }
                    .disabled(!busy)
                    .accessibilityIdentifier("ghostftp.cloud.cancel")
                }
                Text(L10n.text("Google Drive account browsing requires a separate OAuth authorization flow. Sharing a Drive page link is not a signed object URL."))
                    .font(.caption).foregroundStyle(.secondary)
            }
            .frame(maxWidth: 760, alignment: .leading)
            .padding(28)
        }
        .tint(premiumLine)
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
        // A signed object URL is a bearer credential. The validated value
        // remains in the local transfer request, not the SwiftUI text field
        // while a system file picker is on screen.
        uploadLink = ""
        let panel = NSOpenPanel()
        panel.canChooseDirectories = false
        panel.canChooseFiles = true
        guard panel.runModal() == .OK, let file = panel.url else { return }
        busy = true
        status = "Uploading cloud object…"
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
        // Never leave the live signed URL visible in view state while the
        // save panel is open or after its cancellation.
        downloadLink = ""
        let panel = NSSavePanel()
        panel.nameFieldStringValue = "cloud-download.bin"
        guard panel.runModal() == .OK, let destination = panel.url else { return }
        busy = true
        status = "Downloading cloud object…"
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
