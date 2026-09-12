import AppKit
import Foundation

private extension NSColor {
    convenience init(rgb: UInt32) {
        let red = CGFloat((rgb >> 16) & 0xff) / 255.0
        let green = CGFloat((rgb >> 8) & 0xff) / 255.0
        let blue = CGFloat(rgb & 0xff) / 255.0
        self.init(srgbRed: red, green: green, blue: blue, alpha: 1.0)
    }
}

private enum GhostPalette {
    static let workspace = NSColor(rgb: 0xEEF1F5)
    static let panel = NSColor(rgb: 0xF6F8FB)
    static let list = NSColor(rgb: 0xFAFBFD)
    static let text = NSColor(rgb: 0x111827)
    static let muted = NSColor(rgb: 0x667085)
    static let accent = NSColor(rgb: 0x2563EB)
    static let border = NSColor(rgb: 0xD7DDE6)
}

private final class PanelView: NSView {
    override init(frame frameRect: NSRect) {
        super.init(frame: frameRect)
        wantsLayer = true
        layer?.backgroundColor = GhostPalette.panel.cgColor
        layer?.borderColor = GhostPalette.border.cgColor
        layer?.borderWidth = 1
        layer?.cornerRadius = 12
    }

    required init?(coder: NSCoder) {
        nil
    }
}

private final class AppDelegate: NSObject, NSApplicationDelegate, NSWindowDelegate {
    private var window: NSWindow?

