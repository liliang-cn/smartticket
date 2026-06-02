import Foundation
import SwiftUI

/// Owns authentication state and the current user. Drives top-level routing.
@MainActor
@Observable
final class AuthStore {
    enum Phase { case loading, signedOut, signedIn }

    var phase: Phase = .loading
    var user: UserInfo?

    init() {
        NotificationCenter.default.addObserver(
            forName: .authExpired, object: nil, queue: .main
        ) { [weak self] _ in
            Task { @MainActor in self?.handleExpiry() }
        }
    }

    /// On launch: if we hold a token, resolve the current user; else sign out.
    func bootstrap() async {
        #if DEBUG
        // UI-screenshot convenience, gated behind a launch argument so it never
        // runs in normal use: `-demo 1 -demoURL <url> -demoEmail <e> -demoPassword <p>`.
        if UserDefaults.standard.bool(forKey: "demo") {
            if let u = UserDefaults.standard.string(forKey: "demoURL") { AppConfig.shared.baseURL = u }
            let e = UserDefaults.standard.string(forKey: "demoEmail") ?? "admin@smartticket.local"
            let p = UserDefaults.standard.string(forKey: "demoPassword") ?? "admin123"
            do { try await login(email: e, password: p) } catch { phase = .signedOut }
            return
        }
        #endif
        guard TokenStore.access != nil else { phase = .signedOut; return }
        do {
            let me: UserInfo = try await APIClient.shared.get("/auth/me")
            user = me
            phase = .signedIn
        } catch {
            TokenStore.clear()
            phase = .signedOut
        }
    }

    func login(email: String, password: String) async throws {
        let res = try await APIClient.shared.login(email: email, password: password)
        TokenStore.set(access: res.tokens.accessToken, refresh: res.tokens.refreshToken)
        user = res.user
        phase = .signedIn
    }

    func logout() {
        TokenStore.clear()
        user = nil
        phase = .signedOut
    }

    private func handleExpiry() {
        TokenStore.clear()
        user = nil
        phase = .signedOut
    }
}
