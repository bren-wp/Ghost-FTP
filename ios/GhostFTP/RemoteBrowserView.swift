import SwiftUI
import UniformTypeIdentifiers

struct RemoteBrowserView: View {
    @ObservedObject var store: SessionStore

    @State private var searchText = ""
    @State private var showingImporter = false
    @State private var showingNewFolder = false
    @State private var newFolderName = ""
    @State private var showingGoToPath = false
    @State private var goToPath = ""
    @State private var showingRename = false
    @State private var renameEntry: RemoteEntry?
    @State private var renameName = ""
    @State private var deleteEntry: RemoteEntry?

    private var visibleEntries: [RemoteEntry] {
        let query = searchText.trimmingCharacters(in: .whitespacesAndNewlines).lowercased()
        guard !query.isEmpty else { return store.entries }
        return store.entries.filter { $0.name.lowercased().contains(query) }
    }

    var body: some View {
        List {
            Section {
                HStack(spacing: 8) {
                    Image(systemName: "folder")
                    Text(store.currentPath)
                        .font(.system(.body, design: .monospaced))
                        .lineLimit(1)
                        .truncationMode(.middle)
                    Spacer()
                    if store.busy { ProgressView() }
                }
            }

            if let diagnostics = store.hostingDiagnostics {
                Section("Shared hosting") {
                    Label(
                        diagnostics.secure ? "Secure transport" : "Plain FTP · unencrypted",
                        systemImage: diagnostics.secure ? "lock.shield" : "exclamationmark.triangle"
                    )
                    if let webRoot = diagnostics.webRoot {
                        Label("Detected web root: \(webRoot)", systemImage: "globe")
                        Text("Detected from the authenticated root listing. Ghost FTP does not open or save this path automatically.")
                            .font(.caption)
                            .foregroundStyle(.secondary)
                    } else {
                        Label("Authenticated account root ready", systemImage: "externaldrive.connected.to.line.below")
                        Text("No common web-root directory was identified in the initial listing.")
                            .font(.caption)
                            .foregroundStyle(.secondary)
                    }
                }
            }

            if let download = store.downloadedFile {
                Section("Downloaded file") {
                    ShareLink(item: download) {
                        Label("Share or save \(download.lastPathComponent)", systemImage: "square.and.arrow.up")
                    }
                    Button("Clear downloaded copy", role: .destructive) {
                        store.clearDownloadedFile()
                    }
                }
            }

            Section("Remote files") {
                if visibleEntries.isEmpty, !store.busy {
                    VStack(spacing: 8) {
                        Image(systemName: searchText.isEmpty ? "folder" : "magnifyingglass")
                            .font(.title2)
                            .foregroundStyle(.secondary)
                        Text(searchText.isEmpty ? "This directory is empty" : "No matching files")
                            .font(.headline)
                        Text(searchText.isEmpty ? "Use the menu to upload files or create a folder." : "Change or clear the filter to see more files.")
                            .font(.footnote)
                            .foregroundStyle(.secondary)
                            .multilineTextAlignment(.center)
                    }
                    .frame(maxWidth: .infinity)
                    .padding(.vertical, 22)
                }

                ForEach(visibleEntries) { entry in
                    Button {
                        if entry.isDirectory {
                            do { store.openDirectory(try RemotePath.child(store.currentPath, entry.name)) }
                            catch { store.errorMessage = error.localizedDescription }
                        } else {
                            store.download(entry)
                        }
                    } label: {
                        HStack(spacing: 12) {
                            Image(systemName: entry.isDirectory ? "folder.fill" : "doc")
                                .frame(width: 24)
                            VStack(alignment: .leading, spacing: 3) {
                                Text(entry.name)
                                    .foregroundStyle(.primary)
                                    .lineLimit(1)
                                Text(entry.displaySize)
                                    .font(.caption)
                                    .foregroundStyle(.secondary)
                            }
                            Spacer()
                            if entry.isDirectory {
                                Image(systemName: "chevron.right")
                                    .font(.caption)
                                    .foregroundStyle(.secondary)
                            } else {
                                Image(systemName: "arrow.down.circle")
                                    .foregroundStyle(.secondary)
                            }
                        }
                        .contentShape(Rectangle())
                    }
                    .buttonStyle(.plain)
                    .disabled(store.busy)
                    .contextMenu {
                        if !entry.isDirectory {
                            Button {
                                store.download(entry)
                            } label: {
                                Label("Download", systemImage: "arrow.down.circle")
                            }
                        }
                        Button {
                            renameEntry = entry
                            renameName = entry.name
                            showingRename = true
                        } label: {
                            Label("Rename", systemImage: "pencil")
                        }
                        Button(role: .destructive) {
                            deleteEntry = entry
                        } label: {
                            Label("Delete", systemImage: "trash")
                        }
                    }
                }
            }
        }
        .searchable(text: $searchText, placement: .navigationBarDrawer(displayMode: .always), prompt: "Filter files")
        .navigationTitle("Ghost FTP")
        .navigationBarTitleDisplayMode(.inline)
        .toolbar {
            ToolbarItem(placement: .navigationBarLeading) {
                Button {
                    store.goUp()
                } label: {
                    Label("Up", systemImage: "arrow.up")
                }
                .disabled(store.busy || store.currentPath == "/")
            }
            ToolbarItemGroup(placement: .navigationBarTrailing) {
                Button {
                    store.refresh()
                } label: {
                    Label("Refresh", systemImage: "arrow.clockwise")
                }
                .disabled(store.busy)

                Menu {
                    Button {
                        showingImporter = true
                    } label: {
                        Label("Upload files", systemImage: "square.and.arrow.up")
                    }
                    Button {
                        newFolderName = ""
                        showingNewFolder = true
                    } label: {
                        Label("New folder", systemImage: "folder.badge.plus")
                    }
                    Button {
                        goToPath = store.currentPath
                        showingGoToPath = true
                    } label: {
                        Label("Go to path", systemImage: "location")
                    }
                    if store.hasSavedConnection {
                        Divider()
                        Button(role: .destructive) {
                            store.forgetSavedConnection()
                        } label: {
                            Label("Forget saved connection", systemImage: "trash")
                        }
                    }
                    Divider()
                    Button(role: .destructive) {
                        store.disconnect()
                    } label: {
                        Label("Disconnect", systemImage: "bolt.slash")
                    }
                } label: {
                    Image(systemName: "ellipsis.circle")
                }
                .disabled(store.busy)
            }
        }
        .safeAreaInset(edge: .bottom) {
            VStack(spacing: 8) {
                if let detail = store.transferDetail {
                    VStack(alignment: .leading, spacing: 6) {
                        if let fraction = store.transferFraction {
                            ProgressView(value: fraction)
                        } else {
                            ProgressView()
                                .controlSize(.small)
                        }
                        HStack(spacing: 8) {
                            Text(detail)
                                .lineLimit(2)
                                .foregroundStyle(.secondary)
                            Spacer(minLength: 8)
                            if store.canStopAfterCurrent {
                                Button {
                                    store.requestStopAfterCurrent()
                                } label: {
                                    Label("Stop after file", systemImage: "stop.circle")
                                }
                                .buttonStyle(.bordered)
                                .controlSize(.small)
                            }
                        }
                        .font(.caption)
                    }
                    .padding(.horizontal, 14)
                }

                HStack(spacing: 8) {
                    Circle()
                        .fill(store.busy ? Color.orange : Color.green)
                        .frame(width: 7, height: 7)
                    Text(store.status)
                        .lineLimit(1)
                    Spacer()
                    Text("\(visibleEntries.count) items")
                        .foregroundStyle(.secondary)
                }
                .font(.caption)
                .foregroundStyle(.secondary)
                .padding(.horizontal, 14)
            }
            .padding(.vertical, 9)
            .background(.thinMaterial)
        }
        .fileImporter(isPresented: $showingImporter, allowedContentTypes: [.data], allowsMultipleSelection: true) { result in
            switch result {
            case .success(let urls): store.upload(urls)
            case .failure(let error): store.errorMessage = error.localizedDescription
            }
        }
        .alert("New folder", isPresented: $showingNewFolder) {
            TextField("Folder name", text: $newFolderName)
            Button("Cancel", role: .cancel) {}
            Button("Create") { store.createFolder(named: newFolderName) }
        }
        .alert("Go to path", isPresented: $showingGoToPath) {
            TextField("/public_html", text: $goToPath)
                .textInputAutocapitalization(.never)
                .autocorrectionDisabled()
            Button("Cancel", role: .cancel) {}
            Button("Open") { store.openDirectory(goToPath) }
        } message: {
            Text("Enter a canonical remote path beginning with '/'.")
        }
        .alert("Rename", isPresented: $showingRename) {
            TextField("New name", text: $renameName)
            Button("Cancel", role: .cancel) { renameEntry = nil }
            Button("Save") {
                if let entry = renameEntry { store.rename(entry, to: renameName) }
                renameEntry = nil
            }
        }
        .confirmationDialog(
            "Delete \(deleteEntry?.name ?? "item")?",
            isPresented: Binding(
                get: { deleteEntry != nil },
                set: { if !$0 { deleteEntry = nil } }
            ),
            titleVisibility: .visible
        ) {
            Button("Delete", role: .destructive) {
                if let entry = deleteEntry { store.delete(entry) }
                deleteEntry = nil
            }
            Button("Cancel", role: .cancel) { deleteEntry = nil }
        } message: {
            Text("This action cannot be undone.")
        }
    }
}
