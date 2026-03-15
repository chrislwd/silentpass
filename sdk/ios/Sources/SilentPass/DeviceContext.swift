import Foundation
#if canImport(CoreTelephony)
import CoreTelephony
#endif
#if canImport(UIKit)
import UIKit
#endif

/// Collects device and network context for verification requests.
public struct DeviceContext {
    public let ipAddress: String?
    public let userAgent: String
    public let networkType: NetworkType
    public let carrierName: String?
    public let mobileCountryCode: String?
    public let mobileNetworkCode: String?
    public let isCellularAvailable: Bool
    public let deviceModel: String
    public let osVersion: String

    public enum NetworkType: String {
        case cellular
        case wifi
        case unknown
    }
}

/// Gathers device context from the current device.
public final class DeviceContextCollector {

    public static func collect() -> DeviceContext {
        let networkInfo = collectNetworkInfo()

        var deviceModel = "Unknown"
        var osVersion = "Unknown"
        #if canImport(UIKit)
        deviceModel = UIDevice.current.model
        osVersion = UIDevice.current.systemVersion
        #endif

        let userAgent = "SilentPass-iOS/1.0 (\(deviceModel); iOS \(osVersion))"

        return DeviceContext(
            ipAddress: nil, // Resolved server-side from request IP
            userAgent: userAgent,
            networkType: networkInfo.networkType,
            carrierName: networkInfo.carrierName,
            mobileCountryCode: networkInfo.mcc,
            mobileNetworkCode: networkInfo.mnc,
            isCellularAvailable: networkInfo.cellularAvailable,
            deviceModel: deviceModel,
            osVersion: osVersion
        )
    }

    private struct NetworkInfo {
        var networkType: DeviceContext.NetworkType = .unknown
        var carrierName: String?
        var mcc: String?
        var mnc: String?
        var cellularAvailable: Bool = false
    }

    private static func collectNetworkInfo() -> NetworkInfo {
        var info = NetworkInfo()

        #if canImport(CoreTelephony) && !targetEnvironment(simulator)
        let networkInfo = CTTelephonyNetworkInfo()

        if let carriers = networkInfo.serviceSubscriberCellularProviders {
            if let primary = carriers.values.first {
                info.carrierName = primary.carrierName
                info.mcc = primary.mobileCountryCode
                info.mnc = primary.mobileNetworkCode
                info.cellularAvailable = true
            }
        }

        if let radioTech = networkInfo.serviceCurrentRadioAccessTechnology?.values.first {
            info.cellularAvailable = true
            _ = radioTech // Available for detailed logging
            info.networkType = .cellular
        }
        #endif

        return info
    }
}
