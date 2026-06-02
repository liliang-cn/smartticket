import Foundation
import SwiftUI

/// The languages the app ships, with native display names — mirrors the web console.
struct AppLanguage: Identifiable, Hashable {
    let code: String          // nil-equivalent handled by `systemDefault`
    let label: String
    var id: String { code }
}

enum AppLanguages {
    /// `code == ""` means "follow the device language".
    static let systemDefault = AppLanguage(code: "", label: "System")
    static let all: [AppLanguage] = [
        systemDefault,
        AppLanguage(code: "en", label: "English"),
        AppLanguage(code: "zh", label: "中文"),
        AppLanguage(code: "es", label: "Español"),
        AppLanguage(code: "de", label: "Deutsch"),
        AppLanguage(code: "ja", label: "日本語"),
        AppLanguage(code: "ko", label: "한국어"),
        AppLanguage(code: "it", label: "Italiano"),
    ]
}

/// Holds the user's chosen UI language and exposes a `Locale` the root view
/// injects into the environment, so SwiftUI re-renders instantly on change.
@MainActor
@Observable
final class LanguageStore {
    static let shared = LanguageStore()
    private let key = "st.lang"

    /// Empty string = follow the system language.
    var code: String {
        didSet { UserDefaults.standard.set(code, forKey: key) }
    }

    private init() {
        code = UserDefaults.standard.string(forKey: key) ?? ""
    }

    /// The locale to inject, or `nil` to defer to the system.
    var locale: Locale? {
        code.isEmpty ? nil : Locale(identifier: code)
    }
}
