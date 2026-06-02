import Foundation

/// The backend's standard response envelope: `{ success, data, error, meta }`.
private struct Envelope<T: Decodable>: Decodable {
    let data: T?
    let error: APIErrorBody?
    let meta: PageMeta?
}

struct APIErrorBody: Decodable {
    let code: String?
    let message: String?
}

enum APIError: LocalizedError {
    case notConfigured
    case unauthorized
    case server(status: Int, message: String)
    case decoding(String)
    case transport(String)
    case empty

    var errorDescription: String? {
        switch self {
        case .notConfigured: return "Set a valid server URL in Settings."
        case .unauthorized: return "Your session has expired. Please sign in again."
        case .server(_, let message): return message
        case .decoding(let m): return "Unexpected response: \(m)"
        case .transport(let m): return m
        case .empty: return "The server returned no data."
        }
    }
}

/// Async REST client for the SmartTicket API. Attaches the bearer token,
/// transparently refreshes it once on a 401, and unwraps the data envelope.
actor APIClient {
    static let shared = APIClient()

    private let session: URLSession
    private var refreshing: Task<Bool, Never>?

    init() {
        let cfg = URLSessionConfiguration.default
        cfg.timeoutIntervalForRequest = 30
        // Preserve the bearer token across redirects. The API issues trailing-
        // slash 301s (e.g. `/users/` → `/users`); without this the Authorization
        // header is dropped on the followed request, producing a spurious 401.
        session = URLSession(configuration: cfg, delegate: RedirectAuthDelegate(), delegateQueue: nil)
    }

    // MARK: Public verbs

    func get<T: Decodable>(_ path: String, query: [String: String?] = [:]) async throws -> T {
        try await send(path: path, method: "GET", query: query, body: Optional<Data>.none).0
    }

    /// GET a paginated list endpoint, returning items + pagination meta.
    func getList<T: Decodable>(_ path: String, query: [String: String?] = [:]) async throws -> (items: [T], meta: PageMeta?) {
        let (items, meta): ([T], PageMeta?) = try await send(path: path, method: "GET", query: query, body: Optional<Data>.none)
        return (items, meta)
    }

    func post<T: Decodable, B: Encodable>(_ path: String, body: B) async throws -> T {
        try await send(path: path, method: "POST", query: [:], body: body).0
    }

    @discardableResult
    func put<T: Decodable, B: Encodable>(_ path: String, body: B) async throws -> T {
        try await send(path: path, method: "PUT", query: [:], body: body).0
    }

    // MARK: Auth (no envelope on login/refresh)

    func login(email: String, password: String) async throws -> LoginResponse {
        let url = try makeURL(path: "/auth/login", query: [:])
        var req = URLRequest(url: url)
        req.httpMethod = "POST"
        req.setValue("application/json", forHTTPHeaderField: "Content-Type")
        req.httpBody = try JSONEncoder().encode(["email": email, "password": password])
        let (data, resp) = try await perform(req)
        guard let http = resp as? HTTPURLResponse else { throw APIError.transport("No response") }
        if http.statusCode == 401 { throw APIError.server(status: 401, message: String(localized: "Invalid email or password")) }
        guard (200..<300).contains(http.statusCode) else { throw decodeError(data, status: http.statusCode) }
        do { return try JSONDecoder().decode(LoginResponse.self, from: data) }
        catch { throw APIError.decoding(String(describing: error)) }
    }

    // MARK: Core request

    /// Returns (decoded data, meta). Decodes `Envelope<T>` and unwraps `.data`.
    private func send<T: Decodable, B: Encodable>(
        path: String, method: String, query: [String: String?], body: B?, isRetry: Bool = false
    ) async throws -> (T, PageMeta?) {
        let url = try makeURL(path: path, query: query)
        var req = URLRequest(url: url)
        req.httpMethod = method
        req.setValue("application/json", forHTTPHeaderField: "Accept")
        if let token = TokenStore.access {
            req.setValue("Bearer \(token)", forHTTPHeaderField: "Authorization")
        }
        if let body {
            req.setValue("application/json", forHTTPHeaderField: "Content-Type")
            req.httpBody = try JSONEncoder().encode(body)
        }

        let (data, resp) = try await perform(req)
        guard let http = resp as? HTTPURLResponse else { throw APIError.transport("No response") }

        if http.statusCode == 401 {
            // Try a single refresh, then retry the original request once.
            if !isRetry, await attemptRefresh() {
                return try await send(path: path, method: method, query: query, body: body, isRetry: true)
            }
            TokenStore.clear()
            await MainActor.run { NotificationCenter.default.post(name: .authExpired, object: nil) }
            throw APIError.unauthorized
        }
        guard (200..<300).contains(http.statusCode) else { throw decodeError(data, status: http.statusCode) }

        do {
            let env = try JSONDecoder().decode(Envelope<T>.self, from: data)
            guard let value = env.data else { throw APIError.empty }
            return (value, env.meta)
        } catch let e as APIError {
            throw e
        } catch {
            // Some endpoints may return a bare value rather than an envelope.
            if let value = try? JSONDecoder().decode(T.self, from: data) { return (value, nil) }
            throw APIError.decoding(String(describing: error))
        }
    }

    // MARK: Refresh

    private func attemptRefresh() async -> Bool {
        if let task = refreshing { return await task.value }
        let task = Task<Bool, Never> { [self] in
            guard let refresh = TokenStore.refresh,
                  let url = try? makeURL(path: "/auth/refresh", query: [:]) else { return false }
            var req = URLRequest(url: url)
            req.httpMethod = "POST"
            req.setValue("application/json", forHTTPHeaderField: "Content-Type")
            req.httpBody = try? JSONEncoder().encode(["refresh_token": refresh])
            guard let (data, resp) = try? await perform(req),
                  let http = resp as? HTTPURLResponse, (200..<300).contains(http.statusCode) else { return false }
            // Tokens may be top-level or under `data`.
            if let r = try? JSONDecoder().decode(RefreshResponse.self, from: data) {
                TokenStore.set(access: r.tokens.accessToken, refresh: r.tokens.refreshToken)
                return true
            }
            if let env = try? JSONDecoder().decode(Envelope<RefreshResponse>.self, from: data), let r = env.data {
                TokenStore.set(access: r.tokens.accessToken, refresh: r.tokens.refreshToken)
                return true
            }
            return false
        }
        refreshing = task
        let ok = await task.value
        refreshing = nil
        return ok
    }

    // MARK: Helpers

    private func perform(_ req: URLRequest) async throws -> (Data, URLResponse) {
        do { return try await session.data(for: req) }
        catch { throw APIError.transport(error.localizedDescription) }
    }

    private func makeURL(path: String, query: [String: String?]) throws -> URL {
        guard let root = AppConfig.shared.apiRoot else { throw APIError.notConfigured }
        guard var comps = URLComponents(url: root.appendingPathComponent(path), resolvingAgainstBaseURL: false) else {
            throw APIError.notConfigured
        }
        let items = query.compactMap { key, value -> URLQueryItem? in
            guard let value, !value.isEmpty else { return nil }
            return URLQueryItem(name: key, value: value)
        }
        if !items.isEmpty { comps.queryItems = items }
        guard let url = comps.url else { throw APIError.notConfigured }
        return url
    }

    private func decodeError(_ data: Data, status: Int) -> APIError {
        if let env = try? JSONDecoder().decode(Envelope<EmptyValue>.self, from: data),
           let message = env.error?.message, !message.isEmpty {
            return .server(status: status, message: message)
        }
        return .server(status: status, message: "Request failed (\(status)).")
    }
}

/// Placeholder for endpoints whose envelope data we don't need to decode.
private struct EmptyValue: Decodable {}

/// Re-attaches the `Authorization` header when URLSession follows a redirect,
/// which it otherwise strips. Without this, the API's trailing-slash 301s sign
/// the user out mid-session.
private final class RedirectAuthDelegate: NSObject, URLSessionTaskDelegate {
    func urlSession(
        _ session: URLSession,
        task: URLSessionTask,
        willPerformHTTPRedirection response: HTTPURLResponse,
        newRequest request: URLRequest,
        completionHandler: @escaping (URLRequest?) -> Void
    ) {
        var req = request
        if req.value(forHTTPHeaderField: "Authorization") == nil,
           let auth = task.originalRequest?.value(forHTTPHeaderField: "Authorization") {
            req.setValue(auth, forHTTPHeaderField: "Authorization")
        }
        completionHandler(req)
    }
}

extension Notification.Name {
    /// Posted when a refresh fails and the user must re-authenticate.
    static let authExpired = Notification.Name("st.authExpired")
}
