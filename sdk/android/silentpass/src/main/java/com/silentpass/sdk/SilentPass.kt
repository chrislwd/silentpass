package com.silentpass.sdk

import android.content.Context

/**
 * SilentPass Android SDK - Main entry point.
 *
 * Usage:
 * ```kotlin
 * val silentPass = SilentPass(
 *     context = applicationContext,
 *     config = SilentPassConfig(
 *         apiKey = "sk_your_api_key",
 *         appID = "your_app_id"
 *     )
 * )
 *
 * when (val result = silentPass.verify("+6281234567890", "ID", UseCase.SIGNUP)) {
 *     is VerificationResult.Verified -> {
 *         // Phone verified, send result.token to your backend
 *     }
 *     is VerificationResult.OTPRequired -> {
 *         // Show OTP input, then call silentPass.checkOTP(result.sessionID, code)
 *     }
 *     is VerificationResult.Denied -> {
 *         // Blocked by risk check
 *     }
 *     is VerificationResult.Error -> {
 *         // Handle error
 *     }
 * }
 * ```
 */
class SilentPass(
    private val context: Context,
    private val config: SilentPassConfig,
) {
    private val apiClient = APIClient(config)

    // MARK: - High-Level API

    /**
     * Performs the full verification flow: silent verify → auto-fallback to OTP if needed.
     * This is the recommended entry point for most use cases.
     */
    suspend fun verify(
        phoneNumber: String,
        countryCode: String,
        useCase: UseCase = UseCase.SIGNUP,
    ): VerificationResult {
        return try {
            val deviceCtx = DeviceContextCollector.collect(context)

            // Step 1: Create session
            val session = createSession(
                phoneNumber = phoneNumber,
                countryCode = countryCode,
                verificationType = VerificationType.SILENT_OR_OTP,
                useCase = useCase,
                deviceContext = deviceCtx,
            )

            // Step 2: Attempt silent verification
            if (session.recommendedAction == "silent_verify") {
                val silentResult = silentVerify(session.sessionID)

                when (silentResult.verificationStatus) {
                    VerificationStatus.VERIFIED ->
                        VerificationResult.Verified(token = silentResult.token ?: "")
                    VerificationStatus.FALLBACK_REQUIRED ->
                        VerificationResult.OTPRequired(sessionID = session.sessionID)
                    VerificationStatus.FAILED ->
                        VerificationResult.OTPRequired(sessionID = session.sessionID)
                }
            } else {
                // Silent not available
                VerificationResult.OTPRequired(sessionID = session.sessionID)
            }
        } catch (e: SilentPassException) {
            VerificationResult.Error(e)
        }
    }

    /**
     * Sends an OTP to the user via the specified channel.
     */
    suspend fun sendOTP(
        sessionID: String,
        channel: OTPChannel = OTPChannel.SMS,
        locale: String? = null,
    ): OTPSendResponse {
        return apiClient.post(
            "/v1/verification/otp/send",
            OTPSendRequest(sessionID = sessionID, channel = channel.value, locale = locale),
        )
    }

    /**
     * Verifies an OTP code entered by the user.
     */
    suspend fun checkOTP(sessionID: String, code: String): OTPCheckResponse {
        return apiClient.post(
            "/v1/verification/otp/check",
            OTPCheckRequest(sessionID = sessionID, code = code),
        )
    }

    // MARK: - Low-Level API

    /**
     * Creates a verification session.
     */
    suspend fun createSession(
        phoneNumber: String,
        countryCode: String,
        verificationType: VerificationType = VerificationType.SILENT_OR_OTP,
        useCase: UseCase = UseCase.SIGNUP,
        deviceContext: DeviceContext? = null,
    ): SessionResponse {
        val ctx = deviceContext ?: DeviceContextCollector.collect(context)

        return apiClient.post(
            "/v1/verification/session",
            CreateSessionRequest(
                appID = config.appID,
                phoneNumber = phoneNumber,
                countryCode = countryCode,
                verificationType = verificationType.value,
                useCase = useCase.value,
                deviceContext = DeviceContextPayload(userAgent = ctx.userAgent),
            ),
        )
    }

    /**
     * Executes silent verification for an existing session.
     */
    suspend fun silentVerify(sessionID: String): SilentVerifyResponse {
        return apiClient.post(
            "/v1/verification/silent",
            SilentVerifyRequest(sessionID = sessionID),
        )
    }
}
