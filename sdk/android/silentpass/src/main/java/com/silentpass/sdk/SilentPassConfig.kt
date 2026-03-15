package com.silentpass.sdk

/**
 * Configuration for the SilentPass SDK.
 *
 * @param baseURL API base URL
 * @param apiKey API key for authentication
 * @param appID Application identifier
 * @param timeoutMs Request timeout in milliseconds
 * @param sandbox Enable sandbox mode for development
 */
data class SilentPassConfig(
    val baseURL: String = "https://api.silentpass.io",
    val apiKey: String,
    val appID: String,
    val timeoutMs: Long = 15_000,
    val sandbox: Boolean = false,
)
