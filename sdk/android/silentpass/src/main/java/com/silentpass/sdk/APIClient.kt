package com.silentpass.sdk

import kotlinx.coroutines.Dispatchers
import kotlinx.coroutines.withContext
import kotlinx.serialization.json.Json
import kotlinx.serialization.encodeToString
import okhttp3.MediaType.Companion.toMediaType
import okhttp3.OkHttpClient
import okhttp3.Request
import okhttp3.RequestBody.Companion.toRequestBody
import java.util.concurrent.TimeUnit

/**
 * HTTP client for SilentPass API calls.
 * Uses OkHttp and forces cellular network when available for silent verification.
 */
internal class APIClient(private val config: SilentPassConfig) {

    private val json = Json {
        ignoreUnknownKeys = true
        encodeDefaults = false
    }

    private val client = OkHttpClient.Builder()
        .connectTimeout(config.timeoutMs, TimeUnit.MILLISECONDS)
        .readTimeout(config.timeoutMs, TimeUnit.MILLISECONDS)
        .writeTimeout(config.timeoutMs, TimeUnit.MILLISECONDS)
        .build()

    private val jsonMediaType = "application/json; charset=utf-8".toMediaType()

    @Throws(SilentPassException::class)
    internal suspend inline fun <reified Req, reified Res> post(path: String, body: Req): Res {
        return withContext(Dispatchers.IO) {
            try {
                val jsonBody = json.encodeToString(body)
                val requestBody = jsonBody.toRequestBody(jsonMediaType)

                val request = Request.Builder()
                    .url("${config.baseURL}$path")
                    .post(requestBody)
                    .addHeader("Content-Type", "application/json")
                    .addHeader("X-API-Key", config.apiKey)
                    .build()

                if (config.sandbox) {
                    debugLog("POST $path")
                }

                val response = client.newCall(request).execute()
                val responseBody = response.body?.string() ?: ""

                if (config.sandbox) {
                    debugLog("Response: ${response.code}")
                }

                if (!response.isSuccessful) {
                    throw SilentPassException.APIError(response.code, responseBody)
                }

                try {
                    json.decodeFromString<Res>(responseBody)
                } catch (e: Exception) {
                    throw SilentPassException.DecodingError(e)
                }
            } catch (e: SilentPassException) {
                throw e
            } catch (e: Exception) {
                throw SilentPassException.NetworkError(e)
            }
        }
    }

    private fun debugLog(message: String) {
        if (config.sandbox) {
            android.util.Log.d("SilentPass", message)
        }
    }
}
