import Foundation

// MARK: - Request Models

public struct CreateSessionRequest: Encodable {
    let appID: String
    let phoneNumber: String
    let countryCode: String
    let verificationType: String
    let useCase: String
    let deviceContext: DeviceContextPayload?
    let callbackURL: String?

    enum CodingKeys: String, CodingKey {
        case appID = "app_id"
        case phoneNumber = "phone_number"
        case countryCode = "country_code"
        case verificationType = "verification_type"
        case useCase = "use_case"
        case deviceContext = "device_context"
        case callbackURL = "callback_url"
    }
}

struct DeviceContextPayload: Encodable {
    let ipAddress: String?
    let userAgent: String

    enum CodingKeys: String, CodingKey {
        case ipAddress = "ip_address"
        case userAgent = "user_agent"
    }
}

// MARK: - Response Models

public struct SessionResponse: Decodable {
    public let sessionID: String
    public let recommendedAction: String
    public let expiresAt: String

    enum CodingKeys: String, CodingKey {
        case sessionID = "session_id"
        case recommendedAction = "recommended_action"
        case expiresAt = "expires_at"
    }
}

public struct SilentVerifyResponse: Decodable {
    public let status: VerificationStatus
    public let confidenceScore: Double?
    public let telcoSignal: String?
    public let token: String?

    enum CodingKeys: String, CodingKey {
        case status
        case confidenceScore = "confidence_score"
        case telcoSignal = "telco_signal"
        case token
    }
}

public struct OTPSendResponse: Decodable {
    public let deliveryStatus: String
    public let resendAfterSeconds: Int

    enum CodingKeys: String, CodingKey {
        case deliveryStatus = "delivery_status"
        case resendAfterSeconds = "resend_after_seconds"
    }
}

public struct OTPCheckResponse: Decodable {
    public let status: VerificationStatus
    public let token: String?
    public let attemptsLeft: Int

    enum CodingKeys: String, CodingKey {
        case status
        case token
        case attemptsLeft = "attempts_left"
    }
}

// MARK: - Enums

public enum VerificationStatus: String, Decodable {
    case verified
    case fallbackRequired = "fallback_required"
    case failed
}

public enum VerificationType: String {
    case silent
    case silentOrOTP = "silent_or_otp"
    case otpOnly = "otp_only"
}

public enum UseCase: String {
    case signup
    case login
    case transaction
    case phoneChange = "phone_change"
}

public enum OTPChannel: String {
    case sms
    case whatsapp
    case voice
}

// MARK: - Errors

public enum SilentPassError: Error, LocalizedError {
    case invalidConfig(String)
    case networkError(Error)
    case apiError(Int, String)
    case decodingError(Error)
    case sessionExpired
    case cellularUnavailable

    public var errorDescription: String? {
        switch self {
        case .invalidConfig(let msg): return "Invalid config: \(msg)"
        case .networkError(let err): return "Network error: \(err.localizedDescription)"
        case .apiError(let code, let msg): return "API error \(code): \(msg)"
        case .decodingError(let err): return "Decoding error: \(err.localizedDescription)"
        case .sessionExpired: return "Session has expired"
        case .cellularUnavailable: return "Cellular network is not available"
        }
    }
}
