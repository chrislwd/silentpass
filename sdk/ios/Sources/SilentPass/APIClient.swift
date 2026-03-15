import Foundation

/// HTTP client for SilentPass API calls.
final class APIClient {
    private let config: SilentPassConfig
    private let session: URLSession

    init(config: SilentPassConfig) {
        self.config = config
        let sessionConfig = URLSessionConfiguration.default
        sessionConfig.timeoutIntervalForRequest = config.timeout
        // Force cellular for silent verification when possible
        sessionConfig.allowsCellularAccess = true
        self.session = URLSession(configuration: sessionConfig)
    }

    func post<T: Decodable>(_ path: String, body: Encodable) async throws -> T {
        guard let url = URL(string: config.baseURL + path) else {
            throw SilentPassError.invalidConfig("Invalid URL: \(config.baseURL + path)")
        }

        var request = URLRequest(url: url)
        request.httpMethod = "POST"
        request.setValue("application/json", forHTTPHeaderField: "Content-Type")
        request.setValue(config.apiKey, forHTTPHeaderField: "X-API-Key")

        let encoder = JSONEncoder()
        request.httpBody = try encoder.encode(body)

        if config.sandbox {
            debugLog("POST \(path)")
        }

        let (data, response) = try await session.data(for: request)

        guard let httpResponse = response as? HTTPURLResponse else {
            throw SilentPassError.networkError(URLError(.badServerResponse))
        }

        if config.sandbox {
            debugLog("Response: \(httpResponse.statusCode)")
        }

        guard (200...299).contains(httpResponse.statusCode) else {
            let errorBody = String(data: data, encoding: .utf8) ?? "Unknown error"
            throw SilentPassError.apiError(httpResponse.statusCode, errorBody)
        }

        do {
            return try JSONDecoder().decode(T.self, from: data)
        } catch {
            throw SilentPassError.decodingError(error)
        }
    }

    /// Creates a URLSession configured to prefer cellular network.
    /// This is critical for silent verification which relies on the mobile
    /// network context to verify the phone number.
    func createCellularSession() -> URLSession {
        let config = URLSessionConfiguration.default
        config.allowsCellularAccess = true
        config.allowsExpensiveNetworkAccess = true
        config.allowsConstrainedNetworkAccess = true
        // Prefer cellular over WiFi
        config.multipathServiceType = .handover
        return URLSession(configuration: config)
    }

    private func debugLog(_ message: String) {
        #if DEBUG
        print("[SilentPass] \(message)")
        #endif
    }
}
