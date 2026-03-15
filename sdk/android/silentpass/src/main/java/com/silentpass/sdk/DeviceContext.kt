package com.silentpass.sdk

import android.content.Context
import android.net.ConnectivityManager
import android.net.NetworkCapabilities
import android.os.Build
import android.telephony.SubscriptionManager
import android.telephony.TelephonyManager

/**
 * Collects device and network context for verification requests.
 */
data class DeviceContext(
    val userAgent: String,
    val networkType: NetworkType,
    val carrierName: String?,
    val mobileCountryCode: String?,
    val mobileNetworkCode: String?,
    val isCellularAvailable: Boolean,
    val isDualSim: Boolean,
    val activeSimCount: Int,
    val deviceModel: String,
    val osVersion: String,
) {
    enum class NetworkType { CELLULAR, WIFI, UNKNOWN }
}

/**
 * Gathers device and network context from the Android device.
 */
object DeviceContextCollector {

    fun collect(context: Context): DeviceContext {
        val telephonyManager = context.getSystemService(Context.TELEPHONY_SERVICE) as? TelephonyManager
        val connectivityManager = context.getSystemService(Context.CONNECTIVITY_SERVICE) as? ConnectivityManager

        val networkType = detectNetworkType(connectivityManager)
        val simInfo = detectSimInfo(context, telephonyManager)

        val deviceModel = "${Build.MANUFACTURER} ${Build.MODEL}"
        val osVersion = "Android ${Build.VERSION.RELEASE} (API ${Build.VERSION.SDK_INT})"
        val userAgent = "SilentPass-Android/1.0 ($deviceModel; $osVersion)"

        return DeviceContext(
            userAgent = userAgent,
            networkType = networkType,
            carrierName = telephonyManager?.networkOperatorName,
            mobileCountryCode = telephonyManager?.networkOperator?.take(3),
            mobileNetworkCode = telephonyManager?.networkOperator?.drop(3),
            isCellularAvailable = networkType == DeviceContext.NetworkType.CELLULAR || simInfo.hasActiveSim,
            isDualSim = simInfo.isDualSim,
            activeSimCount = simInfo.activeSimCount,
            deviceModel = deviceModel,
            osVersion = osVersion,
        )
    }

    private fun detectNetworkType(cm: ConnectivityManager?): DeviceContext.NetworkType {
        cm ?: return DeviceContext.NetworkType.UNKNOWN
        val network = cm.activeNetwork ?: return DeviceContext.NetworkType.UNKNOWN
        val caps = cm.getNetworkCapabilities(network) ?: return DeviceContext.NetworkType.UNKNOWN

        return when {
            caps.hasTransport(NetworkCapabilities.TRANSPORT_CELLULAR) -> DeviceContext.NetworkType.CELLULAR
            caps.hasTransport(NetworkCapabilities.TRANSPORT_WIFI) -> DeviceContext.NetworkType.WIFI
            else -> DeviceContext.NetworkType.UNKNOWN
        }
    }

    private data class SimInfo(val isDualSim: Boolean, val activeSimCount: Int, val hasActiveSim: Boolean)

    private fun detectSimInfo(context: Context, tm: TelephonyManager?): SimInfo {
        if (tm == null) return SimInfo(false, 0, false)

        return try {
            if (Build.VERSION.SDK_INT >= Build.VERSION_CODES.LOLLIPOP_MR1) {
                val sm = context.getSystemService(Context.TELEPHONY_SUBSCRIPTION_SERVICE) as? SubscriptionManager
                val count = sm?.activeSubscriptionInfoCount ?: 0
                SimInfo(isDualSim = count > 1, activeSimCount = count, hasActiveSim = count > 0)
            } else {
                val hasSim = tm.simState == TelephonyManager.SIM_STATE_READY
                SimInfo(isDualSim = false, activeSimCount = if (hasSim) 1 else 0, hasActiveSim = hasSim)
            }
        } catch (e: SecurityException) {
            // Permission not granted
            SimInfo(isDualSim = false, activeSimCount = 0, hasActiveSim = false)
        }
    }
}
