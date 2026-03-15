package com.silentpass.sdk

import kotlinx.serialization.SerialName
import kotlinx.serialization.Serializable

// -- Enums --

enum class VerificationType(val value: String) {
    SILENT("silent"),
    SILENT_OR_OTP("silent_or_otp"),
    OTP_ONLY("otp_only"),
}

enum class UseCase(val value: String) {
    SIGNUP("signup"),
    LOGIN("login"),
    TRANSACTION("transaction"),
    PHONE_CHANGE("phone_change"),
}

enum class OTPChannel(val value: String) {
    SMS("sms"),
    WHATSAPP("whatsapp"),
    VOICE("voice"),
}

enum class VerificationStatus(val value: String) {
    VERIFIED("verified"),
    FALLBACK_REQUIRED("fallback_required"),
    FAILED("failed");

    companion object {
        fun from(value: String): VerificationStatus =
            entries.firstOrNull { it.value == value } ?: FAILED
    }
}

// -- Request Models --

@Serializable
data class CreateSessionRequest(
    @SerialName("app_id") val appID: String,
    @SerialName("phone_number") val phoneNumber: String,
    @SerialName("country_code") val countryCode: String,
    @SerialName("verification_type") val verificationType: String,
    @SerialName("use_case") val useCase: String,
    @SerialName("device_context") val deviceContext: DeviceContextPayload? = null,
    @SerialName("callback_url") val callbackURL: String? = null,
)

@Serializable
data class DeviceContextPayload(
    @SerialName("ip_address") val ipAddress: String? = null,
    @SerialName("user_agent") val userAgent: String,
)

@Serializable
data class SilentVerifyRequest(
    @SerialName("session_id") val sessionID: String,
)

@Serializable
data class OTPSendRequest(
    @SerialName("session_id") val sessionID: String,
    val channel: String,
    val locale: String? = null,
)

@Serializable
data class OTPCheckRequest(
    @SerialName("session_id") val sessionID: String,
    val code: String,
)

// -- Response Models --

@Serializable
data class SessionResponse(
    @SerialName("session_id") val sessionID: String,
    @SerialName("recommended_action") val recommendedAction: String,
    @SerialName("expires_at") val expiresAt: String,
)

@Serializable
data class SilentVerifyResponse(
    val status: String,
    @SerialName("confidence_score") val confidenceScore: Double? = null,
    @SerialName("telco_signal") val telcoSignal: String? = null,
    val token: String? = null,
) {
    val verificationStatus: VerificationStatus
        get() = VerificationStatus.from(status)
}

@Serializable
data class OTPSendResponse(
    @SerialName("delivery_status") val deliveryStatus: String,
    @SerialName("resend_after_seconds") val resendAfterSeconds: Int,
)

@Serializable
data class OTPCheckResponse(
    val status: String,
    val token: String? = null,
    @SerialName("attempts_left") val attemptsLeft: Int,
) {
    val verificationStatus: VerificationStatus
        get() = VerificationStatus.from(status)
}

// -- Result --

sealed class VerificationResult {
    /** Phone verified successfully. */
    data class Verified(val token: String) : VerificationResult()
    /** Silent verification failed, OTP required. */
    data class OTPRequired(val sessionID: String) : VerificationResult()
    /** Blocked by risk check. */
    data class Denied(val reason: String) : VerificationResult()
    /** Error occurred. */
    data class Error(val exception: SilentPassException) : VerificationResult()
}

// -- Exceptions --

sealed class SilentPassException(message: String, cause: Throwable? = null) : Exception(message, cause) {
    class InvalidConfig(msg: String) : SilentPassException("Invalid config: $msg")
    class NetworkError(cause: Throwable) : SilentPassException("Network error: ${cause.message}", cause)
    class APIError(val code: Int, val body: String) : SilentPassException("API error $code: $body")
    class DecodingError(cause: Throwable) : SilentPassException("Decoding error: ${cause.message}", cause)
    class CellularUnavailable : SilentPassException("Cellular network is not available")
}
