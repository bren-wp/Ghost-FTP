import Foundation

struct RemoteEntry: Identifiable, Hashable, Sendable {
    let name: String
    let isDirectory: Bool
    let size: Int64
    let modifiedAt: Date?

    var id: String { (isDirectory ? "d:" : "f:") + name }

    var displaySize: String {
        guard !isDirectory else { return "Folder" }
        return ByteCountFormatter.string(fromByteCount: max(0, size), countStyle: .file)
    }
}

enum RemotePath {
    static func validateName(_ raw: String) throws -> String {
        let name = raw.trimmingCharacters(in: .whitespacesAndNewlines)
        guard !name.isEmpty else { throw ValidationError("A name is required.") }
        guard name != ".", name != ".." else { throw ValidationError("That name is not allowed.") }
        guard !name.contains("/"), !name.contains("\\"), !name.contains("\0") else {
            throw ValidationError("Names cannot contain path separators or NUL characters.")
        }
        return name
    }

    static func normalizeDirectory(_ raw: String) throws -> String {
        guard raw.hasPrefix("/") else { throw ValidationError("Remote paths must start with '/'.") }
        guard !raw.contains("\\"), !raw.contains("\0"), !raw.contains("//") else {
            throw ValidationError("Remote path is not canonical.")
        }
        if raw == "/" { return "/" }

        let parts = raw.dropFirst().split(separator: "/", omittingEmptySubsequences: false)
        guard !parts.isEmpty else { return "/" }
        for part in parts {
            guard !part.isEmpty, part != ".", part != ".." else {
                throw ValidationError("Remote path contains an unsafe component.")
            }
        }
        return "/" + parts.joined(separator: "/")
    }

    static func child(_ parent: String, _ rawName: String) throws -> String {
        let directory = try normalizeDirectory(parent)
        let name = try validateName(rawName)
        return directory == "/" ? "/\(name)" : "\(directory)/\(name)"
    }

    static func parent(_ raw: String) throws -> String {
        let path = try normalizeDirectory(raw)
        guard path != "/" else { return "/" }
        let pieces = path.split(separator: "/")
        guard pieces.count > 1 else { return "/" }
        return "/" + pieces.dropLast().joined(separator: "/")
    }
}

enum FTPPathMapper {
    static func normalizeLoginRoot(_ raw: String?) throws -> String {
        guard var root = raw?.trimmingCharacters(in: .whitespacesAndNewlines), !root.isEmpty, root != "." else {
            return ""
        }
        root = root.replacingOccurrences(of: "\\", with: "/")
        guard !root.contains("\0"), !root.contains("//") else {
            throw ValidationError("FTP server returned a noncanonical login directory.")
        }
        while root.count > 1, root.hasSuffix("/") { root.removeLast() }
        if root == "/" { return root }

        let check = root.hasPrefix("/") ? String(root.dropFirst()) : root
        let pieces = check.split(separator: "/", omittingEmptySubsequences: false)
        guard !pieces.isEmpty else { throw ValidationError("FTP server returned an unsafe login directory.") }
        for part in pieces {
            guard !part.isEmpty, part != ".", part != ".." else {
                throw ValidationError("FTP server returned an unsafe login directory.")
            }
        }
        return root
    }

    static func map(loginRoot rawRoot: String?, uiPath rawPath: String) throws -> String {
        let root = try normalizeLoginRoot(rawRoot)
        let path = try RemotePath.normalizeDirectory(rawPath)
        if path == "/" { return root.isEmpty ? "." : root }

        let relative = String(path.dropFirst())
        if root.isEmpty || root == "." { return relative }
        if root == "/" { return "/" + relative }
        return root + "/" + relative
    }
}
