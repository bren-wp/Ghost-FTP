import SwiftUI
import UniformTypeIdentifiers

struct RemoteBrowserView: View {
    @ObservedObject var store: SessionStore

    @State private var showingImporter = false
    @State private var showingNewFolder = false
    @State private var newFolderName = ""
    @State private var showingRename = false
    @State private var renameEntry: RemoteEntry?
    @State private var renameName = ""
    @State private var deleteEntry: RemoteEntry?

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
                if store.entries.isEmpty, !store.busy {
                    Text("This directory is empty.")
                        .foregroundStyle(.secondary)
                }

                ForEach(store.entries) { entry in
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
                                    .foregroundStyle(.tertiary)
                            } else {
                                Image(systemName: "arrow.down.circle")
                                    .foregroundStyle(.secondary)
                            }
                        }
                    }
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
        .navigationTitle("ByFTP")
        .navigationBarTitleDisplayMode(.inline)
        .toolbar {
            ToolbarItemGroup(placement: .navigationBarLeading) {
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
                        Label("Upload file", systemImage: "arrow.up.doc")
                    }
                    Button {
                        newFolderName = ""
                        showingNewFolder = true
                    } label: {
                        Label("New folder", systemImage: "folder.badge.plus")
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
            Text(store.status)
                .font(.caption)
                .foregroundStyle(.secondary)
                .frame(maxWidth: .infinity)
                .padding(.vertical, 8)
                .background(.thinMaterial)
        }
        .fileImporter(isPresented: $showingImporter, allowedContentTypes: [.data]) { result in
            switch result {
            case .success(let url): store.upload(url)
            case .failure(let error): store.errorMessage = error.localizedDescription
            }
        }
        .alert("New folder", isPresented: $showingNewFolder) {
            TextField("Folder name", text: $newFolderName)
            Button("Cancel", role: .cancel) {}
            Button("Create") { store.createFolder(named: newFolderName) }
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