    func applicationDidFinishLaunching(_ notification: Notification) {
        NSApp.setActivationPolicy(.regular)
        NSApp.appearance = NSAppearance(named: .aqua)

        let window = NSWindow(
            contentRect: NSRect(x: 0, y: 0, width: 1500, height: 960),
            styleMask: [.titled, .closable, .miniaturizable, .resizable],
            backing: .buffered,
            defer: false
        )
        window.title = "Ghost FTP \(productVersion()) — Brendigo"
        window.minSize = NSSize(width: 1100, height: 720)
        window.isReleasedWhenClosed = false
        window.delegate = self
        window.center()

        let root = NSView(frame: window.contentView?.bounds ?? .zero)
        root.autoresizingMask = [.width, .height]
        root.wantsLayer = true
        root.layer?.backgroundColor = GhostPalette.workspace.cgColor
        window.contentView = root

        let title = label("Ghost FTP", size: 28, weight: .bold, color: GhostPalette.text)
        let subtitle = label(
            "macOS parity development surface",
            size: 14,
            weight: .regular,
            color: GhostPalette.muted
        )
        let badge = label("ENGINE BRIDGE IN PROGRESS", size: 12, weight: .semibold, color: GhostPalette.accent)

        let foundation = PanelView(frame: .zero)
        let foundationTitle = label("Native AppKit foundation", size: 18, weight: .semibold, color: GhostPalette.text)
        let foundationBody = wrappingLabel(
            "This bundle proves the dedicated native macOS build, universal Intel + Apple Silicon executable, product identity and Ghost FTP desktop palette. Windows remains the canonical visual and behavior reference.",
            size: 14
        )

        let parity = PanelView(frame: .zero)
        let parityTitle = label("Fail-closed parity rule", size: 18, weight: .semibold, color: GhostPalette.text)
        let parityBody = wrappingLabel(
            "FTP controls are intentionally not drawn until each action is connected to the same internal/api.Engine behavior used by Windows/Linux. Ghost FTP does not count placeholder or dead controls as parity.",
            size: 14
        )

        [title, subtitle, badge, foundation, parity].forEach {
            $0.translatesAutoresizingMaskIntoConstraints = false
            root.addSubview($0)
        }
        [foundationTitle, foundationBody].forEach {
            $0.translatesAutoresizingMaskIntoConstraints = false
            foundation.addSubview($0)
        }
        [parityTitle, parityBody].forEach {
            $0.translatesAutoresizingMaskIntoConstraints = false
            parity.addSubview($0)
        }

        NSLayoutConstraint.activate([
            title.leadingAnchor.constraint(equalTo: root.leadingAnchor, constant: 30),
            title.topAnchor.constraint(equalTo: root.topAnchor, constant: 26),
            subtitle.leadingAnchor.constraint(equalTo: title.leadingAnchor),
            subtitle.topAnchor.constraint(equalTo: title.bottomAnchor, constant: 3),
            badge.trailingAnchor.constraint(equalTo: root.trailingAnchor, constant: -30),
            badge.centerYAnchor.constraint(equalTo: title.centerYAnchor),

            foundation.leadingAnchor.constraint(equalTo: root.leadingAnchor, constant: 30),
            foundation.trailingAnchor.constraint(equalTo: root.centerXAnchor, constant: -8),
            foundation.topAnchor.constraint(equalTo: subtitle.bottomAnchor, constant: 28),
            foundation.heightAnchor.constraint(greaterThanOrEqualToConstant: 210),

            parity.leadingAnchor.constraint(equalTo: root.centerXAnchor, constant: 8),
            parity.trailingAnchor.constraint(equalTo: root.trailingAnchor, constant: -30),
            parity.topAnchor.constraint(equalTo: foundation.topAnchor),
            parity.heightAnchor.constraint(equalTo: foundation.heightAnchor),

            foundationTitle.leadingAnchor.constraint(equalTo: foundation.leadingAnchor, constant: 22),
            foundationTitle.trailingAnchor.constraint(equalTo: foundation.trailingAnchor, constant: -22),
            foundationTitle.topAnchor.constraint(equalTo: foundation.topAnchor, constant: 22),
            foundationBody.leadingAnchor.constraint(equalTo: foundationTitle.leadingAnchor),
            foundationBody.trailingAnchor.constraint(equalTo: foundationTitle.trailingAnchor),
            foundationBody.topAnchor.constraint(equalTo: foundationTitle.bottomAnchor, constant: 14),
            foundationBody.bottomAnchor.constraint(lessThanOrEqualTo: foundation.bottomAnchor, constant: -22),

            parityTitle.leadingAnchor.constraint(equalTo: parity.leadingAnchor, constant: 22),
            parityTitle.trailingAnchor.constraint(equalTo: parity.trailingAnchor, constant: -22),
            parityTitle.topAnchor.constraint(equalTo: parity.topAnchor, constant: 22),
            parityBody.leadingAnchor.constraint(equalTo: parityTitle.leadingAnchor),
            parityBody.trailingAnchor.constraint(equalTo: parityTitle.trailingAnchor),
            parityBody.topAnchor.constraint(equalTo: parityTitle.bottomAnchor, constant: 14),
            parityBody.bottomAnchor.constraint(lessThanOrEqualTo: parity.bottomAnchor, constant: -22),
        ])

        window.makeKeyAndOrderFront(nil)
        NSApp.activate(ignoringOtherApps: true)
        self.window = window
    }

    func applicationShouldTerminateAfterLastWindowClosed(_ sender: NSApplication) -> Bool {
        true
    }

    private func productVersion() -> String {
        (Bundle.main.object(forInfoDictionaryKey: "CFBundleShortVersionString") as? String) ?? "dev"
    }

    private func label(_ text: String, size: CGFloat, weight: NSFont.Weight, color: NSColor) -> NSTextField {
        let field = NSTextField(labelWithString: text)
        field.font = NSFont.systemFont(ofSize: size, weight: weight)
        field.textColor = color
        field.backgroundColor = .clear
        return field
    }

    private func wrappingLabel(_ text: String, size: CGFloat) -> NSTextField {
        let field = label(text, size: size, weight: .regular, color: GhostPalette.muted)
        field.maximumNumberOfLines = 0
        field.lineBreakMode = .byWordWrapping
        field.setContentCompressionResistancePriority(.defaultLow, for: .horizontal)
        return field
    }
}

let application = NSApplication.shared
let delegate = AppDelegate()
application.delegate = delegate
application.run()
