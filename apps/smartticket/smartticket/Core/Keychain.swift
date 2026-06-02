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
enum TokenStore {
    private static let accessKey = "access"
    private static let refreshKey = "refresh"

    static var access: String? { Keychain.get(accessKey) }
    static var refresh: String? { Keychain.get(refreshKey) }

    static func set(access: String, refresh: String) {
        Keychain.set(access, for: accessKey)
        Keychain.set(refresh, for: refreshKey)
    }

    static func clear() {
        Keychain.delete(accessKey)
        Keychain.delete(refreshKey)
    }
}
