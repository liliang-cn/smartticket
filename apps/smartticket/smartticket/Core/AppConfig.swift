import Foundation

/// App-wide configuration. The API base URL is user-configurable (the platform
/// is self-hosted, so every deployment lives on a different host) and persisted.
@Observable
final class AppConfig {
    static let shared = AppConfig()

    private let baseURLKey = "st.baseURL"
    /// The live demo deployment — a sensible default the user can override.
    static let defaultBaseURL = "https://smartticket.superleo.app"

    var baseURL: String {
        didSet { UserDefaults.standard.set(baseURL, forKey: baseURLKey) }
    }

    private init() {
        baseURL = UserDefaults.standard.string(forKey: baseURLKey) ?? Self.defaultBaseURL
    }

    /// Fully-qualified API root, e.g. `https://host/api/v1`.
    var apiRoot: URL? {
        let trimmed = baseURL.trimmingCharacters(in: .whitespaces)
            .trimmingCharacters(in: CharacterSet(charactersIn: "/"))
        return URL(string: "\(trimmed)/api/v1")
    }
}
