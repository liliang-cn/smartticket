import Foundation
import Security

/// Minimal Keychain-backed store for the JWT access/refresh tokens.
enum Keychain {
    private static let service = "superleo.smartticket.tokens"

    static func set(_ value: String?, for key: String) {
        guard let value, let data = value.data(using: .utf8) else {
            delete(key)
            return
        }
        let query: [String: Any] = [
            kSecClass as String: kSecClassGenericPassword,
            kSecAttrService as String: service,
            kSecAttrAccount as String: key,
        ]
        let attrs: [String: Any] = [kSecValueData as String: data]
        let status = SecItemUpdate(query as CFDictionary, attrs as CFDictionary)
        if status == errSecItemNotFound {
            var add = query
            add[kSecValueData as String] = data
            add[kSecAttrAccessible as String] = kSecAttrAccessibleAfterFirstUnlock
            SecItemAdd(add as CFDictionary, nil)
        }
    }

    static func get(_ key: String) -> String? {
        let query: [String: Any] = [
            kSecClass as String: kSecClassGenericPassword,
            kSecAttrService as String: service,
            kSecAttrAccount as String: key,
            kSecReturnData as String: true,
            kSecMatchLimit as String: kSecMatchLimitOne,
        ]
        var item: CFTypeRef?
        guard SecItemCopyMatching(query as CFDictionary, &item) == errSecSuccess,
              let data = item as? Data else { return nil }
        return String(data: data, encoding: .utf8)
    }

    static func delete(_ key: String) {
        let query: [String: Any] = [
            kSecClass as String: kSecClassGenericPassword,
            kSecAttrService as String: service,
            kSecAttrAccount as String: key,
        ]
        SecItemDelete(query as CFDictionary)
    }
}

/// Token storage facade used by the API client and auth store.
///
/// Tokens are cached in memory and mirrored to the Keychain. The in-memory copy
/// keeps the current session working even when the Keychain is unavailable
/// (e.g. unsigned builds, or transient `SecItem` failures); the Keychain copy
/// provides persistence across app launches.
enum TokenStore {
    private static let accessKey = "access"
    private static let refreshKey = "refresh"

    nonisolated(unsafe) private static var memAccess: String?
    nonisolated(unsafe) private static var memRefresh: String?

    static var access: String? { memAccess ?? Keychain.get(accessKey) }
    static var refresh: String? { memRefresh ?? Keychain.get(refreshKey) }

    static func set(access: String, refresh: String) {
        memAccess = access
        memRefresh = refresh
        Keychain.set(access, for: accessKey)
        Keychain.set(refresh, for: refreshKey)
    }

    static func clear() {
        memAccess = nil
        memRefresh = nil
        Keychain.delete(accessKey)
        Keychain.delete(refreshKey)
    }
}
