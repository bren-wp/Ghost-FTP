import Foundation

/// Native macOS localization catalog, shared with the desktop 24-language
/// dictionaries. All lookups preserve the original English key as a safe
/// fallback; transport protocol tokens and user-provided paths are never
/// interpreted as translations.
enum L10n {
    static let supportedLanguages = ["en", "hr", "cs", "sk", "hu", "ro", "bg", "el", "tr", "uk", "da", "sv", "no", "de", "fr", "es", "it", "pt", "nl", "pl", "sl", "sr", "bs", "mk"]

    private static let translations: [String: [String: String]] = {
        guard let path = Bundle.module.url(forResource: "ghostftp-translations", withExtension: "json"),
              let contents = try? Data(contentsOf: path),
              let parsed = try? JSONSerialization.jsonObject(with: contents) as? [String: Any],
              let strings = parsed["strings"] as? [String: [String: String]] else {
            return [:]
        }
        return strings
    }()

    static func text(_ english: String, language: String? = nil) -> String {
        let identifier = (language ?? Locale.preferredLanguages.first ?? "en")
            .lowercased().split(separator: "-").first.map(String.init) ?? "en"
        let locale = (identifier == "nb" || identifier == "nn") ? "no" : identifier
        if locale == "en" { return english }
        guard supportedLanguages.contains(locale),
              let value = translations[locale]?[english], !value.isEmpty else {
            return english
        }
        return value
    }

    /// Used by CI-backed tests to distinguish a translated value from an
    /// accidental English fallback; content certification is a separate gate.
    static func hasTranslation(_ key: String, language: String) -> Bool {
        let result = translations[language]?[key]
        return result != nil && !(result?.isEmpty ?? true)
    }
}
