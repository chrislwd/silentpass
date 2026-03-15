package com.silentpass.sdk

import kotlinx.serialization.json.Json
import org.junit.Assert.*
import org.junit.Test

class ModelsTest {

    private val json = Json { ignoreUnknownKeys = true }

    @Test
    fun `test config defaults`() {
        val config = SilentPassConfig(apiKey = "sk_test", appID = "app1")
        assertEquals("https://api.silentpass.io", config.baseURL)
        assertEquals(15_000L, config.timeoutMs)
        assertFalse(config.sandbox)
    }

    @Test
    fun `test verification types`() {
        assertEquals("silent", VerificationType.SILENT.value)
        assertEquals("silent_or_otp", VerificationType.SILENT_OR_OTP.value)
        assertEquals("otp_only", VerificationType.OTP_ONLY.value)
    }

    @Test
    fun `test use cases`() {
        assertEquals("signup", UseCase.SIGNUP.value)
        assertEquals("login", UseCase.LOGIN.value)
        assertEquals("transaction", UseCase.TRANSACTION.value)
        assertEquals("phone_change", UseCase.PHONE_CHANGE.value)
    }

    @Test
    fun `test session response decoding`() {
        val response = json.decodeFromString<SessionResponse>("""
            {"session_id":"abc-123","recommended_action":"silent_verify","expires_at":"2026-03-15T10:00:00Z"}
        """.trimIndent())
        assertEquals("abc-123", response.sessionID)
        assertEquals("silent_verify", response.recommendedAction)
    }

    @Test
    fun `test silent verify response - verified`() {
        val response = json.decodeFromString<SilentVerifyResponse>("""
            {"status":"verified","confidence_score":0.98,"telco_signal":"match","token":"sv_123"}
        """.trimIndent())
        assertEquals(VerificationStatus.VERIFIED, response.verificationStatus)
        assertEquals(0.98, response.confidenceScore!!, 0.01)
        assertEquals("sv_123", response.token)
    }

    @Test
    fun `test silent verify response - fallback`() {
        val response = json.decodeFromString<SilentVerifyResponse>("""
            {"status":"fallback_required","telco_signal":"timeout"}
        """.trimIndent())
        assertEquals(VerificationStatus.FALLBACK_REQUIRED, response.verificationStatus)
        assertNull(response.token)
    }

    @Test
    fun `test OTP check response`() {
        val response = json.decodeFromString<OTPCheckResponse>("""
            {"status":"verified","token":"otp_456","attempts_left":0}
        """.trimIndent())
        assertEquals(VerificationStatus.VERIFIED, response.verificationStatus)
        assertEquals("otp_456", response.token)
        assertEquals(0, response.attemptsLeft)
    }

    @Test
    fun `test OTP check response - failed`() {
        val response = json.decodeFromString<OTPCheckResponse>("""
            {"status":"failed","attempts_left":2}
        """.trimIndent())
        assertEquals(VerificationStatus.FAILED, response.verificationStatus)
        assertNull(response.token)
        assertEquals(2, response.attemptsLeft)
    }

    @Test
    fun `test verification status from string`() {
        assertEquals(VerificationStatus.VERIFIED, VerificationStatus.from("verified"))
        assertEquals(VerificationStatus.FALLBACK_REQUIRED, VerificationStatus.from("fallback_required"))
        assertEquals(VerificationStatus.FAILED, VerificationStatus.from("failed"))
        assertEquals(VerificationStatus.FAILED, VerificationStatus.from("unknown_value"))
    }

    @Test
    fun `test create session request serialization`() {
        val request = CreateSessionRequest(
            appID = "app1",
            phoneNumber = "+628123",
            countryCode = "ID",
            verificationType = "silent_or_otp",
            useCase = "signup",
            deviceContext = DeviceContextPayload(userAgent = "test-agent"),
        )
        val serialized = json.encodeToString(CreateSessionRequest.serializer(), request)
        assertTrue(serialized.contains("\"app_id\":\"app1\""))
        assertTrue(serialized.contains("\"phone_number\":\"+628123\""))
        assertTrue(serialized.contains("\"user_agent\":\"test-agent\""))
    }

    @Test
    fun `test exception messages`() {
        val errors = listOf(
            SilentPassException.InvalidConfig("bad url"),
            SilentPassException.APIError(401, "unauthorized"),
            SilentPassException.CellularUnavailable(),
        )
        for (error in errors) {
            assertNotNull(error.message)
        }
    }

    @Test
    fun `test verification result types`() {
        val verified = VerificationResult.Verified("token123")
        assertTrue(verified is VerificationResult.Verified)
        assertEquals("token123", verified.token)

        val otpRequired = VerificationResult.OTPRequired("session-id")
        assertTrue(otpRequired is VerificationResult.OTPRequired)

        val denied = VerificationResult.Denied("sim_swap")
        assertTrue(denied is VerificationResult.Denied)
    }
}
