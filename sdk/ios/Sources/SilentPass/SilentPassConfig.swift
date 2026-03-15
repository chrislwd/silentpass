import Foundation

/// Configuration for the SilentPass SDK.
public struct SilentPassConfig {
    /// API base URL.
    public let baseURL: String
    /// API key for authentication.
    public let apiKey: String
    /// Application identifier.
    public let appID: String
    /// Request timeout in seconds.
    public let timeout: TimeInterval
    /// Enable sandbox mode (logs debug info).
    public let sandbox: Bool

    public init(
        baseURL: String = "https://api.silentpass.io",
        apiKey: String,
        appID: String,
        timeout: TimeInterval = 15,
        sandbox: Bool = false
    ) {
        self.baseURL = baseURL
        self.apiKey = apiKey
        self.appID = appID
        self.timeout = timeout
        self.sandbox = sandbox
    }
}
