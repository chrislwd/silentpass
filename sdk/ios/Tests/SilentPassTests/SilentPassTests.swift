import XCTest
@testable import SilentPass

final class SilentPassTests: XCTestCase {

    func testConfigDefaults() {
        let config = SilentPassConfig(apiKey: "sk_test", appID: "app1")
        XCTAssertEqual(config.baseURL, "https://api.silentpass.io")
        XCTAssertEqual(config.timeout, 15)
        XCTAssertFalse(config.sandbox)
    }

    func testConfigCustom() {
        let config = SilentPassConfig(
            baseURL: "http://localhost:8080",
            apiKey: "sk_test_key",
            appID: "my_app",
            timeout: 30,
            sandbox: true
        )
        XCTAssertEqual(config.baseURL, "http://localhost:8080")
        XCTAssertEqual(config.apiKey, "sk_test_key")
        XCTAssertTrue(config.sandbox)
    }

    func testVerificationTypes() {
        XCTAssertEqual(VerificationType.silent.rawValue, "silent")
        XCTAssertEqual(VerificationType.silentOrOTP.rawValue, "silent_or_otp")
        XCTAssertEqual(VerificationType.otpOnly.rawValue, "otp_only")
    }

    func testUseCases() {
        XCTAssertEqual(UseCase.signup.rawValue, "signup")
        XCTAssertEqual(UseCase.login.rawValue, "login")
        XCTAssertEqual(UseCase.transaction.rawValue, "transaction")
        XCTAssertEqual(UseCase.phoneChange.rawValue, "phone_change")
    }

    func testOTPChannels() {
        XCTAssertEqual(OTPChannel.sms.rawValue, "sms")
        XCTAssertEqual(OTPChannel.whatsapp.rawValue, "whatsapp")
        XCTAssertEqual(OTPChannel.voice.rawValue, "voice")
    }

    func testSessionResponseDecoding() throws {
        let json = """
        {"session_id":"abc-123","recommended_action":"silent_verify","expires_at":"2026-03-15T10:00:00Z"}
        """
        let data = json.data(using: .utf8)!
        let response = try JSONDecoder().decode(SessionResponse.self, from: data)
        XCTAssertEqual(response.sessionID, "abc-123")
        XCTAssertEqual(response.recommendedAction, "silent_verify")
    }

    func testSilentVerifyResponseDecoding() throws {
        let json = """
        {"status":"verified","confidence_score":0.98,"telco_signal":"match","token":"sv_token_123"}
        """
        let data = json.data(using: .utf8)!
        let response = try JSONDecoder().decode(SilentVerifyResponse.self, from: data)
        XCTAssertEqual(response.status, .verified)
        XCTAssertEqual(response.confidenceScore, 0.98)
        XCTAssertEqual(response.token, "sv_token_123")
    }

    func testFallbackResponseDecoding() throws {
        let json = """
        {"status":"fallback_required","telco_signal":"timeout"}
        """
        let data = json.data(using: .utf8)!
        let response = try JSONDecoder().decode(SilentVerifyResponse.self, from: data)
        XCTAssertEqual(response.status, .fallbackRequired)
        XCTAssertNil(response.token)
    }

    func testOTPCheckResponseDecoding() throws {
        let json = """
        {"status":"verified","token":"otp_token_456","attempts_left":0}
        """
        let data = json.data(using: .utf8)!
        let response = try JSONDecoder().decode(OTPCheckResponse.self, from: data)
        XCTAssertEqual(response.status, .verified)
        XCTAssertEqual(response.token, "otp_token_456")
        XCTAssertEqual(response.attemptsLeft, 0)
    }

    func testErrorDescriptions() {
        let errors: [SilentPassError] = [
            .invalidConfig("bad url"),
            .apiError(401, "unauthorized"),
            .sessionExpired,
            .cellularUnavailable,
        ]
        for error in errors {
            XCTAssertNotNil(error.errorDescription)
        }
    }

    func testDeviceContextCollector() {
        let context = DeviceContextCollector.collect()
        XCTAssertFalse(context.userAgent.isEmpty)
        XCTAssertFalse(context.deviceModel.isEmpty)
    }
}
