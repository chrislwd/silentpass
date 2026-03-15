import Foundation

/// SilentPass iOS SDK - Main entry point.
///
/// Usage:
/// ```swift
/// let sp = SilentPass(config: SilentPassConfig(
///     apiKey: "sk_your_api_key",
///     appID: "your_app_id"
/// ))
///
/// let result = try await sp.verify(
///     phoneNumber: "+6281234567890",
///     countryCode: "ID",
///     useCase: .signup
/// )
///
/// switch result {
/// case .verified(let token):
///     // Phone verified, send token to your backend
/// case .otpRequired(let sessionID):
///     // Silent failed, show OTP input
/// case .denied(let reason):
///     // Blocked by risk check
/// }
/// ```
public final class SilentPass {

    private let config: SilentPassConfig
    private let apiClient: APIClient

    public init(config: SilentPassConfig) {
        self.config = config
        self.apiClient = APIClient(config: config)
    }

    // MARK: - High-Level API

    /// Performs the full verification flow: silent verify → auto-fallback to OTP if needed.
    /// This is the recommended entry point for most use cases.
    public func verify(
        phoneNumber: String,
        countryCode: String,
        useCase: UseCase = .signup
    ) async throws -> VerificationResult {
        let context = DeviceContextCollector.collect()

        // Step 1: Create session
        let session = try await createSession(
            phoneNumber: phoneNumber,
            countryCode: countryCode,
            verificationType: .silentOrOTP,
            useCase: useCase,
            context: context
        )

        // Step 2: Attempt silent verification
        if session.recommendedAction == "silent_verify" {
            let silentResult = try await silentVerify(sessionID: session.sessionID)

            switch silentResult.status {
            case .verified:
                return .verified(token: silentResult.token ?? "")
            case .fallbackRequired:
                return .otpRequired(sessionID: session.sessionID)
            case .failed:
                return .otpRequired(sessionID: session.sessionID)
            }
        }

        // Silent not available, OTP needed
        return .otpRequired(sessionID: session.sessionID)
    }

    /// Sends an OTP to the user via the specified channel.
    public func sendOTP(
        sessionID: String,
        channel: OTPChannel = .sms,
        locale: String? = nil
    ) async throws -> OTPSendResponse {
        struct Request: Encodable {
            let sessionID: String
            let channel: String
            let locale: String?

            enum CodingKeys: String, CodingKey {
                case sessionID = "session_id"
                case channel
                case locale
            }
        }

        return try await apiClient.post("/v1/verification/otp/send", body: Request(
            sessionID: sessionID,
            channel: channel.rawValue,
            locale: locale
        ))
    }

    /// Verifies an OTP code entered by the user.
    public func checkOTP(sessionID: String, code: String) async throws -> OTPCheckResponse {
        struct Request: Encodable {
            let sessionID: String
            let code: String

            enum CodingKeys: String, CodingKey {
                case sessionID = "session_id"
                case code
            }
        }

        return try await apiClient.post("/v1/verification/otp/check", body: Request(
            sessionID: sessionID,
            code: code
        ))
    }

    // MARK: - Low-Level API

    /// Creates a verification session.
    public func createSession(
        phoneNumber: String,
        countryCode: String,
        verificationType: VerificationType = .silentOrOTP,
        useCase: UseCase = .signup,
        context: DeviceContext? = nil
    ) async throws -> SessionResponse {
        let ctx = context ?? DeviceContextCollector.collect()

        let request = CreateSessionRequest(
            appID: config.appID,
            phoneNumber: phoneNumber,
            countryCode: countryCode,
            verificationType: verificationType.rawValue,
            useCase: useCase.rawValue,
            deviceContext: DeviceContextPayload(
                ipAddress: ctx.ipAddress,
                userAgent: ctx.userAgent
            ),
            callbackURL: nil
        )

        return try await apiClient.post("/v1/verification/session", body: request)
    }

    /// Executes silent verification for an existing session.
    public func silentVerify(sessionID: String) async throws -> SilentVerifyResponse {
        struct Request: Encodable {
            let sessionID: String
            enum CodingKeys: String, CodingKey {
                case sessionID = "session_id"
            }
        }

        return try await apiClient.post("/v1/verification/silent", body: Request(sessionID: sessionID))
    }
}

// MARK: - Result Type

/// Result of a verification attempt.
public enum VerificationResult {
    /// Phone number verified successfully. Contains the verification token.
    case verified(token: String)
    /// Silent verification failed, OTP is required. Contains the session ID for OTP flow.
    case otpRequired(sessionID: String)
    /// Verification denied by risk check.
    case denied(reason: String)
}
